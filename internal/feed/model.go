package feed

import (
	"time"
)

type Feed struct {
	ID int64
	FeedURL string
	SiteURL string
	Title string
	ETag string
	LastModified string
	LastFetchedAt *time.Time
	LastError *string 
	CreatedAt time.Time
	UpdatedAt time.Time
}
