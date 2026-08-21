package cmd

import (
	"fmt"
	"strings"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done [task title]",
	Short: "Move a task to 'done' status",
	// We use MinimumNArgs(1) so it works WITH or WITHOUT manual quotes
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, _ := storage.LoadTasks()
		var suggestions []string

		for _, t := range list.Tasks {
			// Suggest anything that isn't already done
			if t.Status != "done" {
				// If the title has spaces, we MUST quote it for the shell
				if strings.Contains(t.Title, " ") {
					suggestions = append(suggestions, fmt.Sprintf("\"%s\"", t.Title))
				} else {
					suggestions = append(suggestions, t.Title)
				}
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Join all args with a space in case the user didn't use quotes manually
		targetTitle := strings.Join(args, " ")
		targetTitle = strings.Trim(targetTitle, "\"")

		list, _ := storage.LoadTasks()
		found := false

		for i, t := range list.Tasks {
			if strings.EqualFold(t.Title, targetTitle) {
				list.Tasks[i].Status = "done"
				found = true
				break
			}
		}

		if found {
			storage.SaveTasks(list)
			fmt.Printf(" %s Finished: %s\n", ui.GetStatusDot("done"), targetTitle)
		} else {
			fmt.Printf(" %s Task not found: %s\n", ui.RedStyle.Render("●"), targetTitle)
		}
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
