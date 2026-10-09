package opml

import (
	"encoding/xml"
	"fmt"
	"io"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title string `xml:"title"`
}

type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	Text        string    `xml:"text,attr"`
	Title       string    `xml:"title,attr,omitempty"`
	Type        string    `xml:"type,attr,omitempty"`
	XMLURL      string    `xml:"xmlUrl,attr,omitempty"`
	HTMLURL     string    `xml:"htmlUrl,attr,omitempty"`
	Outlines    []Outline `xml:"outline,omitempty"` 
}

func Generate(title string, feeds []FeedItem) ([]byte, error) {
	doc := OPML{
		Version: "2.0",
		Head: Head{
			Title: title,
		},
		Body: Body{
			Outlines: make([]Outline, 0, len(feeds)),
		},
	}

	for _, f := range feeds {
		t := f.Title
		if t == "" {
			t = f.FeedURL
		}
		doc.Body.Outlines = append(doc.Body.Outlines, Outline{
			Text:    t,
			Title:   t,
			Type:    "rss",
			XMLURL:  f.FeedURL,
			HTMLURL: f.SiteURL,
		})
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal opml: %w", err)
	}

	header := []byte(xml.Header)
	return append(header, output...), nil
}

type FeedItem struct {
	Title   string
	FeedURL string
	SiteURL string
}

func Parse(r io.Reader) ([]string, error) {
	var doc OPML
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode opml: %w", err)
	}

	var urls []string
	var walk func(outlines []Outline)
	walk = func(outlines []Outline) {
		for _, o := range outlines {
			if o.XMLURL != "" {
				urls = append(urls, o.XMLURL)
			}
			if len(o.Outlines) > 0 {
				walk(o.Outlines)
			}
		}
	}

	walk(doc.Body.Outlines)
	return urls, nil
}
