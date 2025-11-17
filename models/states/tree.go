package states

import "github.com/xhd2015/todo/models"

// TreeEntry wraps either a log entry or a note for unified tree rendering
type TreeEntry struct {
	Type   models.LogEntryViewType
	Prefix string
	IsLast bool

	// for all
	Entry *models.LogEntryView

	Log         *TreeLog
	Note        *TreeNote // TODO: remove
	FocusedItem *TreeFocusedItem
	Group       *TreeGroup
}

// TreeLog represents a flattened log entry
type TreeLog struct {
}

// TreeNote represents a flattened note
type TreeNote struct {
	Note    *models.NoteView
	EntryID int64 // ID of the entry that owns this note
}

// TreeFocusedItem represents the focused root path
type TreeFocusedItem struct {
	RootPath []string
}

// TreeGroup represents a group entry
type TreeGroup struct {
	ID   int64
	Name string
}

func (c *TreeEntry) Text() string {
	switch c.Type {
	case models.LogEntryViewType_Log:
		if c.Entry != nil && c.Entry.Data != nil {
			return c.Entry.Data.Text
		}
		return ""
	case models.LogEntryViewType_Note:
		if c.Entry != nil && c.Entry.Data != nil {
			return c.Entry.Data.Text
		}
		// TODO: remove this
		return c.Note.Note.Data.Text
	case models.LogEntryViewType_FocusedItem:
		if c.FocusedItem != nil && len(c.FocusedItem.RootPath) > 0 {
			// Join the path components with " > " separator
			result := c.FocusedItem.RootPath[0]
			for i := 1; i < len(c.FocusedItem.RootPath); i++ {
				result += " > " + c.FocusedItem.RootPath[i]
			}
			return result
		}
		return ""
	case models.LogEntryViewType_Group:
		if c.Group != nil {
			return c.Group.Name
		}
		return ""
	}
	return ""
}
