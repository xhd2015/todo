package run

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/internal/config"
	"github.com/xhd2015/todo/models"
)

const exportHelp = `
export <json_file>

Export all todos and their notes to a JSON file.
`

const importHelp = `
import <json_file>

Import todos and their notes from a JSON file.
Entries with the same text content will be skipped.
`

type ExportData struct {
	Entries []ExportEntry `json:"entries"`
}

type ExportEntry struct {
	Data  *models.LogEntry `json:"data"`
	Notes []ExportNote     `json:"notes"`
}

type ExportNote struct {
	Data *models.Note `json:"data"`
}

func handleExport(args []string) error {
	var storageType string
	var serverAddr string
	var serverToken string

	args, err := flags.String("--storage", &storageType).
		String("--server-addr", &serverAddr).
		String("--server-token", &serverToken).
		Help("-h,--help", exportHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("export requires exactly one argument: <json_file>")
	}

	// Apply config defaults
	storageConfig, err := ApplyConfigDefaults(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}
	storageType = storageConfig.StorageType
	serverAddr = storageConfig.ServerAddr
	serverToken = storageConfig.ServerToken

	// Validate server-addr is provided when storage type is server
	if storageType == "server" && serverAddr == "" {
		return fmt.Errorf("--server-addr is required when --storage=server")
	}

	jsonFile := args[0]

	logManager, _, err := CreateLogManager(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}

	// Initialize with history to export all data including completed todos
	err = logManager.InitWithHistory(true)
	if err != nil {
		return err
	}

	exportData := ExportData{
		Entries: make([]ExportEntry, 0, len(logManager.Entries)),
	}

	for _, entry := range logManager.Entries {
		exportEntry := ExportEntry{
			Data:  entry.Data,
			Notes: make([]ExportNote, 0, len(entry.Notes)),
		}

		for _, note := range entry.Notes {
			exportEntry.Notes = append(exportEntry.Notes, ExportNote{
				Data: note.Data,
			})
		}

		exportData.Entries = append(exportData.Entries, exportEntry)
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(jsonFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Exported %d entries to %s\n", len(exportData.Entries), jsonFile)
	return nil
}

func handleImport(args []string) error {
	var storageType string
	var serverAddr string
	var serverToken string

	args, err := flags.String("--storage", &storageType).
		String("--server-addr", &serverAddr).
		String("--server-token", &serverToken).
		Help("-h,--help", importHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("import requires exactly one argument: <json_file>")
	}

	// Apply config defaults
	storageConfig, err := ApplyConfigDefaults(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}
	storageType = storageConfig.StorageType
	serverAddr = storageConfig.ServerAddr
	serverToken = storageConfig.ServerToken

	// Validate server-addr is provided when storage type is server
	if storageType == "server" && serverAddr == "" {
		return fmt.Errorf("--server-addr is required when --storage=server")
	}

	jsonFile := args[0]

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var importData ExportData
	err = json.Unmarshal(data, &importData)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	logManager, _, err := CreateLogManager(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}

	err = logManager.Init()
	if err != nil {
		return err
	}

	// Create a set of existing entry texts for deduplication
	existingTexts := make(map[string]bool)
	for _, entry := range logManager.Entries {
		existingTexts[strings.TrimSpace(entry.Data.Text)] = true
	}

	// Map old entry IDs to new entry IDs for parent-child relationships
	oldToNewIDMap := make(map[int64]int64)

	// Sort entries to ensure parents are imported before children
	sortedEntries := make([]ExportEntry, 0, len(importData.Entries))
	entryMap := make(map[int64]ExportEntry)

	// Build entry map and identify root entries (no parent)
	var rootEntries []ExportEntry
	for _, entry := range importData.Entries {
		entryMap[entry.Data.ID] = entry
		if entry.Data.ParentID == 0 {
			rootEntries = append(rootEntries, entry)
		}
	}

	// Recursively add entries in parent-first order
	var addEntriesRecursively func(entries []ExportEntry)
	addEntriesRecursively = func(entries []ExportEntry) {
		for _, entry := range entries {
			sortedEntries = append(sortedEntries, entry)
			// Find children of this entry
			var children []ExportEntry
			for _, candidate := range importData.Entries {
				if candidate.Data.ParentID == entry.Data.ID {
					children = append(children, candidate)
				}
			}
			if len(children) > 0 {
				addEntriesRecursively(children)
			}
		}
	}

	addEntriesRecursively(rootEntries)

	imported := 0
	skipped := 0

	// Import entries in sorted order
	for _, importEntry := range sortedEntries {
		trimmedText := strings.TrimSpace(importEntry.Data.Text)
		if existingTexts[trimmedText] {
			skipped++
			continue
		}

		// Determine the new parent ID
		var newParentID int64
		if importEntry.Data.ParentID != 0 {
			if mappedParentID, exists := oldToNewIDMap[importEntry.Data.ParentID]; exists {
				newParentID = mappedParentID
			} else {
				// Parent not found, skip this entry or make it root
				newParentID = 0
			}
		}

		// Add the entry with all original fields preserved
		entryID, err := logManager.LogEntryService.Add(models.LogEntry{
			Text:            importEntry.Data.Text,
			Done:            importEntry.Data.Done,
			DoneTime:        importEntry.Data.DoneTime,
			CreateTime:      importEntry.Data.CreateTime,
			UpdateTime:      importEntry.Data.UpdateTime,
			AdjustedTopTime: importEntry.Data.AdjustedTopTime,
			HighlightLevel:  importEntry.Data.HighlightLevel,
			ParentID:        newParentID,
		})
		if err != nil {
			return fmt.Errorf("failed to add entry: %w", err)
		}

		// Map old ID to new ID
		oldToNewIDMap[importEntry.Data.ID] = entryID

		// Add notes with all original fields preserved
		for _, note := range importEntry.Notes {
			_, err := logManager.LogNoteService.Add(entryID, models.Note{
				Text:       note.Data.Text,
				CreateTime: note.Data.CreateTime,
				UpdateTime: note.Data.UpdateTime,
			})
			if err != nil {
				return fmt.Errorf("failed to add note: %w", err)
			}
		}

		imported++
		existingTexts[trimmedText] = true
	}

	fmt.Printf("Imported %d entries, skipped %d duplicates from %s\n", imported, skipped, jsonFile)
	return nil
}

func handleConfig(args []string) error {
	// print config path
	configPath, err := config.GetConfigJSONFile()
	if err != nil {
		return err
	}

	fmt.Println(configPath)

	return nil
}
