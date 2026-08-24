package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/nicoewok/dotdo-win/pkg/service"
	"github.com/nicoewok/dotdo-win/pkg/store"
	"github.com/nicoewok/dotdo-win/pkg/task"
)

//go:embed assets/scientifica.ttf
var scientificaFontData []byte

//go:embed assets/DotGothic16.ttf
var dotGothicFontData []byte

//go:embed assets/icon.png
var appIconBytes []byte

var (
	bgDark       = color.NRGBA{R: 14, G: 14, B: 18, A: 255}
	cardBg       = color.NRGBA{R: 24, G: 24, B: 30, A: 255}
	inputBg      = color.NRGBA{R: 35, G: 35, B: 45, A: 255}
	accentRed    = color.NRGBA{R: 235, G: 75, B: 75, A: 255}
	accentGreen  = color.NRGBA{R: 85, G: 215, B: 130, A: 255}
	accentOrange = color.NRGBA{R: 255, G: 170, B: 50, A: 255}
	textPrimary  = color.NRGBA{R: 245, G: 245, B: 250, A: 255}
	textMuted    = color.NRGBA{R: 160, G: 160, B: 175, A: 255}
)

const bunnyASCII = `  ⠏⢣ ⠏⢣
⢠⡶⠧⠧⠶⠧⠧⠶⢶⡄
⡜         ⢣
⢸   ⠛   ⠛  ⢣
 ⢣      Y  ⢸
 ⢸      "  ⡜
 ⡜        ⢸
⠺⡜         ⡜
 ⠙⠒⠤⣀⣀⣇⣸⣇⣸`

type UIState struct {
	svc                *service.Service
	th                 *material.Theme
	bunnyTh            *material.Theme
	activeView         string // "list", "add", or "github_connect"
	showDone           bool   // toggle to show or hide completed tasks
	isSyncDropdownOpen bool
	addBtn             widget.Clickable
	backBtn            widget.Clickable
	pullBtn            widget.Clickable
	pushBtn            widget.Clickable
	syncDropdownBtn    widget.Clickable
	githubBtn          widget.Clickable
	purgeBtn           widget.Clickable
	toggleDoneBtn      widget.Clickable
	submitBtn          widget.Clickable
	cancelBtn          widget.Clickable
	syncMsg            string
	titleEditor        widget.Editor
	dueEditor          widget.Editor
	delBtns            map[int]*widget.Clickable
	statusBtns         map[int]*widget.Clickable

	// GitHub Connection View Widgets & State
	githubBackBtn  widget.Clickable
	patConnectBtn  widget.Clickable
	openPatUrlBtn  widget.Clickable
	patTokenEditor widget.Editor
	githubMsg      string

	// Add Task View & Date Picker State
	addErrMsg        string
	togglePickerBtn  widget.Clickable
	isDatePickerOpen bool
	pickerYear       int
	pickerMonth      time.Month
	prevMonthBtn     widget.Clickable
	nextMonthBtn     widget.Clickable
	calendarDayBtns  [42]widget.Clickable
}

func main() {
	svc := service.NewService()
	if err := svc.Init(); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	go func() {
		w := new(app.Window)
		fixedSize := app.Size(unit.Dp(560), unit.Dp(780))
		w.Option(
			app.Title("dotdo"),
			fixedSize,
			app.MinSize(unit.Dp(560), unit.Dp(780)),
			app.MaxSize(unit.Dp(560), unit.Dp(780)),
		)

		uiState := &UIState{
			svc:        svc,
			th:         nil,
			activeView: "list",
			showDone:   false,
			syncMsg:    "Ready",
			delBtns:    make(map[int]*widget.Clickable),
			statusBtns: make(map[int]*widget.Clickable),
		}
		uiState.titleEditor.SingleLine = true
		uiState.dueEditor.SingleLine = true
		uiState.patTokenEditor.SingleLine = true

		if err := run(w, uiState); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func openURL(rawURL string) {
	if rawURL == "" {
		return
	}
	err := exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	if err != nil {
		_ = exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Start-Process '%s'", rawURL)).Start()
	}
}

func run(w *app.Window, state *UIState) error {
	state.th = material.NewTheme()
	if faces, err := opentype.ParseCollection(scientificaFontData); err == nil && len(faces) > 0 {
		state.th.Shaper = text.NewShaper(text.WithCollection(faces))
	}
	state.th.TextSize = unit.Sp(18)
	state.th.Bg = bgDark
	state.th.Fg = textPrimary
	state.th.ContrastBg = accentRed

	state.bunnyTh = material.NewTheme()
	if bunnyFaces, err := opentype.ParseCollection(dotGothicFontData); err == nil && len(bunnyFaces) > 0 {
		state.bunnyTh.Shaper = text.NewShaper(text.WithCollection(bunnyFaces))
	} else {
		state.bunnyTh.Shaper = state.th.Shaper
	}

	var ops op.Ops
	var listLayout layout.List
	listLayout.Axis = layout.Vertical

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Handle View Navigation & Button Clicks
			if state.addBtn.Clicked(gtx) {
				state.activeView = "add"
				state.isSyncDropdownOpen = false
				state.titleEditor.SetText("")
				state.dueEditor.SetText("")
				state.addErrMsg = ""
				now := time.Now()
				state.pickerYear = now.Year()
				state.pickerMonth = now.Month()
				state.isDatePickerOpen = false
			}

			if state.backBtn.Clicked(gtx) || state.cancelBtn.Clicked(gtx) || state.githubBackBtn.Clicked(gtx) {
				state.activeView = "list"
				state.isSyncDropdownOpen = false
				state.addErrMsg = ""
			}

			if state.pullBtn.Clicked(gtx) {
				state.syncMsg = "Pulling..."
				state.isSyncDropdownOpen = false
				go func() {
					if err := state.svc.Pull(); err != nil {
						if !store.IsGithubConnected(state.svc.StorageDir()) {
							state.syncMsg = "Can't pull! Please connect your GitHub account by using the dropdown on the right."
						} else {
							state.syncMsg = fmt.Sprintf("Pull Err: %v", err)
						}
					} else {
						state.syncMsg = "Pulled"
						time.AfterFunc(3*time.Second, func() {
							if state.syncMsg == "Pulled" {
								state.syncMsg = "Ready"
								w.Invalidate()
							}
						})
					}
					w.Invalidate()
				}()
			}

			if state.pushBtn.Clicked(gtx) {
				state.syncMsg = "Pushing..."
				state.isSyncDropdownOpen = false
				go func() {
					if err := state.svc.Push(); err != nil {
						if !store.IsGithubConnected(state.svc.StorageDir()) {
							state.syncMsg = "Can't push! Please connect your GitHub account by using the dropdown on the right."
						} else {
							state.syncMsg = fmt.Sprintf("Push Err: %v", err)
						}
					} else {
						state.syncMsg = "Pushed"
						time.AfterFunc(3*time.Second, func() {
							if state.syncMsg == "Pushed" {
								state.syncMsg = "Ready"
								w.Invalidate()
							}
						})
					}
					w.Invalidate()
				}()
			}

			if state.syncDropdownBtn.Clicked(gtx) {
				state.isSyncDropdownOpen = !state.isSyncDropdownOpen
			}

			if state.githubBtn.Clicked(gtx) {
				state.isSyncDropdownOpen = false
				if store.IsGithubConnected(state.svc.StorageDir()) {
					_ = state.svc.DisconnectGithub()
					state.syncMsg = "Disconnected"
				} else {
					state.activeView = "github_connect"
					state.githubMsg = ""
				}
			}

			if state.openPatUrlBtn.Clicked(gtx) {
				openURL("https://github.com/settings/tokens/new?scopes=repo&description=dotdo-windows")
			}

			if state.patConnectBtn.Clicked(gtx) {
				token := strings.TrimSpace(state.patTokenEditor.Text())

				if token == "" {
					state.githubMsg = "Please enter a valid Personal Access Token."
				} else {
					state.githubMsg = "Validating token & connecting..."
					go func() {
						if err := state.svc.ConfigureGithub(token, "", ".dotdo"); err != nil {
							state.githubMsg = fmt.Sprintf("Connection error: %v", err)
						} else {
							state.syncMsg = "Connected"
							state.githubMsg = "Successfully connected to GitHub!"
							state.activeView = "list"
						}
						w.Invalidate()
					}()
				}
			}

			if state.toggleDoneBtn.Clicked(gtx) {
				state.showDone = !state.showDone
			}

			if state.purgeBtn.Clicked(gtx) {
				_, _ = state.svc.RemoveDoneTasks()
			}

			if state.togglePickerBtn.Clicked(gtx) {
				state.isDatePickerOpen = !state.isDatePickerOpen
				if state.isDatePickerOpen {
					dueStr := strings.TrimSpace(state.dueEditor.Text())
					if parsed, err := time.Parse("2006-01-02", dueStr); err == nil && len(dueStr) == 10 {
						state.pickerYear = parsed.Year()
						state.pickerMonth = parsed.Month()
					} else {
						now := time.Now()
						state.pickerYear = now.Year()
						state.pickerMonth = now.Month()
					}
				}
			}

			if state.prevMonthBtn.Clicked(gtx) {
				d := time.Date(state.pickerYear, state.pickerMonth, 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
				state.pickerYear = d.Year()
				state.pickerMonth = d.Month()
			}

			if state.nextMonthBtn.Clicked(gtx) {
				d := time.Date(state.pickerYear, state.pickerMonth, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
				state.pickerYear = d.Year()
				state.pickerMonth = d.Month()
			}

			if state.isDatePickerOpen && state.pickerYear > 0 {
				firstDay := time.Date(state.pickerYear, state.pickerMonth, 1, 0, 0, 0, 0, time.Local)
				weekdayOffset := (int(firstDay.Weekday()) + 6) % 7
				daysInMonth := time.Date(state.pickerYear, state.pickerMonth+1, 0, 0, 0, 0, 0, time.Local).Day()
				for i := 0; i < 42; i++ {
					dayNum := i - weekdayOffset + 1
					if dayNum >= 1 && dayNum <= daysInMonth {
						if state.calendarDayBtns[i].Clicked(gtx) {
							selected := time.Date(state.pickerYear, state.pickerMonth, dayNum, 0, 0, 0, 0, time.Local)
							state.dueEditor.SetText(selected.Format("2006-01-02"))
							state.addErrMsg = ""
							break
						}
					}
				}
			}

			submitTask := func() {
				dueStr := strings.TrimSpace(state.dueEditor.Text())
				var dueDate time.Time
				validDate := true
				if dueStr != "" {
					if parsed, err := time.Parse("2006-01-02", dueStr); err == nil && len(dueStr) == 10 {
						dueDate = parsed
					} else {
						validDate = false
					}
				}
				if validDate {
					title := strings.TrimSpace(state.titleEditor.Text())
					if title == "" {
						state.addErrMsg = "Please enter a task title."
					} else {
						_, _ = state.svc.AddTask(title, dueDate)
						state.activeView = "list"
						state.addErrMsg = ""
					}
				}
			}

			if state.activeView == "add" {
				for {
					ev, ok := gtx.Event(
						key.Filter{Focus: &state.titleEditor, Required: key.ModCtrl, Name: key.NameReturn},
						key.Filter{Focus: &state.titleEditor, Required: key.ModCtrl, Name: key.NameEnter},
						key.Filter{Focus: &state.dueEditor, Required: key.ModCtrl, Name: key.NameReturn},
						key.Filter{Focus: &state.dueEditor, Required: key.ModCtrl, Name: key.NameEnter},
						key.Filter{Required: key.ModCtrl, Name: key.NameReturn},
						key.Filter{Required: key.ModCtrl, Name: key.NameEnter},
						key.Filter{Focus: &state.titleEditor, Required: key.ModShortcut, Name: key.NameReturn},
						key.Filter{Focus: &state.titleEditor, Required: key.ModShortcut, Name: key.NameEnter},
						key.Filter{Focus: &state.dueEditor, Required: key.ModShortcut, Name: key.NameReturn},
						key.Filter{Focus: &state.dueEditor, Required: key.ModShortcut, Name: key.NameEnter},
						key.Filter{Required: key.ModShortcut, Name: key.NameReturn},
						key.Filter{Required: key.ModShortcut, Name: key.NameEnter},
					)
					if !ok {
						break
					}
					if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
						submitTask()
					}
				}
			}

			if state.submitBtn.Clicked(gtx) {
				submitTask()
			}

			// Fill Window Background
			paint.Fill(gtx.Ops, bgDark)

			// Fetch tasks filtered by showDone state
			tasks, _ := state.svc.ListTasks(!state.showDone)

			// Handle Status Cycle & Delete Button Clicks
			for _, t := range tasks {
				sBtn, sExists := state.statusBtns[t.ID]
				if !sExists {
					sBtn = new(widget.Clickable)
					state.statusBtns[t.ID] = sBtn
				}
				if sBtn.Clicked(gtx) {
					nextStatus := "doing"
					switch t.Status {
					case "todo":
						nextStatus = "doing"
					case "doing":
						nextStatus = "done"
					case "done":
						nextStatus = "todo"
					}
					_, _ = state.svc.SetTaskStatus(t.Title, nextStatus)
					tasks, _ = state.svc.ListTasks(!state.showDone)
					break
				}

				dBtn, dExists := state.delBtns[t.ID]
				if !dExists {
					dBtn = new(widget.Clickable)
					state.delBtns[t.ID] = dBtn
				}
				if dBtn.Clicked(gtx) {
					_, _ = state.svc.DeleteTaskByID(t.ID)
					tasks, _ = state.svc.ListTasks(!state.showDone)
					break
				}
			}

			// Root Window Layout: Stacked Overlay Architecture
			layout.Stack{}.Layout(gtx,
				// Layer 1: Main Application View
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    unit.Dp(18),
						Bottom: unit.Dp(18),
						Left:   unit.Dp(20),
						Right:  unit.Dp(20),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						switch state.activeView {
						case "add":
							return renderAddView(gtx, state)
						case "github_connect":
							return renderGithubConnectView(gtx, state)
						default:
							return renderListView(gtx, state, tasks, &listLayout)
						}
					})
				}),

				// Layer 2: Floating Dropdown Overlay anchored to North-East (Far Right)
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					if !state.isSyncDropdownOpen || state.activeView != "list" {
						return layout.Dimensions{}
					}
					return layout.Inset{
						Top:   unit.Dp(78),
						Right: unit.Dp(36),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top:    unit.Dp(8),
									Bottom: unit.Dp(8),
									Left:   unit.Dp(10),
									Right:  unit.Dp(10),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									githubLabel := "Connect Github"
									if store.IsGithubConnected(state.svc.StorageDir()) {
										githubLabel = "Disconnect Github"
									}

									btn := material.Button(state.th, &state.githubBtn, githubLabel)
									btn.TextSize = unit.Sp(16)
									btn.Background = cardBg
									btn.Color = textPrimary
									return btn.Layout(gtx)
								})
							})
						})
					})
				}),
			)

			e.Frame(gtx.Ops)
		}
	}
}

func renderHeaderTitle(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := material.H4(th, "DOT")
			h.TextSize = unit.Sp(30)
			h.Color = textPrimary
			return h.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := material.H4(th, "●")
			h.TextSize = unit.Sp(30)
			h.Color = accentRed
			return h.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := material.H4(th, "DO")
			h.TextSize = unit.Sp(30)
			h.Color = textPrimary
			return h.Layout(gtx)
		}),
	)
}

func renderListView(gtx layout.Context, state *UIState, tasks []task.Task, listLayout *layout.List) layout.Dimensions {
	allTasks, _ := state.svc.ListTasks(false)
	pendingCount := 0
	doneCount := 0
	for _, t := range allTasks {
		if t.Status == "done" {
			doneCount++
		} else {
			pendingCount++
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Header Banner
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(14),
					Bottom: unit.Dp(14),
					Left:   unit.Dp(18),
					Right:  unit.Dp(18),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Bunny Art in White
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.bunnyTh, bunnyASCII)
							lbl.TextSize = unit.Sp(16)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
						// App Title with Evenly Spaced Red Dot (DOT ● DO)
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return renderHeaderTitle(gtx, state.th)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									sub := material.Body2(state.th, fmt.Sprintf("● %s", state.syncMsg))
									sub.TextSize = unit.Sp(16)
									sub.Color = textMuted
									return sub.Layout(gtx)
								}),
							)
						}),
						// Action Buttons: + Add, Stacked Pull & Push, & Attached Dropdown (▼)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.addBtn, "+ Add")
									btn.TextSize = unit.Sp(16)
									btn.Background = accentRed
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),

								// Stacked Pull (top) & Push (bottom) Buttons
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := material.Button(state.th, &state.pullBtn, "⬇ Pull")
											btn.TextSize = unit.Sp(13)
											btn.Background = inputBg
											btn.Color = accentGreen
											return btn.Layout(gtx)
										}),
										layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := material.Button(state.th, &state.pushBtn, "⬆ Push")
											btn.TextSize = unit.Sp(13)
											btn.Background = inputBg
											btn.Color = accentOrange
											return btn.Layout(gtx)
										}),
									)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),

								// Attached Dropdown Button (▼)
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									arrBtn := material.Button(state.th, &state.syncDropdownBtn, "▼")
									arrBtn.TextSize = unit.Sp(14)
									arrBtn.Background = inputBg
									arrBtn.Color = textPrimary
									return arrBtn.Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),

		// 2. Section Header with Show/Hide Done Toggle
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(state.th, fmt.Sprintf("TASKS (%d)", len(tasks)))
					lbl.Font.Weight = font.Bold
					lbl.TextSize = unit.Sp(18)
					lbl.Color = textMuted
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					toggleLabel := "Hide Done"
					if !state.showDone {
						toggleLabel = "Show Done"
					}
					tBtn := material.Button(state.th, &state.toggleDoneBtn, toggleLabel)
					tBtn.TextSize = unit.Sp(16)
					tBtn.Background = inputBg
					tBtn.Color = textMuted
					return tBtn.Layout(gtx)
				}),
			)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),

		// 3. Task List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(tasks) == 0 {
				return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    unit.Dp(28),
						Bottom: unit.Dp(28),
						Left:   unit.Dp(18),
						Right:  unit.Dp(18),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						msg := "No tasks found. Click '+ Add' to create one!"
						if !state.showDone && doneCount > 0 {
							msg = "No pending tasks. Click 'Show Done' to see completed tasks."
						}
						lbl := material.Body1(state.th, msg)
						lbl.TextSize = unit.Sp(18)
						lbl.Color = textMuted
						return lbl.Layout(gtx)
					})
				})
			}

			return listLayout.Layout(gtx, len(tasks), func(gtx layout.Context, index int) layout.Dimensions {
				t := tasks[index]
				return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return renderTaskRowCard(gtx, state, t)
				})
			})
		}),

		// 4. Footer Status Bar with Remove Done Button
		layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body2(state.th, fmt.Sprintf("● %d Pending  |  ✔ %d Completed", pendingCount, doneCount))
					lbl.TextSize = unit.Sp(16)
					lbl.Color = textMuted
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if doneCount == 0 {
						return layout.Dimensions{}
					}
					btn := material.Button(state.th, &state.purgeBtn, "Remove Done")
					btn.TextSize = unit.Sp(16)
					btn.Background = inputBg
					btn.Color = accentRed
					return btn.Layout(gtx)
				}),
			)
		}),
	)
}

func renderAddView(gtx layout.Context, state *UIState) layout.Dimensions {
	event.Op(gtx.Ops, &state.submitBtn)

	dueStr := strings.TrimSpace(state.dueEditor.Text())
	isDateValid := true
	if dueStr != "" {
		if _, err := time.Parse("2006-01-02", dueStr); err != nil || len(dueStr) != 10 {
			isDateValid = false
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header Banner with Back Button
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(14),
					Bottom: unit.Dp(14),
					Left:   unit.Dp(18),
					Right:  unit.Dp(18),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.bunnyTh, bunnyASCII)
							lbl.TextSize = unit.Sp(16)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return renderHeaderTitle(gtx, state.th)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.backBtn, "← Back")
							btn.TextSize = unit.Sp(18)
							btn.Background = inputBg
							btn.Color = textPrimary
							return btn.Layout(gtx)
						}),
					)
				})
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),

		// Add Task Form Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(22),
					Bottom: unit.Dp(22),
					Left:   unit.Dp(22),
					Right:  unit.Dp(22),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h := material.H6(state.th, "+ CREATE NEW TASK")
							h.TextSize = unit.Sp(22)
							h.Color = accentRed
							return h.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),

						// Title Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Task Title:")
							lbl.TextSize = unit.Sp(18)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top:    unit.Dp(10),
									Bottom: unit.Dp(10),
									Left:   unit.Dp(12),
									Right:  unit.Dp(12),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(state.th, &state.titleEditor, "Enter task title...")
									e.TextSize = unit.Sp(18)
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),

						// Due Date Header with Calendar Toggle
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body1(state.th, "Due Date (optional):")
									lbl.TextSize = unit.Sp(18)
									lbl.Color = textPrimary
									return lbl.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									toggleLabel := "📅 Pick Date ▼"
									if state.isDatePickerOpen {
										toggleLabel = "📅 Close Picker ▲"
									}
									btn := material.Button(state.th, &state.togglePickerBtn, toggleLabel)
									btn.TextSize = unit.Sp(14)
									btn.Background = inputBg
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
							)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),

						// Due Date Input Box
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top:    unit.Dp(10),
									Bottom: unit.Dp(10),
									Left:   unit.Dp(12),
									Right:  unit.Dp(12),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(state.th, &state.dueEditor, "YYYY-MM-DD (e.g. 2026-12-31)")
									e.TextSize = unit.Sp(18)
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						// Live Date Format Verification Feedback (Only shown when invalid)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if isDateValid {
								return layout.Dimensions{}
							}
							return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(state.th, "✕ Invalid date format (use YYYY-MM-DD)")
								lbl.TextSize = unit.Sp(14)
								lbl.Color = accentRed
								return lbl.Layout(gtx)
							})
						}),

						// Optional Calendar View
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !state.isDatePickerOpen {
								return layout.Dimensions{}
							}
							return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return renderDatePicker(gtx, state)
							})
						}),

						// Global Form Error Message (if any)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if state.addErrMsg == "" {
								return layout.Dimensions{}
							}
							return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(state.th, state.addErrMsg)
								lbl.TextSize = unit.Sp(15)
								lbl.Color = accentRed
								return lbl.Layout(gtx)
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),

						// Submit & Cancel Action Buttons
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.cancelBtn, "Cancel")
									btn.TextSize = unit.Sp(18)
									btn.Background = inputBg
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.submitBtn, "Create Task")
									btn.TextSize = unit.Sp(18)
									if !isDateValid {
										btn.Background = inputBg
										btn.Color = textMuted
										gtx = gtx.Disabled()
									} else {
										btn.Background = accentRed
										btn.Color = textPrimary
									}
									return btn.Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		}),
	)
}

func renderDatePicker(gtx layout.Context, state *UIState) layout.Dimensions {
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()

	if state.pickerYear == 0 || state.pickerMonth == 0 {
		state.pickerYear = todayYear
		state.pickerMonth = todayMonth
	}

	dueStr := strings.TrimSpace(state.dueEditor.Text())
	var selectedYear int
	var selectedMonth time.Month
	var selectedDay int
	if parsed, err := time.Parse("2006-01-02", dueStr); err == nil && len(dueStr) == 10 {
		selectedYear = parsed.Year()
		selectedMonth = parsed.Month()
		selectedDay = parsed.Day()
	}

	firstDay := time.Date(state.pickerYear, state.pickerMonth, 1, 0, 0, 0, 0, time.Local)
	weekdayOffset := (int(firstDay.Weekday()) + 6) % 7 // Monday = 0
	daysInMonth := time.Date(state.pickerYear, state.pickerMonth+1, 0, 0, 0, 0, 0, time.Local).Day()

	return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
			Left:   unit.Dp(12),
			Right:  unit.Dp(12),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Month Navigation Header: [ ◀ ]  MONTH YEAR  [ ▶ ]
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.prevMonthBtn, "◀")
							btn.TextSize = unit.Sp(14)
							btn.Background = cardBg
							btn.Color = textPrimary
							btn.Inset = layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}
							return btn.Layout(gtx)
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							monthTitle := fmt.Sprintf("%s %d", strings.ToUpper(state.pickerMonth.String()), state.pickerYear)
							lbl := material.Body1(state.th, monthTitle)
							lbl.TextSize = unit.Sp(16)
							lbl.Font.Weight = font.Bold
							lbl.Color = textPrimary
							lbl.Alignment = text.Middle
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.nextMonthBtn, "▶")
							btn.TextSize = unit.Sp(14)
							btn.Background = cardBg
							btn.Color = textPrimary
							btn.Inset = layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}
							return btn.Layout(gtx)
						}),
					)
				}),

				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),

				// Today Sub-indicator
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body2(state.th, fmt.Sprintf("● Today: %s (Red)", now.Format("Jan 02, 2006")))
					lbl.TextSize = unit.Sp(13)
					lbl.Color = textMuted
					lbl.Alignment = text.Middle
					return lbl.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

				// Day of Week Header: MO TU WE TH FR SA SU
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					days := []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}
					var flexChildren []layout.FlexChild
					for _, d := range days {
						dayName := d
						flexChildren = append(flexChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body2(state.th, dayName)
							lbl.TextSize = unit.Sp(13)
							lbl.Font.Weight = font.Bold
							lbl.Color = textMuted
							lbl.Alignment = text.Middle
							return lbl.Layout(gtx)
						}))
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, flexChildren...)
				}),

				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),

				// Calendar Grid (up to 6 rows)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					var rowChildren []layout.FlexChild
					for row := 0; row < 6; row++ {
						r := row
						hasDaysInRow := false
						for col := 0; col < 7; col++ {
							idx := r*7 + col
							dayNum := idx - weekdayOffset + 1
							if dayNum >= 1 && dayNum <= daysInMonth {
								hasDaysInRow = true
								break
							}
						}
						if !hasDaysInRow {
							continue
						}

						rowChildren = append(rowChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							var colChildren []layout.FlexChild
							for col := 0; col < 7; col++ {
								c := col
								idx := r*7 + c
								dayNum := idx - weekdayOffset + 1
								if dayNum < 1 || dayNum > daysInMonth {
									colChildren = append(colChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Spacer{Height: unit.Dp(26)}.Layout(gtx)
									}))
								} else {
									d := dayNum
									colChildren = append(colChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{
											Top:    unit.Dp(2),
											Bottom: unit.Dp(2),
											Left:   unit.Dp(2),
											Right:  unit.Dp(2),
										}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											isToday := state.pickerYear == todayYear && state.pickerMonth == todayMonth && d == todayDay
											isSelected := state.pickerYear == selectedYear && state.pickerMonth == selectedMonth && d == selectedDay

											btn := material.Button(state.th, &state.calendarDayBtns[idx], fmt.Sprintf("%d", d))
											btn.TextSize = unit.Sp(14)
											btn.CornerRadius = unit.Dp(4)
											btn.Inset = layout.Inset{
												Top:    unit.Dp(6),
												Bottom: unit.Dp(6),
												Left:   unit.Dp(2),
												Right:  unit.Dp(2),
											}

											if isToday {
												btn.Background = accentRed
												btn.Color = textPrimary
												btn.Font.Weight = font.Bold
											} else if isSelected {
												btn.Background = accentGreen
												btn.Color = bgDark
												btn.Font.Weight = font.Bold
											} else {
												btn.Background = cardBg
												btn.Color = textPrimary
												btn.Font.Weight = font.Normal
											}
											return btn.Layout(gtx)
										})
									}))
								}
							}
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, colChildren...)
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rowChildren...)
				}),
			)
		})
	})
}

func renderGithubConnectView(gtx layout.Context, state *UIState) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header Banner
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(14),
					Bottom: unit.Dp(14),
					Left:   unit.Dp(18),
					Right:  unit.Dp(18),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.bunnyTh, bunnyASCII)
							lbl.TextSize = unit.Sp(16)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							h := material.H5(state.th, "CONNECT GITHUB")
							h.TextSize = unit.Sp(22)
							h.Color = textPrimary
							return h.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.githubBackBtn, "← Back")
							btn.TextSize = unit.Sp(18)
							btn.Background = inputBg
							btn.Color = textPrimary
							return btn.Layout(gtx)
						}),
					)
				})
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

		// Content Card - Streamlined PAT Setup
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(24),
					Bottom: unit.Dp(24),
					Left:   unit.Dp(24),
					Right:  unit.Dp(24),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h := material.H6(state.th, "CONNECT WITH PERSONAL ACCESS TOKEN")
							h.TextSize = unit.Sp(18)
							h.Color = accentGreen
							return h.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

						// Prominent Big Red Repository Requirement Warning
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							req := material.Body1(state.th, "REQUIRED: You MUST create a private repo named '.dotdo' on your GitHub account!")
							req.Font.Weight = font.Bold
							req.TextSize = unit.Sp(16)
							req.Color = accentRed
							return req.Layout(gtx)
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.openPatUrlBtn, "Open GitHub PAT Generator Page")
							btn.TextSize = unit.Sp(16)
							btn.Background = inputBg
							btn.Color = accentOrange
							return btn.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							tip := material.Body2(state.th, "Note: GitHub defaults Expiration to 30 days. Be sure to change\nExpiration to 'No expiration' or '1 year' so sync keeps working!")
							tip.TextSize = unit.Sp(13)
							tip.Color = accentOrange
							return tip.Layout(gtx)
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

						// Token Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Personal Access Token (PAT):")
							lbl.TextSize = unit.Sp(16)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(state.th, &state.patTokenEditor, "Paste ghp_... or github_pat_... token here")
									e.TextSize = unit.Sp(16)
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

						// PAT Connect Button
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.patConnectBtn, "Save & Connect GitHub")
							btn.TextSize = unit.Sp(18)
							btn.Background = accentGreen
							btn.Color = bgDark
							return btn.Layout(gtx)
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),

						// Status / Info message
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if state.githubMsg == "" {
								return layout.Dimensions{}
							}
							lbl := material.Body2(state.th, state.githubMsg)
							lbl.TextSize = unit.Sp(14)
							lbl.Color = accentOrange
							return lbl.Layout(gtx)
						}),
					)
				})
			})
		}),
	)
}

func renderCard(gtx layout.Context, bg color.NRGBA, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			rr := gtx.Dp(unit.Dp(8))
			clipOp := clip.RRect{
				Rect: image.Rectangle{Max: gtx.Constraints.Min},
				SE:   rr, SW: rr, NW: rr, NE: rr,
			}.Op(gtx.Ops)
			paint.FillShape(gtx.Ops, bg, clipOp)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(w),
	)
}

func renderTaskRowCard(gtx layout.Context, state *UIState, t task.Task) layout.Dimensions {
	return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(12),
			Bottom: unit.Dp(12),
			Left:   unit.Dp(16),
			Right:  unit.Dp(16),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Interactive Status Full Circle Dot Button (todo -> doing -> done -> todo)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					sBtn, exists := state.statusBtns[t.ID]
					if !exists {
						sBtn = new(widget.Clickable)
						state.statusBtns[t.ID] = sBtn
					}

					label := "● TODO"
					dotColor := accentGreen

					switch t.Status {
					case "doing":
						label = "● DOING"
						dotColor = accentOrange
					case "done":
						label = "● DONE"
						dotColor = accentRed
					}

					statusBtn := material.Button(state.th, sBtn, label)
					statusBtn.TextSize = unit.Sp(16)
					statusBtn.Background = inputBg
					statusBtn.Color = dotColor
					return statusBtn.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),

				// Task Title & Due Date
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, t.Title)
							lbl.TextSize = unit.Sp(20)
							if t.Status == "done" {
								lbl.Color = textMuted
								return layout.Stack{}.Layout(gtx,
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return lbl.Layout(gtx)
									}),
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										lineThickness := gtx.Dp(unit.Dp(2))
										if lineThickness < 1 {
											lineThickness = 1
										}
										yCenter := gtx.Constraints.Min.Y / 2
										yMin := yCenter - lineThickness/2
										yMax := yMin + lineThickness
										lineRect := image.Rect(0, yMin, gtx.Constraints.Min.X, yMax)
										paint.FillShape(gtx.Ops, textMuted, clip.Rect(lineRect).Op())
										return layout.Dimensions{Size: gtx.Constraints.Min}
									}),
								)
							}
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if t.Due.IsZero() {
								return layout.Dimensions{}
							}
							dateStr := t.Due.Format("Jan 02, 2006")
							isExpired := t.Status != "done" && time.Now().After(t.Due)

							dueText := fmt.Sprintf("Due: %s", dateStr)
							if isExpired {
								dueText = fmt.Sprintf("EXPIRED: %s", dateStr)
							}

							dueLbl := material.Body2(state.th, dueText)
							dueLbl.TextSize = unit.Sp(16)
							if isExpired {
								dueLbl.Color = accentRed
							} else {
								dueLbl.Color = textMuted
							}
							return dueLbl.Layout(gtx)
						}),
					)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),

				// ID Badge
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					badge := material.Body2(state.th, fmt.Sprintf("#%d", t.ID))
					badge.TextSize = unit.Sp(16)
					badge.Color = textMuted
					return badge.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),

				// Remove Button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn, exists := state.delBtns[t.ID]
					if !exists {
						btn = new(widget.Clickable)
						state.delBtns[t.ID] = btn
					}
					delBtn := material.Button(state.th, btn, "×")
					delBtn.TextSize = unit.Sp(18)
					delBtn.Background = inputBg
					delBtn.Color = textMuted
					return delBtn.Layout(gtx)
				}),
			)
		})
	})
}
