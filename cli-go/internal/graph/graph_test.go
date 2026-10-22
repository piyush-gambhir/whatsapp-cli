package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetRejectsOversizedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A valid JSON prefix must not hide bytes beyond the response limit.
		_, _ = w.Write([]byte(`{}` + strings.Repeat(" ", 32<<20)))
	}))
	defer server.Close()
	c := &Client{Base: server.URL, Token: "test"}
	if _, err := c.Get(context.Background(), "/123", nil); err == nil || !strings.Contains(err.Error(), "exceeds 32 MiB") {
		t.Fatalf("expected response limit error, got %v", err)
	}
}

func TestGetRejectsUnfollowedRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	c := &Client{Base: server.URL, Token: "test"}
	if _, err := c.Get(context.Background(), "/123", nil); err == nil {
		t.Fatal("expected error for non-success status")
	}
}
