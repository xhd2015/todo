package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/todo/models"
)

// ExportData represents the structure for exporting entries
type ExportData struct {
	Entries []ExportEntry `json:"entries"`
}

// ExportEntry represents a single entry with its notes for export
type ExportEntry struct {
	Data  *models.LogEntry `json:"data"`
	Notes []ExportNote     `json:"notes"`
}

// ExportNote represents a single note for export
type ExportNote struct {
	Data *models.Note `json:"data"`
}

// ExportVisibleEntries exports the currently visible entries to a JSON file
func ExportVisibleEntries(filename string, visibleEntries []TreeEntry) error {
	// Validate filename
	if strings.TrimSpace(filename) == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	// Create export data structure
	exportData := ExportData{
		Entries: make([]ExportEntry, 0, len(visibleEntries)),
	}

	// Convert visible entries to export format
	for _, wrapperEntry := range visibleEntries {
		if wrapperEntry.Type == models.LogEntryViewType_Log && wrapperEntry.Log != nil {
			entry := wrapperEntry.Entry

			exportEntry := ExportEntry{
				Data:  entry.Data,
				Notes: make([]ExportNote, 0, len(entry.Notes)),
			}

			// Add notes
			for _, note := range entry.Notes {
				exportEntry.Notes = append(exportEntry.Notes, ExportNote{
					Data: note.Data,
				})
			}

			exportData.Entries = append(exportData.Entries, exportEntry)
		}
		// Note: We could also export notes separately if needed in the future
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
