package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/models/states"
)

func MainInput(state *State, fullEntries []states.TreeEntry) *dom.Node {
	placeholder := "add todo"
	if state.IsSearchActive {
		placeholder = "search todos (ESC to exit search)"
	}

	return SearchInput(InputProps{
		Placeholder: placeholder,
		State:       &state.Input,
		OnKeyDown: func(event *dom.DOMEvent) bool {
			keyEvent := event.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeUp:
				if state.LastSelectedEntry.ID != 0 {
					if state.LastSelectedEntry.EntryType == models.LogEntryViewType_Log {
						state.Select(state.LastSelectedEntry.EntryType, state.LastSelectedEntry.ID)
					}
					state.LastSelectedEntry = models.EntryIdentity{}
					state.Input.Focused = false
					event.PreventDefault()
				}
			case dom.KeyTypeEsc:
				if state.IsSearchActive {
					state.ClearSearch()
					return true
				}
				if state.FocusedEntry.ID != 0 {
					state.FocusedEntry = models.EntryIdentity{}
					return true
				}
			case dom.KeyTypeCtrlC:
				if state.IsSearchActive {
					state.ClearSearch()
					event.PreventDefault()
					event.StopPropagation()
				}
			}
			return false
		},
		OnEnter: func(s string) bool {
			state.StatusBar.Error = ""
			if strings.TrimSpace(s) == "" {
				return false
			}

			// Check if there's an ongoing submission
			if state.SubmitState.IsSubmitting() {
				state.StatusBar.Error = "submission in progress, please wait..."
				return false
			}

			// Handle search mode
			if state.IsSearchActive {
				state.IsSearchActive = false
				state.SearchQuery = ""

				// if any entry is match, select the first one
				if len(fullEntries) > 0 {
					// first match
					var foundID int64
					for _, wrapperEntry := range fullEntries {
						if wrapperEntry.Type == models.LogEntryViewType_Log && wrapperEntry.Log != nil {
							if len(wrapperEntry.Entry.MatchTexts) > 0 {
								foundID = wrapperEntry.Entry.Data.ID
								break
							}
						}
					}
					if foundID == 0 {
						// Find first log entry
						for _, wrapperEntry := range fullEntries {
							if wrapperEntry.Type == models.LogEntryViewType_Log && wrapperEntry.Log != nil {
								foundID = wrapperEntry.Entry.Data.ID
								break
							}
						}
					}
					state.Select(models.LogEntryViewType_Log, foundID)
					state.SelectFromSource = states.SelectedSource_Search
				}
				return true
			}

			if s == "exit" || s == "quit" || s == "q" {
				state.Quit()
				return true
			}

			// Handle special commands starting with /
			if strings.HasPrefix(s, "/") {
				// Handle /export command with filename
				if filename, found := strings.CutPrefix(s, "/export "); found {
					filename = strings.TrimSpace(filename)
					if filename == "" {
						state.StatusBar.Error = "export requires a filename: /export <filename.json>"
						return true
					}

					// Export visible entries
					err := ExportVisibleEntries(filename, fullEntries)
					if err != nil {
						state.StatusBar.Error = fmt.Sprintf("export failed: %v", err)
					} else {
						state.StatusBar.Error = fmt.Sprintf("exported %d entries to %s", len(fullEntries), filename)
					}
					return true
				}

				switch s {
				case "/history":
					// Toggle ShowHistory and refresh entries
					wasShowingHistory := state.ShowHistory
					state.ShowHistory = !state.ShowHistory

					// If we're turning off history mode, reset all v states
					if wasShowingHistory && !state.ShowHistory {
						state.ResetAllChildrenVisibility()
					}

					if state.RefreshEntries != nil {
						state.Enqueue(func(ctx context.Context) error {
							return state.RefreshEntries(ctx)
						})
					}
					return true
				case "/note", "/notes":
					// Toggle ShowNotes (global notes mode)
					state.ShowNotes = !state.ShowNotes
					return true
				case "/reload", "/refresh":
					if state.RefreshEntries != nil {
						state.Enqueue(func(ctx context.Context) error {
							return state.RefreshEntries(ctx)
						})
					}
					return true
				case "/zen":
					state.ZenMode = !state.ZenMode
					return true
				case "/expandall":
					state.ExpandAll = !state.ExpandAll
					return true
				case "/config":
					// show config page
					configState := loadConfigPageState()
					state.Routes.Push(states.ConfigRoute(configState))
					return true
				case "/h", "/happening":
					// Set loading state and navigate to happening list page
					state.Happening.Loading = true
					state.Happening.Error = ""
					state.Happening.Happenings = nil

					state.Routes.Push(states.HappeningListRoute())

					// Start async loading of happening data
					state.Enqueue(func(ctx context.Context) error {
						if state.Happening.LoadHappenings == nil {
							state.Happening.Error = "LoadHappenings is not set"
							return nil
						}
						happenings, err := state.Happening.LoadHappenings(ctx)
						if err != nil {
							state.Happening.Error = err.Error()
							return err
						}
						// Update the state with loaded data
						state.Happening.Loading = false
						state.Happening.Happenings = happenings
						state.Happening.SelectedItemIndex = len(happenings) - 1
						return nil
					})

					return true
				case "/hstat":
					// Load state on first access
					if state.HumanState.LoadStateOnce != nil {
						state.HumanState.LoadStateOnce()
					}
					// Navigate to human states page
					state.Routes.Push(states.HumanStateRoute())
					return true
				case "/help":
					// Navigate to help page
					state.Routes.Push(states.HelpRoute())
					return true
				case "/learning":
					// Load materials on first access
					if state.Learning.LoadMaterialsOnce != nil {
						state.Learning.LoadMaterialsOnce()
					}
					// Navigate to learning materials page
					state.Routes.Push(states.LearningRoute())
					return true
				case "/switch":
					// Toggle view mode between default and group
					if state.ViewMode == states.ViewMode_Default {
						state.ViewMode = states.ViewMode_Group
					} else {
						state.ViewMode = states.ViewMode_Default
					}
					return true
				default:
					// Unknown command, do nothing
					state.StatusBar.Error = "unknown command: " + s
					return true
				}
			}

			// Log the content before attempting to add it
			log.Infof(context.Background(), "attempting to add todo: %s", s)

			state.Enqueue(func(ctx context.Context) error {
				return state.OnAdd(ctx, models.LogEntryViewType_Log, s)
			})
			return true
		},
		OnSearchChange: func(query string) {
			state.SearchQuery = query
		},
		OnSearchActivate: func() {
			state.IsSearchActive = true
		},
		OnSearchDeactivate: func() {
			state.IsSearchActive = false
			state.SearchQuery = ""
		},
	})
}
