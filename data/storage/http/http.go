package http

import (
	"context"
	"encoding/json"
	"fmt"

	http_request "github.com/xhd2015/go-http-request"
	"github.com/xhd2015/todo/data/storage"
	applog "github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
)

// ServerResponse wraps all server responses
type ServerResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// makeRequest makes an HTTP request and unwraps the server response
// api is the API path that omits the prefix "/api/todo/termui", e.g. "/entries/list"
func (c *Client) makeRequest(ctx context.Context, api string, reqData any, respData any) error {

	// Log the request with full JSON
	if reqJSON, err := json.Marshal(reqData); err == nil {
		applog.Infof(ctx, "HTTP Request: %s %s, payload: %s", "POST", c.serverAddr+api, string(reqJSON))
	} else {
		applog.Infof(ctx, "HTTP Request: %s %s, payload marshal error: %v", "POST", c.serverAddr+api, err)
	}

	req := http_request.New()
	if c.serverAuthToken != "" {
		req = req.Header("Authorization", "Bearer "+c.serverAuthToken)
	}

	var serverResp ServerResponse
	err := req.PostJSON(ctx, c.serverAddr+api, reqData, &serverResp)
	if err != nil {
		applog.Errorf(ctx, "HTTP Request failed: %s %s, error: %v", "POST", c.serverAddr+api, err)
		return fmt.Errorf("request failed: %w", err)
	}

	// Log the response with length only
	responseLength := len(serverResp.Data)
	applog.Infof(ctx, "HTTP Response: %s %s, code: %d, msg: %s, data_length: %d", "POST", c.serverAddr+api, serverResp.Code, serverResp.Msg, responseLength)

	if serverResp.Code != 0 {
		applog.Errorf(ctx, "HTTP Server error: %s %s, code: %d, msg: %s", "POST", c.serverAddr+api, serverResp.Code, serverResp.Msg)
		return fmt.Errorf("server error (code %d): %s", serverResp.Code, serverResp.Msg)
	}

	if respData != nil && len(serverResp.Data) > 0 {
		// Directly unmarshal the raw JSON data
		err = json.Unmarshal(serverResp.Data, respData)
		if err != nil {
			applog.Errorf(ctx, "HTTP Response unmarshal failed: %s %s, error: %v", "POST", c.serverAddr+api, err)
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return nil
}

type Client struct {
	serverAddr      string
	serverAuthToken string
}

func NewClient(serverAddr string, serverAuthToken string) *Client {
	return &Client{
		serverAddr:      serverAddr,
		serverAuthToken: serverAuthToken,
	}
}

// LogEntryHttpService implements storage.LogEntryService
type LogEntryHttpService struct {
	client *Client
}

func NewLogEntryService(client *Client) storage.LogEntryService {
	return &LogEntryHttpService{client: client}
}

func (s *LogEntryHttpService) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	var response struct {
		Entries []models.LogEntry `json:"entries"`
		Total   int64             `json:"total"`
	}

	err := s.client.makeRequest(context.Background(), "/entries/list", options, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entries: %w", err)
	}

	return response.Entries, response.Total, nil
}

func (s *LogEntryHttpService) Add(entry models.LogEntry) (int64, error) {
	var response struct {
		ID int64 `json:"id"`
	}

	err := s.client.makeRequest(context.Background(), "/entries/add", entry, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to add entry: %w", err)
	}

	return response.ID, nil
}

func (s *LogEntryHttpService) Delete(id int64) error {
	params := struct {
		ID int64 `json:"id"`
	}{ID: id}

	err := s.client.makeRequest(context.Background(), "/entries/delete", params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	return nil
}

func (s *LogEntryHttpService) Update(id int64, update models.LogEntryOptional) error {
	params := struct {
		ID     int64                   `json:"id"`
		Update models.LogEntryOptional `json:"update"`
	}{ID: id, Update: update}

	err := s.client.makeRequest(context.Background(), "/entries/update", params, nil)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	return nil
}

func (s *LogEntryHttpService) Move(id int64, newParentID int64) error {
	params := struct {
		ID          int64 `json:"id"`
		NewParentID int64 `json:"new_parent_id"`
	}{ID: id, NewParentID: newParentID}

	err := s.client.makeRequest(context.Background(), "/entries/move", params, nil)
	if err != nil {
		return fmt.Errorf("failed to move entry: %w", err)
	}

	return nil
}

func (s *LogEntryHttpService) GetTree(ctx context.Context, id int64, includeHistory bool) ([]models.LogEntry, error) {
	var response struct {
		Entries []models.LogEntry `json:"entries"`
	}

	params := struct {
		ID             int64 `json:"id"`
		IncludeHistory bool  `json:"include_history"`
	}{ID: id, IncludeHistory: includeHistory}

	err := s.client.makeRequest(ctx, "/entries/getTree", params, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree entries: %w", err)
	}

	return response.Entries, nil
}

// LogNoteHttpService implements storage.LogNoteService
type LogNoteHttpService struct {
	client *Client
}

func NewLogNoteService(client *Client) storage.LogNoteService {
	return &LogNoteHttpService{client: client}
}

func (s *LogNoteHttpService) List(entryID int64, options storage.LogNoteListOptions) ([]models.Note, int64, error) {
	var response struct {
		Notes []models.Note `json:"notes"`
		Total int64         `json:"total"`
	}

	params := struct {
		EntryID int64                      `json:"entry_id"`
		Options storage.LogNoteListOptions `json:"options"`
	}{EntryID: entryID, Options: options}

	err := s.client.makeRequest(context.Background(), "/notes/list", params, &response)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notes: %w", err)
	}

	return response.Notes, response.Total, nil
}

func (s *LogNoteHttpService) ListForEntries(entryIDs []int64) (map[int64][]models.Note, error) {
	var response struct {
		NotesMap map[int64][]models.Note `json:"notes_map"`
	}

	params := struct {
		EntryIDs []int64 `json:"entry_ids"`
	}{EntryIDs: entryIDs}

	err := s.client.makeRequest(context.Background(), "/notes/listForEntries", params, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes for entries: %w", err)
	}

	return response.NotesMap, nil
}

func (s *LogNoteHttpService) Add(entryID int64, note models.Note) (int64, error) {
	var response struct {
		ID int64 `json:"id"`
	}

	params := struct {
		EntryID int64       `json:"entry_id"`
		Note    models.Note `json:"note"`
	}{EntryID: entryID, Note: note}

	err := s.client.makeRequest(context.Background(), "/notes/add", params, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to add note: %w", err)
	}

	return response.ID, nil
}

func (s *LogNoteHttpService) Delete(entryID int64, noteID int64) error {
	params := struct {
		EntryID int64 `json:"entry_id"`
		NoteID  int64 `json:"note_id"`
	}{EntryID: entryID, NoteID: noteID}

	err := s.client.makeRequest(context.Background(), "/notes/delete", params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

func (s *LogNoteHttpService) Update(entryID int64, noteID int64, update models.NoteOptional) error {
	params := struct {
		EntryID int64               `json:"entry_id"`
		NoteID  int64               `json:"note_id"`
		Update  models.NoteOptional `json:"update"`
	}{EntryID: entryID, NoteID: noteID, Update: update}

	err := s.client.makeRequest(context.Background(), "/notes/update", params, nil)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}
