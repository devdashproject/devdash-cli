package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
