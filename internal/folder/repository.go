package folder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("folder not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, f *Folder) error {
	query := `
		INSERT INTO folders (name)
		VALUES (?)
		RETURNING id, created_at;
	`

	err := r.db.QueryRowContext(ctx, query, f.Name).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert folder: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Folder, error) {
	var f Folder
	query := `
		SELECT id, name, created_at
		FROM folders
		WHERE id = ?;
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(&f.ID, &f.Name, &f.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get folder: %w", err)
	}

	return &f, nil
}

func (r *Repository) List(ctx context.Context) ([]*Folder, error) {
	query := `
		SELECT id, name, created_at
		FROM folders
		ORDER BY name COLLATE NOCASE;
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	defer rows.Close()

	var folders []*Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan folder: %w", err)
		}
		folders = append(folders, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate folders: %w", err)
	}

	return folders, nil
}

// Delete removes the folder and moves its feeds back to the top level.
// The feeds are detached explicitly instead of relying on ON DELETE SET NULL,
// since the foreign_keys pragma is only set on one pooled connection.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "UPDATE feeds SET folder_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE folder_id = ?;", id)
	if err != nil {
		return fmt.Errorf("detach feeds: %w", err)
	}

	res, err := tx.ExecContext(ctx, "DELETE FROM folders WHERE id = ?;", id)
	if err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
