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
