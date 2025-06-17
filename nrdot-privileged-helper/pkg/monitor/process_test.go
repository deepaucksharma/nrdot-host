package monitor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessMonitor(t *testing.T) {
	// Skip tests if not running on Linux
	if _, err := os.Stat("/proc"); os.IsNotExist(err) {
		t.Skip("Skipping process monitor tests: /proc not available")
	}

	pm := NewProcessMonitor()

	t.Run("GetProcessInfo for current process", func(t *testing.T) {
		pid := int32(os.Getpid())
		info, err := pm.GetProcessInfo(pid)
		require.NoError(t, err)
		
		assert.Equal(t, pid, info.PID)
		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Cmdline)
		assert.Greater(t, info.MemoryRSS, uint64(0))
		assert.Greater(t, info.NumThreads, int32(0))
	})

	t.Run("GetProcessInfo for init process", func(t *testing.T) {
		// PID 1 should always exist
		info, err := pm.GetProcessInfo(1)
		if err != nil {
			t.Skip("Cannot access PID 1, skipping test")
		}
		
		assert.Equal(t, int32(1), info.PID)
		assert.NotEmpty(t, info.Name)
	})

	t.Run("GetProcessInfo for non-existent process", func(t *testing.T) {
		// Use a very high PID that's unlikely to exist
		_, err := pm.GetProcessInfo(999999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListProcesses", func(t *testing.T) {
		pids, err := pm.ListProcesses()
		require.NoError(t, err)
		
		assert.NotEmpty(t, pids)
		
		// Current process should be in the list
		currentPID := int32(os.Getpid())
		found := false
		for _, pid := range pids {
			if pid == currentPID {
				found = true
				break
			}
		}
		assert.True(t, found, "Current process not found in process list")
	})

	t.Run("GetProcessesByName", func(t *testing.T) {
		// Get info about current process to know its name
		currentPID := int32(os.Getpid())
		currentInfo, err := pm.GetProcessInfo(currentPID)
		require.NoError(t, err)
		
		// Search for processes with the same name
		processes, err := pm.GetProcessesByName(currentInfo.Name)
		require.NoError(t, err)
		
		// Should find at least the current process
		assert.NotEmpty(t, processes)
		
		found := false
		for _, proc := range processes {
			if proc.PID == currentPID {
				found = true
				break
			}
		}
		assert.True(t, found, "Current process not found when searching by name")
	})
}

func TestValidatePID(t *testing.T) {
	tests := []struct {
		name    string
		pid     int32
		wantErr bool
	}{
		{
			name:    "valid PID",
			pid:     1234,
			wantErr: false,
		},
		{
			name:    "zero PID",
			pid:     0,
			wantErr: true,
		},
		{
			name:    "negative PID",
			pid:     -1,
			wantErr: true,
		},
		{
			name:    "max valid PID",
			pid:     4194304,
			wantErr: false,
		},
		{
			name:    "exceeds max PID",
			pid:     4194305,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePID(tt.pid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}