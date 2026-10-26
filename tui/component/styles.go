package component

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("238"))

	footerStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("238"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(2)

	// Borderless Parent Table Styles
	tblHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Bold(true)

	rowIDStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	rowSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	rowNameStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	downSpeedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true) // Emerald Green
	upSpeedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))            // Cyan
	metricStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	progressStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Amber

	// Indented Child Peer Table Styles
	peerHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Bold(true)
	peerTreeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	peerRowStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	peerErrStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	emptyStateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)

	barCompleteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Emerald Green
	barInprogressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Amber / Yellow
	barRemainingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238")) // Muted Dark Gray
	pctStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)
