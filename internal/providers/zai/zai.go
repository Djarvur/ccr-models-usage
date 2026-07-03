// Package zai implements the z.ai monitoring API adapter.
package zai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

// Adapter is the z.ai implementation of provider.Adapter.
type Adapter struct {
	client    *http.Client
	endpoints []string
}

// New returns a new z.ai Adapter configured to talk to the public
// monitoring endpoints.
func New() *Adapter {
	return &Adapter{
		client: &http.Client{Timeout: httpClientTimeout},
		endpoints: []string{
			"https://api.z.ai/api/monitor/usage/quota/limit",
			"https://open.bigmodel.cn/api/monitor/usage/quota/limit",
		},
	}
}

// httpClientTimeout is the timeout for the HTTP client used to talk
// to the z.ai monitoring API.
const httpClientTimeout = 10 * time.Second

// Host returns the canonical hostname the adapter services.
func (a *Adapter) Host() string { return "api.z.ai" }

// NeedsSessionCreds reports that the z.ai adapter needs no
// session credentials beyond the API key.
func (a *Adapter) NeedsSessionCreds() bool { return false }

// ErrAuthFailed is returned by Fetch when the API rejects the key.
var ErrAuthFailed = errors.New("z.ai: auth failed")

// ErrUnexpectedStatus is wrapped around non-2xx responses from the
// final endpoint. The message carries the last seen HTTP status.
var ErrUnexpectedStatus = errors.New("z.ai: unexpected HTTP status")

// Fetch queries the z.ai monitoring API and returns the limits it
// reports. The level (tariff name) is exposed via the FetchResult.Level
// field; the renderer uses it for the header.
func (a *Adapter) Fetch(ctx context.Context, apiKey string) (provider.FetchResult, error) {
	body, status, err := a.fetchOnce(ctx, a.endpoints[0], apiKey)
	if err != nil {
		return provider.FetchResult{}, err
	}

	if status == http.StatusNotFound {
		// 404 on the international endpoint => fall back to CN.
		var cnErr error

		body, status, cnErr = a.fetchOnce(ctx, a.endpoints[1], apiKey)
		if cnErr != nil {
			return provider.FetchResult{}, cnErr
		}
	}

	if status < 200 || status >= 300 {
		return provider.FetchResult{}, fmt.Errorf("%w: %d", ErrUnexpectedStatus, status)
	}

	return decode(body)
}

func (a *Adapter) fetchOnce(ctx context.Context, url, apiKey string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("z.ai: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return body, resp.StatusCode, ErrAuthFailed
	}

	return body, resp.StatusCode, nil
}

// apiResponse is the top-level shape returned by the z.ai endpoint.
type apiResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Level  string       `json:"level"`
		Limits []apiLimitV1 `json:"limits"`
	} `json:"data"`
}

type apiLimitV1 struct {
	Type          string  `json:"type"`
	Percentage    float64 `json:"percentage"`
	Remaining     float64 `json:"remaining"`
	NextResetTime int64   `json:"nextResetTime"`
}

func decode(body []byte) (provider.FetchResult, error) {
	var resp apiResponse

	err := json.Unmarshal(body, &resp)
	if err != nil {
		return provider.FetchResult{}, fmt.Errorf("decode: %w", err)
	}

	out := make(provider.Limits, 0, len(resp.Data.Limits))
	for _, item := range resp.Data.Limits {
		out = append(out, convertLimit(item))
	}

	return provider.FetchResult{Limits: out, Level: resp.Data.Level}, nil
}

func convertLimit(in apiLimitV1) provider.Limit {
	out := provider.Limit{
		Label:   in.Type,
		UsedPct: in.Percentage,
	}
	if in.Remaining > 0 {
		out.Detail = fmt.Sprintf("remaining %d", int(in.Remaining))
	}

	if in.NextResetTime > 0 {
		ts := time.UnixMilli(in.NextResetTime)
		out.ResetAt = &ts
	}

	return out
}
