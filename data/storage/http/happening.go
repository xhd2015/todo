package http

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

// OptionalNumber represents an optional number that can be null, empty string, or a number
type OptionalNumber string

func (c *OptionalNumber) UnmarshalJSON(data []byte) error {
	*c = OptionalNumber(data)
	return nil
}

func (c OptionalNumber) Int64() (int64, error) {
	if len(c) == 0 {
		return 0, nil
	}
	var s string
	if strings.HasPrefix(string(c), "\"") {
		var err error
		s, err = strconv.Unquote(string(c))
		if err != nil {
			return 0, err
		}
		if s == "" {
			return 0, nil
		}
	} else {
		s = string(c)
		if s == "null" {
			return 0, nil
		}
	}
	return strconv.ParseInt(s, 10, 64)
}

// TodoRecordStatus represents the status of a todo record
type TodoRecordStatus string

const (
	TodoRecordStatus_Init    TodoRecordStatus = ""
	TodoRecordStatus_Doing   TodoRecordStatus = "doing"
	TodoRecordStatus_Pending TodoRecordStatus = "pending"
	TodoRecordStatus_Pause   TodoRecordStatus = "pause"
	TodoRecordStatus_Expire  TodoRecordStatus = "expire"
	TodoRecordStatus_Done    TodoRecordStatus = "done"
	TodoRecordStatus_Archive TodoRecordStatus = "archived"
)

// RequestBase replicates ListHappeningReqBase from lifelog
type RequestBase struct {
	Search string  `json:"search"`
	ID     int64   `json:"id"`
	IDs    []int64 `json:"ids"`

	ContextBefore int `json:"contextBefore"`
	ContextAhead  int `json:"contextAhead"`

	IncludeAutogen  string  `json:"includeAutogen"`
	NewerThanID     int64   `json:"newerThanID"`
	NeedCollections bool    `json:"needCollections"`
	CollectionIDs   []int64 `json:"collectionIDs"`

	ThreadID int64 `json:"threadID"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// Request replicates ListHappeningReq from lifelog
type Request struct {
	*RequestBase
	TodoID      OptionalNumber   `json:"todoID"`
	ThreadForID OptionalNumber   `json:"threadForID"`
	TodoStatus  TodoRecordStatus `json:"todoStatus"`
}

// HappeningHttpService implements storage.HappeningService
type HappeningHttpService struct {
	client *Client
}

func NewHappeningService(client *Client) storage.HappeningService {
	return &HappeningHttpService{client: client}
}

func (s *HappeningHttpService) List(options storage.HappeningListOptions) ([]*models.Happening, int64, error) {
	// Set default limit to 20 if limit is 0
	limit := options.Limit
	if limit == 0 {
		limit = 20
	}

	// Convert HappeningListOptions to Request
	req := &Request{
		RequestBase: &RequestBase{
			Search: options.Filter,
			Offset: options.Offset,
			Limit:  limit,
		},
	}

	// Set sort order based on SortBy and SortOrder
	// Note: The lifelog API might handle sorting differently,
	// but we're mapping the basic fields for now

	var response struct {
		Happenings []*models.Happening `json:"happenings"`
		Total      int64               `json:"total"`
	}

	err := s.client.makeRequest(context.Background(), "/happening/list", req, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list happenings: %w", err)
	}

	return response.Happenings, response.Total, nil
}

func (s *HappeningHttpService) Add(ctx context.Context, happening *models.Happening) (*models.Happening, error) {
	if happening == nil {
		return nil, fmt.Errorf("happening cannot be nil")
	}
	if happening.Content == "" {
		return nil, fmt.Errorf("happening content cannot be empty")
	}

	// Create request payload for adding happening
	req := struct {
		Content string `json:"content"`
		Scope   string `json:"scope"`
	}{
		Content: happening.Content,
		Scope:   "", // Use default scope
	}

	var response struct {
		Happening *models.Happening `json:"happening"`
	}

	err := s.client.makeRequest(ctx, "/happening/add", req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add happening: %w", err)
	}

	if response.Happening == nil {
		return nil, fmt.Errorf("server returned nil happening")
	}

	return response.Happening, nil
}

func (s *HappeningHttpService) Update(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error) {
	if update == nil {
		return nil, fmt.Errorf("update cannot be nil")
	}

	// Create request payload for updating happening
	req := struct {
		ID   int64                     `json:"id"`
		Data *models.HappeningOptional `json:"data"`
	}{
		ID:   id,
		Data: update,
	}

	var response struct {
		Happening *models.Happening `json:"happening"`
	}

	err := s.client.makeRequest(ctx, "/happening/update", req, &response)
	if err != nil {
		return nil, fmt.Errorf("http update: %w", err)
	}

	if response.Happening == nil {
		return nil, fmt.Errorf("server returned nil happening")
	}

	return response.Happening, nil
}

func (s *HappeningHttpService) Delete(ctx context.Context, id int64) error {
	// Create request payload for deleting happening
	req := struct {
		ID int64 `json:"id"`
	}{
		ID: id,
	}

	var response struct {
		Success bool `json:"success"`
	}

	err := s.client.makeRequest(ctx, "/happening/delete", req, &response)
	if err != nil {
		return fmt.Errorf("failed to delete happening: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("server reported failure to delete happening")
	}

	return nil
}
