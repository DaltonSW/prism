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

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true) // Orange-Red

	packageSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Faint(true) // Dark Gray
)

// --- Global Application Styles ---
var (
	// AppOverallOutputStyle is the top-level style that wraps all the display output.
	AppOverallOutputStyle = lipgloss.NewStyle().
		AlignHorizontal(lipgloss.Center).
		Padding(1, 2).
		Margin(1, 2)
)

// --- Specific Table/Summary Styles ---
var (
	// Style for the main container holding the overall summary table
	OverallSummaryTableWrapperStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#8AFFFB")).
					Padding(0, 1).
					Margin(1, 0). // Margin between package blocks and summary
					Align(lipgloss.Center)

	// Style for the overall summary table headers/keys (e.g., "Total", "Passed")
	overallSummaryTableHeaderStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#FF8AFF"))

	// Style for the overall summary table values (e.g., "10", "8")
	overallSummaryTableValueStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#A0A0A0"))

	// Style for the package test table
	pkgTableStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center)

	// Style for the header row of the package test table
	pkgTableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Bold(true).
				Underline(true).
				Padding(0, 1)

	// Default cell style for the package test table (used as base)
	pkgTableCellStyle = lipgloss.NewStyle().
				Padding(0, 1)

	// Column-specific styles for the package test table
	// These widths (1, 4, 8) are the *content* widths, `pkgTableCellStyle.Padding` adds to total.
	pkgTableIconColStyle     = lipgloss.NewStyle().Width(1).Align(lipgloss.Center)
	pkgTableStatusColStyle   = lipgloss.NewStyle().Width(4).Align(lipgloss.Left)
	pkgTableDurationColStyle = lipgloss.NewStyle().Width(8).Align(lipgloss.Right)
)

var PrismHeader = `         /\
        /  \ #########
       /    \ ########
 #### /      \ #######
     /        \ ######
    /          \ #####
   /            \ ####
   ---------------`
