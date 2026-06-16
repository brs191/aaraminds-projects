package adapter

import (
	"context"
	"testing"

	armrg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

// pagedQuerier is a mock argQuerier that returns a fixed sequence of pages and
// records the SkipToken received on each call (to assert it is threaded).
type pagedQuerier struct {
	pages     [][]map[string]interface{}
	tokens    []*string // SkipToken to return per page; last must be nil
	gotTokens []*string // SkipToken received per call
	callIndex int
}

func sptr(s string) *string { return &s }

func (p *pagedQuerier) Resources(_ context.Context, q armrg.QueryRequest, _ *armrg.ClientResourcesOptions) (armrg.ClientResourcesResponse, error) {
	var recv *string
	if q.Options != nil {
		recv = q.Options.SkipToken
	}
	p.gotTokens = append(p.gotTokens, recv)
	i := p.callIndex
	p.callIndex++
	return armrg.ClientResourcesResponse{QueryResponse: armrg.QueryResponse{
		Data:      p.pages[i],
		SkipToken: p.tokens[i],
	}}, nil
}

// C-1 regression: a query whose results span multiple Resource Graph pages must
// follow the SkipToken and return ALL rows — not just the first ~1000-row page.
func TestRunKQL_FollowsSkipTokenAcrossPages(t *testing.T) {
	q := &pagedQuerier{
		pages: [][]map[string]interface{}{
			{{"name": "r1"}, {"name": "r2"}},
			{{"name": "r3"}, {"name": "r4"}},
			{{"name": "r5"}},
		},
		tokens: []*string{sptr("t1"), sptr("t2"), nil},
	}
	a := &adapter{subscriptionID: "sub-1"}
	rows, err := a.runKQL(context.Background(), q, "Resources | project name")
	if err != nil {
		t.Fatalf("runKQL: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows across 3 pages, got %d (single-page truncation bug?)", len(rows))
	}
	if q.callIndex != 3 {
		t.Fatalf("expected 3 page requests, got %d", q.callIndex)
	}
	if q.gotTokens[0] != nil {
		t.Errorf("page 1 must send no SkipToken, got %v", q.gotTokens[0])
	}
	if q.gotTokens[1] == nil || *q.gotTokens[1] != "t1" {
		t.Errorf("page 2 must send SkipToken t1, got %v", q.gotTokens[1])
	}
	if q.gotTokens[2] == nil || *q.gotTokens[2] != "t2" {
		t.Errorf("page 3 must send SkipToken t2, got %v", q.gotTokens[2])
	}
}

func TestRunKQL_SinglePageNoToken(t *testing.T) {
	q := &pagedQuerier{
		pages:  [][]map[string]interface{}{{{"name": "only"}}},
		tokens: []*string{nil},
	}
	a := &adapter{subscriptionID: "sub-1"}
	rows, err := a.runKQL(context.Background(), q, "Resources")
	if err != nil {
		t.Fatalf("runKQL: %v", err)
	}
	if len(rows) != 1 || q.callIndex != 1 {
		t.Fatalf("single page: rows=%d calls=%d (want 1/1)", len(rows), q.callIndex)
	}
}

func TestRunKQL_EmptyResult(t *testing.T) {
	q := &pagedQuerier{pages: [][]map[string]interface{}{nil}, tokens: []*string{nil}}
	a := &adapter{subscriptionID: "sub-1"}
	rows, err := a.runKQL(context.Background(), q, "Resources")
	if err != nil {
		t.Fatalf("runKQL: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}
