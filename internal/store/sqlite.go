package store

import (
	"database/sql"
	"errors"
	"time"

	"go-cloud-ide/internal/apperr"
	_ "modernc.org/sqlite"
)

type Status string

const (
	StatusCreating Status = "creating"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusError    Status = "error"
)

type Workspace struct {
	ID          string
	ContainerID string
	Volume      string
	Port        string
	Status      Status
	CreatedAt   time.Time
	LastActive  time.Time
}

type Store struct {
	db *sql.DB
}

// New opens the SQLite workspace store and ensures its schema exists.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, apperr.E("store.open", apperr.KindInternal, "failed to open workspace store", err)
	}

	schema := `
    CREATE TABLE IF NOT EXISTS workspaces (
        id TEXT PRIMARY KEY,
        container_id TEXT,
        volume TEXT,
        port TEXT,
        status TEXT DEFAULT 'creating',
        created_at DATETIME,
        last_active DATETIME
    );`

	if _, err = db.Exec(schema); err != nil {
		return nil, apperr.E("store.migrate", apperr.KindInternal, "failed to initialize workspace store", err)
	}

	// Migration: Add status column if it doesn't exist (for old databases)
	if _, err = db.Exec(`ALTER TABLE workspaces ADD COLUMN status TEXT DEFAULT 'creating'`); err != nil {
		// Ignore error - column probably already exists
	}

	return &Store{db: db}, nil
}

// Save persists a workspace record to the database.
func (s *Store) Save(ws *Workspace) error {
	_, err := s.db.Exec(`INSERT INTO workspaces (id, container_id, volume, port, status, created_at, last_active) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.ContainerID, ws.Volume, ws.Port, ws.Status, ws.CreatedAt, ws.LastActive)
	return apperr.E("store.save", apperr.KindInternal, "failed to save workspace", err)
}

// Update persists changes to an existing workspace.
func (s *Store) Update(ws *Workspace) error {
	result, err := s.db.Exec(`UPDATE workspaces SET container_id = ?, volume = ?, port = ?, status = ?, last_active = ? WHERE id = ?`,
		ws.ContainerID, ws.Volume, ws.Port, ws.Status, ws.LastActive, ws.ID)
	if err != nil {
		return apperr.E("store.update", apperr.KindInternal, "failed to update workspace", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperr.E("store.update.rows_affected", apperr.KindInternal, "failed to update workspace", err)
	}

	if rowsAffected == 0 {
		return apperr.New("store.update", apperr.KindNotFound, "workspace not found")
	}

	return nil
}

// UpdateStatus updates only the status field of a workspace.
func (s *Store) UpdateStatus(id string, status Status) error {
	result, err := s.db.Exec(`UPDATE workspaces SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return apperr.E("store.update_status", apperr.KindInternal, "failed to update workspace status", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperr.E("store.update_status.rows_affected", apperr.KindInternal, "failed to update workspace status", err)
	}

	if rowsAffected == 0 {
		return apperr.New("store.update_status", apperr.KindNotFound, "workspace not found")
	}

	return nil
}

// Get loads a single workspace by its ID.
func (s *Store) Get(id string) (*Workspace, error) {
	row := s.db.QueryRow(`SELECT id, container_id, volume, port, status, created_at, last_active FROM workspaces WHERE id = ?`, id)
	ws := &Workspace{}
	var createdAt, lastActive string
	err := row.Scan(&ws.ID, &ws.ContainerID, &ws.Volume, &ws.Port, &ws.Status, &createdAt, &lastActive)
	if err == nil {
		ws.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		ws.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActive)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.New("store.get", apperr.KindNotFound, "workspace not found")
	}

	if err != nil {
		return nil, apperr.E("store.get", apperr.KindInternal, "failed to load workspace", err)
	}

	return ws, err
}

// List returns all tracked workspaces from the database.
func (s *Store) List() ([]*Workspace, error) {
	rows, err := s.db.Query(`SELECT id, container_id, volume, port, status, created_at, last_active FROM workspaces`)
	if err != nil {
		return nil, apperr.E("store.list.query", apperr.KindInternal, "failed to list workspaces", err)
	}
	defer rows.Close()

	var list []*Workspace
	for rows.Next() {
		ws := &Workspace{}
		var createdAt, lastActive string
		if err := rows.Scan(&ws.ID, &ws.ContainerID, &ws.Volume, &ws.Port, &ws.Status, &createdAt, &lastActive); err != nil {
			return nil, apperr.E("store.list.scan", apperr.KindInternal, "failed to read workspace list", err)
		}
		ws.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		ws.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActive)
		list = append(list, ws)
	}

	if err := rows.Err(); err != nil {
		return nil, apperr.E("store.list.rows", apperr.KindInternal, "failed to read workspace list", err)
	}

	return list, nil
}

// ListByStatus returns workspaces filtered by status.
func (s *Store) ListByStatus(status Status) ([]*Workspace, error) {
	rows, err := s.db.Query(`SELECT id, container_id, volume, port, status, created_at, last_active FROM workspaces WHERE status = ?`, status)
	if err != nil {
		return nil, apperr.E("store.list_by_status.query", apperr.KindInternal, "failed to list workspaces", err)
	}
	defer rows.Close()

	var list []*Workspace
	for rows.Next() {
		ws := &Workspace{}
		var createdAt, lastActive string
		if err := rows.Scan(&ws.ID, &ws.ContainerID, &ws.Volume, &ws.Port, &ws.Status, &createdAt, &lastActive); err != nil {
			return nil, apperr.E("store.list_by_status.scan", apperr.KindInternal, "failed to read workspace list", err)
		}
		ws.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		ws.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActive)
		list = append(list, ws)
	}

	if err := rows.Err(); err != nil {
		return nil, apperr.E("store.list_by_status.rows", apperr.KindInternal, "failed to read workspace list", err)
	}

	return list, nil
}

// Delete removes a workspace record and reports when it does not exist.
func (s *Store) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return apperr.E("store.delete", apperr.KindInternal, "failed to delete workspace", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperr.E("store.delete.rows_affected", apperr.KindInternal, "failed to delete workspace", err)
	}

	if rowsAffected == 0 {
		return apperr.New("store.delete", apperr.KindNotFound, "workspace not found")
	}

	return nil
}

// UpdateLastActive records a fresh heartbeat time for a workspace.
func (s *Store) UpdateLastActive(id string) error {
	result, err := s.db.Exec(`UPDATE workspaces SET last_active = ? WHERE id = ?`, time.Now(), id)
	if err != nil {
		return apperr.E("store.update_last_active", apperr.KindInternal, "failed to update workspace heartbeat", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperr.E("store.update_last_active.rows_affected", apperr.KindInternal, "failed to update workspace heartbeat", err)
	}

	if rowsAffected == 0 {
		return apperr.New("store.update_last_active", apperr.KindNotFound, "workspace not found")
	}

	return nil
}
