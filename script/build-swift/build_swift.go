package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/xgo/support/cmd"
)

func main() {
	err := handle(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handle(args []string) error {
	return handleBuildSwift(args)
}

func handleBuildSwift(args []string) error {
	fmt.Println("Building todo-sticker-swift...")
	fmt.Println("==============================")

	var fullLog bool
	n := len(args)
	for i := 0; i < n; i++ {
		arg := args[i]
		if arg == "--full-log" {
			fullLog = true
			continue
		}
	}

	// Build the macOS app using xcodebuild
	// xcodebuild -project todo-sticker.xcodeproj -scheme todo-sticker -destination 'platform=macOS' build
	baseCmd := `xcodebuild -project todo-sticker.xcodeproj -scheme todo-sticker -destination 'platform=macOS' build 2>&1`

	if !fullLog {
		// Filter output to show only important messages
		baseCmd = baseCmd + ` | grep -B 3 -A 10 -E "(error|failed|BUILD SUCCEEDED|BUILD FAILED)" | tail -60`
	}

	err := cmd.Debug().Dir("todo-sticker-swift").Run("bash", "-c", baseCmd)
	if err != nil {
		return err
	}

	fmt.Println("")
	fmt.Println("Build todo-sticker-swift successfully.")
	return nil
}
