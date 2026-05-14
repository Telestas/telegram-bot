package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	ID       int64
	Text     string
	Priority bool
}

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    text        TEXT NOT NULL,
    priority    INTEGER NOT NULL DEFAULT 0,
    closed      INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_open ON tasks(closed, priority DESC, id);
`

func newStore(path string) (*Store, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Create(text string, priority bool) (int64, error) {
	now := time.Now().Unix()
	p := 0
	if priority {
		p = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO tasks (text, priority, closed, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		text, p, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListOpen() ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id, text, priority FROM tasks WHERE closed = 0 ORDER BY priority DESC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var (
			t Task
			p int
		)
		if err := rows.Scan(&t.ID, &t.Text, &p); err != nil {
			return nil, err
		}
		t.Priority = p == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateText(id int64, text string) (int64, error) {
	res, err := s.db.Exec(
		`UPDATE tasks SET text = ?, updated_at = ? WHERE id = ? AND closed = 0`,
		text, time.Now().Unix(), id,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// TogglePriority flips the priority flag of an open task and returns its new
// value. Returns (nil, nil) if no open task with that id exists.
func (s *Store) TogglePriority(id int64) (*bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var p int
	err = tx.QueryRow(`SELECT priority FROM tasks WHERE id = ? AND closed = 0`, id).Scan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	newP := 1 - p
	if _, err := tx.Exec(
		`UPDATE tasks SET priority = ?, updated_at = ? WHERE id = ?`,
		newP, time.Now().Unix(), id,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	b := newP == 1
	return &b, nil
}

func (s *Store) MarkDone(id int64) (int64, error) {
	res, err := s.db.Exec(
		`UPDATE tasks SET closed = 1, updated_at = ? WHERE id = ? AND closed = 0`,
		time.Now().Unix(), id,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) Delete(id int64) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
