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
			profile TEXT,
			character_deltas TEXT,
			turn_count INTEGER,
			total_turn_count INTEGER DEFAULT 0,
			history TEXT,
			updated_at DATETIME,
			character_updated_at DATETIME
		)
	`)
	if err != nil {
		return err
	}

	s.db.Exec(`ALTER TABLE sessions ADD COLUMN profile TEXT`)
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN total_turn_count INTEGER DEFAULT 0`)
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN character_deltas TEXT`)
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN character_updated_at DATETIME`)

	return nil
}

func (s *Store) Get(id string) (*Session, error) {
	row := s.db.QueryRow(`
		SELECT id, summary, COALESCE(profile, ''), COALESCE(character_deltas, ''), turn_count, COALESCE(total_turn_count, 0), history, updated_at, COALESCE(character_updated_at, '')
		FROM sessions
		WHERE id = ?
	`, id)

	var session Session
	var historyJSON string
	var updatedAt string
	var characterUpdatedAt string

	err := row.Scan(&session.ID, &session.Summary, &session.Profile, &session.CharacterDeltas, &session.TurnCount, &session.TotalTurnCount, &historyJSON, &updatedAt, &characterUpdatedAt)
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
	if characterUpdatedAt != "" {
		session.CharacterUpdatedAt, _ = time.Parse(time.RFC3339, characterUpdatedAt)
	}

	return &session, nil
}

func (s *Store) Save(session *Session) error {
	historyJSON, err := json.Marshal(session.History)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	var characterUpdatedAt string
	if !session.CharacterUpdatedAt.IsZero() {
		characterUpdatedAt = session.CharacterUpdatedAt.Format(time.RFC3339)
	}

	_, err = s.db.Exec(`
		INSERT INTO sessions (id, summary, profile, character_deltas, turn_count, total_turn_count, history, updated_at, character_updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			summary = excluded.summary,
			profile = excluded.profile,
			character_deltas = excluded.character_deltas,
			turn_count = excluded.turn_count,
			total_turn_count = excluded.total_turn_count,
			history = excluded.history,
			updated_at = excluded.updated_at,
			character_updated_at = excluded.character_updated_at
	`, session.ID, session.Summary, session.Profile, session.CharacterDeltas, session.TurnCount, session.TotalTurnCount, string(historyJSON), session.UpdatedAt.Format(time.RFC3339), characterUpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
