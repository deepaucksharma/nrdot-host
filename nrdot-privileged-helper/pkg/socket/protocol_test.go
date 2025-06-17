package socket

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *Request
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "request is nil",
		},
		{
			name: "missing request ID",
			req: &Request{
				Type: RequestTypeProcessInfo,
				PID:  123,
			},
			wantErr: true,
			errMsg:  "request ID is required",
		},
		{
			name: "valid process info request",
			req: &Request{
				Type:      RequestTypeProcessInfo,
				PID:       123,
				RequestID: "req-123",
			},
			wantErr: false,
		},
		{
			name: "process info with invalid PID",
			req: &Request{
				Type:      RequestTypeProcessInfo,
				PID:       0,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "invalid PID",
		},
		{
			name: "valid list processes request",
			req: &Request{
				Type:      RequestTypeListProcesses,
				RequestID: "req-123",
			},
			wantErr: false,
		},
		{
			name: "valid processes by name request",
			req: &Request{
				Type:      RequestTypeProcessesByName,
				Name:      "test-process",
				RequestID: "req-123",
			},
			wantErr: false,
		},
		{
			name: "processes by name with empty name",
			req: &Request{
				Type:      RequestTypeProcessesByName,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "process name is required",
		},
		{
			name: "processes by name with long name",
			req: &Request{
				Type:      RequestTypeProcessesByName,
				Name:      string(make([]byte, 300)),
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "process name too long",
		},
		{
			name: "unknown request type",
			req: &Request{
				Type:      "unknown",
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "unknown request type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestEncodeDecode(t *testing.T) {
	req := &Request{
		Type:      RequestTypeProcessInfo,
		PID:       12345,
		RequestID: "test-req-123",
	}

	// Encode
	data, err := EncodeRequest(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Decode
	decoded, err := DecodeRequest(data)
	require.NoError(t, err)
	
	assert.Equal(t, req.Type, decoded.Type)
	assert.Equal(t, req.PID, decoded.PID)
	assert.Equal(t, req.RequestID, decoded.RequestID)
}

func TestResponseEncodeDecode(t *testing.T) {
	testData := &ProcessInfoData{
		PID:       12345,
		Name:      "test-process",
		MemoryRSS: 1024 * 1024,
	}

	resp := &Response{
		Success:   true,
		Data:      testData,
		RequestID: "test-req-123",
	}

	// Encode
	data, err := EncodeResponse(resp)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Decode
	decoded, err := DecodeResponse(data)
	require.NoError(t, err)
	
	assert.Equal(t, resp.Success, decoded.Success)
	assert.Equal(t, resp.RequestID, decoded.RequestID)
	assert.Empty(t, decoded.Error)
	
	// Check that data is preserved (as map[string]interface{})
	dataMap, ok := decoded.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(testData.PID), dataMap["pid"])
	assert.Equal(t, testData.Name, dataMap["name"])
}

func TestNewErrorResponse(t *testing.T) {
	err := errors.New("test error")
	resp := NewErrorResponse("req-123", err)
	
	assert.False(t, resp.Success)
	assert.Equal(t, "test error", resp.Error)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.Nil(t, resp.Data)
}

func TestNewSuccessResponse(t *testing.T) {
	data := &ProcessListData{
		PIDs: []int32{1, 2, 3},
	}
	resp := NewSuccessResponse("req-123", data)
	
	assert.True(t, resp.Success)
	assert.Empty(t, resp.Error)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, data, resp.Data)
}

func TestInvalidJSON(t *testing.T) {
	t.Run("invalid request JSON", func(t *testing.T) {
		_, err := DecodeRequest([]byte("invalid json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode request")
	})

	t.Run("invalid response JSON", func(t *testing.T) {
		_, err := DecodeResponse([]byte("invalid json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})
}

func TestProtocolSerialization(t *testing.T) {
	// Test that our protocol can handle various data types
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "process info data",
			data: &ProcessInfoData{
				PID:        123,
				PPID:       1,
				Name:       "test",
				Cmdline:    "/usr/bin/test --flag",
				Executable: "/usr/bin/test",
				MemoryRSS:  1024 * 1024,
				MemoryVMS:  2048 * 1024,
				UID:        1000,
				GID:        1000,
				NumThreads: 4,
				State:      "S",
			},
		},
		{
			name: "process list data",
			data: &ProcessListData{
				PIDs: []int32{1, 2, 3, 100, 200, 300},
			},
		},
		{
			name: "array of process info",
			data: []*ProcessInfoData{
				{PID: 1, Name: "init"},
				{PID: 2, Name: "kthreadd"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response with data
			resp := NewSuccessResponse("test-123", tt.data)
			
			// Encode to JSON
			encoded, err := json.Marshal(resp)
			require.NoError(t, err)
			
			// Decode back
			var decoded Response
			err = json.Unmarshal(encoded, &decoded)
			require.NoError(t, err)
			
			assert.True(t, decoded.Success)
			assert.Equal(t, "test-123", decoded.RequestID)
			assert.NotNil(t, decoded.Data)
		})
	}
}