package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xhd2015/todo/data/storage"
	"github.com/xhd2015/todo/models"
)

type SQLiteStore struct {
	db *sql.DB
}

type LogEntrySQLiteStore struct {
	*SQLiteStore
}

type LogNoteSQLiteStore struct {
	*SQLiteStore
}

type HappeningSQLiteStore struct {
	*SQLiteStore
}

type StateRecordingSQLiteStore struct {
	*SQLiteStore
}

func New(filePath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}

	if err := store.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) createTables() error {
	createLogEntriesTable := `
	CREATE TABLE IF NOT EXISTS log_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text TEXT NOT NULL,
		done BOOLEAN NOT NULL DEFAULT 0,
		done_time DATETIME,
		create_time DATETIME NOT NULL,
		update_time DATETIME NOT NULL,
		adjusted_top_time INTEGER NOT NULL DEFAULT 0,
		highlight_level INTEGER NOT NULL DEFAULT 0,
		collapsed BOOLEAN NOT NULL DEFAULT 0,
		parent_id INTEGER NOT NULL DEFAULT 0
	);`

	createNotesTable := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		create_time DATETIME NOT NULL,
		update_time DATETIME NOT NULL,
		FOREIGN KEY (entry_id) REFERENCES log_entries(id) ON DELETE CASCADE
	);`

	createHappeningsTable := `
	CREATE TABLE IF NOT EXISTS happenings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		create_time DATETIME NOT NULL,
		update_time DATETIME NOT NULL
	);`

	createStatesTable := `
	CREATE TABLE IF NOT EXISTS states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		parent_state_record_id INTEGER NOT NULL DEFAULT 0,
		score REAL NOT NULL DEFAULT 0.0,
		scope TEXT NOT NULL DEFAULT '',
		create_time DATETIME NOT NULL,
		update_time DATETIME NOT NULL
	);`

	createStateEventsTable := `
	CREATE TABLE IF NOT EXISTS state_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_record_id INTEGER NOT NULL,
		record_data TEXT NOT NULL DEFAULT '',
		delta_score REAL NOT NULL DEFAULT 0.0,
		description TEXT NOT NULL DEFAULT '',
		details TEXT NOT NULL DEFAULT '',
		scope TEXT NOT NULL DEFAULT '',
		create_time DATETIME NOT NULL,
		update_time DATETIME NOT NULL,
		FOREIGN KEY (state_record_id) REFERENCES states(id) ON DELETE CASCADE
	);`

	if _, err := s.db.Exec(createLogEntriesTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(createNotesTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(createHappeningsTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(createStatesTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(createStateEventsTable); err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func NewLogEntryService(filePath string) (storage.LogEntryService, error) {
	store, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &LogEntrySQLiteStore{SQLiteStore: store}, nil
}

func NewLogNoteService(filePath string) (storage.LogNoteService, error) {
	store, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &LogNoteSQLiteStore{SQLiteStore: store}, nil
}

func NewHappeningService(filePath string) (storage.HappeningService, error) {
	store, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &HappeningSQLiteStore{SQLiteStore: store}, nil
}

func NewStateRecordingService(filePath string) (storage.StateRecordingService, error) {
	store, err := New(filePath)
	if err != nil {
		return nil, err
	}
	return &StateRecordingSQLiteStore{SQLiteStore: store}, nil
}

// LogEntry service methods
func (les *LogEntrySQLiteStore) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	var whereClause []string
	var args []interface{}

	if options.Filter != "" {
		whereClause = append(whereClause, "text LIKE ?")
		args = append(args, "%"+options.Filter+"%")
	}

	// Handle history filtering
	if !options.IncludeHistory {
		// Filter out entries that are done and have done_time before today
		whereClause = append(whereClause, "(done = 0 OR done_time IS NULL OR date(done_time) >= date('now'))")
	}

	where := ""
	if len(whereClause) > 0 {
		where = "WHERE " + strings.Join(whereClause, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM log_entries %s", where)
	var total int64
	if err := les.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build main query
	orderBy := "ORDER BY id ASC"
	if options.SortBy != "" {
		direction := "ASC"
		if options.SortOrder == "desc" {
			direction = "DESC"
		}
		if options.SortBy == "create_time" {
			// Special handling for create_time: if AdjustedTopTime is set, use it for priority
			orderBy = fmt.Sprintf("ORDER BY CASE WHEN adjusted_top_time != 0 THEN adjusted_top_time ELSE strftime('%%s', create_time) * 1000 END %s", direction)
		} else {
			orderBy = fmt.Sprintf("ORDER BY %s %s", options.SortBy, direction)
		}
	}

	limit := ""
	if options.Limit > 0 {
		limit = fmt.Sprintf("LIMIT %d", options.Limit)
		if options.Offset > 0 {
			limit += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	query := fmt.Sprintf("SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id FROM log_entries %s %s %s",
		where, orderBy, limit)

	rows, err := les.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []models.LogEntry
	for rows.Next() {
		var entry models.LogEntry
		var createTime, updateTime string
		var doneTime *string

		if err := rows.Scan(&entry.ID, &entry.Text, &entry.Done, &doneTime, &createTime, &updateTime, &entry.AdjustedTopTime, &entry.HighlightLevel, &entry.Collapsed, &entry.ParentID); err != nil {
			return nil, 0, err
		}

		if entry.CreateTime, err = tryParseTime(createTime); err != nil {
			return nil, 0, err
		}
		if entry.UpdateTime, err = tryParseTime(updateTime); err != nil {
			return nil, 0, err
		}
		if doneTime != nil {
			if parsedDoneTime, err := tryParseTime(*doneTime); err != nil {
				return nil, 0, err
			} else {
				entry.DoneTime = &parsedDoneTime
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (les *LogEntrySQLiteStore) Add(entry models.LogEntry) (int64, error) {
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}

	query := `INSERT INTO log_entries (text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id) 
		  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var doneTimeStr interface{}
	if entry.DoneTime != nil {
		doneTimeStr = entry.DoneTime.Format("2006-01-02 15:04:05")
	}

	result, err := les.db.Exec(query, entry.Text, entry.Done, doneTimeStr,
		entry.CreateTime.Format("2006-01-02 15:04:05"),
		entry.UpdateTime.Format("2006-01-02 15:04:05"),
		entry.AdjustedTopTime,
		entry.HighlightLevel,
		entry.Collapsed,
		entry.ParentID)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (les *LogEntrySQLiteStore) Delete(id int64) error {
	// Delete notes first (cascade should handle this, but being explicit)
	if _, err := les.db.Exec("DELETE FROM notes WHERE entry_id = ?", id); err != nil {
		return err
	}

	result, err := les.db.Exec("DELETE FROM log_entries WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	return nil
}

func (les *LogEntrySQLiteStore) Update(id int64, update models.LogEntryOptional) error {
	var setParts []string
	var args []interface{}

	if update.Text != nil {
		setParts = append(setParts, "text = ?")
		args = append(args, *update.Text)
	}
	if update.Done != nil {
		setParts = append(setParts, "done = ?")
		args = append(args, *update.Done)
	}
	if update.DoneTime != nil {
		setParts = append(setParts, "done_time = ?")
		if *update.DoneTime != nil {
			args = append(args, formatTime(**update.DoneTime))
		} else {
			args = append(args, nil)
		}
	}
	if update.CreateTime != nil {
		setParts = append(setParts, "create_time = ?")
		args = append(args, formatTime(*update.CreateTime))
	}
	if update.UpdateTime != nil {
		setParts = append(setParts, "update_time = ?")
		args = append(args, formatTime(*update.UpdateTime))
	} else {
		setParts = append(setParts, "update_time = ?")
		args = append(args, formatTime(time.Now()))
	}
	if update.AdjustedTopTime != nil {
		setParts = append(setParts, "adjusted_top_time = ?")
		args = append(args, *update.AdjustedTopTime)
	}
	if update.HighlightLevel != nil {
		setParts = append(setParts, "highlight_level = ?")
		args = append(args, *update.HighlightLevel)
	}
	if update.Collapsed != nil {
		setParts = append(setParts, "collapsed = ?")
		args = append(args, *update.Collapsed)
	}
	if update.ParentID != nil {
		setParts = append(setParts, "parent_id = ?")
		args = append(args, *update.ParentID)
	}

	if len(setParts) == 0 {
		return nil // Nothing to update
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE log_entries SET %s WHERE id = ?", strings.Join(setParts, ", "))

	result, err := les.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	return nil
}

func (les *LogEntrySQLiteStore) Move(id int64, newParentID int64) error {
	result, err := les.db.Exec("UPDATE log_entries SET parent_id = ?, update_time = ? WHERE id = ?", newParentID, formatTime(time.Now()), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("log entry with id %d not found", id)
	}

	return nil
}

func (les *LogEntrySQLiteStore) GetTree(ctx context.Context, id int64, includeHistory bool) ([]models.LogEntry, error) {
	// Use recursive CTE to get all descendants of the root entry
	var query string
	if includeHistory {
		query = `
			WITH RECURSIVE descendants AS (
				-- Base case: the root entry
				SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id
				FROM log_entries 
				WHERE id = ?
				
				UNION ALL
				
				-- Recursive case: children of entries already in the result
				SELECT e.id, e.text, e.done, e.done_time, e.create_time, e.update_time, e.adjusted_top_time, e.highlight_level, e.collapsed, e.parent_id
				FROM log_entries e
				INNER JOIN descendants d ON e.parent_id = d.id
			)
			SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id
			FROM descendants
			ORDER BY parent_id, id
		`
	} else {
		query = `
			WITH RECURSIVE descendants AS (
				-- Base case: the root entry
				SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id
				FROM log_entries 
				WHERE id = ?
				
				UNION ALL
				
				-- Recursive case: children of entries already in the result (excluding done entries)
				SELECT e.id, e.text, e.done, e.done_time, e.create_time, e.update_time, e.adjusted_top_time, e.highlight_level, e.collapsed, e.parent_id
				FROM log_entries e
				INNER JOIN descendants d ON e.parent_id = d.id
				WHERE NOT (e.done = 1 AND e.done_time IS NOT NULL)
			)
			SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, collapsed, parent_id
			FROM descendants
			ORDER BY parent_id, id
		`
	}

	rows, err := les.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.LogEntry
	for rows.Next() {
		var entry models.LogEntry
		var createTime, updateTime string
		var doneTime *string

		if err := rows.Scan(&entry.ID, &entry.Text, &entry.Done, &doneTime, &createTime, &updateTime, &entry.AdjustedTopTime, &entry.HighlightLevel, &entry.Collapsed, &entry.ParentID); err != nil {
			return nil, err
		}

		if entry.CreateTime, err = tryParseTime(createTime); err != nil {
			return nil, err
		}
		if entry.UpdateTime, err = tryParseTime(updateTime); err != nil {
			return nil, err
		}
		if doneTime != nil {
			if parsedDoneTime, err := tryParseTime(*doneTime); err != nil {
				return nil, err
			} else {
				entry.DoneTime = &parsedDoneTime
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// LogNote service methods
func (lns *LogNoteSQLiteStore) List(entryID int64, options storage.LogNoteListOptions) ([]models.Note, int64, error) {
	var whereClause []string
	var args []interface{}

	whereClause = append(whereClause, "entry_id = ?")
	args = append(args, entryID)

	if options.Filter != "" {
		whereClause = append(whereClause, "text LIKE ?")
		args = append(args, "%"+options.Filter+"%")
	}

	where := "WHERE " + strings.Join(whereClause, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notes %s", where)
	var total int64
	if err := lns.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build main query
	orderBy := "ORDER BY id ASC"
	if options.SortBy != "" {
		direction := "ASC"
		if options.SortOrder == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", options.SortBy, direction)
	}

	limit := ""
	if options.Limit > 0 {
		limit = fmt.Sprintf("LIMIT %d", options.Limit)
		if options.Offset > 0 {
			limit += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	query := fmt.Sprintf("SELECT id, entry_id, text, create_time, update_time FROM notes %s %s %s",
		where, orderBy, limit)

	rows, err := lns.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var note models.Note
		var createTime, updateTime string

		if err := rows.Scan(&note.ID, &note.EntryID, &note.Text, &createTime, &updateTime); err != nil {
			return nil, 0, err
		}

		if note.CreateTime, err = tryParseTime(createTime); err != nil {
			return nil, 0, err
		}
		if note.UpdateTime, err = tryParseTime(updateTime); err != nil {
			return nil, 0, err
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

func (lns *LogNoteSQLiteStore) ListForEntries(entryIDs []int64) (map[int64][]models.Note, error) {
	if len(entryIDs) == 0 {
		return make(map[int64][]models.Note), nil
	}

	result := make(map[int64][]models.Note)

	// Initialize empty slices for all requested entry IDs
	for _, entryID := range entryIDs {
		result[entryID] = []models.Note{}
	}

	// Build IN clause with placeholders
	placeholders := make([]string, len(entryIDs))
	args := make([]interface{}, len(entryIDs))
	for i, entryID := range entryIDs {
		placeholders[i] = "?"
		args[i] = entryID
	}

	query := fmt.Sprintf("SELECT id, entry_id, text, create_time, update_time FROM notes WHERE entry_id IN (%s) ORDER BY entry_id, id",
		strings.Join(placeholders, ", "))

	rows, err := lns.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var note models.Note
		var createTime, updateTime string

		if err := rows.Scan(&note.ID, &note.EntryID, &note.Text, &createTime, &updateTime); err != nil {
			return nil, err
		}

		if note.CreateTime, err = tryParseTime(createTime); err != nil {
			return nil, err
		}
		if note.UpdateTime, err = tryParseTime(updateTime); err != nil {
			return nil, err
		}

		result[note.EntryID] = append(result[note.EntryID], note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (lns *LogNoteSQLiteStore) Add(entryID int64, note models.Note) (int64, error) {
	// Check if entry exists
	var exists bool
	if err := lns.db.QueryRow("SELECT EXISTS(SELECT 1 FROM log_entries WHERE id = ?)", entryID).Scan(&exists); err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("log entry with id %d not found", entryID)
	}

	if note.CreateTime.IsZero() {
		note.CreateTime = time.Now()
	}
	if note.UpdateTime.IsZero() {
		note.UpdateTime = time.Now()
	}

	query := `INSERT INTO notes (entry_id, text, create_time, update_time) 
			  VALUES (?, ?, ?, ?)`

	result, err := lns.db.Exec(query, entryID, note.Text,
		formatTime(note.CreateTime),
		formatTime(note.UpdateTime))
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (lns *LogNoteSQLiteStore) Delete(entryID int64, noteID int64) error {
	result, err := lns.db.Exec("DELETE FROM notes WHERE id = ? AND entry_id = ?", noteID, entryID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
	}

	return nil
}

func (lns *LogNoteSQLiteStore) Update(entryID int64, noteID int64, update models.NoteOptional) error {
	var setParts []string
	var args []interface{}

	if update.Text != nil {
		setParts = append(setParts, "text = ?")
		args = append(args, *update.Text)
	}
	if update.CreateTime != nil {
		setParts = append(setParts, "create_time = ?")
		args = append(args, formatTime(*update.CreateTime))
	}
	if update.UpdateTime != nil {
		setParts = append(setParts, "update_time = ?")
		args = append(args, formatTime(*update.UpdateTime))
	} else {
		setParts = append(setParts, "update_time = ?")
		args = append(args, formatTime(time.Now()))
	}

	if len(setParts) == 0 {
		return nil // Nothing to update
	}

	args = append(args, noteID, entryID)
	query := fmt.Sprintf("UPDATE notes SET %s WHERE id = ? AND entry_id = ?", strings.Join(setParts, ", "))

	result, err := lns.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note with id %d not found for entry %d", noteID, entryID)
	}

	return nil
}

// Happening service methods
func (hss *HappeningSQLiteStore) List(options storage.HappeningListOptions) ([]*models.Happening, int64, error) {
	var whereClause []string
	var args []interface{}

	if options.Filter != "" {
		whereClause = append(whereClause, "content LIKE ?")
		args = append(args, "%"+options.Filter+"%")
	}

	where := ""
	if len(whereClause) > 0 {
		where = "WHERE " + strings.Join(whereClause, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM happenings %s", where)
	var total int64
	if err := hss.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build main query
	orderBy := "ORDER BY id ASC"
	if options.SortBy != "" {
		direction := "ASC"
		if options.SortOrder == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", options.SortBy, direction)
	}

	limit := ""
	if options.Limit > 0 {
		limit = fmt.Sprintf("LIMIT %d", options.Limit)
		if options.Offset > 0 {
			limit += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	query := fmt.Sprintf("SELECT id, content, create_time, update_time FROM happenings %s %s %s",
		where, orderBy, limit)

	rows, err := hss.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var happenings []*models.Happening
	for rows.Next() {
		var happening models.Happening
		var createTime, updateTime string

		if err := rows.Scan(&happening.ID, &happening.Content, &createTime, &updateTime); err != nil {
			return nil, 0, err
		}

		if happening.CreateTime, err = tryParseTime(createTime); err != nil {
			return nil, 0, err
		}
		if happening.UpdateTime, err = tryParseTime(updateTime); err != nil {
			return nil, 0, err
		}

		happenings = append(happenings, &happening)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return happenings, total, nil
}

func (hss *HappeningSQLiteStore) Add(ctx context.Context, happening *models.Happening) (*models.Happening, error) {
	if happening == nil {
		return nil, fmt.Errorf("happening cannot be nil")
	}
	if happening.Content == "" {
		return nil, fmt.Errorf("happening content cannot be empty")
	}

	now := time.Now()

	// Insert the happening
	query := `INSERT INTO happenings (content, create_time, update_time) VALUES (?, ?, ?)`
	result, err := hss.db.ExecContext(ctx, query, happening.Content, formatTime(now), formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("failed to insert happening: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted ID: %w", err)
	}

	// Return the new happening
	newHappening := &models.Happening{
		ID:         id,
		Content:    happening.Content,
		CreateTime: now,
		UpdateTime: now,
	}

	return newHappening, nil
}

func (hss *HappeningSQLiteStore) Update(ctx context.Context, id int64, update *models.HappeningOptional) (*models.Happening, error) {
	if update == nil {
		return nil, fmt.Errorf("update cannot be nil")
	}

	// First get the existing happening
	var existing models.Happening
	var createTime, updateTime string
	err := hss.db.QueryRowContext(ctx, "SELECT id, content, create_time, update_time FROM happenings WHERE id = ?", id).Scan(
		&existing.ID, &existing.Content, &createTime, &updateTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("happening with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get existing happening: %w", err)
	}

	existing.CreateTime, err = tryParseTime(createTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse create time: %w", err)
	}
	existing.UpdateTime, err = tryParseTime(updateTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update time: %w", err)
	}

	// Apply the optional updates
	existing.Update(update)

	// Update the happening in database
	query := `UPDATE happenings SET content = ?, update_time = ? WHERE id = ?`
	result, err := hss.db.ExecContext(ctx, query, existing.Content, formatTime(existing.UpdateTime), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update happening: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("happening with id %d not found", id)
	}

	return &existing, nil
}

func (hss *HappeningSQLiteStore) Delete(ctx context.Context, id int64) error {
	result, err := hss.db.ExecContext(ctx, "DELETE FROM happenings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete happening: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("happening with id %d not found", id)
	}

	return nil
}

// StateRecordingService methods
func (srs *StateRecordingSQLiteStore) GetState(ctx context.Context, name string) (*models.State, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	var state models.State
	var createTime, updateTime string

	query := `SELECT id, name, description, parent_state_record_id, score, scope, create_time, update_time 
			  FROM states WHERE name = ?`

	err := srs.db.QueryRowContext(ctx, query, name).Scan(
		&state.ID, &state.Name, &state.Description, &state.ParentStateRecordID,
		&state.Score, &state.Scope, &createTime, &updateTime)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state not found")
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	state.CreateTime, err = tryParseTime(createTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse create time: %w", err)
	}
	state.UpdateTime, err = tryParseTime(updateTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update time: %w", err)
	}

	return &state, nil
}

func (srs *StateRecordingSQLiteStore) RecordStateEvent(ctx context.Context, name string, deltaScore float64) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Start a transaction
	tx, err := srs.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the state by name
	var stateID int64
	var currentScore float64
	var scope string
	err = tx.QueryRowContext(ctx, "SELECT id, score, scope FROM states WHERE name = ?", name).Scan(&stateID, &currentScore, &scope)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("state not found")
		}
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Update the state score
	newScore := currentScore + deltaScore
	updateTime := time.Now()
	_, err = tx.ExecContext(ctx, "UPDATE states SET score = ?, update_time = ? WHERE id = ?",
		newScore, formatTime(updateTime), stateID)
	if err != nil {
		return fmt.Errorf("failed to update state score: %w", err)
	}

	// Create and insert the state event
	now := time.Now()
	eventQuery := `INSERT INTO state_events (state_record_id, record_data, delta_score, description, details, scope, create_time, update_time) 
				   VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, eventQuery, stateID, "", deltaScore, "", "", scope, formatTime(now), formatTime(now))
	if err != nil {
		return fmt.Errorf("failed to insert state event: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (srs *StateRecordingSQLiteStore) CreateState(ctx context.Context, state *models.State) (*models.State, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}
	if state.Name == "" {
		return nil, fmt.Errorf("state name cannot be empty")
	}

	// Check if state with same name already exists
	var exists bool
	err := srs.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM states WHERE name = ?)", state.Name).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check if state exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("state with this name already exists")
	}

	now := time.Now()

	// Insert the state
	query := `INSERT INTO states (name, description, parent_state_record_id, score, scope, create_time, update_time) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := srs.db.ExecContext(ctx, query,
		state.Name, state.Description, state.ParentStateRecordID,
		state.Score, state.Scope, formatTime(now), formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("failed to insert state: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted ID: %w", err)
	}

	// Return the new state
	newState := &models.State{
		ID:                  id,
		Name:                state.Name,
		Description:         state.Description,
		ParentStateRecordID: state.ParentStateRecordID,
		Score:               state.Score,
		Scope:               state.Scope,
		CreateTime:          now,
		UpdateTime:          now,
	}

	return newState, nil
}

func (srs *StateRecordingSQLiteStore) ListStates(ctx context.Context, scope string) ([]*models.State, error) {
	var whereClause string
	var args []interface{}

	if scope != "" {
		whereClause = "WHERE scope LIKE ?"
		args = append(args, "%"+scope+"%")
	}

	query := fmt.Sprintf(`SELECT id, name, description, parent_state_record_id, score, scope, create_time, update_time 
						  FROM states %s ORDER BY name`, whereClause)

	rows, err := srs.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	defer rows.Close()

	var states []*models.State
	for rows.Next() {
		var state models.State
		var createTime, updateTime string

		err := rows.Scan(&state.ID, &state.Name, &state.Description, &state.ParentStateRecordID,
			&state.Score, &state.Scope, &createTime, &updateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan state: %w", err)
		}

		state.CreateTime, err = tryParseTime(createTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse create time: %w", err)
		}
		state.UpdateTime, err = tryParseTime(updateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse update time: %w", err)
		}

		states = append(states, &state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return states, nil
}

func (srs *StateRecordingSQLiteStore) GetStateEvents(ctx context.Context, stateID int64, limit int) ([]*models.StateEvent, error) {
	var limitClause string
	if limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", limit)
	}

	query := fmt.Sprintf(`SELECT id, state_record_id, record_data, delta_score, description, details, scope, create_time, update_time 
						  FROM state_events WHERE state_record_id = ? ORDER BY create_time DESC %s`, limitClause)

	rows, err := srs.db.QueryContext(ctx, query, stateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query state events: %w", err)
	}
	defer rows.Close()

	var events []*models.StateEvent
	for rows.Next() {
		var event models.StateEvent
		var createTime, updateTime string

		err := rows.Scan(&event.ID, &event.StateRecordID, &event.RecordData, &event.DeltaScore,
			&event.Description, &event.Details, &event.Scope, &createTime, &updateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan state event: %w", err)
		}

		event.CreateTime, err = tryParseTime(createTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse create time: %w", err)
		}
		event.UpdateTime, err = tryParseTime(updateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse update time: %w", err)
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}
