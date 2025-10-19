package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/xgo/support/cmd"
)

func main() {
	// Load .env file (defaults to ".env" in the current directory)
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found") // Or handle as fatal if required
	}

	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

const help = `
setup-lifelog - Setup a lifelog project

Options:
  --lifelog-project <project>  The project to setup
  -h,--help                    Show this help message

Examples:
	go run ./script/setup-lifelog
	go run ./script/setup-lifelog merge-into-master
`

func Handle(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "merge-into-master":
			return mergeIntoMaster(args[1:])
		}
	}

	var lifelogProject string
	// "github.com/xhd2015/less-gen/flags"
	args, err := flags.
		String("--lifelog-project", &lifelogProject).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra args: %s", strings.Join(args, " "))
	}

	if lifelogProject == "" {
		lifelogProject = os.Getenv("LIFELOG_PROJECT")
	}
	if lifelogProject == "" {
		return fmt.Errorf("requires --lifelog-project or LIFELOG_PROJECT env")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pwdLifelog := filepath.Join(pwd, "lifelog")

	stat, _ := os.Stat(pwdLifelog)
	if stat != nil {
		return fmt.Errorf("lifelog already exists")
	}

	err = cmd.Dir(lifelogProject).Run("git", "worktree", "add", pwdLifelog)
	if err != nil {
		return err
	}

	return nil
}

func mergeIntoMaster(args []string) error {
	// Parse arguments to get lifelog project path
	var lifelogProject string
	_, err := flags.
		String("--lifelog-project", &lifelogProject).
		Parse(args)
	if err != nil {
		return err
	}

	if lifelogProject == "" {
		lifelogProject = os.Getenv("LIFELOG_PROJECT")
	}
	if lifelogProject == "" {
		return fmt.Errorf("requires --lifelog-project or LIFELOG_PROJECT env")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pwdLifelog := filepath.Join(pwd, "lifelog")

	// Check if $PWD/lifelog exists
	if _, err := os.Stat(pwdLifelog); os.IsNotExist(err) {
		return fmt.Errorf("lifelog directory does not exist at %s", pwdLifelog)
	}

	// Get current branch in $PWD/lifelog
	currentBranch, err := getCurrentBranch(pwdLifelog)
	if err != nil {
		return fmt.Errorf("failed to get current branch in %s: %w", pwdLifelog, err)
	}

	// Check if worktree is clean in $PWD/lifelog
	if !isWorktreeClean(pwdLifelog) {
		return fmt.Errorf("worktree is not clean in %s", pwdLifelog)
	}

	// this file is insane
	err = cmd.Dir(lifelogProject).Run("git", "restore", "swift/lifelog.xcodeproj/project.xcworkspace/xcuserdata/xhd2015.xcuserdatad/UserInterfaceState.xcuserstate")
	if err != nil {
		return fmt.Errorf("failed to restore UserInterfaceState.xcuserstate: %w", err)
	}

	// Check if worktree is clean in lifelogProject
	if !isWorktreeClean(lifelogProject) {
		return fmt.Errorf("worktree is not clean in %s", lifelogProject)
	}

	// Ensure the branch is master in lifelogProject
	masterBranch, err := getCurrentBranch(lifelogProject)
	if err != nil {
		return fmt.Errorf("failed to get current branch in %s: %w", lifelogProject, err)
	}
	if masterBranch != "master" {
		return fmt.Errorf("lifelogProject is not on master branch, current branch: %s", masterBranch)
	}

	// Do git merge with the branch from $PWD/lifelog
	fmt.Printf("Merging branch '%s' from %s into master at %s\n", currentBranch, pwdLifelog, lifelogProject)
	err = cmd.Dir(lifelogProject).Stdin(os.Stdin).Run("git", "merge", currentBranch)
	if err != nil {
		return fmt.Errorf("failed to merge branch %s: %w", currentBranch, err)
	}

	fmt.Printf("Successfully merged branch '%s' into master\n", currentBranch)

	// print follow-up commands:

	pushOriginCmd := fmt.Sprintf("cd %q && git push origin master", lifelogProject)
	removeWorktreeCmd := fmt.Sprintf("cd %q && git worktree remove \"$PWD\"", pwdLifelog)
	removeBranchCmd := fmt.Sprintf("cd %q && git branch -D %s", lifelogProject, currentBranch)

	fmt.Printf("You can now push origin to remote: (%s)\n", pushOriginCmd)
	fmt.Printf("Then remove the worktree: (%s)\n", removeWorktreeCmd)
	fmt.Printf("Finally remove the branch: (%s)\n", removeBranchCmd)

	fmt.Printf("Combined commands: \n")
	cmds := []string{pushOriginCmd, removeWorktreeCmd, removeBranchCmd}
	for _, cmd := range cmds {
		fmt.Printf("  (%s)\n", cmd)
	}
	return nil
}

// getCurrentBranch returns the current git branch name for the given directory
func getCurrentBranch(dir string) (string, error) {
	output, err := cmd.Dir(dir).Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// isWorktreeClean checks if the git worktree is clean (no uncommitted changes)
func isWorktreeClean(dir string) bool {
	output, err := cmd.Dir(dir).Output("git", "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == ""
}
