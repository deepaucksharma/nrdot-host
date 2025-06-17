package socket

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-privileged-helper/pkg/monitor"
)

// Client represents a client to the privileged helper
type Client struct {
	socketPath string
	timeout    time.Duration
	
	mu         sync.Mutex
	requestID  int64
}

// NewClient creates a new privileged helper client
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    10 * time.Second,
	}
}

// GetProcessInfo retrieves information about a specific process
func (c *Client) GetProcessInfo(pid int32) (*monitor.ProcessInfo, error) {
	if err := monitor.ValidatePID(pid); err != nil {
		return nil, err
	}

	req := &Request{
		Type:      RequestTypeProcessInfo,
		PID:       pid,
		RequestID: c.generateRequestID(),
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("request failed: %s", resp.Error)
	}

	// Convert response data
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data type")
	}

	info := &monitor.ProcessInfo{}
	
	// Parse response fields
	if v, ok := getInt32(data, "pid"); ok {
		info.PID = v
	}
	if v, ok := getInt32(data, "ppid"); ok {
		info.PPID = v
	}
	if v, ok := getString(data, "name"); ok {
		info.Name = v
	}
	if v, ok := getString(data, "cmdline"); ok {
		info.Cmdline = v
	}
	if v, ok := getString(data, "executable"); ok {
		info.Executable = v
	}
	if v, ok := getUint64(data, "memory_rss"); ok {
		info.MemoryRSS = v
	}
	if v, ok := getUint64(data, "memory_vms"); ok {
		info.MemoryVMS = v
	}
	if v, ok := getInt32(data, "uid"); ok {
		info.UID = v
	}
	if v, ok := getInt32(data, "gid"); ok {
		info.GID = v
	}
	if v, ok := getInt32(data, "num_threads"); ok {
		info.NumThreads = v
	}
	if v, ok := getString(data, "state"); ok {
		info.State = v
	}

	return info, nil
}

// ListProcesses returns a list of all process PIDs
func (c *Client) ListProcesses() ([]int32, error) {
	req := &Request{
		Type:      RequestTypeListProcesses,
		RequestID: c.generateRequestID(),
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("request failed: %s", resp.Error)
	}

	// Convert response data
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data type")
	}

	pidsInterface, ok := data["pids"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid PIDs data type")
	}

	var pids []int32
	for _, pidInterface := range pidsInterface {
		if pid, ok := toInt32(pidInterface); ok {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

// GetProcessesByName returns all processes matching the given name
func (c *Client) GetProcessesByName(name string) ([]*monitor.ProcessInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("process name is required")
	}

	req := &Request{
		Type:      RequestTypeProcessesByName,
		Name:      name,
		RequestID: c.generateRequestID(),
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("request failed: %s", resp.Error)
	}

	// Convert response data
	dataList, ok := resp.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data type")
	}

	var processes []*monitor.ProcessInfo
	for _, item := range dataList {
		data, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		info := &monitor.ProcessInfo{}
		
		// Parse response fields
		if v, ok := getInt32(data, "pid"); ok {
			info.PID = v
		}
		if v, ok := getInt32(data, "ppid"); ok {
			info.PPID = v
		}
		if v, ok := getString(data, "name"); ok {
			info.Name = v
		}
		if v, ok := getString(data, "cmdline"); ok {
			info.Cmdline = v
		}
		if v, ok := getString(data, "executable"); ok {
			info.Executable = v
		}
		if v, ok := getUint64(data, "memory_rss"); ok {
			info.MemoryRSS = v
		}
		if v, ok := getUint64(data, "memory_vms"); ok {
			info.MemoryVMS = v
		}
		if v, ok := getInt32(data, "uid"); ok {
			info.UID = v
		}
		if v, ok := getInt32(data, "gid"); ok {
			info.GID = v
		}
		if v, ok := getInt32(data, "num_threads"); ok {
			info.NumThreads = v
		}
		if v, ok := getString(data, "state"); ok {
			info.State = v
		}

		processes = append(processes, info)
	}

	return processes, nil
}

// sendRequest sends a request to the server and waits for response
func (c *Client) sendRequest(req *Request) (*Response, error) {
	// Connect to socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to privileged helper: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send request
	reqData, err := EncodeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	writer := bufio.NewWriter(conn)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	if err := writer.WriteByte('\n'); err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	respData, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Decode response
	resp, err := DecodeResponse(respData[:len(respData)-1]) // Remove newline
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify request ID matches
	if resp.RequestID != req.RequestID {
		return nil, fmt.Errorf("request ID mismatch")
	}

	return resp, nil
}

// generateRequestID generates a unique request ID
func (c *Client) generateRequestID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.requestID++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), c.requestID)
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) (string, bool) {
	v, ok := data[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getInt32(data map[string]interface{}, key string) (int32, bool) {
	v, ok := data[key]
	if !ok {
		return 0, false
	}
	return toInt32(v)
}

func getUint64(data map[string]interface{}, key string) (uint64, bool) {
	v, ok := data[key]
	if !ok {
		return 0, false
	}
	
	switch n := v.(type) {
	case float64:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case uint64:
		return n, true
	default:
		return 0, false
	}
}

func toInt32(v interface{}) (int32, bool) {
	switch n := v.(type) {
	case float64:
		return int32(n), true
	case int32:
		return n, true
	case int64:
		return int32(n), true
	default:
		return 0, false
	}
}