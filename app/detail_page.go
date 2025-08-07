package app

import (
	"fmt"

	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/todo/models"
)

func DetailPage(state *State, id int64) *dom.Node {
	item := state.Entries.Get(id)
	if item == nil {
		return dom.Text(fmt.Sprintf("not found: %d", id))
	}

	return dom.Div(dom.DivProps{
		OnKeyDown: func(d *dom.DOMEvent) {
			keyEvent := d.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEsc:
				state.EnteredEntryID = 0
			}
		},
	},
		dom.Text(item.Data.Text),

		dom.H1(dom.DivProps{}, dom.Text("Notes")),

		func() *dom.Node {
			notes := item.Notes

			if len(notes) == 0 {
				return dom.Fragment(dom.Text("No notes"), dom.Br())
			}
			inputState := &item.DetailPage.EditInputState
			var children []*dom.Node
			for _, note := range notes {
				isSelected := item.DetailPage.SelectedNoteID == note.Data.ID

				if item.DetailPage.SelectedNoteMode == models.SelectedNoteMode_Editing && isSelected {
					children = append(children, dom.Input(dom.InputProps{
						Value:          inputState.Value,
						Focused:        inputState.Focused,
						CursorPosition: inputState.CursorPosition,
						OnCursorMove: func(position int) {
							inputState.CursorPosition = position
						},
						OnChange: func(value string) {
							inputState.Value = value
						},
						OnKeyDown: func(e *dom.DOMEvent) {
							keyEvent := e.KeydownEvent
							switch keyEvent.KeyType {
							case dom.KeyTypeUp, dom.KeyTypeDown:
								e.PreventDefault()
							case dom.KeyTypeEsc:
								item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Default
								e.StopPropagation()
							case dom.KeyTypeCtrlC:
								item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Default
								e.StopPropagation()
							case dom.KeyTypeEnter:
								state.OnUpdateNote(item.Data.ID, note.Data.ID, inputState.Value)
								item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Default
							}
						},
					}))
					continue
				}

				children = append(children, dom.Li(dom.ListItemProps{
					Selected: isSelected,
					Focused:  item.DetailPage.SelectedNoteMode == models.SelectedNoteMode_Default && isSelected,
					OnFocus: func() {
						item.DetailPage.SelectedNoteID = note.Data.ID
					},
					OnBlur: func() {
						item.DetailPage.SelectedNoteID = 0
					},
					Focusable: dom.Focusable(true),
					OnKeyDown: func(e *dom.DOMEvent) {
						keyEvent := e.KeydownEvent
						switch keyEvent.KeyType {
						default:
							key := string(keyEvent.Runes)
							switch key {
							case "e":
								item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Editing
								inputState.FocusWithText(note.Data.Text)
							case "d":
								item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Deleting
							}
						}
					},
				}, dom.Text(note.Data.Text)))

				if item.DetailPage.SelectedNoteMode == models.SelectedNoteMode_Deleting && isSelected {
					children = append(children, ConfirmDialog(ConfirmDialogProps{
						PromptText:     "Delete Note",
						DeleteText:     "[Delete]",
						CancelText:     "[Cancel]",
						SelectedButton: item.DetailPage.ConfirmDeleteButton,
						OnDelete: func() {
							state.OnDeleteNote(item.Data.ID, note.Data.ID)
							item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Default
						},
						OnCancel: func() {
							item.DetailPage.SelectedNoteMode = models.SelectedNoteMode_Default
						},
						OnNavigateRight: func() {
							item.DetailPage.ConfirmDeleteButton = 1
						},
						OnNavigateLeft: func() {
							item.DetailPage.ConfirmDeleteButton = 0
						},
					}))
				}
			}
			return dom.Ul(dom.DivProps{}, children...)
		}(),

		SearchInput(InputProps{
			Placeholder: "add note",
			State:       &item.DetailPage.InputState,
			OnEnter: func(value string) bool {
				state.OnAddNote(item.Data.ID, value)
				return true
			},
		}),
	)
}
