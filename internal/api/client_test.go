package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	// Override BaseURL to skip /api prefix for this test
	client.BaseURL = server.URL

	data, err := client.Do("GET", "", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var result map[string]string
	json.Unmarshal(data, &result)
	if result["status"] != "ok" {
		t.Errorf("got %v", result)
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type: %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "created-123", "subject": body["subject"]})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	data, err := client.Do("POST", "", map[string]string{"subject": "Test"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	var result map[string]string
	json.Unmarshal(data, &result)
	if result["id"] != "created-123" {
		t.Errorf("got %v", result)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("GET", "/missing", nil)
	if err == nil {
		t.Fatal("should return error for 404")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "Not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Not found")
	}
}

func TestClientNoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("should not send auth header when token is empty, got: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := New(server.URL, "", "test")
	client.BaseURL = server.URL

	_, err := client.Do("GET", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONGeneric(t *testing.T) {
	data := []byte(`{"id": "test-123", "subject": "Hello"}`)

	type Item struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}

	result, err := JSON[Item](data, nil)
	if err != nil {
		t.Fatalf("JSON() failed: %v", err)
	}
	if result.ID != "test-123" {
		t.Errorf("ID = %q", result.ID)
	}
	if result.Subject != "Hello" {
		t.Errorf("Subject = %q", result.Subject)
	}
}

func TestJSONPropagatesError(t *testing.T) {
	_, err := JSON[map[string]string](nil, &APIError{StatusCode: 500, Message: "fail"})
	if err == nil {
		t.Fatal("should propagate error")
	}
}

func TestClientUpgradeMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(410)
		json.NewEncoder(w).Encode(map[string]string{
			"error":           "gone",
			"upgrade_message": "This endpoint was removed in v0.5.0. Run devdash self-update.",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("GET", "/deprecated", nil)
	if err == nil {
		t.Fatal("should return error for 410")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 410 {
		t.Errorf("StatusCode = %d, want 410", apiErr.StatusCode)
	}
	// Verify upgrade_message is parsed and formatted with upgrade guidance
	expectedMsg := "CLI update required: This endpoint was removed in v0.5.0. Run devdash self-update.\nRun: devdash self-update"
	if apiErr.Message != expectedMsg {
		t.Errorf("Message = %q, want %q", apiErr.Message, expectedMsg)
	}
}

func TestClientEntitlementTrialExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "Your free trial has ended. Upgrade to a paid plan to make changes.",
			"code":        "TRIAL_EXPIRED",
			"limit":       nil,
			"used":        nil,
			"plan":        "trial",
			"trialEndsAt": time.Now().Add(-6 * 24 * time.Hour).Format(time.RFC3339),
			"upgradeUrl":  "https://devdash.dev/upgrade",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("POST", "/beads", map[string]string{"subject": "x"})
	if err == nil {
		t.Fatal("should return error for 403")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.Code != "TRIAL_EXPIRED" {
		t.Errorf("Code = %q, want TRIAL_EXPIRED", apiErr.Code)
	}
	if !apiErr.IsEntitlementError() {
		t.Error("IsEntitlementError() = false, want true")
	}
	if !strings.Contains(apiErr.Message, "Your free trial has ended") {
		t.Errorf("Message missing server text: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "Trial ended 6 days ago.") {
		t.Errorf("Message missing trial-ended detail: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "Upgrade: https://devdash.dev/upgrade") {
		t.Errorf("Message missing upgrade path: %q", apiErr.Message)
	}
}

func TestClientEntitlementProjectLimitReached(t *testing.T) {
	limit := 1
	used := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "Free trial is limited to 1 project. Upgrade to create more.",
			"code":        "PROJECT_LIMIT_REACHED",
			"limit":       limit,
			"used":        used,
			"plan":        "trial",
			"trialEndsAt": time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
			"upgradeUrl":  "https://devdash.dev/upgrade",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("POST", "/projects", map[string]string{"name": "x"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "PROJECT_LIMIT_REACHED" {
		t.Errorf("Code = %q, want PROJECT_LIMIT_REACHED", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "Limit: 1/1 projects used") {
		t.Errorf("Message missing limit detail: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "Upgrade: https://devdash.dev/upgrade") {
		t.Errorf("Message missing upgrade path: %q", apiErr.Message)
	}
}

func TestClientEntitlementBeadLimitReached(t *testing.T) {
	limit := 20
	used := 20
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Free trial is limited to 20 tasks. Upgrade to add more.",
			"code":       "BEAD_LIMIT_REACHED",
			"limit":      limit,
			"used":       used,
			"plan":       "trial",
			"upgradeUrl": "https://devdash.dev/upgrade",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("POST", "/beads", map[string]string{"subject": "x"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "BEAD_LIMIT_REACHED" {
		t.Errorf("Code = %q, want BEAD_LIMIT_REACHED", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "Limit: 20/20 tasks used") {
		t.Errorf("Message missing limit detail: %q", apiErr.Message)
	}
}

func TestClientEntitlementUnrecognizedCodeFallsBackToPlainError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Forbidden",
			"code":  "SOME_OTHER_CODE",
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "test")
	client.BaseURL = server.URL

	_, err := client.Do("POST", "/beads", map[string]string{"subject": "x"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.IsEntitlementError() {
		t.Error("IsEntitlementError() = true, want false for unrecognized code")
	}
	if apiErr.Message != "Forbidden" {
		t.Errorf("Message = %q, want plain passthrough %q", apiErr.Message, "Forbidden")
	}
}

func TestClientUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "devdash-cli-go/0.4.0" {
			t.Errorf("User-Agent = %q, want %q", ua, "devdash-cli-go/0.4.0")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := New(server.URL, "test-token", "0.4.0")
	client.BaseURL = server.URL

	data, err := client.Do("GET", "", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	var result map[string]string
	json.Unmarshal(data, &result)
	if result["status"] != "ok" {
		t.Errorf("got %v", result)
	}
}
