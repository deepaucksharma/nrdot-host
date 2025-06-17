package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the API client for nrdot-api-server
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new API client
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetStatus gets the current system status
func (c *Client) GetStatus() (*Status, error) {
	var status Status
	err := c.get("/api/v1/status", &status)
	return &status, err
}

// ValidateConfig validates configuration
func (c *Client) ValidateConfig(config []byte) (*ValidationResult, error) {
	var result ValidationResult
	err := c.post("/api/v1/config/validate", config, &result)
	return &result, err
}

// GenerateConfig generates OTel config from NRDOT config
func (c *Client) GenerateConfig(config []byte) ([]byte, error) {
	resp, err := c.postRaw("/api/v1/config/generate", config)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ApplyConfig applies new configuration
func (c *Client) ApplyConfig(config []byte) (*ApplyResult, error) {
	var result ApplyResult
	err := c.post("/api/v1/config/apply", config, &result)
	return &result, err
}

// StartCollector starts the collector
func (c *Client) StartCollector() (*OperationResult, error) {
	var result OperationResult
	err := c.post("/api/v1/collector/start", nil, &result)
	return &result, err
}

// StopCollector stops the collector
func (c *Client) StopCollector() (*OperationResult, error) {
	var result OperationResult
	err := c.post("/api/v1/collector/stop", nil, &result)
	return &result, err
}

// RestartCollector restarts the collector
func (c *Client) RestartCollector() (*OperationResult, error) {
	var result OperationResult
	err := c.post("/api/v1/collector/restart", nil, &result)
	return &result, err
}

// GetLogs gets recent logs
func (c *Client) GetLogs(lines int) (string, error) {
	resp, err := c.getRaw(fmt.Sprintf("/api/v1/collector/logs?lines=%d", lines))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// StreamLogs streams logs in real-time
func (c *Client) StreamLogs() (io.ReadCloser, error) {
	resp, err := c.getRaw("/api/v1/collector/logs?follow=true")
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetMetrics gets current metrics
func (c *Client) GetMetrics() (*Metrics, error) {
	var metrics Metrics
	err := c.get("/api/v1/metrics", &metrics)
	return &metrics, err
}

// Helper methods

func (c *Client) get(path string, result interface{}) error {
	resp, err := c.getRaw(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s (status %d)", string(body), resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) getRaw(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}

func (c *Client) post(path string, data []byte, result interface{}) error {
	resp, err := c.postRaw(path, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s (status %d)", string(body), resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

func (c *Client) postRaw(path string, data []byte) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}