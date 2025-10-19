package app

import (
	"context"
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/happening_list"
	"github.com/xhd2015/todo/app/human_state"
	"github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
)

type RouteType int

const (
	RouteType_Main RouteType = iota
	RouteType_Detail
	RouteType_Config
	RouteType_HappeningList
	RouteType_HumanState
)

type Routes []Route

type Route struct {
	Type              RouteType
	MainPage          *MainPageState
	DetailPage        *DetailPageState
	ConfigPage        *ConfigPageState
	HappeningListPage *HappeningListPageState
	HumanStatePage    *HumanStatePageState
}

func (routes *Routes) Push(route Route) {
	*routes = append(*routes, route)
}

func (routes *Routes) Pop() {
	*routes = (*routes)[:len(*routes)-1]
}

func (routes *Routes) Last() Route {
	return (*routes)[len(*routes)-1]
}

type MainPageState struct {
	Entries []TreeEntry
}

type DetailPageState struct {
	EntryID int64
}

type ConfigPhase int

const (
	ConfigPhase_PickingStorageType ConfigPhase = iota
	ConfigPhase_PickingStorageDetail
)

type StorageType int

const (
	StorageType_LocalFile StorageType = iota
	StorageType_LocalSqlite
	StorageType_Server
)

type ConfigPageState struct {
	ConfigPhase ConfigPhase

	SelectedStorageType StorageType
	PickingStorageType  StorageType

	ServerAddr      models.InputState
	ServerAuthToken models.InputState

	ConfirmButtonFocused bool
	CancelButtonFocused  bool
}

type HappeningListPageState struct {
	// This can be empty since happening state is now in main State
}

type HumanStatePageState struct {
	// This can be empty since human state is now in main State
}

func DetailRoute(entryID int64) Route {
	return Route{
		Type: RouteType_Detail,
		DetailPage: &DetailPageState{
			EntryID: entryID,
		},
	}
}

func ConfigRoute(state ConfigPageState) Route {
	return Route{
		Type:       RouteType_Config,
		ConfigPage: &state,
	}
}

func HappeningListRoute() Route {
	return Route{
		Type:              RouteType_HappeningList,
		HappeningListPage: &HappeningListPageState{},
	}
}

func HumanStateRoute() Route {
	return Route{
		Type:           RouteType_HumanState,
		HumanStatePage: &HumanStatePageState{},
	}
}

// HappeningListPage renders the happening list page
func HappeningListPage(state *State) *dom.Node {
	happeningState := &state.Happening

	if happeningState.Loading {
		return dom.Div(dom.DivProps{},
			dom.Text("Loading happenings..."),
		)
	}

	if happeningState.Error != "" {
		return dom.Div(dom.DivProps{},
			dom.Text("Error loading happenings: "+happeningState.Error),
		)
	}

	return happening_list.HappeningList(happening_list.HappeningListProps{
		Items:         happeningState.Happenings,
		FocusedItemID: happeningState.FocusedItemID,
		OnFocusItem: func(id int64) {
			happeningState.FocusedItemID = id
		},
		OnBlurItem: func(id int64) {
			happeningState.FocusedItemID = 0
		},
		InputState: &happeningState.Input,
		OnNavigateBack: func() {
			// Navigate back to main page by popping the current route
			state.Routes.Pop()
		},
		OnAddHappening: func(text string) {
			// Add new happening using backend API
			state.Enqueue(func(ctx context.Context) error {
				if happeningState.AddHappening == nil {
					return fmt.Errorf("AddHappening function not available")
				}

				// Add via backend service
				newHappening, err := happeningState.AddHappening(ctx, text)
				if err != nil {
					return fmt.Errorf("failed to add happening: %w", err)
				}

				// Add to local list for immediate UI update
				happeningState.Happenings = append(happeningState.Happenings, newHappening)
				return nil
			})
		},
		OnReload: func() {
			// Reload happenings by setting loading state and fetching fresh data
			if len(happeningState.Happenings) == 0 {
				happeningState.Loading = true
			}
			happeningState.Error = ""

			state.Enqueue(func(ctx context.Context) error {
				log.Infof(ctx, "Reload happenings")
				if happeningState.LoadHappenings == nil {
					happeningState.Error = "LoadHappenings is not set"
					return nil
				}
				happenings, err := happeningState.LoadHappenings(ctx)
				if err != nil {
					happeningState.Error = err.Error()
					return err
				}
				// Update the state with loaded data
				happeningState.Loading = false
				happeningState.Happenings = happenings
				return nil
			})
		},
		// Edit/Delete functionality
		EditingItemID:       happeningState.EditingItemID,
		EditInputState:      &happeningState.EditInputState,
		DeletingItemID:      happeningState.DeletingItemID,
		DeleteConfirmButton: happeningState.DeleteConfirmButton,
		OnEditItem: func(id int64) {
			// Find the happening to edit
			for _, happening := range happeningState.Happenings {
				if happening.ID == id {
					happeningState.EditingItemID = id
					happeningState.EditInputState.Value = happening.Content
					happeningState.EditInputState.Focused = true
					happeningState.EditInputState.CursorPosition = len(happening.Content)
					break
				}
			}
		},
		OnDeleteItem: func(id int64) {
			happeningState.DeletingItemID = id
			happeningState.DeleteConfirmButton = 0 // Default to Delete button
		},
		OnSaveEdit: func(id int64, content string) {
			// Update happening using backend API
			state.Enqueue(func(ctx context.Context) error {
				if happeningState.UpdateHappening == nil {
					return fmt.Errorf("UpdateHappening function not available")
				}

				// Create update with only the content field
				update := &models.HappeningOptional{
					Content: &content,
				}

				// Update via backend service
				updatedHappening, err := happeningState.UpdateHappening(ctx, id, update)
				if err != nil {
					return fmt.Errorf("update: %w", err)
				}

				// Update local list for immediate UI update
				for i, happening := range happeningState.Happenings {
					if happening.ID == id {
						happeningState.Happenings[i] = updatedHappening
						break
					}
				}

				// Reset edit state
				happeningState.EditingItemID = 0
				happeningState.EditInputState.Reset()
				return nil
			})
		},
		OnCancelEdit: func(e *dom.DOMEvent) {
			happeningState.EditingItemID = 0
			happeningState.EditInputState.Reset()
			if e != nil {
				e.StopPropagation()
			}
		},
		OnConfirmDelete: func(e *dom.DOMEvent, id int64) {
			// Delete happening using backend API
			state.Enqueue(func(ctx context.Context) error {
				if happeningState.DeleteHappening == nil {
					return fmt.Errorf("DeleteHappening function not available")
				}

				// Delete via backend service
				err := happeningState.DeleteHappening(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to delete happening: %w", err)
				}

				// Remove from local list for immediate UI update
				for i, happening := range happeningState.Happenings {
					if happening.ID == id {
						happeningState.Happenings = append(happeningState.Happenings[:i], happeningState.Happenings[i+1:]...)
						break
					}
				}

				// Reset delete state
				happeningState.DeletingItemID = 0
				return nil
			})
		},
		OnCancelDelete: func(e *dom.DOMEvent) {
			happeningState.DeletingItemID = 0
		},
		OnNavigateDeleteConfirm: func(direction int) {
			happeningState.DeleteConfirmButton += direction
			if happeningState.DeleteConfirmButton < 0 {
				happeningState.DeleteConfirmButton = 1
			}
			if happeningState.DeleteConfirmButton > 1 {
				happeningState.DeleteConfirmButton = 0
			}
		},
	})
}

// HumanStatePage renders the human state page
func HumanStatePage(state *State) *dom.Node {
	return human_state.HumanStatePage(
		state.HumanState,
		func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent != nil {
				if keyEvent.KeyType == dom.KeyTypeEsc {
					state.Routes.Pop()
					return
				}
				return
			}
		},
	)
}

func RenderRoute(state *State, route Route) *dom.Node {
	switch route.Type {
	case RouteType_Detail:
		return DetailPage(state, route.DetailPage.EntryID)
	case RouteType_Config:
		return ConfigPage(state)
	case RouteType_HappeningList:
		return HappeningListPage(state)
	case RouteType_HumanState:
		return HumanStatePage(state)
	default:
		return dom.Text(fmt.Sprintf("unknown route: %d", route.Type), styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		})
	}
}
