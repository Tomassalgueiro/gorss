package article

import (
	"context"
	"database/sql"
	"strings"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("article not found")

type Repository struct {
	db *sql.DB
}

type ListFilter struct {
	FeedID *int64
	UnreadOnly bool
	StarredOnly bool
	Limit int
	Offset int
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
		       is_read,
		       is_starred,
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
			&a.IsRead,
			&a.IsStarred,
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
		       is_read,
		       is_starred,
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
		&a.IsRead,
		&a.IsStarred,
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

func (r *Repository) UpdateStatus(ctx context.Context, id int64, isRead, isStarred *bool) (*Article, error) {
	var setClauses []string
	var args []any

	if isRead != nil {
		setClauses = append(setClauses, "is_read = ?")
		val := 0
		if *isRead {
			val = 1
		}
		args = append(args, val)
	}

	if isStarred != nil {
		setClauses = append(setClauses, "is_starred = ?")
		val := 0
		if *isStarred {
			val = 1
		}
		args = append(args, val)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE articles
		SET %s
		WHERE id = ?;
	`, strings.Join(setClauses, ", "))

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update article status: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) MarkFeedAsRead(ctx context.Context, feedID int64) error {
	query := `
		UPDATE articles
		SET is_read = 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE feed_id = ? AND is_read = 0;
	`

	_, err := r.db.ExecContext(ctx, query, feedID)
	if err != nil {
		return fmt.Errorf("mark feed as read: %w", err)
	}

	return nil
}

func (r *Repository) ListAll(ctx context.Context, filter ListFilter) ([]*Article, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var whereClauses []string
	var args []any

	if filter.FeedID != nil {
		whereClauses = append(whereClauses, "feed_id = ?")
		args = append(args, *filter.FeedID)
	}

	if filter.UnreadOnly {
		whereClauses = append(whereClauses, "is_read = 0")
	}

	if filter.StarredOnly {
		whereClauses = append(whereClauses, "is_starred = 1")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id,
		       feed_id,
		       guid,
		       url,
		       title,
		       content,
		       published_at,
		       is_read,
		       is_starred,
		       created_at,
		       updated_at
		FROM articles
		%s
		ORDER BY COALESCE(published_at, created_at) DESC, id DESC
		LIMIT ? OFFSET ?;
	`, whereSQL)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
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
			&a.IsRead,
			&a.IsStarred,
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

	if articles == nil {
		articles = []*Article{}
	}

	return articles, nil
}
