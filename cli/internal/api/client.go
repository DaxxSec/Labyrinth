package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
