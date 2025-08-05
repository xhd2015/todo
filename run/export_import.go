package run

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/less-gen/flags"
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
	Text       string       `json:"text"`
	Done       bool         `json:"done"`
	CreateTime time.Time    `json:"create_time"`
	UpdateTime time.Time    `json:"update_time"`
	Notes      []ExportNote `json:"notes"`
}

type ExportNote struct {
	Text       string    `json:"text"`
	CreateTime time.Time `json:"create_time"`
	UpdateTime time.Time `json:"update_time"`
}

func handleExport(args []string) error {
	var storageType string = "sqlite"

	args, err := flags.String("--storage", &storageType).
		Help("-h,--help", exportHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("export requires exactly one argument: <json_file>")
	}

	jsonFile := args[0]

	logManager, err := CreateLogManager(storageType)
	if err != nil {
		return err
	}

	err = logManager.Init()
	if err != nil {
		return err
	}

	exportData := ExportData{
		Entries: make([]ExportEntry, 0, len(logManager.Entries)),
	}

	for _, entry := range logManager.Entries {
		exportEntry := ExportEntry{
			Text:       entry.Text,
			Done:       entry.Done,
			CreateTime: entry.CreateTime,
			UpdateTime: entry.UpdateTime,
			Notes:      make([]ExportNote, 0, len(entry.Notes)),
		}

		for _, note := range entry.Notes {
			exportEntry.Notes = append(exportEntry.Notes, ExportNote{
				Text:       note.Text,
				CreateTime: note.CreateTime,
				UpdateTime: note.UpdateTime,
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
	var storageType string = "sqlite"

	args, err := flags.String("--storage", &storageType).
		Help("-h,--help", importHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("import requires exactly one argument: <json_file>")
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

	logManager, err := CreateLogManager(storageType)
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
		existingTexts[strings.TrimSpace(entry.Text)] = true
	}

	imported := 0
	skipped := 0

	for _, importEntry := range importData.Entries {
		trimmedText := strings.TrimSpace(importEntry.Text)
		if existingTexts[trimmedText] {
			skipped++
			continue
		}

		// Add the entry
		entryID, err := logManager.LogEntryService.Add(models.LogEntry{
			Text:       importEntry.Text,
			Done:       importEntry.Done,
			CreateTime: importEntry.CreateTime,
			UpdateTime: importEntry.UpdateTime,
		})
		if err != nil {
			return fmt.Errorf("failed to add entry: %w", err)
		}

		// Add notes
		for _, note := range importEntry.Notes {
			_, err := logManager.LogNoteService.Add(entryID, models.Note{
				Text:       note.Text,
				CreateTime: note.CreateTime,
				UpdateTime: note.UpdateTime,
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
