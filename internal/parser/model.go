package parser

import "time"

type Item struct {
	GUID string
	URL string
	Title string
	Content string
	PublishedAt *time.Time
}

type Feed struct {
	Title string
	SiteURL string
	Items []Item
}
