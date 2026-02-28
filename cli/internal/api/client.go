package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the LABYRINTH dashboard API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a dashboard API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Healthy checks if the dashboard API is reachable.
func (c *Client) Healthy() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/api/stats")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// FetchStats retrieves aggregate stats from /api/stats.
func (c *Client) FetchStats() (Stats, error) {
	var stats Stats
	if err := c.getJSON("/api/stats", &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

// FetchSessions retrieves the session list from /api/sessions.
func (c *Client) FetchSessions() ([]SessionEntry, error) {
	var sessions []SessionEntry
	if err := c.getJSON("/api/sessions", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// FetchEvents retrieves the unified event stream from /api/events.
func (c *Client) FetchEvents(limit int) ([]ForensicEvent, error) {
	var resp EventsResponse
	path := fmt.Sprintf("/api/events?limit=%d", limit)
	if err := c.getJSON(path, &resp); err != nil {
		return nil, err
	}
	return resp.Events, nil
}

// FetchAuthEvents retrieves auth capture events from /api/auth.
func (c *Client) FetchAuthEvents(limit int) ([]AuthEvent, error) {
	var resp AuthResponse
	path := fmt.Sprintf("/api/auth?limit=%d", limit)
	if err := c.getJSON(path, &resp); err != nil {
		return nil, err
	}
	return resp.AuthEvents, nil
}

// FetchContainers retrieves Docker container status from /api/containers.
func (c *Client) FetchContainers() (*ContainersResponse, error) {
	var resp ContainersResponse
	if err := c.getJSON("/api/containers", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FetchLayers retrieves layer status from /api/layers.
func (c *Client) FetchLayers() ([]LayerStatus, error) {
	var resp LayersResponse
	if err := c.getJSON("/api/layers", &resp); err != nil {
		return nil, err
	}
	return resp.Layers, nil
}

// FetchSessionDetail retrieves full event timeline for one session.
func (c *Client) FetchSessionDetail(sessionID string) (*SessionDetail, error) {
	var detail SessionDetail
	path := fmt.Sprintf("/api/sessions/%s", sessionID)
	if err := c.getJSON(path, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// FetchPrompts retrieves captured AI prompts from /api/prompts.
func (c *Client) FetchPrompts() ([]CapturedPrompt, error) {
	var resp PromptsResponse
	if err := c.getJSON("/api/prompts", &resp); err != nil {
		return nil, err
	}
	return resp.Prompts, nil
}

// ResetSessions sends a POST /api/reset to clear sessions and forensic data.
func (c *Client) ResetSessions() (*ResetResponse, error) {
	var result ResetResponse
	if err := c.postJSON("/api/reset", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FetchL4Mode retrieves the current L4 interceptor mode from /api/l4/mode.
func (c *Client) FetchL4Mode() (L4ModeResponse, error) {
	var resp L4ModeResponse
	if err := c.getJSON("/api/l4/mode", &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SetL4Mode changes the L4 interceptor mode via POST /api/l4/mode.
func (c *Client) SetL4Mode(mode string) error {
	body := fmt.Sprintf(`{"mode":"%s"}`, mode)
	resp, err := c.httpClient.Post(c.baseURL+"/api/l4/mode", "application/json",
		strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set mode failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// FetchL4Intel retrieves captured intelligence from /api/l4/intel.
func (c *Client) FetchL4Intel() ([]L4IntelSummary, error) {
	var resp L4IntelResponse
	if err := c.getJSON("/api/l4/intel", &resp); err != nil {
		return nil, err
	}
	return resp.Intel, nil
}

func (c *Client) postJSON(path string, target interface{}) error {
	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}

// BaseURL returns the client's base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// MultiClient wraps multiple Client instances for aggregated queries across environments.
type MultiClient struct {
	clients []*Client
	names   []string
}

// NewMultiClient creates a multi-environment API client.
func NewMultiClient(urls, names []string) *MultiClient {
	mc := &MultiClient{names: names}
	for _, url := range urls {
		mc.clients = append(mc.clients, NewClient(url))
	}
	return mc
}

// FetchAggregateStats queries all environments and sums their stats.
func (mc *MultiClient) FetchAggregateStats() (Stats, error) {
	var agg Stats
	var lastErr error

	for _, c := range mc.clients {
		s, err := c.FetchStats()
		if err != nil {
			lastErr = err
			continue
		}
		agg.ActiveSessions += s.ActiveSessions
		agg.CapturedPrompts += s.CapturedPrompts
		agg.TotalEvents += s.TotalEvents
		agg.AuthAttempts += s.AuthAttempts
		agg.HTTPRequests += s.HTTPRequests
		agg.L3Activations += s.L3Activations
		agg.L4Interceptions += s.L4Interceptions
		agg.ActiveContainers += s.ActiveContainers
		if s.MaxDepthReached > agg.MaxDepthReached {
			agg.MaxDepthReached = s.MaxDepthReached
		}
	}

	if agg.TotalEvents == 0 && lastErr != nil {
		return agg, lastErr
	}
	return agg, nil
}

// FetchAggregateSessions queries all environments and merges their session lists.
func (mc *MultiClient) FetchAggregateSessions() ([]SessionEntry, error) {
	var all []SessionEntry
	for i, c := range mc.clients {
		sessions, err := c.FetchSessions()
		if err != nil {
			continue
		}
		envName := ""
		if i < len(mc.names) {
			envName = mc.names[i]
		}
		for _, s := range sessions {
			s.Environment = envName
			all = append(all, s)
		}
	}
	return all, nil
}

func (c *Client) getJSON(path string, target interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}
