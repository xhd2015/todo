package app

import (
	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/emojis"
)

// AppState renders the application status bar
func AppStatusBar(state *State) *dom.Node {
	// Build status bar nodes
	var nodes []*dom.Node

	// Left side: dot, storage, error, requesting
	nodes = append(nodes, dom.Text("•", styles.Style{
		Bold:  true,
		Color: colors.GREEN_SUCCESS,
	}))
	if state.StatusBar.Storage != "" {
		nodes = append(nodes, dom.Text(state.StatusBar.Storage, styles.Style{
			Bold:  true,
			Color: colors.GREY_TEXT,
		}))
	}
	if state.StatusBar.Error != "" {
		nodes = append(nodes, dom.Text("  "+state.StatusBar.Error, styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		}))
	}
	if state.Requesting() {
		nodes = append(nodes, dom.Text("  •", styles.Style{
			Bold:  true,
			Color: colors.GREEN_SUCCESS,
		}))
		nodes = append(nodes, dom.Text("Request...", styles.Style{
			Bold:  true,
			Color: colors.GREEN_SUCCESS,
		}))
	}

	// Spacer to push modes to the right
	hasRightContent := state.ZenMode || state.ShowHistory || state.ShowNotes || state.FocusedEntry.IsSet() || state.ViewMode != ViewMode_Default
	if hasRightContent {
		nodes = append(nodes, dom.Spacer(dom.WithMaxSize(40)))

		// Right side: modes
		var modeCount int
		if state.FocusedEntry.IsSet() {
			nodes = append(nodes, dom.Text(emojis.FOCUSED, styles.Style{
				Bold: true,
			}))
			modeCount++
		}
		if state.ZenMode {
			if modeCount > 0 {
				nodes = append(nodes, dom.Text(" ", styles.Style{}))
			}
			nodes = append(nodes, dom.Text("zen", styles.Style{
				Bold:  true,
				Color: colors.GREY_TEXT,
			}))
			modeCount++
		}
		if state.ShowHistory {
			if modeCount > 0 {
				nodes = append(nodes, dom.Text(" ", styles.Style{}))
			}
			nodes = append(nodes, dom.Text("history", styles.Style{
				Bold:  true,
				Color: colors.GREY_TEXT,
			}))
			modeCount++
		}
		if state.ShowNotes {
			if modeCount > 0 {
				nodes = append(nodes, dom.Text(" ", styles.Style{}))
			}
			nodes = append(nodes, dom.Text("notes", styles.Style{
				Bold:  true,
				Color: colors.GREY_TEXT,
			}))
			modeCount++
		}
		if state.ViewMode != ViewMode_Default {
			if modeCount > 0 {
				nodes = append(nodes, dom.Text(" ", styles.Style{}))
			}
			nodes = append(nodes, dom.Text("group:on", styles.Style{
				Bold:  true,
				Color: "cyan",
			}))
		}
	}

	return dom.HDiv(dom.DivProps{Width: UIWidth}, nodes...)
}
