package http

import (
	"context"
	"fmt"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// StateRecordingHttpService implements storage.StateRecordingService
type StateRecordingHttpService struct {
	client *Client
}

func NewStateRecordingService(client *Client) storage.StateRecordingService {
	return &StateRecordingHttpService{client: client}
}

func (s *StateRecordingHttpService) GetState(ctx context.Context, name string) (*models.State, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	// Create request payload for getting state
	req := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}

	var response struct {
		State *models.State `json:"state"`
	}

	err := s.client.makeRequest(ctx, "/state/get", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	if response.State == nil {
		return nil, fmt.Errorf("state not found")
	}

	return response.State, nil
}

func (s *StateRecordingHttpService) RecordStateEvent(ctx context.Context, name string, deltaScore float64) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Create request payload for recording state event
	req := struct {
		Name       string  `json:"name"`
		DeltaScore float64 `json:"delta_score"`
	}{
		Name:       name,
		DeltaScore: deltaScore,
	}

	var response struct {
		Success bool `json:"success"`
	}

	err := s.client.makeRequest(ctx, "/state/recordEvent", req, &response)
	if err != nil {
		return fmt.Errorf("failed to record state event: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("server reported failure to record state event")
	}

	return nil
}

func (s *StateRecordingHttpService) CreateState(ctx context.Context, state *models.State) (*models.State, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}
	if state.Name == "" {
		return nil, fmt.Errorf("state name cannot be empty")
	}

	// Create request payload for creating state
	req := struct {
		State *models.State `json:"state"`
	}{
		State: state,
	}

	var response struct {
		State *models.State `json:"state"`
	}

	err := s.client.makeRequest(ctx, "/state/create", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	if response.State == nil {
		return nil, fmt.Errorf("server returned nil state")
	}

	return response.State, nil
}

func (s *StateRecordingHttpService) ListStates(ctx context.Context, scope string) ([]*models.State, error) {
	// Create request payload for listing states
	req := struct {
		Scope string `json:"scope"`
	}{
		Scope: scope,
	}

	var response struct {
		States []*models.State `json:"states"`
	}

	err := s.client.makeRequest(ctx, "/state/list", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}

	return response.States, nil
}

func (s *StateRecordingHttpService) GetStateEvents(ctx context.Context, stateID int64, limit int) ([]*models.StateEvent, error) {
	// Create request payload for getting state events
	req := struct {
		StateID int64 `json:"state_id"`
		Limit   int   `json:"limit"`
	}{
		StateID: stateID,
		Limit:   limit,
	}

	var response struct {
		Events []*models.StateEvent `json:"events"`
	}

	err := s.client.makeRequest(ctx, "/state/events", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get state events: %w", err)
	}

	return response.Events, nil
}
