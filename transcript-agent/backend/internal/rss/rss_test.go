package rss

import (
	"testing"
	"time"
)

const fixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>Reliable Systems Weekly</title>
    <description>A show about shipping small reliable systems.</description>
    <image>
      <url>https://example.com/cover.png</url>
      <title>Reliable Systems Weekly</title>
      <link>https://example.com</link>
    </image>
    <item>
      <title>Episode 1: Approval Gates</title>
      <description>Why the human gate matters.</description>
      <guid isPermaLink="false">ep-001</guid>
      <pubDate>Mon, 06 Jul 2026 09:30:00 +0000</pubDate>
      <itunes:duration>01:02:03</itunes:duration>
      <enclosure url="https://cdn.example.com/ep1.mp3" type="audio/mpeg" length="1234"/>
    </item>
    <item>
      <title>Episode 2: Immutable Raw Output</title>
      <description>Never edit the source of truth.</description>
      <guid>ep-002</guid>
      <pubDate>Tue, 07 Jul 2026 10:00:00 GMT</pubDate>
      <itunes:duration>3725</itunes:duration>
      <enclosure url="https://cdn.example.com/ep2.m4a" type="audio/mp4" length="5678"/>
    </item>
    <item>
      <title>Episode 3: No GUID Here</title>
      <description>Falls back to the enclosure URL.</description>
      <itunes:duration>45:10</itunes:duration>
      <enclosure url="https://cdn.example.com/ep3.mp3" type="audio/mpeg" length="42"/>
    </item>
    <item>
      <title>Not an episode: no enclosure</title>
      <description>Blog-post-only item, must be dropped.</description>
      <guid>post-001</guid>
    </item>
  </channel>
</rss>`

func TestParseRSSFixture(t *testing.T) {
	feed, err := Parse([]byte(fixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if feed.Title != "Reliable Systems Weekly" {
		t.Errorf("title = %q", feed.Title)
	}
	if feed.Description != "A show about shipping small reliable systems." {
		t.Errorf("description = %q", feed.Description)
	}
	if feed.ImageURL != "https://example.com/cover.png" {
		t.Errorf("image = %q", feed.ImageURL)
	}
	if len(feed.Items) != 3 {
		t.Fatalf("items = %d, want 3 (enclosure-less item dropped)", len(feed.Items))
	}

	ep1 := feed.Items[0]
	if ep1.GUID != "ep-001" || ep1.EnclosureURL != "https://cdn.example.com/ep1.mp3" {
		t.Errorf("ep1 guid/enclosure = %q / %q", ep1.GUID, ep1.EnclosureURL)
	}
	if ep1.DurationSeconds == nil || *ep1.DurationSeconds != 1*3600+2*60+3 {
		t.Errorf("ep1 duration (hh:mm:ss) = %v, want 3723", ep1.DurationSeconds)
	}
	if ep1.PublishedAt == nil {
		t.Fatal("ep1 published_at is nil")
	}
	want := time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC)
	if !ep1.PublishedAt.Equal(want) {
		t.Errorf("ep1 published_at = %v, want %v", ep1.PublishedAt, want)
	}

	ep2 := feed.Items[1]
	if ep2.DurationSeconds == nil || *ep2.DurationSeconds != 3725 {
		t.Errorf("ep2 duration (plain seconds) = %v, want 3725", ep2.DurationSeconds)
	}
	if ep2.PublishedAt == nil {
		t.Error("ep2 published_at (RFC1123/GMT) is nil")
	}

	ep3 := feed.Items[2]
	if ep3.GUID != "https://cdn.example.com/ep3.mp3" {
		t.Errorf("ep3 guid fallback = %q, want the enclosure URL", ep3.GUID)
	}
	if ep3.DurationSeconds == nil || *ep3.DurationSeconds != 45*60+10 {
		t.Errorf("ep3 duration (mm:ss) = %v, want 2710", ep3.DurationSeconds)
	}
	if ep3.PublishedAt != nil {
		t.Errorf("ep3 published_at = %v, want nil", ep3.PublishedAt)
	}
}

func TestParseRejectsNonRSS(t *testing.T) {
	if _, err := Parse([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><title>x</title></feed>`)); err == nil {
		t.Fatal("Atom root accepted, want error")
	}
	if _, err := Parse([]byte(`not xml at all`)); err == nil {
		t.Fatal("garbage accepted, want error")
	}
}

func TestParseDurationForms(t *testing.T) {
	cases := map[string]struct {
		secs int
		ok   bool
	}{
		"3600":     {3600, true},
		"01:00:00": {3600, true},
		"5:03":     {303, true},
		"90.5":     {90, true},
		"":         {0, false},
		"abc":      {0, false},
		"1:2:3:4":  {0, false},
	}
	for in, want := range cases {
		got, ok := parseDuration(in)
		if ok != want.ok || (ok && got != want.secs) {
			t.Errorf("parseDuration(%q) = %d,%v; want %d,%v", in, got, ok, want.secs, want.ok)
		}
	}
}
