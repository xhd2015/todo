package run

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/models"
	"golang.org/x/term"
)

const listHelp = `
list

`

func handleList(args []string) error {
	var storageType string = "sqlite" // default to sqlite
	args, err := flags.String("--storage", &storageType).
		Help("-h,--help", listHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra argument: %s", strings.Join(args, " "))
	}

	logManager, err := CreateLogManager(storageType)
	if err != nil {
		return err
	}

	err = logManager.Init()
	if err != nil {
		return err
	}

	strikethroughStyle := lipgloss.NewStyle().Strikethrough(true)
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	// Recursive function to render entries with proper indentation
	var renderEntryRecursive func(entry *models.EntryView, depth int)
	renderEntryRecursive = func(entry *models.EntryView, depth int) {
		// Create indentation (2 spaces per depth level)
		indent := strings.Repeat("  ", depth)

		// Choose bullet based on completion status
		bullet := "•"
		if entry.Data.Done {
			bullet = "✓"
		}

		// Apply styling
		text := entry.Data.Text
		if entry.Data.Done && isTTY {
			text = strikethroughStyle.Render(text)
		}

		// Print the entry with indentation
		fmt.Printf("%s%s %s\n", indent, bullet, text)

		// Recursively render children
		for _, child := range entry.Children {
			renderEntryRecursive(child, depth+1)
		}
	}

	// Render only top-level entries (ParentID == 0) and their children
	for _, entry := range logManager.Entries {
		if entry.Data.ParentID == 0 {
			renderEntryRecursive(entry, 0)
		}
	}

	return nil
}
