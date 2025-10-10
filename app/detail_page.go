package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/models"
)

type ChildNotesSection struct {
	Entry *models.LogEntryView
	Path  string
	Notes []*models.NoteView
}

func collectChildrenNotes(entry *models.LogEntryView, path []string) []ChildNotesSection {
	var sections []ChildNotesSection

	var traverse func(*models.LogEntryView, []string)
	traverse = func(e *models.LogEntryView, currentPath []string) {
		if len(e.Notes) > 0 {
			pathStr := strings.Join(currentPath, " / ")
			sections = append(sections, ChildNotesSection{
				Entry: e,
				Path:  pathStr,
				Notes: e.Notes,
			})
		}

		for _, child := range e.Children {
			childPath := append(currentPath, child.Data.Text)
			traverse(child, childPath)
		}
	}

	for _, child := range entry.Children {
		childPath := append(path, child.Data.Text)
		traverse(child, childPath)
	}

	return sections
}

func DetailPage(state *State, id int64) *dom.Node {
	item := state.Entries.Get(id)
	if item == nil {
		return dom.Text(fmt.Sprintf("not found: %d", id))
	}

	return dom.Div(dom.DivProps{
		OnKeyDown: func(e *dom.DOMEvent) {
			keyEvent := e.KeydownEvent
			switch keyEvent.KeyType {
			case dom.KeyTypeEsc:
				if len(state.Routes) > 0 {
					state.Routes.Pop()
					e.StopPropagation()
				}
			}
		},
	},
		dom.Div(dom.DivProps{}, dom.Text(item.Data.Text)),

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
				val := strings.TrimSpace(value)
				if val == "" {
					return true
				}
				state.Enqueue(func(ctx context.Context) error {
					return state.OnAddNote(item.Data.ID, val)
				})
				return true
			},
		}),

		// children notes
		func() *dom.Node {
			childrenSections := collectChildrenNotes(item, []string{})

			if len(childrenSections) == 0 {
				return dom.Fragment()
			}

			var children []*dom.Node
			children = append(children, dom.H2(dom.DivProps{}, dom.Text("Children Notes")))

			for _, section := range childrenSections {
				id := section.Entry.Data.ID
				selected := item.DetailPage.SelectedChildEntryID == id

				children = append(children, dom.TextWithProps(section.Path+":", dom.TextNodeProps{
					Focused:   selected,
					Focusable: true,
					Style: styles.Style{
						Color: func() string {
							if selected {
								return colors.GREEN_SUCCESS
							}
							return ""
						}(),
					},
					OnFocus: func() {
						item.DetailPage.SelectedChildEntryID = id
					},
					OnBlur: func() {
						item.DetailPage.SelectedChildEntryID = 0
					},
					OnKeyDown: func(d *dom.DOMEvent) {
						keyEvent := d.KeydownEvent
						switch keyEvent.KeyType {
						case dom.KeyTypeEnter:
							nextItem := state.Entries.Get(id)
							if nextItem != nil {
								state.Routes.Push(DetailRoute(id))
								nextItem.DetailPage.InputState.Reset()
							}
						}
					},
				}))
				children = append(children, dom.Br())

				var noteNodes []*dom.Node
				for _, note := range section.Notes {
					noteNodes = append(noteNodes, dom.Li(dom.ListItemProps{}, dom.Text("  - "+note.Data.Text)))
				}
				children = append(children, dom.Ul(dom.DivProps{}, noteNodes...))
				children = append(children, dom.Br())
			}

			return dom.Div(dom.DivProps{}, children...)
		}(),
	)
}
