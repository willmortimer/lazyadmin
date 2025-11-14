package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *HTTPClient) Request(ctx context.Context, method, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)), nil
}
