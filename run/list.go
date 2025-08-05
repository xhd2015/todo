package run

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xhd2015/less-gen/flags"
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
	for _, entry := range logManager.Entries {
		text := entry.Text
		if entry.Done && isTTY {
			text = strikethroughStyle.Render(text)
		}
		fmt.Printf("- %s\n", text)
	}

	return nil
}
