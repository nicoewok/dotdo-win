package cmd

import (
	"fmt"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks (even done ones)",

	// Show bunny logo on every command
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "completion" || cmd.Name() == "help" || cmd.Name() == "__complete" {
			return
		}
		if cmd.HasParent() && cmd.Parent().Name() == "completion" {
			return
		}

		fmt.Print("\033[H\033[2J")

		fmt.Println(ui.GetBunny())
	},
	Run: func(cmd *cobra.Command, args []string) {
		list, _ := storage.LoadTasks()

		list.SortByDueDate()

		for _, t := range list.Tasks {
			dot := ui.GetStatusDot(t.Status)
			date := ui.FormatDueDate(t.Due)
			fmt.Printf("  %s %s%s\n", dot, t.Title, date)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
