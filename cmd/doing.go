package cmd

import (
	"fmt"
	"strings"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var doingCmd = &cobra.Command{
	Use:   "doing [task title]",
	Short: "Move a task to 'doing' status",
	// We use MinimumNArgs(1) so it works WITH or WITHOUT manual quotes
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, _ := storage.LoadTasks()
		var suggestions []string
		for _, t := range list.Tasks {
			if t.Status == "todo" {
				// The Magic: Wrap the title in double quotes for the shell
				// This ensures the shell handles spaces correctly
				quotedTitle := fmt.Sprintf("\"%s\"", t.Title)
				suggestions = append(suggestions, quotedTitle)
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Join all args with a space in case the user didn't use quotes manually
		targetTitle := strings.Join(args, " ")

		// Clean up any accidental double-quotes the shell might have passed through
		targetTitle = strings.Trim(targetTitle, "\"")

		list, _ := storage.LoadTasks()
		found := false

		for i, t := range list.Tasks {
			if strings.EqualFold(t.Title, targetTitle) {
				list.Tasks[i].Status = "doing"
				found = true
				break
			}
		}

		if found {
			storage.SaveTasks(list)
			fmt.Printf(" %s Focused on: %s\n", ui.GetStatusDot("doing"), targetTitle)
		} else {
			fmt.Printf(" %s Task not found: %s\n", ui.RedStyle.Render("●"), targetTitle)
		}
	},
}

func init() {
	rootCmd.AddCommand(doingCmd)
}
