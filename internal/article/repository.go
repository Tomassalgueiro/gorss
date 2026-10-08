package article

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("article not found")

type Repository struct {
	db *sql.DB
}

func NewReposiroty(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateArticles(ctx context.Context, articles []Article) error {
	if len(articles) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO articles (
			feed_id,
			guid,
			url,
			title,
			content,
			published_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(feed_id, guid) DO NOTHING;
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for _, a := range articles {
		_, err := stmt.ExecContext(
			ctx,
			a.FeedID,
			a.GUID,
			a.URL,
			a.Title,
			a.Content,
			a.PublishedAt,
		)
		if err != nil {
			return fmt.Errorf("insert article guid %q: %w", a.GUID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) ListByFeed(ctx context.Context, feedID int64, limit, offset int) ([]*Article, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id,
		       feed_id,
		       guid,
		       url,
		       title,
		       content,
		       published_at,
		       created_at,
		       updated_at
		FROM articles
		WHERE feed_id = ?
		ORDER BY COALESCE(published_at, created_at) DESC, id DESC
		LIMIT ? OFFSET ?;
	`

	rows, err := r.db.QueryContext(ctx, query, feedID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query articles by feed: %w", err)
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		var a Article
		err := rows.Scan(
			&a.ID,
			&a.FeedID,
			&a.GUID,
			&a.URL,
			&a.Title,
			&a.Content,
			&a.PublishedAt,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}

	return articles, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Article, error) {
	query := `
		SELECT id,
		       feed_id,
		       guid,
		       url,
		       title,
		       content,
		       published_at,
		       created_at,
		       updated_at
		FROM articles
		WHERE id = ?;
	`

	var a Article
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.FeedID,
		&a.GUID,
		&a.URL,
		&a.Title,
		&a.Content,
		&a.PublishedAt,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get article: %w", err)
	}

	return &a, nil
}
