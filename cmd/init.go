package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicoewok/dotdo/internal/storage"
	"github.com/nicoewok/dotdo/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the dotdo directory and storage",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		dotFolderPath := filepath.Join(home, ".dotdo")
		configPath := filepath.Join(dotFolderPath, "tasks.json")

		// 1. Create the directory (~/.dotdo)
		if _, err := os.Stat(dotFolderPath); os.IsNotExist(err) {
			err := os.MkdirAll(dotFolderPath, 0755)
			if err != nil {
				fmt.Printf(" %s Failed to create directory: %v\n", ui.RedStyle.Render("●"), err)
				return
			}
			fmt.Printf(" %s Created directory: %s\n", ui.GetStatusDot("todo"), dotFolderPath)
		} else {
			fmt.Printf(" %s Directory already exists: %s\n", ui.GreyStyle.Render("○"), dotFolderPath)
		}

		// 2. Create the tasks.json file if it doesn't exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			emptyList := storage.List{Tasks: []storage.Task{}}
			fileData, _ := json.MarshalIndent(emptyList, "", "  ")

			err := os.WriteFile(configPath, fileData, 0644)
			if err != nil {
				fmt.Printf(" %s Failed to create tasks.json: %v\n", ui.RedStyle.Render("●"), err)
				return
			}
			fmt.Printf(" %s Initialized storage: %s\n", ui.GetStatusDot("todo"), configPath)
		} else {
			fmt.Printf(" %s Storage already exists: %s\n", ui.GreyStyle.Render("○"), configPath)
		}

		// 3. Instruction for Git Sync
		fmt.Println("\n" + ui.GreyStyle.Render(" NEXT STEPS for Git Sync:"))
		fmt.Println("Be sure you have a private Git repository for saving your tasks, Git installed on your system and are authenticated with your Git provider on your current CLI.")
		fmt.Println(" 1. cd " + dotFolderPath)
		fmt.Println(" 2. git init")
		fmt.Println(" 3. git remote add origin <your-private-repo-url>")

		setupAutocomplete()
		fmt.Println("\n" + ui.GreenStyle.Render("Initialization complete!"))
	},
}

func setupAutocomplete() {
	shellPath := os.Getenv("SHELL")
	var rcFile string
	var shellType string

	if strings.Contains(shellPath, "zsh") {
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
		shellType = "zsh"
	} else {
		rcFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
		shellType = "bash"
	}

	// Check if we've already added it
	f, err := os.Open(rcFile)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "dotdo completion") {
				fmt.Printf(" %s Autocomplete already configured in %s\n", ui.GreyStyle.Render("○"), rcFile)
				f.Close()
				return
			}
		}
		f.Close()
	}

	// Append the source command
	line := fmt.Sprintf("\nsource <(dotdo completion %s) # dotdo-completion\n", shellType)

	f, err = os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf(" %s Could not update %s: %v\n", ui.RedStyle.Render("●"), rcFile, err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		fmt.Printf(" %s Failed to write to %s\n", ui.RedStyle.Render("●"), rcFile)
	} else {
		fmt.Printf(" %s Autocomplete added to %s\n", ui.GetStatusDot("todo"), rcFile)
		fmt.Println(ui.GreyStyle.Render("   Restart your terminal or run: source " + rcFile))
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
