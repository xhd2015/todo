package run

import (
	"fmt"

	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/data/storage/filestore"
	"github.com/xhd2015/todo/data/storage/sqlite"
	"github.com/xhd2015/todo/internal/config"
)

func createLogServices(storageType string) (storage.LogEntryService, storage.LogNoteService, error) {
	var logEntryService storage.LogEntryService
	var logNoteService storage.LogNoteService

	switch storageType {
	case "sqlite":
		sqliteFile, err := config.GetSqliteFile()
		if err != nil {
			return nil, nil, err
		}

		sqliteStore, err := sqlite.New(sqliteFile)
		if err != nil {
			return nil, nil, err
		}

		logEntryService = &sqlite.LogEntrySQLiteStore{
			SQLiteStore: sqliteStore,
		}
		logNoteService = &sqlite.LogNoteSQLiteStore{
			SQLiteStore: sqliteStore,
		}
	case "file":
		recordFile, err := config.GetRecordJSONFile()
		if err != nil {
			return nil, nil, err
		}

		filestoreStore, err := filestore.New(recordFile)
		if err != nil {
			return nil, nil, err
		}
		logEntryService = &filestore.LogEntryFileStore{
			FileStore: filestoreStore,
		}
		logNoteService = &filestore.LogNoteFileStore{
			FileStore: filestoreStore,
		}

	default:
		return nil, nil, fmt.Errorf("unsupported storage type: %s, available: sqlite, file", storageType)
	}

	return logEntryService, logNoteService, nil
}

func CreateLogManager(storageType string) (*data.LogManager, error) {
	logEntryService, logNoteService, err := createLogServices(storageType)
	if err != nil {
		return nil, err
	}

	logManager := data.NewLogManager(logEntryService, logNoteService)
	return logManager, nil
}
