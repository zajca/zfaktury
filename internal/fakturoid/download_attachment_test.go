package fakturoid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadAttachment_Success(t *testing.T) {
	fileContent := []byte("%PDF-1.4 fake pdf content for testing")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth and headers.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			t.Errorf("unexpected Authorization header: %q", auth)
		}
		ua := r.Header.Get("User-Agent")
		if !strings.Contains(ua, "ZFaktury") {
			t.Errorf("unexpected User-Agent: %q", ua)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(fileContent)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	data, contentType, err := client.DownloadAttachment(context.Background(), srv.URL+"/attachments/123/download")
	if err != nil {
		t.Fatalf("DownloadAttachment() error: %v", err)
	}

	if contentType != "application/pdf" {
		t.Errorf("contentType = %q, want application/pdf", contentType)
	}
	if len(data) != len(fileContent) {
		t.Errorf("data length = %d, want %d", len(data), len(fileContent))
	}
	if string(data) != string(fileContent) {
		t.Errorf("data content mismatch")
	}
}

func TestDownloadAttachment_DefaultContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally do not set Content-Type header.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary data"))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, contentType, err := client.DownloadAttachment(context.Background(), srv.URL+"/file")
	if err != nil {
		t.Fatalf("DownloadAttachment() error: %v", err)
	}

	// When no Content-Type is set, the Go HTTP server defaults to text/plain or similar.
	// The client should handle this; if truly empty it defaults to application/octet-stream.
	if contentType == "" {
		t.Error("expected non-empty content type")
	}
}

func TestDownloadAttachment_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not Found"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, _, err := client.DownloadAttachment(context.Background(), srv.URL+"/missing")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected HTTP 404 in error, got: %v", err)
	}
}

func TestDownloadAttachment_NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, _, err := client.DownloadAttachment(context.Background(), srv.URL+"/empty")
	if err == nil {
		t.Fatal("expected error for 204 response")
	}
	if !strings.Contains(err.Error(), "HTTP 204") {
		t.Errorf("expected HTTP 204 in error, got: %v", err)
	}
}

func TestDownloadAttachment_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal Server Error"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, _, err := client.DownloadAttachment(context.Background(), srv.URL+"/error")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 in error, got: %v", err)
	}
}

func TestDownloadAttachment_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := newTestClient(srv.URL)

	_, _, err := client.DownloadAttachment(ctx, srv.URL+"/file")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
