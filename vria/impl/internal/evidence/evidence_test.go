package evidence

import (
	"testing"
	"time"

	"github.com/aaraminds/vria/internal/enums"
)

var window = 30 * 24 * time.Hour // monthly default (06 §8)

func TestFreshnessCadence(t *testing.T) {
	now := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		age  time.Duration
		want enums.Freshness
	}{
		{10 * 24 * time.Hour, enums.Fresh},
		{45 * 24 * time.Hour, enums.Aging}, // 1 missed window
		{70 * 24 * time.Hour, enums.Stale}, // 2+ missed windows
	}
	for _, c := range cases {
		got := FreshnessFor(now.Add(-c.age), now, window)
		if got != c.want {
			t.Fatalf("age %v: got %s want %s", c.age, got, c.want)
		}
	}
	if FreshnessFor(time.Time{}, now, window) != enums.FreshnessUnknown {
		t.Fatal("zero snapshot time must be Unknown")
	}
}

func TestNextCheckComputable(t *testing.T) {
	last := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	next := NextSustainmentCheck(last, window)
	if next != last.Add(window) {
		t.Fatal("next check must be last + reporting window")
	}
}

func TestConflictPrefersAuthoritative(t *testing.T) {
	old := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	c := ResolveConflict([]Source{
		{SourceID: "sec", Value: 40, Authority: enums.Secondary, RetrievedAt: newer},
		{SourceID: "auth", Value: 25, Authority: enums.Authoritative, RetrievedAt: old},
	})
	if !c.Resolved || c.Preferred.SourceID != "auth" {
		t.Fatalf("must prefer authoritative source: %+v", c)
	}
	if len(c.Others) != 1 {
		t.Fatal("conflicting value must be surfaced, never dropped or averaged")
	}
}

func TestConflictUnresolvedWithoutAuthority(t *testing.T) {
	now := time.Now()
	c := ResolveConflict([]Source{
		{SourceID: "a", Value: 10, Authority: enums.Secondary, RetrievedAt: now.Add(-time.Hour)},
		{SourceID: "b", Value: 12, Authority: enums.Secondary, RetrievedAt: now},
	})
	if c.Resolved {
		t.Fatal("two secondary sources must surface an unresolved conflict")
	}
	if c.Preferred.SourceID != "b" {
		t.Fatal("newer source preferred for display")
	}
}
