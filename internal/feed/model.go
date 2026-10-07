package feed

import (
	"time"
)

type Feed struct {
	id int64
	feed_url string
	site_url string
	title string
	etag string
	last_modified string
	last_fetched_at *time.Time
	last_error string 
	created_at *time.Time
	updated_at *time.Time
}
