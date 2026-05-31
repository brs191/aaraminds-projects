package tools

import (
	"encoding/json"
	"testing"
)

func TestFixturesAreValidJSON(t *testing.T) {
	for _, f := range []string{"fixtures/active_sprint.json", "fixtures/sprint_issues.json"} {
		data, err := fixtures.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
	}
}

func TestSprintIssuesShape(t *testing.T) {
	data, _ := fixtures.ReadFile("fixtures/sprint_issues.json")
	var doc struct {
		Issues []map[string]any `json:"issues"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Issues) == 0 {
		t.Fatal("expected issues in fixture")
	}
	// time-based estimation: the field must be present (value may be null).
	if _, ok := doc.Issues[0]["timeoriginalestimate"]; !ok {
		t.Fatal("expected time-tracking field timeoriginalestimate")
	}
}
