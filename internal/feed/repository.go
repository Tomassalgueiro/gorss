package feed

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"errors"
)

var ErrNotFound = errors.New("feed not found")

type Repository struct {
	db *sql.DB
}

func  NewRepository(db *sql.DB) *Repository{
	r := Repository{db: db}
	return &r
}

func (r *Repository) CreateFeed(ctx context.Context, f *Feed) error {
	query := `
		INSERT INTO feeds (
			feed_url,
			site_url,
			title,
			etag,
			last_modified
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at;
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		f.FeedURL,
		f.SiteURL,
		f.Title,
		f.ETag,
		f.LastModified,
	).Scan(
		&f.ID,
		&f.CreatedAt,
		&f.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert feed: %w", err)
	}

	return nil
}

func (r *Repository) GetFeedByID(ctx context.Context, id int64) (*Feed, error) {
	var f Feed
	query := `
		SELECT id, 
		       feed_url,
		       site_url,
		       title,
		       etag,
		       last_modified,
		       last_fetched_at,
		       last_error,
		       created_at,
		       updated_at
		FROM feeds
		WHERE id = ?;
	`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.FeedURL,
		&f.SiteURL,
		&f.Title,
		&f.ETag,
		&f.LastModified,
		&f.LastFetchedAt,
		&f.LastError,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
	    if errors.Is(err, sql.ErrNoRows) {
		    return nil, ErrNotFound
	    }
    	    return nil, fmt.Errorf("get feed: %w", err)
	}
	return &f, nil
}

func (r *Repository) ListFeeds(ctx context.Context) ([]*Feed, error) {
	query := `
        SELECT id, 
               feed_url, 
               site_url, 
               title, 
               etag, 
               last_modified, 
               last_fetched_at, 
               last_error, 
               created_at, 
               updated_at
        FROM feeds
        ORDER BY created_at DESC;
    	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*Feed
	for rows.Next(){
		var f Feed
		err := rows.Scan(
		    &f.ID,
		    &f.FeedURL,
		    &f.SiteURL,
		    &f.Title,
		    &f.ETag,
		    &f.LastModified,
		    &f.LastFetchedAt,
		    &f.LastError,
		    &f.CreatedAt,
		    &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}
	return feeds, nil
}

func (r *Repository) GetFeedsToFetch(ctx context.Context, olderThan time.Duration, limit int) ([]*Feed, error) {
	threshold := time.Now().UTC().Add(-olderThan)

	query := `
		SELECT id, feed_url, site_url, title, etag, last_modified,
		       last_fetched_at, last_error, created_at, updated_at
		FROM feeds
		WHERE last_fetched_at IS NULL OR last_fetched_at < ?
		ORDER BY last_fetched_at ASC NULLS FIRST
		LIMIT ?;
	`

	rows, err := r.db.QueryContext(ctx, query, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("query feeds to fetch: %w", err)
	}
	defer rows.Close()

	var feeds []*Feed
	for rows.Next() {
		var f Feed
		err := rows.Scan(
			&f.ID,
			&f.FeedURL,
			&f.SiteURL,
			&f.Title,
			&f.ETag,
			&f.LastModified,
			&f.LastFetchedAt,
			&f.LastError,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feed to fetch: %w", err)
		}
		feeds = append(feeds, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds to fetch: %w", err)
	}

	return feeds, nil
}

func (r *Repository) UpdateFeedFetchStatus(ctx context.Context, id int64, etag, lastModified string, fetchErr error) error {
	var errStr *string
	if fetchErr != nil {
		s := fetchErr.Error()
		errStr = &s
	}

	now := time.Now().UTC()

	query := `
		UPDATE feeds
		SET etag = CASE WHEN ? != '' THEN ? ELSE etag END,
		    last_modified = CASE WHEN ? != '' THEN ? ELSE last_modified END,
		    last_fetched_at = ?,
		    last_error = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`

	_, err := r.db.ExecContext(ctx, query, etag, etag, lastModified, lastModified, now, errStr, id)
	if err != nil {
		return fmt.Errorf("update feed fetch status: %w", err)
	}

	return nil
}

func (r *Repository) DeleteFeed(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM feeds WHERE id = ?;", id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
