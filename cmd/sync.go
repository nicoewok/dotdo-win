package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tasks with the remote private repository",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		dotFolderPath := filepath.Join(home, ".dotdo")

		// 1. Check if .git exists
		if _, err := os.Stat(filepath.Join(dotFolderPath, ".git")); os.IsNotExist(err) {
			fmt.Printf(" %s Git is not initialized in %s\n", ui.RedStyle.Render("●"), dotFolderPath)
			fmt.Println(ui.GreyStyle.Render("   Run 'dotdo init' and follow the Git steps."))
			return
		}

		fmt.Println(ui.GreyStyle.Render(" Starting synchronization..."))

		// 2. Commit local changes FIRST (so rebase doesn't panic)
		status, _ := getGitStatus(dotFolderPath)
		if status != "" {
			_ = runGit(dotFolderPath, "add", "tasks.json")
			err := runGit(dotFolderPath, "commit", "-m", "dotdo: auto sync update")
			if err != nil {
				fmt.Printf(" %s Local commit failed: %v\n", ui.RedStyle.Render("●"), err)
				return
			}
			fmt.Println(ui.GreyStyle.Render("   Local changes committed."))
		}

		// 3. Pull latest changes with rebase (merges history cleanly)
		err := runGit(dotFolderPath, "pull", "origin", "master", "--rebase")
		if err != nil {
			fmt.Printf(" %s Pull failed (Conflict?): %v\n", ui.RedStyle.Render("●"), err)
			fmt.Println(ui.GreyStyle.Render("   Manual fix required: cd ~/.dotdo && git rebase --continue"))
			return
		}

		// 4. Push everything back to the server
		err = runGit(dotFolderPath, "push", "origin", "master")
		if err != nil {
			fmt.Printf(" %s Push failed: %v\n", ui.RedStyle.Render("●"), err)
			return
		}

		fmt.Printf(" %s Successfully synced with remote.\n", ui.GetStatusDot("todo"))
	},
}

// Helper to run git commands and capture errors
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}

// Helper to check if there are changes to commit
func getGitStatus(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
