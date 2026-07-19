// Package embeddings defines DIF's embedding provider seam.
package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultHashProviderName = "dif-hash-stub"
	DefaultHashModel        = "hash-stub-v0"
	DefaultHashDimensions   = 16
)

var tokenPattern = regexp.MustCompile(`[A-Za-z0-9]+`)

// Provider abstracts embedding generation so P0 tests can stay offline while a
// later implementation can call the shared RIF/LiteLLM provider.
type Provider interface {
	Name() string
	Model() string
	Dimensions() int
	Embed(ctx context.Context, request Request) (Response, error)
}

// Input is one text item to embed.
type Input struct {
	ID       string
	Text     string
	Metadata map[string]string
}

// Request is a batch embedding request.
type Request struct {
	Inputs []Input
}

// Vector is one embedded item.
type Vector struct {
	ID       string
	Values   []float32
	Metadata map[string]string
}

// Response is the provider output plus metering placeholders.
type Response struct {
	ProviderName string
	Model        string
	Dimensions   int
	Vectors      []Vector
	Usage        Usage
}

// Usage captures non-PII metering fields that can later feed usage_events.
type Usage struct {
	ProviderName        string
	Model               string
	InputCount          int
	EmbeddingCount      int
	EmbeddingDimensions int
	InputTokenEstimate  int
}

// HashProvider is a deterministic offline provider for tests and local
// development. It is not a production embedding model and does not imply a
// persisted vector dimension.
type HashProvider struct {
	ProviderName string
	ModelName    string
	VectorSize   int
}

// NewHashProvider returns a deterministic offline provider.
func NewHashProvider(dimensions int) HashProvider {
	return HashProvider{ProviderName: DefaultHashProviderName, ModelName: DefaultHashModel, VectorSize: dimensions}
}

// Name returns the provider name.
func (p HashProvider) Name() string {
	if strings.TrimSpace(p.ProviderName) == "" {
		return DefaultHashProviderName
	}
	return strings.TrimSpace(p.ProviderName)
}

// Model returns the model identifier.
func (p HashProvider) Model() string {
	if strings.TrimSpace(p.ModelName) == "" {
		return DefaultHashModel
	}
	return strings.TrimSpace(p.ModelName)
}

// Dimensions returns the vector length used by this offline provider.
func (p HashProvider) Dimensions() int {
	if p.VectorSize <= 0 {
		return DefaultHashDimensions
	}
	return p.VectorSize
}

// Embed deterministically maps text to normalized hash vectors.
func (p HashProvider) Embed(ctx context.Context, request Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	if len(request.Inputs) == 0 {
		return Response{}, errors.New("embedding request requires at least one input")
	}
	dimensions := p.Dimensions()
	if dimensions <= 0 || dimensions > 4096 {
		return Response{}, fmt.Errorf("embedding dimensions must be between 1 and 4096, got %d", dimensions)
	}

	vectors := make([]Vector, 0, len(request.Inputs))
	tokenEstimate := 0
	seen := map[string]bool{}
	for _, input := range request.Inputs {
		id := strings.TrimSpace(input.ID)
		if id == "" {
			return Response{}, errors.New("embedding input id is required")
		}
		if seen[id] {
			return Response{}, fmt.Errorf("duplicate embedding input id %q", id)
		}
		seen[id] = true
		text := strings.TrimSpace(input.Text)
		if text == "" {
			return Response{}, fmt.Errorf("embedding input %q text is required", id)
		}
		tokenEstimate += EstimateTokens(text)
		vectors = append(vectors, Vector{
			ID:       id,
			Values:   hashVector(p.Name(), p.Model(), dimensions, text),
			Metadata: sortedMetadata(input.Metadata),
		})
	}
	return Response{
		ProviderName: p.Name(),
		Model:        p.Model(),
		Dimensions:   dimensions,
		Vectors:      vectors,
		Usage: Usage{
			ProviderName:        p.Name(),
			Model:               p.Model(),
			InputCount:          len(request.Inputs),
			EmbeddingCount:      len(vectors),
			EmbeddingDimensions: dimensions,
			InputTokenEstimate:  tokenEstimate,
		},
	}, nil
}

// EstimateTokens returns a deterministic placeholder token estimate for
// metering until the real provider reports token usage.
func EstimateTokens(text string) int {
	return len(tokenPattern.FindAllString(text, -1))
}

func hashVector(providerName, model string, dimensions int, text string) []float32 {
	values := make([]float32, dimensions)
	seed := strings.Join([]string{providerName, model, text}, "\x00")
	var magnitude float64
	for index := 0; index < dimensions; index++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", seed, index)))
		raw := binary.BigEndian.Uint32(sum[:4])
		scaled := (float64(raw)/float64(math.MaxUint32))*2 - 1
		values[index] = float32(scaled)
		magnitude += scaled * scaled
	}
	if magnitude == 0 {
		return values
	}
	norm := math.Sqrt(magnitude)
	for index := range values {
		values[index] = float32(float64(values[index]) / norm)
	}
	return values
}

func sortedMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sorted := make(map[string]string, len(metadata))
	for _, key := range keys {
		sorted[strings.TrimSpace(key)] = strings.TrimSpace(metadata[key])
	}
	return sorted
}
