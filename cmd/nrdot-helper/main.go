// nrdot-helper is a minimal setuid binary for privileged operations
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Operation types
const (
	OpReadFile      = "read_file"
	OpListDir       = "list_dir"
	OpReadProcNet   = "read_proc_net"
	OpCheckPort     = "check_port"
)

// Request represents a privileged operation request
type Request struct {
	Operation string `json:"operation"`
	Path      string `json:"path,omitempty"`
	Port      int    `json:"port,omitempty"`
}

// Response represents the operation result
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Allowed paths for security
var allowedPaths = map[string]bool{
	"/etc/mysql":         true,
	"/etc/postgresql":    true,
	"/etc/redis":         true,
	"/etc/nginx":         true,
	"/etc/apache2":       true,
	"/etc/httpd":         true,
	"/etc/mongodb":       true,
	"/etc/mongod.conf":   true,
	"/etc/elasticsearch": true,
	"/etc/rabbitmq":      true,
	"/etc/kafka":         true,
	"/etc/memcached.conf": true,
	"/proc/net/tcp":      true,
	"/proc/net/tcp6":     true,
	"/proc/net/udp":      true,
	"/proc/net/udp6":     true,
}

func main() {
	// Drop privileges after reading what we need
	defer dropPrivileges()

	// Read request from stdin
	var req Request
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		respondError(fmt.Sprintf("Failed to decode request: %v", err))
		return
	}

	// Validate and execute operation
	switch req.Operation {
	case OpReadFile:
		handleReadFile(req.Path)
	case OpListDir:
		handleListDir(req.Path)
	case OpReadProcNet:
		handleReadProcNet(req.Path)
	case OpCheckPort:
		handleCheckPort(req.Port)
	default:
		respondError(fmt.Sprintf("Unknown operation: %s", req.Operation))
	}
}

func handleReadFile(path string) {
	// Security check
	if !isPathAllowed(path) {
		respondError("Path not allowed")
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		respondError(fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	respondSuccess(map[string]interface{}{
		"path":    path,
		"content": string(data),
		"size":    len(data),
	})
}

func handleListDir(path string) {
	// Security check
	if !isPathAllowed(path) {
		respondError("Path not allowed")
		return
	}

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		respondError(fmt.Sprintf("Failed to list directory: %v", err))
		return
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"size":    entry.Size(),
			"mode":    entry.Mode().String(),
			"is_dir":  entry.IsDir(),
		})
	}

	respondSuccess(map[string]interface{}{
		"path":  path,
		"files": files,
	})
}

func handleReadProcNet(path string) {
	// Only allow specific /proc/net files
	allowed := []string{"/proc/net/tcp", "/proc/net/tcp6", "/proc/net/udp", "/proc/net/udp6"}
	valid := false
	for _, a := range allowed {
		if path == a {
			valid = true
			break
		}
	}

	if !valid {
		respondError("Path not allowed")
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		respondError(fmt.Sprintf("Failed to read proc net: %v", err))
		return
	}

	respondSuccess(map[string]interface{}{
		"path":    path,
		"content": string(data),
	})
}

func handleCheckPort(port int) {
	if port < 1 || port > 65535 {
		respondError("Invalid port number")
		return
	}

	// Try to bind to the port to check if it's in use
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use
		respondSuccess(map[string]interface{}{
			"port":   port,
			"in_use": true,
		})
		return
	}
	listener.Close()

	respondSuccess(map[string]interface{}{
		"port":   port,
		"in_use": false,
	})
}

func isPathAllowed(path string) bool {
	// Check exact match first
	if allowedPaths[path] {
		return true
	}

	// Check if path is under an allowed directory
	for allowed := range allowedPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}

	return false
}

func dropPrivileges() {
	// Get original UID/GID from environment
	origUID := os.Getenv("SUDO_UID")
	origGID := os.Getenv("SUDO_GID")

	if origUID != "" && origGID != "" {
		uid := 0
		gid := 0
		fmt.Sscanf(origUID, "%d", &uid)
		fmt.Sscanf(origGID, "%d", &gid)

		// Drop privileges
		syscall.Setgid(gid)
		syscall.Setuid(uid)
	}
}

func respondSuccess(data interface{}) {
	resp := Response{
		Success: true,
		Data:    data,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

func respondError(err string) {
	resp := Response{
		Success: false,
		Error:   err,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}