package component

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rpsingh21/torrent-cli/internal/app"
	"github.com/rpsingh21/torrent-cli/internal/peer"
)

type tickMsg time.Time
type statUpdateMsg map[string]peer.ManagerStatSnapshot

type cmdResultMsg struct {
	status string
	err    error
}

type managerEntry struct {
	id   string
	snap peer.ManagerStatSnapshot
}

type peerEntry struct {
	addr string
	stat peer.StatSnapshot
}

type model struct {
	app       *app.App
	expandID  string // Manager ID or prefix expanded via /peers <id>
	stats     map[string]peer.ManagerStatSnapshot
	sorted    []managerEntry // Reused slice for deterministic torrent row order
	peerBuf   []peerEntry    // Reused slice for deterministic peer row order
	textInput textinput.Model
	viewport  viewport.Model
	statusMsg string
	isErr     bool
	ready     bool
	width     int
	height    int
}

func InitialModel(app *app.App) model {
	ti := textinput.New()
	ti.Placeholder = "/add <url|file> | /peers <id> | /pause <id> | /resume <id> | /remove <id> | /exit"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.CharLimit = 512

	return model{
		app:       app,
		textInput: ti,
		sorted:    make([]managerEntry, 0, 16),
		peerBuf:   make([]peerEntry, 0, 64),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchStatsCmd(m.app), tickEverySecondCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(fetchStatsCmd(m.app), tickEverySecondCmd())

	case statUpdateMsg:
		m.stats = msg
		m.updateViewportContent()
		return m, nil

	case cmdResultMsg:
		if msg.err != nil {
			m.statusMsg = "✗ " + msg.err.Error()
			m.isErr = true
		} else {
			m.statusMsg = msg.status
			m.isErr = false
		}
		return m, fetchStatsCmd(m.app)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			m.textInput.SetValue("")
			if input == "" {
				return m, nil
			}
			return m, m.handleCommand(input)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		viewportHeight := max(1, msg.Height-headerHeight-footerHeight)

		if !m.ready {
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}

		m.textInput.Width = max(20, msg.Width-6)
		m.updateViewportContent()
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleCommand parses slash commands and returns non-blocking tea.Cmds.
func (m *model) handleCommand(input string) tea.Cmd {
	cmdName, arg, _ := strings.Cut(input, " ")
	arg = strings.TrimSpace(arg)

	switch cmdName {
	case "/exit", "/quit":
		return tea.Quit

	case "/add":
		if arg == "" {
			m.setStatus("Usage: /add <magnet-uri | .torrent>", true)
			return nil
		}
		m.setStatus("⏳ Adding torrent...", false)
		return addTorrentCmd(m.app, arg)

	case "/remove", "/rm":
		if arg == "" {
			m.setStatus("Usage: /remove <id>", true)
			return nil
		}
		if m.expandID != "" && (m.expandID == arg || strings.HasPrefix(m.expandID, arg)) {
			m.expandID = ""
		}
		return removeTorrentCmd(m.app, arg)

	case "/pause":
		if arg == "" {
			m.setStatus("Usage: /pause <id>", true)
			return nil
		}
		return pauseTorrentCmd(m.app, arg)

	case "/resume":
		if arg == "" {
			m.setStatus("Usage: /resume <id>", true)
			return nil
		}
		return resumeTorrentCmd(m.app, arg)

	case "/peers":
		if arg == "" || m.expandID == arg {
			m.expandID = ""
			m.setStatus("Collapsed peers view", false)
		} else {
			m.expandID = arg
			m.setStatus("Expanded peers for "+arg, false)
		}
		m.updateViewportContent()
		return nil

	default:
		m.setStatus("Unknown command: "+cmdName, true)
		return nil
	}
}

func (m *model) setStatus(msg string, isErr bool) {
	m.statusMsg = msg
	m.isErr = isErr
}

func (m *model) updateViewportContent() {
	if !m.ready {
		return
	}

	if len(m.stats) == 0 {
		m.viewport.SetContent(emptyStateStyle.Render(
			"\n  No active downloads. Type /add <magnet-uri | .torrent> below.\n",
		))
		return
	}

	// Reuse slice capacity
	m.sorted = m.sorted[:0]
	for id, snap := range m.stats {
		m.sorted = append(m.sorted, managerEntry{id: id, snap: snap})
	}

	// Sort deterministically by StartAt, then by ID
	slices.SortFunc(m.sorted, func(a, b managerEntry) int {
		if c := a.snap.StartAt.Compare(b.snap.StartAt); c != 0 {
			return c
		}
		return cmp.Compare(a.id, b.id)
	})

	// Dynamically size NAME column so UP and ↑ SPEED never clip on 80-col terminals
	// Fixed columns take ~72 chars total: ID(8) + DOWN(10) + ↓SPEED(11) + UP(10) + ↑SPEED(11) + PIECES(13) + spaces
	nameWidth := min(30, max(12, m.width-74))

	var b strings.Builder
	b.Grow(len(m.sorted) * 256)

	headerLine := fmt.Sprintf("  %-8s %-*s %-10s %-11s %-10s %-11s %-13s",
		"ID", nameWidth, "NAME", "DOWN", "↓ SPEED", "UP", "↑ SPEED", "DONE / PROG",
	)
	b.WriteString(tblHeaderStyle.Render(headerLine))
	b.WriteByte('\n')

	for _, entry := range m.sorted {
		snap := entry.snap

		// Display 8-char short ID for readability while matching full or prefix ID
		shortID := entry.id
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		name := shortID
		if snap.MetaInfo != nil && snap.MetaInfo.Name != "" {
			name = snap.MetaInfo.Name
		}

		isExpanded := m.expandID != "" && (entry.id == m.expandID || strings.HasPrefix(entry.id, m.expandID))
		icon := "▸ "
		idCol := rowIDStyle.Render(padRight(shortID, 8))
		if isExpanded {
			icon = "▾ "
			idCol = rowSelectedStyle.Render(padRight(shortID, 8))
		}

		piecesStr := fmt.Sprintf("%d / %d", snap.Completed, snap.Inprogress)

		b.WriteString(icon)
		b.WriteString(idCol)
		b.WriteByte(' ')
		b.WriteString(rowNameStyle.Render(padRight(name, nameWidth)))
		b.WriteByte(' ')
		b.WriteString(metricStyle.Render(padRight(formatBytes(int64(snap.Downloaded)), 10)))
		b.WriteByte(' ')
		b.WriteString(downSpeedStyle.Render(padRight(formatSpeed(snap.DownloadRate), 11)))
		b.WriteByte(' ')
		b.WriteString(metricStyle.Render(padRight(formatBytes(int64(snap.Uploaded)), 10)))
		b.WriteByte(' ')
		b.WriteString(upSpeedStyle.Render(padRight(formatSpeed(snap.UploadRate), 11)))
		b.WriteByte(' ')
		b.WriteString(progressStyle.Render(padRight(piecesStr, 13)))
		b.WriteByte('\n')

		if isExpanded {
			m.renderChildPeerTable(&b, snap.PeerStat)
		}
	}

	m.viewport.SetContent(b.String())
}

func (m *model) renderChildPeerTable(b *strings.Builder, peers map[string]peer.StatSnapshot) {
	if len(peers) == 0 {
		b.WriteString(peerTreeStyle.Render("  ╰─ "))
		b.WriteString(emptyStateStyle.Render("No active peers connected"))
		b.WriteString("\n\n")
		return
	}

	m.peerBuf = m.peerBuf[:0]
	maxAddrLen := 21
	for addr, st := range peers {
		m.peerBuf = append(m.peerBuf, peerEntry{addr: addr, stat: st})
		if len(addr) > maxAddrLen {
			maxAddrLen = len(addr)
		}
	}

	slices.SortFunc(m.peerBuf, func(a, b peerEntry) int {
		if c := cmp.Compare(b.stat.Downloaded, a.stat.Downloaded); c != 0 {
			return c
		}

		if c := cmp.Compare(b.stat.Uploaded, a.stat.Uploaded); c != 0 {
			return c
		}

		return cmp.Compare(a.addr, b.addr)
	})

	// Other columns + tree prefix ("  ├─ ") take 74 chars total.
	// Expand PEER ADDRESS up to maxAddrLen (max 47 for full IPv6:port) if terminal is wide enough.
	availAddrWidth := max(21, m.width-74)
	addrWidth := min(maxAddrLen, min(47, availAddrWidth))

	subHeader := fmt.Sprintf("%-*s %-10s %-11s %-10s %-11s %-13s %-8s",
		addrWidth, "PEER ADDRESS", "DOWN", "↓ SPEED", "UP", "↑ SPEED", "REQ (OK/SENT)", "ERR/TO",
	)
	b.WriteString(peerTreeStyle.Render("  ├─ "))
	b.WriteString(peerHeaderStyle.Render(subHeader))
	b.WriteByte('\n')

	for i, p := range m.peerBuf {
		branch := "  ├─ "
		if i == len(m.peerBuf)-1 {
			branch = "  ╰─ "
		}

		reqInfo := fmt.Sprintf("%d/%d", p.stat.RequestsCompleted, p.stat.RequestsSent)
		errInfo := fmt.Sprintf("%d/%d", p.stat.Errors, p.stat.Timeouts)

		b.WriteString(peerTreeStyle.Render(branch))
		b.WriteString(peerRowStyle.Render(formatPeerAddr(p.addr, addrWidth)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(formatBytes(p.stat.Downloaded), 10)))
		b.WriteByte(' ')
		b.WriteString(downSpeedStyle.Render(padRight(formatSpeed(p.stat.DownloadRate), 11)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(formatBytes(p.stat.Uploaded), 10)))
		b.WriteByte(' ')
		b.WriteString(upSpeedStyle.Render(padRight(formatSpeed(p.stat.UploadRate), 11)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(reqInfo, 13)))
		b.WriteByte(' ')
		if p.stat.Errors > 0 || p.stat.Timeouts > 0 {
			b.WriteString(peerErrStyle.Render(padRight(errInfo, 8)))
		} else {
			b.WriteString(peerRowStyle.Render(padRight(errInfo, 8)))
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func (m model) headerView() string {
	var totalDown, totalUp int64
	for _, s := range m.stats {
		totalDown += s.DownloadRate
		totalUp += s.UploadRate
	}

	left := "⚡ TORRENT-CLI"
	right := fmt.Sprintf("ACTIVE: %d   ↓ %s   ↑ %s",
		len(m.stats), formatSpeed(totalDown), formatSpeed(totalUp),
	)

	space := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)-4)
	return headerStyle.Width(m.width).Render(left + strings.Repeat(" ", space) + right)
}

func (m model) footerView() string {
	helpLine := helpStyle.Render("Commands: /add • /peers <id> • /pause <id> • /resume <id> • /remove <id> • /exit")
	if m.statusMsg != "" {
		statusRendered := downSpeedStyle.Render(m.statusMsg)
		if m.isErr {
			statusRendered = peerErrStyle.Render(m.statusMsg)
		}
		helpLine = "  " + statusRendered + "  " + helpLine
	}
	return footerStyle.Width(m.width).Render(
		m.textInput.View() + "\n" + helpLine,
	)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing full-screen view..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

// --- Background Tea Commands ---

func fetchStatsCmd(a *app.App) tea.Cmd {
	return func() tea.Msg {
		return statUpdateMsg(a.GetUpdatedManagerStat())
	}
}

func tickEverySecondCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func addTorrentCmd(a *app.App, arg string) tea.Cmd {
	return func() tea.Msg {
		shortID, name, err := a.Add(arg)
		if err != nil {
			return cmdResultMsg{err: err}
		}
		return cmdResultMsg{status: fmt.Sprintf("✓ Added %s (%s)", name, shortID)}
	}
}

func removeTorrentCmd(a *app.App, id string) tea.Cmd {
	return func() tea.Msg {
		shortID, err := a.Remove(id)
		if err != nil {
			return cmdResultMsg{err: err}
		}
		return cmdResultMsg{status: fmt.Sprintf("✓ Removed %s", shortID)}
	}
}

func pauseTorrentCmd(a *app.App, id string) tea.Cmd {
	return func() tea.Msg {
		shortID, err := a.Pause(id)
		if err != nil {
			return cmdResultMsg{err: err}
		}
		return cmdResultMsg{status: fmt.Sprintf("⏸ Paused %s", shortID)}
	}
}

func resumeTorrentCmd(a *app.App, id string) tea.Cmd {
	return func() tea.Msg {
		shortID, err := a.Resume(id)
		if err != nil {
			return cmdResultMsg{err: err}
		}
		return cmdResultMsg{status: fmt.Sprintf("▶ Resumed %s", shortID)}
	}
}

func formatPeerAddr(addr string, width int) string {
	if len(addr) <= width {
		return addr + strings.Repeat(" ", width-len(addr))
	}

	if width <= 10 {
		return padRight(addr, width)
	}

	// Keep the last ":port" + last 4 hex chars of the IP intact
	tailLen := min(11, width/2) // e.g., "7334]:65535" (11 chars)
	headLen := width - tailLen - 1

	return addr[:headLen] + "…" + addr[len(addr)-tailLen:]
}
