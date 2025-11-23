package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("86")  // Cyan
	secondaryColor = lipgloss.Color("212") // Pink
	mutedColor     = lipgloss.Color("241") // Gray
	successColor   = lipgloss.Color("82")  // Green
	warningColor   = lipgloss.Color("214") // Orange
	errorColor     = lipgloss.Color("196") // Red

	// Base styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Width(14)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(primaryColor).
			Padding(0, 1)

	directionTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(secondaryColor).
				Padding(0, 1)

	wasteTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(errorColor).
			Padding(0, 1).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(1, 2)
)
