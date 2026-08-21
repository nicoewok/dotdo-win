package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	White = lipgloss.Color("#FFFFFF")
	Red   = lipgloss.Color("#FF0000")
	Grey  = lipgloss.Color("#555555")
)

func GetBunny() string {
	rawBunny := `
    ⠏⢣ ⠏⢣
  ⢠⡶⠧⠧⠶⠧⠧⠶⢶⡄
  ⡜         ⢣
 ⢸   ⠛   ⠛  ⢣
  ⢣      Y  ⢸
  ⢸      "  ⡜
  ⡜        ⢸
⠺⡜         ⡜
  ⠙⠒⠤⣀⣀⣇⣸⣇⣸
  
 DOT %s DO%s`

	redEye := lipgloss.NewStyle().Foreground(Red).Render("●")

	// FIXED: Removed "ui." prefix because we are already in the ui package
	bunnyText := lipgloss.NewStyle().Foreground(Grey).Render(" bunny\n  ──────────────")

	return lipgloss.NewStyle().
		Foreground(White).
		Render(fmt.Sprintf(rawBunny, redEye, bunnyText))
}
