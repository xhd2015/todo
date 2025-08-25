package app

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

type RouteType int

const (
	RouteType_Main RouteType = iota
	RouteType_Detail
	RouteType_Config
)

type Routes []Route

type Route struct {
	Type       RouteType
	MainPage   *MainPageState
	DetailPage *DetailPageState
	ConfigPage *ConfigPageState
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

func RenderRoute(state *State, route Route) *dom.Node {
	switch route.Type {
	case RouteType_Detail:
		return DetailPage(state, route.DetailPage.EntryID)
	case RouteType_Config:
		return ConfigPage(state)
	default:
		return dom.Text(fmt.Sprintf("unknown route: %d", route.Type), styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		})
	}
}
