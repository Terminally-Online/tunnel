package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			summary TEXT,
			turn_count INTEGER,
			history TEXT,
			updated_at DATETIME
		)
	`)
	return err
}

func (s *Store) Get(id string) (*Session, error) {
	row := s.db.QueryRow(`
		SELECT id, summary, turn_count, history, updated_at
		FROM sessions
		WHERE id = ?
	`, id)

	var session Session
	var historyJSON string
	var updatedAt string

	err := row.Scan(&session.ID, &session.Summary, &session.TurnCount, &historyJSON, &updatedAt)
	if err == sql.ErrNoRows {
		return &Session{
			ID:        id,
			UpdatedAt: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if historyJSON != "" {
		if err := json.Unmarshal([]byte(historyJSON), &session.History); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history: %w", err)
		}
	}

	session.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &session, nil
}

func (s *Store) Save(session *Session) error {
	historyJSON, err := json.Marshal(session.History)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO sessions (id, summary, turn_count, history, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			summary = excluded.summary,
			turn_count = excluded.turn_count,
			history = excluded.history,
			updated_at = excluded.updated_at
	`, session.ID, session.Summary, session.TurnCount, string(historyJSON), session.UpdatedAt.Format(time.RFC3339))

	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
