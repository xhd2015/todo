package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
)

func main() {
	err := handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handle(args []string) error {
	var fullLog bool

	args, err := flags.Bool("--full-log", &fullLog).Parse(args)
	if err != nil {
		return err
	}

	fmt.Println("Building and running todo-sticker-swift...")
	fmt.Println("==========================================")

	swiftProjectDir := "todo-sticker-swift"

	// Build the project
	fmt.Println("Building...")
	buildCmd := `xcodebuild -project todo-sticker.xcodeproj -scheme todo-sticker -destination 'platform=macOS' build 2>&1`
	if !fullLog {
		buildCmd = buildCmd + ` | grep -B 3 -A 10 -E "(error|failed|BUILD SUCCEEDED|BUILD FAILED)" | tail -60`
	}

	err = cmd.Debug().Dir(swiftProjectDir).Run("bash", "-c", buildCmd)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Find the built app
	buildDir := filepath.Join(swiftProjectDir, "build/Release")
	appPath := filepath.Join(buildDir, "todo-sticker.app")

	// Check if app exists in build directory, if not try DerivedData
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		// Try to find in DerivedData, excluding Index.noindex directories
		homeDir, _ := os.UserHomeDir()
		derivedDataPath := filepath.Join(homeDir, "Library/Developer/Xcode/DerivedData")
		findCmd := fmt.Sprintf(`find "%s" -name "todo-sticker.app" -type d | grep -v "Index.noindex" | head -1`, derivedDataPath)

		result, err := cmd.Dir(".").Output("bash", "-c", findCmd)
		if err != nil || len(result) == 0 {
			return fmt.Errorf("could not find built app. Try running from Xcode instead")
		}
		appPath = strings.TrimSpace(string(result))
	}

	fmt.Printf("Running app from: %s\n", appPath)

	// Run the app
	err = cmd.Debug().Run("open", appPath)
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	fmt.Println("")
	fmt.Println("App launched successfully!")
	fmt.Println("Note: If you see permission errors, try running from Xcode instead.")

	return nil
}
