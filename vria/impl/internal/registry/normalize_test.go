package registry

import (
	"errors"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
)

// Real status strings from internal/99_Source_AI_Use_Case_Inventory.md.
func TestNormalizeRealInventoryStatuses(t *testing.T) {
	cases := []struct {
		raw     string
		want    enums.DeliveryStatus
		wantErr error
	}{
		{"Training; PTB not started; pending", enums.DSTraining, nil},
		{"PTB", enums.DSPTBInProgress, nil},
		{"In Progress", enums.DSInProgress, nil},
		{"Not captured", enums.DSUnknown, nil},
		{"", enums.DSUnknown, nil},
		{"Training; PTB/PTO not started/completed/pending mixed", enums.DSUnknown, ErrAmbiguousStatus},
		{"some new status", enums.DSUnknown, ErrUnmappedStatus},
	}
	for _, c := range cases {
		got, err := NormalizeDeliveryStatus(c.raw)
		if got != c.want {
			t.Fatalf("%q: got %s want %s", c.raw, got, c.want)
		}
		if c.wantErr != nil && !errors.Is(err, c.wantErr) {
			t.Fatalf("%q: err %v, want %v", c.raw, err, c.wantErr)
		}
		if c.wantErr == nil && err != nil {
			t.Fatalf("%q: unexpected err %v", c.raw, err)
		}
	}
}

func TestStageImportRejectsWithoutGuessing(t *testing.T) {
	rows := []map[string]string{
		{"use_case_id": "UC-1", "name": "A", "tier": "Tool", "delivery_status": "PTB"},
		{"use_case_id": "", "name": "B", "tier": "Agent", "delivery_status": "Pilot"},
		{"use_case_id": "UC-3", "name": "C", "tier": "???", "delivery_status": "PTB/PTO mixed"},
	}
	staged, rejected := StageImport(rows)
	if len(staged) != 3 || rejected != 2 {
		t.Fatalf("staged=%d rejected=%d, want 3/2", len(staged), rejected)
	}
	if staged[2].Tier != enums.TierUnclassified {
		t.Fatalf("unknown tier must map to Unclassified, got %s", staged[2].Tier)
	}
	if len(staged[0].Errors) != 0 {
		t.Fatalf("clean row must have no errors: %v", staged[0].Errors)
	}
}
