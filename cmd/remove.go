package cmd

import (
	"fmt"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Clear all finished tasks from storage",
	Run: func(cmd *cobra.Command, args []string) {
		list, err := storage.LoadTasks()
		if err != nil {
			fmt.Printf(" %s Error loading tasks: %v\n", ui.RedStyle.Render("●"), err)
			return
		}

		initialCount := len(list.Tasks)
		var keptTasks []storage.Task

		// Filter: Only keep tasks that are NOT done
		for _, t := range list.Tasks {
			if t.Status != "done" {
				keptTasks = append(keptTasks, t)
			}
		}

		removedCount := initialCount - len(keptTasks)

		if removedCount == 0 {
			fmt.Printf(" %s No completed tasks found. Storage is already clean.\n", ui.GreyStyle.Render("○"))
			return
		}

		list.Tasks = keptTasks
		storage.SaveTasks(list)

		fmt.Printf(" %s Successfully removed %d 'done' tasks.\n", ui.GetStatusDot("done"), removedCount)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
