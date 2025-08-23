package run

import (
	"fmt"

	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/data/storage/filestore"
	"github.com/xhd2015/todo/data/storage/http"
	"github.com/xhd2015/todo/data/storage/sqlite"
	"github.com/xhd2015/todo/internal/config"
)

func createLogServices(storageType string, serverAddr string, serverToken string) (storage.LogEntryService, storage.LogNoteService, error) {
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

		logEntryService, err = filestore.NewLogEntryService(recordFile)
		if err != nil {
			return nil, nil, err
		}
		logNoteService, err = filestore.NewLogNoteService(recordFile)
		if err != nil {
			return nil, nil, err
		}
	case "server":
		if serverAddr == "" {
			return nil, nil, fmt.Errorf("requires --server-addr")
		}
		if serverToken == "" {
			return nil, nil, fmt.Errorf("requires --server-token")
		}

		client := http.NewClient(serverAddr, serverToken)
		logEntryService = http.NewLogEntryService(client)
		logNoteService = http.NewLogNoteService(client)

	default:
		return nil, nil, fmt.Errorf("unsupported storage type: %s, available: sqlite, file, server", storageType)
	}

	return logEntryService, logNoteService, nil
}

func CreateLogManager(storageType string, serverAddr string, serverToken string) (*data.LogManager, error) {
	logEntryService, logNoteService, err := createLogServices(storageType, serverAddr, serverToken)
	if err != nil {
		return nil, err
	}

	logManager := data.NewLogManager(logEntryService, logNoteService)
	return logManager, nil
}
