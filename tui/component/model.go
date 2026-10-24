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
	expandID  string // Manager ID or prefix expanded via /details <id>
	stats     map[string]peer.ManagerStatSnapshot
	sorted    []managerEntry // Reused slice for deterministic torrent row order
	peerBuf   []peerEntry    // Reused slice for deterministic peer row order
	textInput textinput.Model
	viewport  viewport.Model
	statusMsg string
	ready     bool
	width     int
	height    int
}

func InitialModel(app *app.App) model {
	ti := textinput.New()
	ti.Placeholder = "/add <magnet|file> | /details <id> | /remove <id> | /exit"
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
	return tea.Batch(fetchStats(m.app), tickEverySecond())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(fetchStats(m.app), tickEverySecond())

	case statUpdateMsg:
		m.stats = msg
		m.updateViewportContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			m.textInput.SetValue("")

			if input == "/exit" || input == "/quit" {
				return m, tea.Quit
			} else if after, ok := strings.CutPrefix(input, "/add "); ok {
				arg := strings.TrimSpace(after)
				if arg != "" {
					m.app.Add(arg)
					m.statusMsg = "Added: " + arg
				}
			} else if after, ok := strings.CutPrefix(input, "/remove "); ok {
				id := strings.TrimSpace(after)
				if id != "" {
					m.app.Remove(id)
					if m.expandID != "" && (m.expandID == id || strings.HasPrefix(m.expandID, id)) {
						m.expandID = ""
					}
					m.statusMsg = "Removed: " + id
				}
			} else if after, ok := strings.CutPrefix(input, "/details"); ok {
				id := strings.TrimSpace(after)
				if id == "" || m.expandID == id {
					m.expandID = ""
					m.statusMsg = "Collapsed peer details"
				} else {
					m.expandID = id
					m.statusMsg = "Expanded details for " + id
				}
				m.updateViewportContent()
			}
			return m, nil
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

	var b strings.Builder
	b.Grow(len(m.sorted) * 256)

	headerLine := fmt.Sprintf("  %-10s %-24s %-11s %-12s %-11s %-12s %-18s",
		"ID", "NAME", "DOWN", "↓ SPEED", "UP", "↑ SPEED", "DONE / INPROG",
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
		idCol := rowIDStyle.Render(padRight(shortID, 10))
		if isExpanded {
			icon = "▾ "
			idCol = rowSelectedStyle.Render(padRight(shortID, 10))
		}

		piecesStr := fmt.Sprintf("%d / %d", snap.Completed, snap.Inprogress)

		b.WriteString(icon)
		b.WriteString(idCol)
		b.WriteByte(' ')
		b.WriteString(rowNameStyle.Render(padRight(name, 24)))
		b.WriteByte(' ')
		b.WriteString(metricStyle.Render(padRight(formatBytes(int64(snap.Downloaded)), 11)))
		b.WriteByte(' ')
		b.WriteString(downSpeedStyle.Render(padRight(formatSpeed(snap.DownloadRate), 12)))
		b.WriteByte(' ')
		b.WriteString(metricStyle.Render(padRight(formatBytes(int64(snap.Uploaded)), 11)))
		b.WriteByte(' ')
		b.WriteString(upSpeedStyle.Render(padRight(formatSpeed(snap.UploadRate), 12)))
		b.WriteByte(' ')
		b.WriteString(progressStyle.Render(padRight(piecesStr, 18)))
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
		b.WriteString(emptyStateStyle.Render("Waiting for active peers..."))
		b.WriteString("\n\n")
		return
	}

	m.peerBuf = m.peerBuf[:0]
	for addr, st := range peers {
		m.peerBuf = append(m.peerBuf, peerEntry{addr: addr, stat: st})
	}

	slices.SortFunc(m.peerBuf, func(a, b peerEntry) int {
		if c := cmp.Compare(b.stat.DownloadRate, a.stat.DownloadRate); c != 0 {
			return c
		}
		return cmp.Compare(a.addr, b.addr)
	})

	subHeader := fmt.Sprintf("%-22s %-11s %-12s %-11s %-12s %-16s %-10s",
		"PEER ADDRESS", "DOWN", "↓ SPEED", "UP", "↑ SPEED", "REQ (DONE/SENT)", "ERR / TO",
	)
	b.WriteString(peerTreeStyle.Render("  ├─ "))
	b.WriteString(peerHeaderStyle.Render(subHeader))
	b.WriteByte('\n')

	for i, p := range m.peerBuf {
		branch := "  ├─ "
		if i == len(m.peerBuf)-1 {
			branch = "  ╰─ "
		}

		reqInfo := fmt.Sprintf("%d / %d", p.stat.RequestsCompleted, p.stat.RequestsSent)
		errInfo := fmt.Sprintf("%d / %d", p.stat.Errors, p.stat.Timeouts)

		b.WriteString(peerTreeStyle.Render(branch))
		b.WriteString(peerRowStyle.Render(padRight(p.addr, 22)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(formatBytes(p.stat.Downloaded), 11)))
		b.WriteByte(' ')
		b.WriteString(downSpeedStyle.Render(padRight(formatSpeed(p.stat.DownloadRate), 12)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(formatBytes(p.stat.Uploaded), 11)))
		b.WriteByte(' ')
		b.WriteString(upSpeedStyle.Render(padRight(formatSpeed(p.stat.UploadRate), 12)))
		b.WriteByte(' ')
		b.WriteString(peerRowStyle.Render(padRight(reqInfo, 16)))
		b.WriteByte(' ')
		if p.stat.Errors > 0 || p.stat.Timeouts > 0 {
			b.WriteString(peerErrStyle.Render(padRight(errInfo, 10)))
		} else {
			b.WriteString(peerRowStyle.Render(padRight(errInfo, 10)))
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
	helpLine := "Commands: /add <magnet|file> • /details <id> (toggle peers) • /remove <id> • /exit"
	if m.statusMsg != "" {
		helpLine = m.statusMsg + "  |  " + helpLine
	}
	return footerStyle.Width(m.width).Render(
		m.textInput.View() + "\n" + helpStyle.Render(helpLine),
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

func fetchStats(a *app.App) tea.Cmd {
	return func() tea.Msg {
		return statUpdateMsg(a.GetUpdatedManagerStat())
	}
}

func tickEverySecond() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
