package process

import (
	"errors"
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

func ProcessExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	_, findErr := os.FindProcess(pid)
	if findErr != nil {
		return false, nil
	}

	return isProcessAlive(pid)
}

func isProcessAlive(pid int) (bool, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}
		return false, fmt.Errorf("failed to find process: %v", err)
	}

	isRunning, err := p.IsRunning()
	if err != nil {
		return false, fmt.Errorf("failed to check if process is running: %v", err)
	}

	return isRunning, nil
}
