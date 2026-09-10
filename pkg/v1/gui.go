package v1

import (
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RunGUI starts the local desktop GUI.
func RunGUI(t *Tester) {
	myApp := app.New()
	myWindow := myApp.NewWindow("Integration Tests")

	// --- Shared State ---
	var (
		logs   []LogEntry
		logsMu sync.Mutex

		// Map StageName -> Status String
		stageStatus = make(map[string]string)
		statusMu    sync.Mutex
	)

	// Initialize status
	for _, s := range t.Stages {
		stageStatus[s.Name] = "Not Run"
	}

	// --- Left Pane: Stage & Action Tree ---
	var leftTree *widget.Tree
	leftTree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				// Root: Return Stage Names
				names := make([]string, len(t.Stages))
				for i, s := range t.Stages {
					names[i] = s.Name
				}
				return names
			}
			// Check if uid is a Stage Name
			// If so, return Action IDs: "StageName:Index"
			actions := GetStageActions(uid)
			if len(actions) > 0 {
				ids := make([]string, len(actions))
				for i := range actions {
					ids[i] = fmt.Sprintf("%s:%d", uid, i)
				}
				return ids
			}
			return nil
		},
		func(uid widget.TreeNodeID) bool {
			// Stages are branches, Actions are leaves
			if uid == "" {
				return true
			}
			if strings.Contains(uid, ":") {
				return false
			} // Action
			return true // Stage
		},
		func(branch bool) fyne.CanvasObject {
			// Template for Node
			return container.NewHBox(
				widget.NewLabel("Template"), // Name/Summary
				layout.NewSpacer(),
				canvas.NewText("Status", theme.ForegroundColor()), // Status (Stage only)
				widget.NewButton("Run", func() {}),                // Run Button
			)
		},
		func(uid widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			statusText := box.Objects[2].(*canvas.Text)
			btn := box.Objects[3].(*widget.Button)

			// Determine if Stage or Action
			if strings.Contains(uid, ":") {
				// Action: "StageName:Index"
				parts := strings.SplitN(uid, ":", 2)
				stageName := parts[0]
				idx, _ := strconv.Atoi(parts[1])

				actions := GetStageActions(stageName)
				if idx < len(actions) {
					action := actions[idx]
					label.SetText("  " + action.Summary) // Indent
					label.TextStyle = fyne.TextStyle{Italic: true}
					statusText.Text = "" // No status for actions
					statusText.Refresh()

					btn.SetText("Run")
					btn.OnTapped = func() {
						go func() {
							defer func() {
								if r := recover(); r != nil {
									var errMsg string
									if te, ok := r.(TestError); ok {
										errMsg = te.Message
									} else {
										errMsg = fmt.Sprintf("%v", r)
									}
									Log(LogTypeInfo, "Manual Run FAILED: "+action.Summary, errMsg)
									fyne.Do(func() {
										dialog.ShowError(fmt.Errorf("Execution Failed: %s", errMsg), myWindow)
									})
								}
							}()

							// Run individual action
							// We don't update stage status for individual run
							Log(LogTypeInfo, "Manual Run: "+action.Summary, "")
							action.Func()
							Log(LogTypeInfo, "Manual Run PASSED: "+action.Summary, "")
						}()
					}
					btn.Show()
				}
			} else {
				// Stage
				stageName := uid
				label.SetText(stageName)
				label.TextStyle = fyne.TextStyle{Bold: true}

				statusMu.Lock()
				st := stageStatus[stageName]
				statusMu.Unlock()

				statusText.Text = st
				if st == "PASSED" {
					statusText.Color = color.NRGBA{R: 0, G: 180, B: 0, A: 255}
				} else if strings.HasPrefix(st, "FAILED") {
					statusText.Color = color.NRGBA{R: 200, G: 0, B: 0, A: 255}
				} else {
					statusText.Color = theme.ForegroundColor()
				}
				statusText.Refresh()

				btn.SetText("Run Stage")
				btn.OnTapped = func() {
					go func() {
						// Run all preceding stages that haven't passed yet
						for _, s := range t.Stages {
							name := s.Name
							statusMu.Lock()
							st := stageStatus[name]
							statusMu.Unlock()
							if name == stageName {
								break
							}
							if st == "PASSED" {
								continue
							}
							fyne.Do(func() {
								statusMu.Lock()
								stageStatus[name] = "Running..."
								statusMu.Unlock()
								leftTree.Refresh()
							})
							err := t.RunStageByName(name)
							fyne.Do(func() {
								statusMu.Lock()
								if err != nil {
									stageStatus[name] = "FAILED"
								} else {
									stageStatus[name] = "PASSED"
								}
								statusMu.Unlock()
								leftTree.Refresh()
							})
							if err != nil {
								return // Stop if a prerequisite stage failed
							}
						}

						fyne.Do(func() {
							statusMu.Lock()
							stageStatus[stageName] = "Running..."
							statusMu.Unlock()
							leftTree.Refresh()
						})
						err := t.RunStageByName(stageName)
						fyne.Do(func() {
							statusMu.Lock()
							if err != nil {
								stageStatus[stageName] = "FAILED"
							} else {
								stageStatus[stageName] = "PASSED"
							}
							statusMu.Unlock()
							leftTree.Refresh()
						})
					}()
				}
				btn.Show()
			}
		},
	)

	// Helper: discover actions without executing real operations (dry-run)
	runDiscoverActions := func() {
		go func() {
			defer func() { recover() }()
			log.Println("Discovering actions via dry-run...")
			t.DryRunAll()
			fyne.Do(func() { leftTree.Refresh() })
		}()
	}

	// --- Right Pane: Log Tree ---
	// Structure: Root -> Stage Logs -> Operation Logs (Click for Popup)
	var rightTree *widget.Tree
	rightTree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			logsMu.Lock()
			defer logsMu.Unlock()

			if uid == "" {
				// Root nodes: stage logs as branches, plus any logs before the first stage
				var ids []string
				sawStage := false
				for i, l := range logs {
					if l.Type == LogTypeStage {
						sawStage = true
						ids = append(ids, fmt.Sprintf("%d", i))
					} else if !sawStage {
						ids = append(ids, fmt.Sprintf("%d", i))
					}
				}
				return ids
			}

			// If uid is a number, it's a log index
			idx, err := strconv.Atoi(uid)
			if err != nil {
				return nil
			}

			if idx >= len(logs) {
				return nil
			}
			parentLog := logs[idx]

			// Level 1: Stage -> Operations
			if parentLog.Type == LogTypeStage {
				var children []string
				// Scan forward until next Stage log
				for i := idx + 1; i < len(logs); i++ {
					l := logs[i]
					if l.Type == LogTypeStage {
						break
					}
					children = append(children, fmt.Sprintf("%d", i))
				}
				return children
			}

			return nil
		},
		func(uid widget.TreeNodeID) bool {
			// Check if branch. Stage logs are branches; root entries before any stage are leaves.
			logsMu.Lock()
			defer logsMu.Unlock()

			if uid == "" {
				return true
			}

			idx, err := strconv.Atoi(uid)
			if err != nil || idx >= len(logs) {
				return false
			}

			l := logs[idx]
			if l.Type == LogTypeStage {
				return true
			}

			// Non-stage logs that are roots (before first stage) are leaves
			return false
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Log Entry")
		},
		func(uid widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			label := o.(*widget.Label)

			idx, _ := strconv.Atoi(uid)
			logsMu.Lock()
			if idx >= len(logs) {
				logsMu.Unlock()
				return
			}
			entry := logs[idx]
			logsMu.Unlock()

			// Icons
			icon := "🔹"
			isStage := false
			switch entry.Type {
			case LogTypeStage:
				icon = "📂"
				isStage = true
			case LogTypeDB:
				icon = "🛢️"
			case LogTypeRedis:
				icon = "🧊"
			case LogTypeRequest:
				icon = "🌍"
			case LogTypeMock:
				icon = "🤖"
			case LogTypeApp:
				icon = "⚙️"
			case LogTypeExpect:
				icon = "🎯"
			case LogTypeError:
				icon = "❌"
			case LogTypeInfo:
				icon = "ℹ️"
			}

			text := fmt.Sprintf("%s %s", icon, entry.Summary)
			if !isStage {
				text = "   " + text
			}
			label.SetText(text)

			if isStage {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
			label.Wrapping = fyne.TextWrapBreak
		},
	)

	var lastClickID string
	var lastClickTime time.Time

	rightTree.OnSelected = func(uid widget.TreeNodeID) {
		// Double Click Logic
		now := time.Now()
		if uid != lastClickID || now.Sub(lastClickTime) > 500*time.Millisecond {
			lastClickID = uid
			lastClickTime = now
			rightTree.Unselect(uid)
			return
		}
		lastClickID = "" // Reset

		idx, err := strconv.Atoi(uid)
		if err != nil {
			rightTree.Unselect(uid)
			return
		}

		logsMu.Lock()
		if idx >= len(logs) {
			logsMu.Unlock()
			rightTree.Unselect(uid)
			return
		}
		entry := logs[idx]
		logsMu.Unlock()

		// Create independent window (non-modal)
		detailWin := myApp.NewWindow(entry.Summary)

		content := widget.NewMultiLineEntry()
		detailText := entry.Detail
		if detailText == "" {
			detailText = "(No details available)"
		}

		fullText := fmt.Sprintf("Summary: %s\n\n%s", entry.Summary, detailText)
		content.SetText(fullText)
		content.Wrapping = fyne.TextWrapWord
		// Enabled to fix grey text issue

		// Scrollable container
		scroll := container.NewScroll(content)

		detailWin.SetContent(scroll)
		detailWin.Resize(fyne.NewSize(600, 400))
		detailWin.Show()

		rightTree.Unselect(uid)
	}

	// --- Handlers ---

	// On Action Update (New operation recorded) -> Refresh Left Tree
	RegisterActionUpdateHandler(func() {
		// We need to refresh the node of the current stage
		// Finding the UID of current stage is tricky without passing it.
		// But refreshing Root might be enough?
		// No, RefreshItem("") refreshes structure.
		// We know 'currentStage' in tester, but not here.
		// Let's just refresh root structure.
		fyne.Do(func() {
			if leftTree != nil {
				leftTree.Refresh()
			}
		})
	})

	// On Log Update -> Refresh Right Tree
	RegisterLogHandler(func(entry LogEntry) {
		logsMu.Lock()
		logs = append(logs, entry)
		logsMu.Unlock()
		fyne.Do(func() {
			if rightTree != nil {
				rightTree.Refresh()
			}
		})
		// Auto-scroll? Tree doesn't support easy auto-scroll to bottom.
	})

	// Layout
	runAllBtn := widget.NewButton("Run All", func() {
		go func() {
			for _, s := range t.Stages {
				name := s.Name
				fyne.Do(func() {
					statusMu.Lock()
					stageStatus[name] = "Running..."
					statusMu.Unlock()
					leftTree.Refresh()
				})
				err := t.RunStageByName(name)
				fyne.Do(func() {
					statusMu.Lock()
					if err != nil {
						stageStatus[name] = "FAILED"
					} else {
						stageStatus[name] = "PASSED"
					}
					statusMu.Unlock()
					leftTree.Refresh()
				})
			}
		}()
	})
	stageHeader := container.NewBorder(nil, nil, nil,
		container.NewHBox(runAllBtn, widget.NewButton("Refresh Actions", func() {
			runDiscoverActions()
		})),
		widget.NewLabelWithStyle("Test Stages", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	split := container.NewHSplit(
		container.NewBorder(stageHeader, nil, nil, nil, leftTree),
		container.NewBorder(widget.NewLabelWithStyle("Operation Logs", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), nil, nil, nil, rightTree),
	)
	split.SetOffset(0.35)

	myWindow.SetContent(split)
	myWindow.Resize(fyne.NewSize(1000, 700))

	// Pre-populate actions via dry-run discovery. This avoids executing real operations.
	runDiscoverActions()

	log.Println("Starting GUI window...")
	myWindow.ShowAndRun()
}
