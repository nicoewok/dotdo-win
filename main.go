package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/opentype"
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
	activeView         string // "list" or "add"
	showDone           bool   // toggle to show or hide completed tasks
	isSyncDropdownOpen bool
	addBtn             widget.Clickable
	backBtn            widget.Clickable
	syncBtn            widget.Clickable
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
			app.Title("DOT ● DO - Task Manager"),
			fixedSize,
			app.MinSize(unit.Dp(560), unit.Dp(780)),
			app.MaxSize(unit.Dp(560), unit.Dp(780)),
		)

		uiState := &UIState{
			svc:        svc,
			activeView: "list",
			showDone:   true,
			delBtns:    make(map[int]*widget.Clickable),
			statusBtns: make(map[int]*widget.Clickable),
			syncMsg:    "Synced",
		}
		uiState.titleEditor.SingleLine = true
		uiState.dueEditor.SingleLine = true

		if err := run(w, uiState); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
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
			}

			if state.backBtn.Clicked(gtx) || state.cancelBtn.Clicked(gtx) {
				state.activeView = "list"
				state.isSyncDropdownOpen = false
			}

			if state.syncBtn.Clicked(gtx) {
				state.syncMsg = "Syncing..."
				state.isSyncDropdownOpen = false
				go func() {
					_ = state.svc.Sync()
					state.syncMsg = "Synced"
					w.Invalidate()
				}()
			}

			if state.syncDropdownBtn.Clicked(gtx) {
				state.isSyncDropdownOpen = !state.isSyncDropdownOpen
			}

			if state.githubBtn.Clicked(gtx) {
				state.isSyncDropdownOpen = false
			}

			if state.toggleDoneBtn.Clicked(gtx) {
				state.showDone = !state.showDone
			}

			if state.purgeBtn.Clicked(gtx) {
				_, _ = state.svc.RemoveDoneTasks()
			}

			if state.submitBtn.Clicked(gtx) {
				title := strings.TrimSpace(state.titleEditor.Text())
				if title != "" {
					var dueDate time.Time
					dueStr := strings.TrimSpace(state.dueEditor.Text())
					if dueStr != "" {
						if parsed, err := time.Parse("2006-01-02", dueStr); err == nil {
							dueDate = parsed
						}
					}
					_, _ = state.svc.AddTask(title, dueDate)
					state.activeView = "list"
				}
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
						if state.activeView == "add" {
							return renderAddView(gtx, state)
						}
						return renderListView(gtx, state, tasks, &listLayout)
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
						// Action Buttons: + Add & Attached Sync Dropdown (Sync ▼)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.addBtn, "+ Add")
									btn.TextSize = unit.Sp(18)
									btn.Background = accentRed
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),

								// Sync Attached Button Pair (Sync + ▼)
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := material.Button(state.th, &state.syncBtn, "Sync")
											btn.TextSize = unit.Sp(18)
											btn.Background = inputBg
											btn.Color = textPrimary
											return btn.Layout(gtx)
										}),
										layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
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

		layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),

		// Add Task Form Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(28),
					Bottom: unit.Dp(28),
					Left:   unit.Dp(24),
					Right:  unit.Dp(24),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h := material.H6(state.th, "+ CREATE NEW TASK")
							h.TextSize = unit.Sp(22)
							h.Color = accentRed
							return h.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),

						// Title Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Task Title:")
							lbl.TextSize = unit.Sp(18)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top:    unit.Dp(12),
									Bottom: unit.Dp(12),
									Left:   unit.Dp(14),
									Right:  unit.Dp(14),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(state.th, &state.titleEditor, "Enter task title...")
									e.TextSize = unit.Sp(18)
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),

						// Due Date Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Due Date (YYYY-MM-DD, optional):")
							lbl.TextSize = unit.Sp(18)
							lbl.Color = textPrimary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return renderCard(gtx, inputBg, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top:    unit.Dp(12),
									Bottom: unit.Dp(12),
									Left:   unit.Dp(14),
									Right:  unit.Dp(14),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(state.th, &state.dueEditor, "e.g. 2026-12-31")
									e.TextSize = unit.Sp(18)
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(28)}.Layout),

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
									btn.Background = accentRed
									btn.Color = textPrimary
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
							} else {
								lbl.Color = textPrimary
							}
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
					delBtn := material.Button(state.th, btn, "✖")
					delBtn.TextSize = unit.Sp(18)
					delBtn.Background = inputBg
					delBtn.Color = accentRed
					return delBtn.Layout(gtx)
				}),
			)
		})
	})
}
