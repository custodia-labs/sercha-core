// Package support provides shared test context and utilities for integration tests.
package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// TestContext holds state shared across scenario steps.
type TestContext struct {
	BaseURL string
	Client  *http.Client

	// Auth state
	Token string

	// Resource IDs
	InstallationID string
	SourceID       string

	// Last response for assertions
	LastStatusCode int
	LastBody       []byte
}

// NewTestContext creates a new test context.
func NewTestContext() *TestContext {
	baseURL := os.Getenv("SERCHA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &TestContext{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Reset clears state between scenarios.
func (c *TestContext) Reset() {
	c.Token = ""
	c.InstallationID = ""
	c.SourceID = ""
	c.LastStatusCode = 0
	c.LastBody = nil
}

// Request makes an HTTP request to the API.
func (c *TestContext) Request(method, path string, body any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	c.LastStatusCode = resp.StatusCode
	c.LastBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	return nil
}

// ParseResponse unmarshals the last response body into v.
func (c *TestContext) ParseResponse(v any) error {
	return json.Unmarshal(c.LastBody, v)
}

// WaitFor polls a condition until it returns true or timeout.
func (c *TestContext) WaitFor(timeout time.Duration, interval time.Duration, condition func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		done, err := condition()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timeout waiting for condition after %v", timeout)
}
