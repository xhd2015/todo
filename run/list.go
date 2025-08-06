package run

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/todo/models"
	"github.com/xhd2015/todo/ui/tree"
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

	// Recursive function to render entries with proper tree connectors
	var renderEntryRecursive func(entry *models.LogEntryView, depth int, ancestorIsLast []bool)
	renderEntryRecursive = func(entry *models.LogEntryView, depth int, ancestorIsLast []bool) {
		// Build tree connector prefix using common utility
		treePrefix := tree.BuildTreePrefix(depth, ancestorIsLast)

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

		// Print the entry with tree connectors
		fmt.Printf("%s%s %s\n", treePrefix, bullet, text)

		// Recursively render children
		for childIndex, child := range entry.Children {
			isLastChild := (childIndex == len(entry.Children)-1)
			// Create ancestor info for child: copy parent's info and add current level
			childAncestorIsLast := make([]bool, depth+1)
			copy(childAncestorIsLast, ancestorIsLast)
			childAncestorIsLast[depth] = isLastChild
			renderEntryRecursive(child, depth+1, childAncestorIsLast)
		}
	}

	// Render only top-level entries (ParentID == 0) and their children
	for _, entry := range logManager.Entries {
		if entry.Data.ParentID == 0 {
			renderEntryRecursive(entry, 0, []bool{})
		}
	}

	return nil
}
