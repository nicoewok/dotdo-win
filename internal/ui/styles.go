package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	Done   = lipgloss.Color("#FF6961")
	Green  = lipgloss.Color("#8CD47E")
	Orange = lipgloss.Color("#FFB54C")

	// Styles
	WhiteStyle = lipgloss.NewStyle().Foreground(White).Bold(true)
	RedStyle   = lipgloss.NewStyle().Foreground(Red)
	GreyStyle  = lipgloss.NewStyle().Foreground(Grey)
	GreenStyle = lipgloss.NewStyle().Foreground(Green)
	DoneStyle  = lipgloss.NewStyle().Foreground(Grey).Strikethrough(true)
)

// GetStatusDot returns a colored dot based on the task status
func GetStatusDot(status string) string {
	switch status {
	case "todo":
		return lipgloss.NewStyle().Foreground(Green).Render("●")
	case "doing":
		return lipgloss.NewStyle().Foreground(Orange).Render("●")
	case "done":
		return lipgloss.NewStyle().Foreground(Done).Render("●")
	default:
		return "○"
	}
}

// FormatDueDate handles the logic for red-tinting expired dates
func FormatDueDate(due time.Time) string {
	if due.IsZero() {
		return ""
	}

	dateStr := due.Format(" (Jan 02)")

	// If the current time is after the due date, make it red
	if time.Now().After(due) {
		return RedStyle.Render(dateStr)
	}

	return GreyStyle.Render(dateStr)
}
