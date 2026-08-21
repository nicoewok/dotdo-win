package cmd

import (
	"fmt"

	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show the dotdo user manual",
	Run: func(cmd *cobra.Command, args []string) {
		// Section Styling
		header := func(s string) string {
			return ui.RedStyle.Bold(true).Render("\n " + s)
		}
		cmdStyle := func(c, desc string) {
			fmt.Printf("  %-25s %s\n", ui.WhiteStyle.Render(c), ui.GreyStyle.Render(desc))
		}

		fmt.Println(header("CORE COMMANDS"))
		cmdStyle("dotdo", "Show pending dashboard")
		cmdStyle("dotdo init", "Initialize storage & folder")
		cmdStyle("dotdo list", "Show all tasks (incl. done)")
		cmdStyle("dotdo sync", "Git pull & push changes")

		fmt.Println(header("MANAGEMENT"))
		cmdStyle("dotdo add [text]", "Add task (no quotes needed)")
		cmdStyle("dotdo add [text] -d [date]", "Add task with YYYY-MM-DD")
		cmdStyle("dotdo doing [tab]", "Focus on a task (autocomplete)")
		cmdStyle("dotdo done [tab]", "Finish a task (autocomplete)")
		cmdStyle("dotdo remove", "Purge all 'done' tasks")

		fmt.Println(header("TIPS"))
		fmt.Println(ui.GreyStyle.Render("  - Use TAB for autocomplete on titles."))
		fmt.Println(ui.GreyStyle.Render("  - Expired dates appear in Red."))
		fmt.Println("")
	},
}

func init() {
	// We remove the default help command to use our own
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.AddCommand(helpCmd)
}
