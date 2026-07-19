// Package rss parses podcast RSS 2.0 feeds (channel metadata + items with
// enclosures and itunes:duration) using only encoding/xml. Library mode's
// poller feeds on this; Atom is intentionally not supported in MVP.
package rss

import (
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// Feed is the parsed channel plus its enclosure-bearing items.
type Feed struct {
	Title       string
	Description string
	ImageURL    string
	Items       []Item
}

// Item is one episode entry. Items without an enclosure URL are dropped
// during parsing (they are not playable episodes). GUID falls back to the
// enclosure URL when the feed omits <guid>.
type Item struct {
	GUID            string
	Title           string
	Description     string
	EnclosureURL    string
	EnclosureType   string
	PublishedAt     *time.Time
	DurationSeconds *int
}

type xmlImage struct {
	URL  string `xml:"url"`       // RSS 2.0 <image><url>
	Href string `xml:"href,attr"` // itunes:image href attribute
}

type xmlEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type xmlItem struct {
	Title       string       `xml:"title"`
	Description string       `xml:"description"`
	GUID        string       `xml:"guid"`
	PubDate     string       `xml:"pubDate"`
	Durations   []string     `xml:"duration"` // itunes:duration (namespace-agnostic)
	Enclosure   xmlEnclosure `xml:"enclosure"`
}

type xmlRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title       string     `xml:"title"`
		Description string     `xml:"description"`
		Images      []xmlImage `xml:"image"` // standard <image> and itunes:image
		Items       []xmlItem  `xml:"item"`
	} `xml:"channel"`
}

// Parse decodes an RSS 2.0 document. Feeds without an <rss> root (e.g. Atom)
// are rejected with FEED_FETCH_FAILED.
func Parse(data []byte) (*Feed, error) {
	var doc xmlRSS
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, domain.E(domain.CodeFeedFetchFailed, "not a parseable RSS 2.0 feed: %v", err)
	}
	feed := &Feed{
		Title:       strings.TrimSpace(doc.Channel.Title),
		Description: strings.TrimSpace(doc.Channel.Description),
	}
	for _, img := range doc.Channel.Images {
		if u := strings.TrimSpace(img.URL); u != "" {
			feed.ImageURL = u
			break
		}
		if h := strings.TrimSpace(img.Href); h != "" && feed.ImageURL == "" {
			feed.ImageURL = h
		}
	}
	for _, it := range doc.Channel.Items {
		encURL := strings.TrimSpace(it.Enclosure.URL)
		if encURL == "" {
			continue // no enclosure, not a playable episode
		}
		guid := strings.TrimSpace(it.GUID)
		if guid == "" {
			guid = encURL // missing guid falls back to the enclosure URL
		}
		item := Item{
			GUID:          guid,
			Title:         strings.TrimSpace(it.Title),
			Description:   strings.TrimSpace(it.Description),
			EnclosureURL:  encURL,
			EnclosureType: strings.TrimSpace(it.Enclosure.Type),
			PublishedAt:   parsePubDate(it.PubDate),
		}
		for _, d := range it.Durations {
			if secs, ok := parseDuration(d); ok {
				item.DurationSeconds = &secs
				break
			}
		}
		feed.Items = append(feed.Items, item)
	}
	return feed, nil
}

// parseDuration handles itunes:duration in hh:mm:ss, mm:ss, and plain-seconds
// forms (integer or decimal).
func parseDuration(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) > 3 {
		return 0, false
	}
	if len(parts) == 1 {
		if n, err := strconv.Atoi(parts[0]); err == nil && n >= 0 {
			return n, true
		}
		if f, err := strconv.ParseFloat(parts[0], 64); err == nil && f >= 0 {
			return int(f), true
		}
		return 0, false
	}
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 {
			return 0, false
		}
		total = total*60 + n
	}
	return total, true
}

var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700",
	time.RFC3339,
}

func parsePubDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}
