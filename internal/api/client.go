package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ExitUser   = 1
	ExitAPI    = 2
	ExitConfig = 3
)

// Client wraps HTTP calls to the Dev-Dash API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New creates an API client.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Body)
}

// Do executes an HTTP request and returns the response body.
func (c *Client) Do(method, path string, body interface{}) ([]byte, error) {
	url := c.BaseURL + "/api" + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
		// Try to extract error message from JSON
		var errResp struct {
			Error          string `json:"error"`
			Message        string `json:"message"`
			UpgradeMessage string `json:"upgrade_message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errResp.UpgradeMessage != "" {
				apiErr.Message = fmt.Sprintf("CLI update required: %s\nRun: devdash self-update", errResp.UpgradeMessage)
			} else if errResp.Error != "" {
				apiErr.Message = errResp.Error
			} else if errResp.Message != "" {
				apiErr.Message = errResp.Message
			}
		}
		return nil, apiErr
	}

	return respBody, nil
}

// Get performs a GET request.
func (c *Client) Get(path string) ([]byte, error) {
	return c.Do("GET", path, nil)
}

// Post performs a POST request.
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	return c.Do("POST", path, body)
}

// Patch performs a PATCH request.
func (c *Client) Patch(path string, body interface{}) ([]byte, error) {
	return c.Do("PATCH", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, error) {
	return c.Do("DELETE", path, nil)
}

// JSON unmarshals the response into the given target.
func JSON[T any](data []byte, err error) (T, error) {
	var target T
	if err != nil {
		return target, err
	}
	if err := json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("failed to parse response: %w", err)
	}
	return target, nil
}
