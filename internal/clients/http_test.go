package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestNewHTTPClient(t *testing.T) {
	baseURL := "https://api.example.com"
	client := NewHTTPClient(baseURL)

	if client.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, baseURL)
	}

	if client.client == nil {
		t.Error("client is nil")
	}

	if client.client.Timeout != 5*time.Second {
		t.Errorf("client.Timeout = %v, want %v", client.client.Timeout, 5*time.Second)
	}
}

func TestHTTPClient_Request(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantOutput     string
		wantErr        bool
		requestMethod  string
		requestPath    string
		serverDelay    time.Duration
		contextTimeout time.Duration
	}{
		{
			name:          "successful GET request",
			statusCode:    http.StatusOK,
			responseBody:  "OK",
			wantOutput:    "HTTP 200 OK",
			wantErr:       false,
			requestMethod: "GET",
			requestPath:   "/users",
		},
		{
			name:          "successful POST request",
			statusCode:    http.StatusCreated,
			responseBody:  "Created",
			wantOutput:    "HTTP 201 Created",
			wantErr:       false,
			requestMethod: "POST",
			requestPath:   "/users",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  "Not Found",
			wantOutput:    "HTTP 404 Not Found",
			wantErr:       false,
			requestMethod: "GET",
			requestPath:   "/nonexistent",
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "Internal Server Error",
			wantOutput:    "HTTP 500 Internal Server Error",
			wantErr:       false,
			requestMethod: "GET",
			requestPath:   "/error",
		},
		{
			name:           "context timeout",
			statusCode:     http.StatusOK,
			responseBody:   "OK",
			wantOutput:     "",
			wantErr:        true,
			requestMethod:  "GET",
			requestPath:    "/slow",
			serverDelay:    2 * time.Second,
			contextTimeout: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverDelay > 0 {
					time.Sleep(tt.serverDelay)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL)

			ctx := context.Background()
			if tt.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
			}

			output, err := client.Request(ctx, tt.requestMethod, tt.requestPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Request() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && output != tt.wantOutput {
				t.Errorf("Request() output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestHTTPClient_Request_InvalidURL(t *testing.T) {
	client := NewHTTPClient("https://invalid-url-that-does-not-exist.local")
	ctx := context.Background()

	_, err := client.Request(ctx, "GET", "/test")
	if err == nil {
		t.Error("Request() error = nil, want error for invalid URL")
	}
}

func TestHTTPClient_Request_PathJoining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users" {
			t.Errorf("Request path = %q, want %q", r.URL.Path, "/api/v1/users")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	ctx := context.Background()

	_, err := client.Request(ctx, "GET", "/api/v1/users")
	if err != nil {
		t.Errorf("Request() error = %v", err)
	}
}

func TestHTTPClient_Request_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := client.Request(ctx, "GET", "/test")
	if err == nil {
		t.Error("Request() error = nil, want error for cancelled context")
		return
	}
	// The error may be wrapped by http.Client, so check the error message
	errMsg := err.Error()
	if errMsg != "context canceled" && errMsg != "context deadline exceeded" && !contains(errMsg, "canceled") {
		t.Errorf("Request() error = %v, want error containing 'canceled'", err)
	}
}
