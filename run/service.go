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

func createLogServices(storageType string, serverAddr string, serverToken string) (storage.LogEntryService, storage.LogNoteService, storage.HappeningService, storage.StateRecordingService, error) {
	var logEntryService storage.LogEntryService
	var logNoteService storage.LogNoteService
	var happeningService storage.HappeningService
	var stateRecordingService storage.StateRecordingService

	switch storageType {
	case "sqlite":
		sqliteFile, err := config.GetSqliteFile()
		if err != nil {
			return nil, nil, nil, nil, err
		}

		sqliteStore, err := sqlite.New(sqliteFile)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		logEntryService = &sqlite.LogEntrySQLiteStore{
			SQLiteStore: sqliteStore,
		}
		logNoteService = &sqlite.LogNoteSQLiteStore{
			SQLiteStore: sqliteStore,
		}
		happeningService = &sqlite.HappeningSQLiteStore{
			SQLiteStore: sqliteStore,
		}
		stateRecordingService = &sqlite.StateRecordingSQLiteStore{
			SQLiteStore: sqliteStore,
		}
	case "file":
		recordFile, err := config.GetRecordJSONFile()
		if err != nil {
			return nil, nil, nil, nil, err
		}

		logEntryService, err = filestore.NewLogEntryService(recordFile)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		logNoteService, err = filestore.NewLogNoteService(recordFile)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		happeningService, err = filestore.NewHappeningService(recordFile)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		stateRecordingService, err = filestore.NewStateRecordingService(recordFile)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	case "server":
		if serverAddr == "" {
			return nil, nil, nil, nil, fmt.Errorf("requires --server-addr")
		}
		if serverToken == "" {
			return nil, nil, nil, nil, fmt.Errorf("requires --server-token")
		}

		client := http.NewClient(serverAddr, serverToken)
		logEntryService = http.NewLogEntryService(client)
		logNoteService = http.NewLogNoteService(client)
		happeningService = http.NewHappeningService(client)
		stateRecordingService = http.NewStateRecordingService(client)

	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported storage type: %s, available: sqlite, file, server", storageType)
	}

	return logEntryService, logNoteService, happeningService, stateRecordingService, nil
}

func CreateLogManager(storageType string, serverAddr string, serverToken string) (*data.LogManager, error) {
	logEntryService, logNoteService, happeningService, stateRecordingService, err := createLogServices(storageType, serverAddr, serverToken)
	if err != nil {
		return nil, err
	}

	logManager := data.NewLogManager(logEntryService, logNoteService, happeningService, stateRecordingService)
	return logManager, nil
}
