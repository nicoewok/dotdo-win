package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	window          fyne.Window
	svc             *Service
	taskList        *fyne.Container
	statusLabel     *widget.Label
	summaryLabel    *widget.Label
	titleEntry      *widget.Entry
	dueEntry        *widget.Entry
	statusMu        sync.Mutex
}

// SetupGUI constructs and attaches the dotdo GUI components to the given Fyne window.
func SetupGUI(window fyne.Window, svc *Service) *GUI {
	g := &GUI{
		window:   window,
		svc:      svc,
		taskList: container.NewVBox(),
	}

	g.buildUI()
	g.refreshTasks()
	return g
}

func (g *GUI) setStatus(msg string) {
	g.statusMu.Lock()
	defer g.statusMu.Unlock()
	if g.statusLabel != nil {
		g.statusLabel.SetText(msg)
	}
}

func (g *GUI) buildUI() {
	// 1. Header with title, sync status indicator, and manual Sync button
	titleLabel := widget.NewLabelWithStyle("DOT ● DO", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
	g.statusLabel = widget.NewLabelWithStyle("● Synced", fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true})
	
	syncBtn := widget.NewButton("Sync", func() {
		g.setStatus("● Syncing...")
		go func() {
			err := g.svc.Sync()
			if err != nil {
				g.setStatus("● Sync Error")
			} else {
				g.setStatus("● Synced")
			}
		}()
	})

	header := container.NewHBox(
		titleLabel,
		layout.NewSpacer(),
		g.statusLabel,
		syncBtn,
	)

	// 2. Task input bar with title entry, optional due date entry, and Add button
	g.titleEntry = widget.NewEntry()
	g.titleEntry.SetPlaceHolder("Add new task...")

	g.dueEntry = widget.NewEntry()
	g.dueEntry.SetPlaceHolder("Due YYYY-MM-DD (optional)")

	addBtn := widget.NewButton("+ Add", func() {
		g.handleAddTask()
	})

	g.titleEntry.OnSubmitted = func(_ string) {
		g.handleAddTask()
	}
	g.dueEntry.OnSubmitted = func(_ string) {
		g.handleAddTask()
	}

	inputGrid := container.NewGridWithColumns(3, g.titleEntry, g.dueEntry, addBtn)

	// 3. Scrollable task list
	scrollableList := container.NewVScroll(g.taskList)

	// 4. Footer status bar with summary and Purge Done action
	g.summaryLabel = widget.NewLabelWithStyle("0 tasks", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	purgeBtn := widget.NewButton("Purge Done", func() {
		count, err := g.svc.RemoveDoneTasks()
		if err != nil {
			g.setStatus(fmt.Sprintf("● Error: %v", err))
			return
		}
		g.setStatus(fmt.Sprintf("● Removed %d finished tasks", count))
		g.refreshTasks()
	})

	footer := container.NewHBox(
		g.summaryLabel,
		layout.NewSpacer(),
		purgeBtn,
	)

	// Main Layout
	mainContent := container.NewBorder(
		container.NewVBox(header, widget.NewSeparator(), inputGrid, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), footer),
		nil,
		nil,
		scrollableList,
	)

	g.window.SetContent(mainContent)
}

func (g *GUI) handleAddTask() {
	title := strings.TrimSpace(g.titleEntry.Text)
	if title == "" {
		g.setStatus("● Error: Title cannot be empty")
		return
	}

	var dueDate time.Time
	dueStr := strings.TrimSpace(g.dueEntry.Text)
	if dueStr != "" {
		parsed, err := time.Parse("2006-01-02", dueStr)
		if err != nil {
			g.setStatus("● Error: Date format must be YYYY-MM-DD")
			return
		}
		dueDate = parsed
	}

	_, err := g.svc.AddTask(title, dueDate)
	if err != nil {
		g.setStatus(fmt.Sprintf("● Error: %v", err))
		return
	}

	g.titleEntry.SetText("")
	g.dueEntry.SetText("")
	g.setStatus("● Task Added")
	g.refreshTasks()
}

func (g *GUI) refreshTasks() {
	tasks, err := g.svc.ListTasks(false)
	if err != nil {
		g.setStatus(fmt.Sprintf("● Error loading tasks: %v", err))
		return
	}

	g.taskList.Objects = nil

	pendingCount := 0
	doneCount := 0

	for _, task := range tasks {
		t := task // local capture
		if t.Status == "done" {
			doneCount++
		} else {
			pendingCount++
		}

		// Checkbox for completion
		check := widget.NewCheck("", func(checked bool) {
			newStatus := "todo"
			if checked {
				newStatus = "done"
			}
			_, err := g.svc.SetTaskStatus(t.Title, newStatus)
			if err != nil {
				g.setStatus(fmt.Sprintf("● Error: %v", err))
			} else {
				g.setStatus("● Updated")
			}
			g.refreshTasks()
		})
		check.Checked = (t.Status == "done")

		// Dot status indicator
		dotSymbol := "○"
		switch t.Status {
		case "todo":
			dotSymbol = "●"
		case "doing":
			dotSymbol = "◐"
		case "done":
			dotSymbol = "✔"
		}
		dotLabel := widget.NewLabelWithStyle(dotSymbol, fyne.TextAlignCenter, fyne.TextStyle{Monospace: true, Bold: true})

		// Title & Due Date formatting
		displayText := t.Title
		if !t.Due.IsZero() {
			dateStr := t.Due.Format("Jan 02")
			if t.Status != "done" && time.Now().After(t.Due) {
				displayText += fmt.Sprintf("  [EXPIRED: %s]", dateStr)
			} else {
				displayText += fmt.Sprintf("  (%s)", dateStr)
			}
		}

		style := fyne.TextStyle{Monospace: true}
		if t.Status == "done" {
			style.Italic = true
		} else if t.Status == "doing" {
			style.Bold = true
		}

		titleLabel := widget.NewLabelWithStyle(displayText, fyne.TextAlignLeading, style)

		// Focus / Doing button
		var actionBtn *widget.Button
		if t.Status == "doing" {
			actionBtn = widget.NewButton("Doing", func() {
				g.svc.SetTaskStatus(t.Title, "todo")
				g.refreshTasks()
			})
		} else if t.Status == "todo" {
			actionBtn = widget.NewButton("Focus", func() {
				g.svc.MarkDoing(t.Title)
				g.refreshTasks()
			})
		}

		// Delete button
		delBtn := widget.NewButton("✖", func() {
			_, err := g.svc.DeleteTaskByID(t.ID)
			if err != nil {
				g.setStatus(fmt.Sprintf("● Error: %v", err))
			} else {
				g.setStatus("● Task Deleted")
			}
			g.refreshTasks()
		})

		// Assemble row layout
		rowElements := []fyne.CanvasObject{check, dotLabel, titleLabel, layout.NewSpacer()}
		if actionBtn != nil {
			rowElements = append(rowElements, actionBtn)
		}
		rowElements = append(rowElements, delBtn)

		row := container.NewHBox(rowElements...)
		g.taskList.Add(row)
	}

	g.taskList.Refresh()
	g.summaryLabel.SetText(fmt.Sprintf("%d PENDING | %d DONE", pendingCount, doneCount))
}
