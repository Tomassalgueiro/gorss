package parser

import (
	"encoding/xml"
	"errors"
	"strings"
	"time"
)

var ErrUnknownFeedFormat = errors.New("unknown feed format: neither valid RSS nor Atom")

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Link  string    `xml:"link"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Links   []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:",innerxml"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
}

func Parse(data []byte) (*Feed, error) {
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err == nil && rss.Channel.Title != "" {
		return normalizeRSS(&rss), nil
	}
	 
	var atom atomFeed
	if err := xml.Unmarshal(data, &atom); err == nil && atom.Title != "" {
		return normalizeAtom(&atom), nil
	}

	return nil, ErrUnknownFeedFormat
}

func normalizeRSS(rss *rssFeed) *Feed {
	feed := &Feed{
		Title:   strings.TrimSpace(rss.Channel.Title),
		SiteURL: strings.TrimSpace(rss.Channel.Link),
		Items:   make([]Item, 0, len(rss.Channel.Items)),
	}

	for _, it := range rss.Channel.Items {
		guid := strings.TrimSpace(it.GUID)
		link := strings.TrimSpace(it.Link)
		if guid == "" {
			guid = link
		}

		item := Item{
			GUID:        guid,
			URL:         link,
			Title:       strings.TrimSpace(it.Title),
			Content:     strings.TrimSpace(it.Description),
			PublishedAt: parseDate(it.PubDate),
		}
		feed.Items = append(feed.Items, item)
	}

	return feed
}

func normalizeAtom(atom *atomFeed) *Feed {
	feed := &Feed{
		Title:   strings.TrimSpace(atom.Title),
		SiteURL: extractAtomLink(atom.Links),
		Items:   make([]Item, 0, len(atom.Entries)),
	}

	for _, entry := range atom.Entries {
		link := extractAtomLink(entry.Links)
		guid := strings.TrimSpace(entry.ID)
		if guid == "" {
			guid = link
		}

		content := strings.TrimSpace(entry.Content)
		if content == "" {
			content = strings.TrimSpace(entry.Summary)
		}

		dateStr := entry.Published
		if dateStr == "" {
			dateStr = entry.Updated
		}

		item := Item{
			GUID:        guid,
			URL:         link,
			Title:       strings.TrimSpace(entry.Title),
			Content:     content,
			PublishedAt: parseDate(dateStr),
		}
		feed.Items = append(feed.Items, item)
	}

	return feed
}

func extractAtomLink(links []atomLink) string {
	if len(links) == 0 {
		return ""
	}
	for _, l := range links {
		if l.Rel == "alternate" || l.Rel == "" {
			if l.Href != "" {
				return strings.TrimSpace(l.Href)
			}
		}
	}
	return strings.TrimSpace(links[0].Href)
}

func parseDate(dateStr string) *time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, dateStr); err == nil {
			utc := t.UTC()
			return &utc
		}
	}

	return nil
}
