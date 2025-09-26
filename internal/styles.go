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

	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0")).Italic(true).MarginLeft(3)

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true) // Orange-Red

	packageSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Faint(true) // Dark Gray
)

// --- Global Application Styles ---
var (
	// AppOverallOutputStyle is the top-level style that wraps all the display output.
	AppOverallOutputStyle = lipgloss.NewStyle().
		AlignHorizontal(lipgloss.Center).
		Margin(1, 1, 0)
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

	// Style for the package test table
	pkgTableStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center)
)

var PrismHeader = `         /\
        /  \ #########
       /    \ ########
 #### /      \ #######
     /        \ ######
    /          \ #####
   /            \ ####
   ---------------`
