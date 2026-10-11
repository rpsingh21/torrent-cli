package component

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true)

	footerStyle = lipgloss.NewStyle().
			Padding(1, 1).
			Border(lipgloss.NormalBorder(), true, false, false, false) // Top border only

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			PaddingLeft(2)
)

type model struct {
	items     map[string]string
	textInput textinput.Model
	viewport  viewport.Model
	nextID    int
	ready     bool // True once we get the initial terminal size
	width     int
	height    int
}

func InitialModel() model {
	ti := textinput.New()
	ti.Placeholder = "/add <value>, /remove <id>, /exit"
	ti.Focus()
	ti.Prompt = "❯ "
	ti.CharLimit = 256

	return model{
		items:     make(map[string]string),
		textInput: ti,
		nextID:    1,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			m.textInput.SetValue("") // Clear input

			if input == "/exit" {
				return m, tea.Quit
			} else if after, ok := strings.CutPrefix(input, "/add "); ok {
				// val := strings.TrimPrefix(input, "/add ")
				idStr := strconv.Itoa(m.nextID)
				m.items[idStr] = after
				m.nextID++
				m.updateViewportContent()
			} else if after, ok := strings.CutPrefix(input, "/remove "); ok {
				// idStr := strings.TrimPrefix(input, "/remove ")
				delete(m.items, after)
				m.updateViewportContent()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for the viewport
		// Header height (~3 lines with borders) + Footer height (~3 lines) = 6 lines
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		viewportHeight := msg.Height - headerHeight - footerHeight

		if !m.ready {
			// Initialize the viewport on the first size message
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.ready = true
			m.updateViewportContent()
		} else {
			// Resize existing viewport on subsequent resizes
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}

		// Ensure text input spans the whole width minus prompt padding
		m.textInput.Width = msg.Width - 4
	}

	// Route messages to components
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateViewportContent refreshes the scrollable list data
func (m *model) updateViewportContent() {
	if !m.ready {
		return
	}

	var b strings.Builder

	if len(m.items) == 0 {
		b.WriteString("\n  (Map is empty. Type /add <value> below)\n")
	} else {
		// Sort keys to maintain UI stability
		keys := make([]int, 0, len(m.items))
		for k := range m.items {
			id, _ := strconv.Atoi(k)
			keys = append(keys, id)
		}
		sort.Ints(keys)

		for _, k := range keys {
			idStr := strconv.Itoa(k)
			// Truncate long strings to prevent wrapping based on terminal width
			line := fmt.Sprintf("[%s] %s", idStr, m.items[idStr])
			if len(line) > m.width-4 {
				line = line[:m.width-7] + "..."
			}
			b.WriteString(itemStyle.Render(line) + "\n")
		}
	}

	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom() // Auto-scroll to the latest items like a chat CLI
}

func (m model) headerView() string {
	title := headerStyle.Width(m.width).Render("🚀 BTUI")
	return title
}

func (m model) footerView() string {
	return footerStyle.Width(m.width).Render(m.textInput.View())
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing full-screen view..."
	}

	// Join the Header, Viewport (Middle), and Footer vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}
