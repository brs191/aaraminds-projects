package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	embeddingMaxAttempts = 3
	embeddingBackoffBase = 250 * time.Millisecond
)

type embeddingHealth struct {
	Status string `json:"status"`
	Model  string `json:"model"`
	Dim    int    `json:"dim"`
}

type embeddingItem struct {
	NodeID string `json:"node_id"`
	Text   string `json:"text"`
}

type embeddingResult struct {
	NodeID    string    `json:"node_id"`
	Embedding []float32 `json:"embedding"`
}

type embeddingClient struct {
	baseURL string
	client  *http.Client
}

func newEmbeddingClient(baseURL string) *embeddingClient {
	return &embeddingClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *embeddingClient) enabled() bool {
	return c != nil && c.baseURL != ""
}

func (c *embeddingClient) Health(ctx context.Context) (*embeddingHealth, error) {
	if !c.enabled() {
		return nil, nil
	}
	var health embeddingHealth
	if err := c.doWithRetry(ctx, http.MethodGet, c.baseURL+"/health", nil, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

func (c *embeddingClient) Embed(ctx context.Context, batch []embeddingItem) ([]embeddingResult, error) {
	if !c.enabled() || len(batch) == 0 {
		return nil, nil
	}
	var out []embeddingResult
	if err := c.doWithRetry(ctx, http.MethodPost, c.baseURL+"/embed", batch, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *embeddingClient) doWithRetry(ctx context.Context, method, url string, payload any, out any) error {
	var bodyBytes []byte
	var err error
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal embedding payload: %w", err)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= embeddingMaxAttempts; attempt++ {
		reqBody := bytes.NewReader(bodyBytes)
		req, reqErr := http.NewRequestWithContext(ctx, method, url, reqBody)
		if reqErr != nil {
			return reqErr
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, callErr := c.client.Do(req)
		if callErr == nil && resp.StatusCode < 500 {
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
				return fmt.Errorf("embedding service %s %s returned HTTP %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(data)))
			}
			if out != nil {
				if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
					return fmt.Errorf("decode embedding response: %w", err)
				}
			}
			return nil
		}

		if callErr == nil {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			lastErr = fmt.Errorf("embedding service %s %s returned HTTP %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(data)))
		} else {
			lastErr = callErr
		}

		if attempt == embeddingMaxAttempts {
			break
		}
		backoff := time.Duration(attempt) * embeddingBackoffBase
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return fmt.Errorf("embedding request failed after %d attempts: %w", embeddingMaxAttempts, lastErr)
}
