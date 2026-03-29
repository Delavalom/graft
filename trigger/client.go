package trigger

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for the Trigger.dev REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Trigger.dev API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// TriggerTask starts a task run and returns the run ID.
func (c *Client) TriggerTask(ctx context.Context, taskID string, payload any) (string, error) {
	body, err := json.Marshal(map[string]any{
		"taskIdentifier": taskID,
		"payload":        payload,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/tasks/trigger", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("trigger: trigger task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("trigger: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("trigger: decode response: %w", err)
	}
	return result.ID, nil
}

// GetRunStatus returns the current status of a task run.
func (c *Client) GetRunStatus(ctx context.Context, runID string) (*RunStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trigger: get run status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("trigger: HTTP %d: %s", resp.StatusCode, respBody)
	}

	var status RunStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("trigger: decode status: %w", err)
	}
	return &status, nil
}

// SubscribeToRun opens an SSE connection to stream run events.
func (c *Client) SubscribeToRun(ctx context.Context, runID string) (<-chan RunEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/runs/"+runID+"/stream", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trigger: subscribe: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("trigger: subscribe HTTP %d", resp.StatusCode)
	}

	ch := make(chan RunEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		parseSSE(ctx, resp.Body, ch)
	}()
	return ch, nil
}

// SendWaitpointToken resumes a paused task by completing a waitpoint.
func (c *Client) SendWaitpointToken(ctx context.Context, token string, data any) error {
	body, err := json.Marshal(map[string]any{
		"token": token,
		"data":  data,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/waitpoints/complete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("trigger: send waitpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("trigger: waitpoint HTTP %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

// parseSSE reads SSE events from a reader and sends them to the channel.
func parseSSE(ctx context.Context, r io.Reader, ch chan<- RunEvent) {
	scanner := bufio.NewScanner(r)
	var eventType, data string

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if data != "" {
				var event RunEvent
				event.Type = eventType
				_ = json.Unmarshal([]byte(data), &event.Data)
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
				eventType = ""
				data = ""
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}
