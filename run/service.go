package run

import (
	"fmt"

	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/data/storage/filestore"
	"github.com/xhd2015/todo/data/storage/http"
	"github.com/xhd2015/todo/data/storage/sqlite"
	"github.com/xhd2015/todo/internal/config"
)

func createLogServices(storageType string, serverAddr string, serverToken string) (*data.Services, error) {
	services := &data.Services{}

	switch storageType {
	case "sqlite":
		sqliteFile, err := config.GetSqliteFile()
		if err != nil {
			return nil, err
		}

		sqliteStore, err := sqlite.New(sqliteFile)
		if err != nil {
			return nil, err
		}

		services.LogEntry = &sqlite.LogEntrySQLiteStore{
			SQLiteStore: sqliteStore,
		}
		services.LogNote = &sqlite.LogNoteSQLiteStore{
			SQLiteStore: sqliteStore,
		}
		services.Happening = &sqlite.HappeningSQLiteStore{
			SQLiteStore: sqliteStore,
		}
		services.StateRecording = &sqlite.StateRecordingSQLiteStore{
			SQLiteStore: sqliteStore,
		}
	case "file":
		recordFile, err := config.GetRecordJSONFile()
		if err != nil {
			return nil, err
		}

		services.LogEntry, err = filestore.NewLogEntryService(recordFile)
		if err != nil {
			return nil, err
		}
		services.LogNote, err = filestore.NewLogNoteService(recordFile)
		if err != nil {
			return nil, err
		}
		services.Happening, err = filestore.NewHappeningService(recordFile)
		if err != nil {
			return nil, err
		}
		services.StateRecording, err = filestore.NewStateRecordingService(recordFile)
		if err != nil {
			return nil, err
		}
	case "server":
		if serverAddr == "" {
			return nil, fmt.Errorf("requires --server-addr")
		}
		if serverToken == "" {
			return nil, fmt.Errorf("requires --server-token")
		}

		client := http.NewClient(serverAddr, serverToken)
		services.LogEntry = http.NewLogEntryService(client)
		services.LogNote = http.NewLogNoteService(client)
		services.Happening = http.NewHappeningService(client)
		services.StateRecording = http.NewStateRecordingService(client)
		services.LearningMaterials = http.NewLearningMaterialsService(client)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s, available: sqlite, file, server", storageType)
	}

	return services, nil
}

func CreateLogManager(storageType string, serverAddr string, serverToken string) (*data.LogManager, *data.Services, error) {
	services, err := createLogServices(storageType, serverAddr, serverToken)
	if err != nil {
		return nil, nil, err
	}

	logManager := data.NewLogManager(services)
	return logManager, services, nil
}
