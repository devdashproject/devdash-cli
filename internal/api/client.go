package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ExitUser   = 1
	ExitAPI    = 2
	ExitConfig = 3
)

// Entitlement error codes returned in the 403 envelope for blocked writes
// (trial expiry / project or bead limits). See EntitlementErrorBody in the
// server's packages/api-types.
const (
	CodeTrialExpired        = "TRIAL_EXPIRED"
	CodeProjectLimitReached = "PROJECT_LIMIT_REACHED"
	CodeBeadLimitReached    = "BEAD_LIMIT_REACHED"
)

func isEntitlementCode(code string) bool {
	switch code {
	case CodeTrialExpired, CodeProjectLimitReached, CodeBeadLimitReached:
		return true
	default:
		return false
	}
}

// Client wraps HTTP calls to the Dev-Dash API.
type Client struct {
	BaseURL    string
	Token      string
	Version    string
	HTTPClient *http.Client
}

// New creates an API client.
func New(baseURL, token, version string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Version: version,
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

	// Entitlement fields, populated when the API returns a 403 envelope for a
	// blocked write (trial expired / project or bead limit reached). Code is
	// empty for non-entitlement errors.
	Code        string
	Limit       *int
	Used        *int
	Plan        string
	TrialEndsAt string
	UpgradeURL  string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Body)
}

// IsEntitlementError reports whether this error is a trial/limit block
// (as opposed to a generic API failure).
func (e *APIError) IsEntitlementError() bool {
	return isEntitlementCode(e.Code)
}

// formatEntitlementMessage builds a clear, actionable message for a blocked
// write: what limit was hit, trial status, and the upgrade path.
func formatEntitlementMessage(message, code string, limit, used *int, trialEndsAt, upgradeURL string) string {
	var b strings.Builder
	if message != "" {
		b.WriteString(message)
	} else {
		b.WriteString("This action is blocked by your plan's limits.")
	}

	switch code {
	case CodeTrialExpired:
		if when := trialEndedLabel(trialEndsAt); when != "" {
			b.WriteString("\n  " + when)
		}
	case CodeProjectLimitReached, CodeBeadLimitReached:
		resource := "projects"
		if code == CodeBeadLimitReached {
			resource = "tasks"
		}
		if limit != nil {
			usedStr := "?"
			if used != nil {
				usedStr = strconv.Itoa(*used)
			}
			b.WriteString(fmt.Sprintf("\n  Limit: %s/%d %s used", usedStr, *limit, resource))
		}
	}

	if upgradeURL != "" {
		b.WriteString("\n  Upgrade: " + upgradeURL)
	}

	return b.String()
}

// trialEndedLabel renders how long ago the trial ended, e.g. "Trial ended
// 6 days ago." Returns "" if trialEndsAt is missing or unparsable.
func trialEndedLabel(trialEndsAt string) string {
	if trialEndsAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, trialEndsAt)
	if err != nil {
		return ""
	}
	days := int(time.Since(t).Hours() / 24)
	switch {
	case days <= 0:
		return "Trial ended today."
	case days == 1:
		return "Trial ended 1 day ago."
	default:
		return fmt.Sprintf("Trial ended %d days ago.", days)
	}
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
	req.Header.Set("User-Agent", "devdash-cli-go/"+c.Version)

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
			Code           string `json:"code"`
			Limit          *int   `json:"limit"`
			Used           *int   `json:"used"`
			Plan           string `json:"plan"`
			TrialEndsAt    string `json:"trialEndsAt"`
			UpgradeURL     string `json:"upgradeUrl"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			apiErr.Code = errResp.Code
			apiErr.Limit = errResp.Limit
			apiErr.Used = errResp.Used
			apiErr.Plan = errResp.Plan
			apiErr.TrialEndsAt = errResp.TrialEndsAt
			apiErr.UpgradeURL = errResp.UpgradeURL

			switch {
			case errResp.UpgradeMessage != "":
				apiErr.Message = fmt.Sprintf("CLI update required: %s\nRun: devdash self-update", errResp.UpgradeMessage)
			case isEntitlementCode(errResp.Code):
				apiErr.Message = formatEntitlementMessage(errResp.Error, errResp.Code, errResp.Limit, errResp.Used, errResp.TrialEndsAt, errResp.UpgradeURL)
			case errResp.Error != "":
				apiErr.Message = errResp.Error
			case errResp.Message != "":
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
