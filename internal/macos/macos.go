package macos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TopCommand represents a command to show a todo item in the macOS floating bar
type TopCommand struct {
	ID       int64         `json:"id"`
	Text     string        `json:"text"`
	Duration time.Duration `json:"duration"`
}

// SendTopCommand sends a command to the macOS app via HTTP to show a floating progress bar
func SendTopCommand(id int64, text string, duration time.Duration) error {
	command := TopCommand{
		ID:       id,
		Text:     text,
		Duration: duration,
	}

	// Marshal command to JSON
	data, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Try multiple ports starting from 4756
	ports := []int{4756, 4757, 4758, 4759, 4760, 4761, 4762, 4763, 4764, 4765}

	client := &http.Client{
		Timeout: 2 * time.Second, // Shorter timeout for multiple attempts
	}

	var lastErr error
	for _, port := range ports {
		url := fmt.Sprintf("http://localhost:%d/command", port)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			lastErr = fmt.Errorf("failed to create HTTP request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send HTTP request to port %d: %w", port, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP request to port %d failed with status: %s", port, resp.Status)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to connect to macOS app on any port (tried %v): %w", ports, lastErr)
}
