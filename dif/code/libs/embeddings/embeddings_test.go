package embeddings

import (
	"context"
	"math"
	"reflect"
	"testing"
)

func TestHashProviderEmbedsDeterministicallyOffline(t *testing.T) {
	t.Parallel()

	provider := NewHashProvider(12)
	request := Request{Inputs: []Input{
		{ID: "passage-1", Text: "The architecture service is owned by Platform Architecture.", Metadata: map[string]string{"source_ref": "golden-admitted@docver-architecture-overview:md:architecture-overview.md#L5-L8"}},
		{ID: "passage-2", Text: "Retry the ingestion job once after verifying the source path is reachable."},
	}}
	first, err := provider.Embed(context.Background(), request)
	if err != nil {
		t.Fatalf("embed first batch: %v", err)
	}
	second, err := provider.Embed(context.Background(), request)
	if err != nil {
		t.Fatalf("embed second batch: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("hash provider embeddings must be deterministic")
	}
	if first.ProviderName != DefaultHashProviderName || first.Model != DefaultHashModel || first.Dimensions != 12 {
		t.Fatalf("unexpected provider metadata: %+v", first)
	}
	if len(first.Vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %+v", first.Vectors)
	}
	for _, vector := range first.Vectors {
		if len(vector.Values) != 12 {
			t.Fatalf("expected 12 dimensions, got %d", len(vector.Values))
		}
		if norm := vectorNorm(vector.Values); norm < 0.99 || norm > 1.01 {
			t.Fatalf("expected normalized vector, norm=%f values=%+v", norm, vector.Values)
		}
	}
}

func TestHashProviderUsageMeteringPlaceholders(t *testing.T) {
	t.Parallel()

	response, err := NewHashProvider(8).Embed(context.Background(), Request{Inputs: []Input{
		{ID: "a", Text: "one two three"},
		{ID: "b", Text: "four five"},
	}})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	expected := Usage{
		ProviderName:        DefaultHashProviderName,
		Model:               DefaultHashModel,
		InputCount:          2,
		EmbeddingCount:      2,
		EmbeddingDimensions: 8,
		InputTokenEstimate:  5,
	}
	if !reflect.DeepEqual(response.Usage, expected) {
		t.Fatalf("usage mismatch: expected %+v got %+v", expected, response.Usage)
	}
}

func TestHashProviderRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		provider HashProvider
		request  Request
	}{
		{name: "empty batch", provider: NewHashProvider(8), request: Request{}},
		{name: "missing id", provider: NewHashProvider(8), request: Request{Inputs: []Input{{Text: "text"}}}},
		{name: "empty text", provider: NewHashProvider(8), request: Request{Inputs: []Input{{ID: "a", Text: " \n\t"}}}},
		{name: "duplicate id", provider: NewHashProvider(8), request: Request{Inputs: []Input{{ID: "a", Text: "one"}, {ID: "a", Text: "two"}}}},
		{name: "invalid dimensions", provider: NewHashProvider(4097), request: Request{Inputs: []Input{{ID: "a", Text: "one"}}}},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := testCase.provider.Embed(context.Background(), testCase.request); err == nil {
				t.Fatal("expected invalid embedding request to fail")
			}
		})
	}
}

func TestHashProviderHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewHashProvider(8).Embed(ctx, Request{Inputs: []Input{{ID: "a", Text: "text"}}}); err == nil {
		t.Fatal("expected cancelled context error")
	}
}

func TestEstimateTokensIsDeterministicPlaceholder(t *testing.T) {
	t.Parallel()

	if tokens := EstimateTokens("PaymentService owner: platform-payments"); tokens != 4 {
		t.Fatalf("unexpected token estimate: %d", tokens)
	}
}

func vectorNorm(values []float32) float64 {
	var sum float64
	for _, value := range values {
		sum += float64(value) * float64(value)
	}
	return math.Sqrt(sum)
}
