package internal

import "github.com/charmbracelet/lipgloss/v2"

// --- Lipgloss Styles (Less harsh colors) ---
var (
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8AFF8A")).Bold(true) // Light Green
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8A80")).Bold(true) // Light Red/Coral
	skipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFAA")).Bold(true) // Pale Yellow

	packageStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8AFFFB")).Bold(true) // Light Aqua
	testNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))            // Off-white
	durationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))            // Medium Gray
	// Adjusted MarginLeft to align failure output with the start of the 'TEST NAME' column.
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0")).Italic(true).MarginLeft(9)

	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8AFF")).Bold(true).Underline(true) // Soft Magenta

	summaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			MarginTop(1).
			BorderForeground(lipgloss.Color("#6A6A6A")) // Darker Gray for border

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true) // Orange-Red

	packageSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Faint(true) // Dark Gray

	// New style for table headers (within package)
	tableHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Bold(true).Underline(true) // Subtle gray underline
)
