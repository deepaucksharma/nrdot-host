package socket

import (
	"encoding/json"
	"fmt"
)

// RequestType defines the type of request
type RequestType string

const (
	// RequestTypeProcessInfo requests information about a specific process
	RequestTypeProcessInfo RequestType = "process_info"
	// RequestTypeListProcesses requests a list of all processes
	RequestTypeListProcesses RequestType = "list_processes"
	// RequestTypeProcessesByName requests processes matching a name
	RequestTypeProcessesByName RequestType = "processes_by_name"
)

// Request represents a request to the privileged helper
type Request struct {
	Type       RequestType `json:"type"`
	PID        int32       `json:"pid,omitempty"`
	Name       string      `json:"name,omitempty"`
	RequestID  string      `json:"request_id"`
}

// Response represents a response from the privileged helper
type Response struct {
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id"`
}

// ProcessInfoData contains process information in the response
type ProcessInfoData struct {
	PID        int32  `json:"pid"`
	PPID       int32  `json:"ppid"`
	Name       string `json:"name"`
	Cmdline    string `json:"cmdline"`
	Executable string `json:"executable"`
	MemoryRSS  uint64 `json:"memory_rss"`
	MemoryVMS  uint64 `json:"memory_vms"`
	Username   string `json:"username"`
	UID        int32  `json:"uid"`
	GID        int32  `json:"gid"`
	NumThreads int32  `json:"num_threads"`
	State      string `json:"state"`
}

// ProcessListData contains a list of process PIDs
type ProcessListData struct {
	PIDs []int32 `json:"pids"`
}

// ValidateRequest validates a request for security
func ValidateRequest(req *Request) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.RequestID == "" {
		return fmt.Errorf("request ID is required")
	}

	switch req.Type {
	case RequestTypeProcessInfo:
		if req.PID <= 0 {
			return fmt.Errorf("invalid PID for process info request")
		}
	case RequestTypeProcessesByName:
		if req.Name == "" {
			return fmt.Errorf("process name is required")
		}
		// Limit name length for security
		if len(req.Name) > 256 {
			return fmt.Errorf("process name too long")
		}
	case RequestTypeListProcesses:
		// No additional validation needed
	default:
		return fmt.Errorf("unknown request type: %s", req.Type)
	}

	return nil
}

// EncodeRequest encodes a request to JSON
func EncodeRequest(req *Request) ([]byte, error) {
	return json.Marshal(req)
}

// DecodeRequest decodes a request from JSON
func DecodeRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}
	return &req, nil
}

// EncodeResponse encodes a response to JSON
func EncodeResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}

// DecodeResponse decodes a response from JSON
func DecodeResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &resp, nil
}

// NewErrorResponse creates an error response
func NewErrorResponse(requestID string, err error) *Response {
	return &Response{
		Success:   false,
		Error:     err.Error(),
		RequestID: requestID,
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(requestID string, data interface{}) *Response {
	return &Response{
		Success:   true,
		Data:      data,
		RequestID: requestID,
	}
}