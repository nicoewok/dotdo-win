package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"strings"
	"time"

	"gioui.org/app"
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
	"github.com/nicoewok/dotdo-win/pkg/task"
)

//go:embed assets/DotGothic16.ttf
var dotFontData []byte

var (
	bgDark       = color.NRGBA{R: 14, G: 14, B: 18, A: 255}
	cardBg       = color.NRGBA{R: 24, G: 24, B: 30, A: 255}
	inputBg      = color.NRGBA{R: 35, G: 35, B: 45, A: 255}
	accentRed    = color.NRGBA{R: 235, G: 75, B: 75, A: 255}
	accentGreen  = color.NRGBA{R: 85, G: 215, B: 130, A: 255}
	accentOrange = color.NRGBA{R: 255, G: 170, B: 50, A: 255}
	textPrimary  = color.NRGBA{R: 240, G: 240, B: 245, A: 255}
	textMuted    = color.NRGBA{R: 150, G: 150, B: 165, A: 255}
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
	svc         *service.Service
	th          *material.Theme
	activeView  string // "list" or "add"
	addBtn      widget.Clickable
	backBtn     widget.Clickable
	syncBtn     widget.Clickable
	purgeBtn    widget.Clickable
	submitBtn   widget.Clickable
	cancelBtn   widget.Clickable
	syncMsg     string
	titleEditor widget.Editor
	dueEditor   widget.Editor
	delBtns     map[int]*widget.Clickable
	statusBtns  map[int]*widget.Clickable
}

func main() {
	svc := service.NewService()
	if err := svc.Init(); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	go func() {
		w := new(app.Window)
		w.Option(app.Title("DOT ● DO - Task Manager"), app.Size(unit.Dp(520), unit.Dp(720)))

		uiState := &UIState{
			svc:        svc,
			activeView: "list",
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
	if faces, err := opentype.ParseCollection(dotFontData); err == nil && len(faces) > 0 {
		state.th.Shaper = text.NewShaper(text.WithCollection(faces))
	}
	state.th.Bg = bgDark
	state.th.Fg = textPrimary
	state.th.ContrastBg = accentRed

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
				state.titleEditor.SetText("")
				state.dueEditor.SetText("")
			}

			if state.backBtn.Clicked(gtx) || state.cancelBtn.Clicked(gtx) {
				state.activeView = "list"
			}

			if state.syncBtn.Clicked(gtx) {
				state.syncMsg = "Syncing..."
				go func() {
					_ = state.svc.Sync()
					state.syncMsg = "Synced"
					w.Invalidate()
				}()
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

			tasks, _ := state.svc.ListTasks(false)

			// Handle Status Cycle & Delete Button Clicks
			for _, t := range tasks {
				// Status Toggle Click (todo -> doing -> done -> todo)
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
					tasks, _ = state.svc.ListTasks(false)
					break
				}

				// Delete Click
				dBtn, dExists := state.delBtns[t.ID]
				if !dExists {
					dBtn = new(widget.Clickable)
					state.delBtns[t.ID] = dBtn
				}
				if dBtn.Clicked(gtx) {
					_, _ = state.svc.DeleteTaskByID(t.ID)
					tasks, _ = state.svc.ListTasks(false)
					break
				}
			}

			// Render Active View
			layout.Inset{
				Top:    unit.Dp(16),
				Bottom: unit.Dp(16),
				Left:   unit.Dp(18),
				Right:  unit.Dp(18),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if state.activeView == "add" {
					return renderAddView(gtx, state)
				}
				return renderListView(gtx, state, tasks, &listLayout)
			})

			e.Frame(gtx.Ops)
		}
	}
}

func renderListView(gtx layout.Context, state *UIState, tasks []task.Task, listLayout *layout.List) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// 1. Header Banner
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(12),
					Bottom: unit.Dp(12),
					Left:   unit.Dp(16),
					Right:  unit.Dp(16),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Bunny Art
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(state.th, bunnyASCII)
							lbl.Color = accentRed
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
						// App Title
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									h := material.H5(state.th, "DOT ● DO")
									h.Color = textPrimary
									return h.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									sub := material.Caption(state.th, fmt.Sprintf("● %s", state.syncMsg))
									sub.Color = textMuted
									return sub.Layout(gtx)
								}),
							)
						}),
						// Action Buttons: + Add & Sync
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.addBtn, "+ Add")
									btn.Background = accentRed
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.syncBtn, "Sync")
									btn.Background = inputBg
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

		// 2. Section Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Overline(state.th, fmt.Sprintf("TASKS (%d) - Click status button to toggle (todo → doing → done)", len(tasks)))
			lbl.Color = textMuted
			return lbl.Layout(gtx)
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),

		// 3. Task List
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(tasks) == 0 {
				return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top:    unit.Dp(24),
						Bottom: unit.Dp(24),
						Left:   unit.Dp(16),
						Right:  unit.Dp(16),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(state.th, "No tasks found. Click '+ Add' to create one!")
						lbl.Color = textMuted
						return lbl.Layout(gtx)
					})
				})
			}

			return listLayout.Layout(gtx, len(tasks), func(gtx layout.Context, index int) layout.Dimensions {
				t := tasks[index]
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return renderTaskRowCard(gtx, state, t)
				})
			})
		}),

		// 4. Footer Status Bar with Remove Done Button
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			pendingCount := 0
			doneCount := 0
			for _, t := range tasks {
				if t.Status == "done" {
					doneCount++
				} else {
					pendingCount++
				}
			}

			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Caption(state.th, fmt.Sprintf("● %d Pending  |  ✔ %d Completed", pendingCount, doneCount))
					lbl.Color = textMuted
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if doneCount == 0 {
						return layout.Dimensions{}
					}
					btn := material.Button(state.th, &state.purgeBtn, "Remove Done")
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
					Top:    unit.Dp(12),
					Bottom: unit.Dp(12),
					Left:   unit.Dp(16),
					Right:  unit.Dp(16),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(state.th, bunnyASCII)
							lbl.Color = accentRed
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							h := material.H5(state.th, "DOT ● DO")
							h.Color = textPrimary
							return h.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(state.th, &state.backBtn, "← Back")
							btn.Background = inputBg
							btn.Color = textPrimary
							return btn.Layout(gtx)
						}),
					)
				})
			})
		}),

		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

		// Add Task Form Card
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderCard(gtx, cardBg, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(24),
					Bottom: unit.Dp(24),
					Left:   unit.Dp(20),
					Right:  unit.Dp(20),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							h := material.H6(state.th, "+ CREATE NEW TASK")
							h.Color = accentRed
							return h.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

						// Title Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Task Title:")
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
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

						// Due Date Field
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, "Due Date (YYYY-MM-DD, optional):")
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
									e := material.Editor(state.th, &state.dueEditor, "e.g. 2026-12-31")
									e.Color = textPrimary
									return e.Layout(gtx)
								})
							})
						}),

						layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),

						// Submit & Cancel Action Buttons
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.cancelBtn, "Cancel")
									btn.Background = inputBg
									btn.Color = textPrimary
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(state.th, &state.submitBtn, "Create Task")
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
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
			Left:   unit.Dp(14),
			Right:  unit.Dp(14),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Interactive Status Button (cycles: todo -> doing -> done -> todo)
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
						label = "◐ DOING"
						dotColor = accentOrange
					case "done":
						label = "✔ DONE"
						dotColor = textMuted
					}

					statusBtn := material.Button(state.th, sBtn, label)
					statusBtn.Background = inputBg
					statusBtn.Color = dotColor
					return statusBtn.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),

				// Task Title & Due Date
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(state.th, t.Title)
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
							dueLbl := material.Caption(state.th, fmt.Sprintf("Due: %s", dateStr))
							dueLbl.Color = textMuted
							return dueLbl.Layout(gtx)
						}),
					)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),

				// ID Badge
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					badge := material.Caption(state.th, fmt.Sprintf("#%d", t.ID))
					badge.Color = textMuted
					return badge.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),

				// Remove Button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn, exists := state.delBtns[t.ID]
					if !exists {
						btn = new(widget.Clickable)
						state.delBtns[t.ID] = btn
					}
					delBtn := material.Button(state.th, btn, "✖")
					delBtn.Background = inputBg
					delBtn.Color = accentRed
					return delBtn.Layout(gtx)
				}),
			)
		})
	})
}
