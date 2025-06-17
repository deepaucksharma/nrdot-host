package socket

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-privileged-helper/pkg/monitor"
	"go.uber.org/zap"
)

// Server represents the privileged helper server
type Server struct {
	socketPath string
	listener   net.Listener
	monitor    *monitor.ProcessMonitor
	logger     *zap.Logger
	
	mu       sync.RWMutex
	shutdown bool
	wg       sync.WaitGroup
}

// NewServer creates a new privileged helper server
func NewServer(socketPath string, logger *zap.Logger) (*Server, error) {
	// Ensure socket directory exists
	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove existing socket: %w", err)
	}

	return &Server{
		socketPath: socketPath,
		monitor:    monitor.NewProcessMonitor(),
		logger:     logger,
	}, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Create Unix domain socket
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions (readable/writable by group)
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.logger.Info("Privileged helper server started", zap.String("socket", s.socketPath))

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptLoop(ctx)

	return nil
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all connections to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All connections closed
	case <-time.After(5 * time.Second):
		s.logger.Warn("Timeout waiting for connections to close")
	}

	// Remove socket file
	os.Remove(s.socketPath)

	s.logger.Info("Privileged helper server stopped")
	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			shutdown := s.shutdown
			s.mu.RUnlock()

			if shutdown {
				return
			}

			s.logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Set connection timeout
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Log connection
	s.logger.Debug("New connection", zap.String("remote", conn.RemoteAddr().String()))

	scanner := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)

	for scanner.Scan() {
		// Reset deadline for each request
		conn.SetDeadline(time.Now().Add(30 * time.Second))

		// Process request
		response := s.processRequest(scanner.Bytes())

		// Send response
		respData, err := EncodeResponse(response)
		if err != nil {
			s.logger.Error("Failed to encode response", zap.Error(err))
			continue
		}

		if _, err := writer.Write(respData); err != nil {
			s.logger.Error("Failed to write response", zap.Error(err))
			return
		}

		if err := writer.WriteByte('\n'); err != nil {
			s.logger.Error("Failed to write newline", zap.Error(err))
			return
		}

		if err := writer.Flush(); err != nil {
			s.logger.Error("Failed to flush response", zap.Error(err))
			return
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("Scanner error", zap.Error(err))
	}
}

// processRequest processes a single request
func (s *Server) processRequest(data []byte) *Response {
	// Decode request
	req, err := DecodeRequest(data)
	if err != nil {
		s.logger.Error("Failed to decode request", zap.Error(err))
		return NewErrorResponse("", fmt.Errorf("invalid request format"))
	}

	// Validate request
	if err := ValidateRequest(req); err != nil {
		s.logger.Error("Invalid request", zap.Error(err), zap.String("type", string(req.Type)))
		return NewErrorResponse(req.RequestID, err)
	}

	// Log request
	s.logger.Debug("Processing request", 
		zap.String("type", string(req.Type)),
		zap.String("request_id", req.RequestID))

	// Process based on request type
	switch req.Type {
	case RequestTypeProcessInfo:
		return s.handleProcessInfo(req)
	case RequestTypeListProcesses:
		return s.handleListProcesses(req)
	case RequestTypeProcessesByName:
		return s.handleProcessesByName(req)
	default:
		return NewErrorResponse(req.RequestID, fmt.Errorf("unknown request type"))
	}
}

// handleProcessInfo handles a process info request
func (s *Server) handleProcessInfo(req *Request) *Response {
	info, err := s.monitor.GetProcessInfo(req.PID)
	if err != nil {
		return NewErrorResponse(req.RequestID, err)
	}

	data := &ProcessInfoData{
		PID:        info.PID,
		PPID:       info.PPID,
		Name:       info.Name,
		Cmdline:    info.Cmdline,
		Executable: info.Executable,
		MemoryRSS:  info.MemoryRSS,
		MemoryVMS:  info.MemoryVMS,
		UID:        info.UID,
		GID:        info.GID,
		NumThreads: info.NumThreads,
		State:      info.State,
	}

	return NewSuccessResponse(req.RequestID, data)
}

// handleListProcesses handles a list processes request
func (s *Server) handleListProcesses(req *Request) *Response {
	pids, err := s.monitor.ListProcesses()
	if err != nil {
		return NewErrorResponse(req.RequestID, err)
	}

	data := &ProcessListData{
		PIDs: pids,
	}

	return NewSuccessResponse(req.RequestID, data)
}

// handleProcessesByName handles a processes by name request
func (s *Server) handleProcessesByName(req *Request) *Response {
	processes, err := s.monitor.GetProcessesByName(req.Name)
	if err != nil {
		return NewErrorResponse(req.RequestID, err)
	}

	// Convert to response format
	var processData []*ProcessInfoData
	for _, info := range processes {
		data := &ProcessInfoData{
			PID:        info.PID,
			PPID:       info.PPID,
			Name:       info.Name,
			Cmdline:    info.Cmdline,
			Executable: info.Executable,
			MemoryRSS:  info.MemoryRSS,
			MemoryVMS:  info.MemoryVMS,
			UID:        info.UID,
			GID:        info.GID,
			NumThreads: info.NumThreads,
			State:      info.State,
		}
		processData = append(processData, data)
	}

	return NewSuccessResponse(req.RequestID, processData)
}