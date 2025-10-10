package app

import (
	"context"
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/happening_list"
	"github.com/xhd2015/todo/models"
)

type RouteType int

const (
	RouteType_Main RouteType = iota
	RouteType_Detail
	RouteType_Config
	RouteType_HappeningList
)

type Routes []Route

type Route struct {
	Type              RouteType
	MainPage          *MainPageState
	DetailPage        *DetailPageState
	ConfigPage        *ConfigPageState
	HappeningListPage *HappeningListPageState
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
	})
}

func RenderRoute(state *State, route Route) *dom.Node {
	switch route.Type {
	case RouteType_Detail:
		return DetailPage(state, route.DetailPage.EntryID)
	case RouteType_Config:
		return ConfigPage(state)
	case RouteType_HappeningList:
		return HappeningListPage(state)
	default:
		return dom.Text(fmt.Sprintf("unknown route: %d", route.Type), styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		})
	}
}
