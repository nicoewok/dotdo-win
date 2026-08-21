package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var dueStr string

var addCmd = &cobra.Command{
	Use:   "add [task]",
	Short: "Add a new task to your list",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskName := strings.Join(args, " ")
		list, _ := storage.LoadTasks()

		var dueDate time.Time
		var err error

		// If the user provided a date via -d or --due
		if dueStr != "" {
			// Layout "2006-01-02" is Go's way of saying YYYY-MM-DD
			dueDate, err = time.Parse("2006-01-02", dueStr)
			if err != nil {
				fmt.Printf(" %s Error: Invalid date format. Use YYYY-MM-DD (e.g. 2026-12-31)\n", ui.GetStatusDot("done"))
				return
			}
		}

		if taskName == "" {
			fmt.Printf(" %s Error: Task title cannot be empty.\n", ui.GetStatusDot("done"))
			return
		}

		// if task is already in the list, we don't add it again
		for _, t := range list.Tasks {
			if t.Title == taskName {
				fmt.Printf(" %s Task already exists: %s\n", ui.GetStatusDot(t.Status), taskName)
				return
			}
		}

		newTask := storage.Task{
			ID:     len(list.Tasks) + 1,
			Title:  taskName,
			Status: "todo",
			Due:    dueDate,
		}

		list.Tasks = append(list.Tasks, newTask)
		storage.SaveTasks(list)
		dot := ui.GetStatusDot(newTask.Status)

		// Subtle feedback
		if !dueDate.IsZero() {
			fmt.Printf(" %s Added: %s (Due: %s)\n", dot, taskName, dueDate.Format("Jan 02"))
		} else {
			fmt.Printf(" %s Added: %s\n", dot, taskName)
		}
	},
}

func init() {
	addCmd.Flags().StringVarP(&dueStr, "due", "d", "", "Set a due date (YYYY-MM-DD)")
	rootCmd.AddCommand(addCmd)
}
