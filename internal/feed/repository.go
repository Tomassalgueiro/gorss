package feed

import (
	"context"
	"database/sql"
	"fmt"
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

