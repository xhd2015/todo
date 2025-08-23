package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/data"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/tree"
	"golang.org/x/term"
)

const listHelp = `
list - Display todo entries in tree format

Options:
  --json                       Output raw JSON data instead of formatted tree
  --include <pattern>          Only include sub-trees containing the pattern (case-insensitive)
  --toggle <id>                Toggle visibility of all children (including history) for the specified entry ID
  --storage <type>             Storage backend: sqlite (default), file, or server
  --server-addr <addr>         Server address (required when --storage=server)
  --server-token <token>       Server authentication token (optional when --storage=server)
  -h,--help                    Show this help message

Examples:
  todo list                    Show all todos in tree format
  todo list --json            Output raw JSON data
  todo list --include "bug"    Show only sub-trees containing "bug"
  todo list --toggle 123      Show all children including history for entry ID 123
  todo list --json --include "feature"  Output JSON for entries containing "feature"
`

func handleList(args []string) error {
	var storageType string
	var serverAddr string
	var serverToken string
	var jsonOutput bool
	var includePattern string
	var showID bool
	var toggleID int64

	args, err := flags.String("--storage", &storageType).
		String("--server-addr", &serverAddr).
		String("--server-token", &serverToken).
		Bool("--json", &jsonOutput).
		String("--include", &includePattern).
		Bool("--show-id", &showID).
		Int("--toggle", &toggleID).
		Help("-h,--help", listHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra argument: %s", strings.Join(args, " "))
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

	logManager, err := CreateLogManager(storageType, serverAddr, serverToken)
	if err != nil {
		return err
	}

	err = logManager.Init()
	if err != nil {
		return err
	}

	// Handle toggle functionality if specified
	if toggleID != 0 {
		err = applyToggleExpansion(logManager, toggleID)
		if err != nil {
			return err
		}
	}

	// Apply pattern filtering if specified
	var filteredEntries []*models.LogEntryView
	if includePattern != "" {
		filteredEntries = filterEntriesByPattern(logManager.Entries, includePattern)
	} else {
		filteredEntries = logManager.Entries
	}

	// Handle JSON output
	if jsonOutput {
		return outputJSON(filteredEntries)
	}

	// Render the filtered entries using the extracted function
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	renderEntries(os.Stdout, isTTY, filteredEntries, showID)

	return nil
}

// renderEntries renders a list of entries with proper tree connectors
func renderEntries(out io.Writer, isTTY bool, entries []*models.LogEntryView, showID bool) {
	tree.RenderEntries(entries, func(prefix string, connector string, entry *models.LogEntryView) {
		io.WriteString(out, prefix+connector+tree.RenderItem(entry, showID, isTTY)+"\n")
	})
}

func RenderToString(entries []*models.LogEntryView, showID bool, simulateTTY bool) string {
	var b bytes.Buffer
	renderEntries(&b, simulateTTY, entries, showID)
	return b.String()
}

// filterEntriesByPattern filters entries to include only sub-trees that contain the pattern
// A sub-tree is included if the entry itself or any of its descendants contain the pattern
func filterEntriesByPattern(entries []*models.LogEntryView, pattern string) []*models.LogEntryView {
	pattern = strings.ToLower(pattern)

	// Helper function to check if an entry or any of its descendants contains the pattern
	var containsPattern func(entry *models.LogEntryView) bool
	containsPattern = func(entry *models.LogEntryView) bool {
		// Check if current entry contains the pattern
		if strings.Contains(strings.ToLower(entry.Data.Text), pattern) {
			return true
		}

		// Check if any child contains the pattern
		for _, child := range entry.Children {
			if containsPattern(child) {
				return true
			}
		}

		return false
	}

	// Filter entries that contain the pattern
	var filtered []*models.LogEntryView
	for _, entry := range entries {
		if containsPattern(entry) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// outputJSON outputs the entries as JSON
func outputJSON(entries []*models.LogEntryView) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

// applyToggleExpansion applies the toggle expansion to the specified entry ID
// This simulates the 'v' command being pressed on the entry
func applyToggleExpansion(logManager *data.LogManager, toggleID int64) error {
	targetEntry, err := logManager.Get(toggleID)
	if err != nil {
		return err
	}

	// Toggle history inclusion state (simulate 'v' command)
	targetEntry.IncludeHistory = !targetEntry.IncludeHistory

	if targetEntry.IncludeHistory {
		// Load all children including history
		ctx := context.Background()
		fullEntry, err := logManager.GetTree(ctx, toggleID, true)
		if err != nil {
			return fmt.Errorf("failed to load all children: %v", err)
		}

		// Replace the entry's children with the full loaded children
		targetEntry.Children = fullEntry.Children
		// Only the target entry should show the (*) indicator, not its children
		// Children should have IncludeHistory = false by default
	} else {
		// When hiding history, mark all children as not including history
		var setChildrenNoHistory func(entry *models.LogEntryView)
		setChildrenNoHistory = func(entry *models.LogEntryView) {
			entry.IncludeHistory = false
			for _, child := range entry.Children {
				setChildrenNoHistory(child)
			}
		}
		setChildrenNoHistory(targetEntry)
	}

	return nil
}
