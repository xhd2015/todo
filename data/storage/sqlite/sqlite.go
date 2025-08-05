package sqlite

import (
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

	if _, err := s.db.Exec(createLogEntriesTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(createNotesTable); err != nil {
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

// LogEntry service methods
func (les *LogEntrySQLiteStore) List(options storage.LogEntryListOptions) ([]models.LogEntry, int64, error) {
	var whereClause []string
	var args []interface{}

	if options.Filter != "" {
		whereClause = append(whereClause, "text LIKE ?")
		args = append(args, "%"+options.Filter+"%")
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

	query := fmt.Sprintf("SELECT id, text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, parent_id FROM log_entries %s %s %s",
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

		if err := rows.Scan(&entry.ID, &entry.Text, &entry.Done, &doneTime, &createTime, &updateTime, &entry.AdjustedTopTime, &entry.HighlightLevel, &entry.ParentID); err != nil {
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

// parsing time "2025-08-05T10:15:43Z" as "2006-01-02 15:04:05": cannot parse "T10:15:43Z" as " "
func tryParseTime(s string) (time.Time, error) {
	if strings.Contains(s, "T") {
		return tryParseStdTime(s)
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

func tryParseStdTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", s)
}

func (les *LogEntrySQLiteStore) Add(entry models.LogEntry) (int64, error) {
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	if entry.UpdateTime.IsZero() {
		entry.UpdateTime = time.Now()
	}

	query := `INSERT INTO log_entries (text, done, done_time, create_time, update_time, adjusted_top_time, highlight_level, parent_id) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	var doneTimeStr interface{}
	if entry.DoneTime != nil {
		doneTimeStr = entry.DoneTime.Format("2006-01-02 15:04:05")
	}

	result, err := les.db.Exec(query, entry.Text, entry.Done, doneTimeStr,
		entry.CreateTime.Format("2006-01-02 15:04:05"),
		entry.UpdateTime.Format("2006-01-02 15:04:05"),
		entry.AdjustedTopTime,
		entry.HighlightLevel,
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
			args = append(args, (*update.DoneTime).Format("2006-01-02 15:04:05"))
		} else {
			args = append(args, nil)
		}
	}
	if update.CreateTime != nil {
		setParts = append(setParts, "create_time = ?")
		args = append(args, update.CreateTime.Format("2006-01-02 15:04:05"))
	}
	if update.UpdateTime != nil {
		setParts = append(setParts, "update_time = ?")
		args = append(args, update.UpdateTime.Format("2006-01-02 15:04:05"))
	} else {
		setParts = append(setParts, "update_time = ?")
		args = append(args, time.Now().Format("2006-01-02 15:04:05"))
	}
	if update.AdjustedTopTime != nil {
		setParts = append(setParts, "adjusted_top_time = ?")
		args = append(args, *update.AdjustedTopTime)
	}
	if update.HighlightLevel != nil {
		setParts = append(setParts, "highlight_level = ?")
		args = append(args, *update.HighlightLevel)
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
		note.CreateTime.Format("2006-01-02 15:04:05"),
		note.UpdateTime.Format("2006-01-02 15:04:05"))
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
		args = append(args, update.CreateTime.Format("2006-01-02 15:04:05"))
	}
	if update.UpdateTime != nil {
		setParts = append(setParts, "update_time = ?")
		args = append(args, update.UpdateTime.Format("2006-01-02 15:04:05"))
	} else {
		setParts = append(setParts, "update_time = ?")
		args = append(args, time.Now().Format("2006-01-02 15:04:05"))
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
