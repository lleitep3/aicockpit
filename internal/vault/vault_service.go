package vault

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// VaultService provides secure secret access where the service controls the namespace
// Packages NEVER specify their namespace - the service determines it from process identity
type VaultService struct {
	socketPath string
	vault      *osVault
	secretKey  []byte
	running    bool
	mu         sync.Mutex
}

// SecretRequest represents a request for a secret
type SecretRequest struct {
	PID        int    `json:"pid"`
	Executable string `json:"executable"`
	Key        string `json:"key"`
	Timestamp  int64  `json:"timestamp"`
}

// SecretResponse represents the response
type SecretResponse struct {
	Value   string `json:"value,omitempty"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// NewVaultService creates a new vault service
func NewVaultService(socketPath string) *VaultService {
	if socketPath == "" {
		socketPath = "/tmp/cockpit-vault.sock"
	}

	return &VaultService{
		socketPath: socketPath,
		vault:      newOSVault(),
		secretKey:  getServiceSecretKey(),
		running:    false,
	}
}

// Start starts the vault service
func (vs *VaultService) Start() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.running {
		return fmt.Errorf("service already running")
	}

	// Remove existing socket
	os.Remove(vs.socketPath)

	listener, err := net.Listen("unix", vs.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	vs.running = true
	fmt.Printf("[VaultService] Started on %s\n", vs.socketPath)

	go vs.acceptConnections(listener)

	return nil
}

// Stop stops the vault service
func (vs *VaultService) Stop() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.running {
		return nil
	}

	vs.running = false
	return os.Remove(vs.socketPath)
}

// acceptConnections accepts incoming connections
func (vs *VaultService) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if vs.running {
				fmt.Printf("[VaultService] Accept error: %v\n", err)
			}
			return
		}

		go vs.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (vs *VaultService) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var request SecretRequest
	if err := decoder.Decode(&request); err != nil {
		vs.sendError(encoder, "invalid request format")
		return
	}

	// Verify process identity
	namespace, err := vs.verifyAndGetNamespace(&request)
	if err != nil {
		vs.logSecurityEvent("identity_verification_failed", fmt.Sprintf("PID: %d, Error: %v", request.PID, err))
		vs.sendError(encoder, fmt.Sprintf("identity verification failed: %v", err))
		return
	}

	// Get secret from the determined namespace only
	namespacedKey := fmt.Sprintf("%s:%s", namespace, request.Key)
	value, err := vs.vault.Get(namespacedKey)

	if err != nil {
		vs.sendError(encoder, fmt.Sprintf("secret not found: %v", err))
		return
	}

	// Send success response
	response := SecretResponse{
		Value:   value,
		Success: true,
	}

	encoder.Encode(response)
	vs.logSecurityEvent("secret_access_granted", fmt.Sprintf("Namespace: %s, Key: %s, PID: %d", namespace, request.Key, request.PID))
}

// verifyAndGetNamespace verifies the process identity and returns the appropriate namespace
func (vs *VaultService) verifyAndGetNamespace(req *SecretRequest) (string, error) {
	// Verify the process exists
	if !vs.processExists(req.PID) {
		return "", fmt.Errorf("process %d does not exist", req.PID)
	}

	// Get the actual executable path for this PID
	actualExePath, err := getProcessPath(req.PID)
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}

	// Verify the executable path matches what the client claims
	if actualExePath != req.Executable {
		return "", fmt.Errorf("executable path mismatch: claimed=%s, actual=%s", req.Executable, actualExePath)
	}

	// Verify the executable is in an authorized location
	if !vs.isAuthorizedLocation(actualExePath) {
		return "", fmt.Errorf("executable not in authorized location: %s", actualExePath)
	}

	// Determine namespace from executable identity
	namespace := vs.determineNamespace(actualExePath)

	return namespace, nil
}

// processExists checks if a process with the given PID exists
func (vs *VaultService) processExists(pid int) bool {
	// Try to send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(os.Signal(nil))
	return err == nil
}

// getProcessPath gets the executable path for a PID
func getProcessPath(pid int) (string, error) {
	// Linux: read /proc/[PID]/exe
	procPath := fmt.Sprintf("/proc/%d/exe", pid)
	path, err := os.Readlink(procPath)
	if err == nil {
		return path, nil
	}

	// Fallback: try using ps command
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// isAuthorizedLocation checks if the executable is in an authorized location
func (vs *VaultService) isAuthorizedLocation(exePath string) bool {
	authorizedLocations := []string{
		"/home/lleite/.cockpit/packages/",
		os.Getenv("HOME") + "/.cockpit/packages/",
		"/usr/local/bin/",
		os.Getenv("HOME") + "/.local/bin/",
	}

	// Also allow dev mode
	if os.Getenv("COCKPIT_DEV_MODE") == "true" {
		return true
	}

	for _, authLoc := range authorizedLocations {
		if strings.HasPrefix(exePath, authLoc) {
			return true
		}
	}

	return false
}

// determineNamespace determines the namespace from the executable path
func (vs *VaultService) determineNamespace(exePath string) string {
	// Extract package name from path
	// Expected path: /home/user/.cockpit/packages/package-name/...
	// Or: /home/user/.local/bin/package-name

	// Try to extract from packages directory
	if strings.Contains(exePath, "/.cockpit/packages/") {
		parts := strings.Split(exePath, "/.cockpit/packages/")
		if len(parts) > 1 {
			packagePart := parts[1]
			// Get the first directory after packages/
			packageParts := strings.Split(packagePart, "/")
			if len(packageParts) > 0 {
				return sanitizeNamespace(packageParts[0])
			}
		}
	}

	// Fallback: use executable name
	exeName := filepath.Base(exePath)
	exeName = strings.TrimSuffix(exeName, filepath.Ext(exeName))
	return sanitizeNamespace(exeName)
}

// sendError sends an error response
func (vs *VaultService) sendError(encoder *json.Encoder, message string) {
	response := SecretResponse{
		Success: false,
		Error:   message,
	}
	encoder.Encode(response)
}

// logSecurityEvent logs security events
func (vs *VaultService) logSecurityEvent(eventType, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[SECURITY %s] %s: %s\n", timestamp, eventType, details)
}

// getServiceSecretKey generates or retrieves the service secret key
func getServiceSecretKey() []byte {
	hostname, _ := os.Hostname()
	userID := os.Getuid()
	data := fmt.Sprintf("cockpit-vault-service|%s|%d", hostname, userID)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// Client for accessing the vault service
type VaultServiceClient struct {
	socketPath string
}

// NewVaultServiceClient creates a new client
func NewVaultServiceClient(socketPath string) *VaultServiceClient {
	if socketPath == "" {
		socketPath = "/tmp/cockpit-vault.sock"
	}

	return &VaultServiceClient{
		socketPath: socketPath,
	}
}

// GetSecret gets a secret from the vault service
// The client doesn't specify namespace - the service determines it from process identity
func (client *VaultServiceClient) GetSecret(key string) (string, error) {
	// Get current process information
	pid := os.Getpid()
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create request
	request := SecretRequest{
		PID:        pid,
		Executable: exePath,
		Key:        key,
		Timestamp:  time.Now().Unix(),
	}

	// Connect to service
	conn, err := net.Dial("unix", client.socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to vault service: %w", err)
	}
	defer conn.Close()

	// Send request
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	var response SecretResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&response); err != nil {
		return "", fmt.Errorf("failed to receive response: %w", err)
	}

	if !response.Success {
		return "", fmt.Errorf("vault service error: %s", response.Error)
	}

	return response.Value, nil
}
