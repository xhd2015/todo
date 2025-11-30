package run

import (
	"context"

	"github.com/xhd2015/todo/app"
	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/models"
)

func HandleToggleCollapsed(ctx context.Context, appState *app.State, logManager *data.LogManager, entryType models.LogEntryViewType, id int64) error {
	// clear search selected entry
	if entryType == models.LogEntryViewType_Log {
		err := logManager.ToggleCollapsed(id)
		if err != nil {
			return err
		}

		appState.Entries = logManager.Entries
		return nil
	} else if entryType == models.LogEntryViewType_Group {
		// in memory group toggle
		appState.GroupCollapseState.Toggle(id)
		return nil
	}
	return nil
}
