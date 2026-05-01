package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchAllSinglePage(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PaginatedResponse[Item]{
			Data:       []Item{{Name: "a"}, {Name: "b"}},
			NextCursor: "",
		})
	}))
	defer server.Close()

	client := New(server.URL, "token", "test")
	client.BaseURL = server.URL + "/api"

	items, err := FetchAll[Item](client, "/items")
	if err != nil {
		t.Fatalf("FetchAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

func TestFetchAllMultiplePages(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(PaginatedResponse[Item]{
				Data:       []Item{{Name: "a"}},
				NextCursor: "page2",
			})
		} else {
			json.NewEncoder(w).Encode(PaginatedResponse[Item]{
				Data:       []Item{{Name: "b"}},
				NextCursor: "",
			})
		}
	}))
	defer server.Close()

	client := New(server.URL, "token", "test")
	client.BaseURL = server.URL + "/api"

	items, err := FetchAll[Item](client, "/items")
	if err != nil {
		t.Fatalf("FetchAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestFetchAllPlainArray(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Some endpoints return plain arrays
		json.NewEncoder(w).Encode([]Item{{Name: "x"}, {Name: "y"}, {Name: "z"}})
	}))
	defer server.Close()

	client := New(server.URL, "token", "test")
	client.BaseURL = server.URL + "/api"

	items, err := FetchAll[Item](client, "/items")
	if err != nil {
		t.Fatalf("FetchAll failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}
