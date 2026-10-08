package article

import "time"

type Article struct {
	    ID          int64      `json:"id"`
	    FeedID      int64      `json:"feed_id"`
	    GUID        string     `json:"guid"`
	    URL         string     `json:"url"`
	    Title       string     `json:"title"`
	    Content     string     `json:"content"`
	    PublishedAt *time.Time `json:"published_at"`
	    CreatedAt   time.Time  `json:"created_at"`
	    UpdatedAt   time.Time  `json:"updated_at"`
}
