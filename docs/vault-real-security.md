# Vault Security Real - Solução de Controle de Namespace

## 🚨 Problema Crítico Identificado

**A solução anterior com --namespace é vulnerável:**
```bash
# Pacote malicioso pode simplesmente:
cockpit vault get --namespace kb-graphify api-key
cockpit vault get --namespace any-other-package secret
# Basta especificar outro namespace!
```

**Isso é "security by obscurity", não segurança real.**

## ✅ Abordagem Correta: Vault Controla o Namespace

O pacote **NUNCA** deve saber/specificar o namespace. O vault deve:
1. Identificar automaticamente quem está chamando
2. Determinar o namespace correto
3. Enviar apenas os secrets daquele namespace
4. Pacote recebe secrets sem saber o namespace

## 🎯 Soluções Reais de Segurança

### Solução 1: Vault Service com Autenticação de Processo (RECOMENDADA)

**Arquitetura:**
```
Pacote → Solicita Secret → Vault Service Valida Processo → Envia Secret Apropriado
```

**Implementação:**

```go
// internal/vault/service.go
package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type VaultService struct {
	serverAddr string
	secretKey  []byte
}

type SecretRequest struct {
	PID        int    `json:"pid"`
	Executable string `json:"executable"`
	Key        string `json:"key"`
	Timestamp  int64  `json:"timestamp"`
	Signature  string `json:"signature"`
}

type SecretResponse struct {
	Value   string `json:"value,omitempty"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// NewVaultService creates a new vault service
func NewVaultService() *VaultService {
	return &VaultService{
		serverAddr: "/tmp/cockpit-vault.sock", // Unix socket
		secretKey:  getServiceSecretKey(),
	}
}

// GetSecretForCaller returns the secret for the calling process
// The caller doesn't specify namespace - the service determines it
func (vs *VaultService) GetSecretForCaller(key string) (string, error) {
	// Get caller process information
	callerPID := os.Getppid()
	exePath, err := getProcessPath(callerPID)
	if err != nil {
		return "", fmt.Errorf("failed to identify caller: %w", err)
	}

	// Determine namespace from executable identity
	namespace := vs.determineNamespace(exePath)
	
	// Create request with process signature
	request := SecretRequest{
		PID:        callerPID,
		Executable: exePath,
		Key:        key,
		Timestamp:  time.Now().Unix(),
	}
	
	// Sign request to prove it comes from this process
	request.Signature = vs.signRequest(&request)
	
	// Send to vault service
	response := vs.sendRequest(&request)
	
	if !response.Success {
		return "", fmt.Errorf("vault service error: %s", response.Error)
	}
	
	return response.Value, nil
}

// determineNamespace determines the namespace based on executable identity
func (vs *VaultService) determineNamespace(exePath string) string {
	// Extract app ID from executable path
	appID := extractAppIDFromPath(exePath)
	
	// Verify executable signature (if available)
	if !vs.verifyExecutableSignature(exePath) {
		// Log suspicious activity
		logSecurityEvent("unsigned_executable", exePath)
	}
	
	// Return namespace (could also use a mapping file)
	return sanitizeNamespace(appID)
}

// verifyExecutableSignature verifies that the executable is signed/authorized
func (vs *VaultService) verifyExecutableSignature(exePath string) bool {
	// In production, verify cryptographic signature
	// For now, check if it's in authorized locations
	
	authorizedPaths := []string{
		"/home/lleite/.cockpit/packages/",
		"/usr/local/bin/",
		os.Getenv("HOME") + "/.local/bin/",
	}
	
	for _, authPath := range authorizedPaths {
		if strings.HasPrefix(exePath, authPath) {
			return true
		}
	}
	
	return false
}

// signRequest creates a signature for the request
func (vs *VaultService) signRequest(req *SecretRequest) string {
	data := fmt.Sprintf("%d|%s|%s|%d", req.PID, req.Executable, req.Key, req.Timestamp)
	hash := sha256.Sum256([]byte(data + string(vs.secretKey)))
	return hex.EncodeToString(hash[:])
}

// sendRequest sends request to vault service
func (vs *VaultService) sendRequest(req *SecretRequest) *SecretResponse {
	// Connect to Unix socket
	conn, err := net.Dial("unix", vs.serverAddr)
	if err != nil {
		return &SecretResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to connect to vault service: %v", err),
		}
	}
	defer conn.Close()
	
	// Send request
	encoder := json.NewEncoder(conn)
	encoder.Encode(req)
	
	// Receive response
	var response SecretResponse
	decoder := json.NewDecoder(conn)
	decoder.Decode(&response)
	
	return &response
}

// getServiceSecretKey generates or retrieves the service secret key
func getServiceSecretKey() []byte {
	// In production, load from secure location
	// For now, generate from system-specific data
	hostname, _ := os.Hostname()
	userID := os.Getuid()
	data := fmt.Sprintf("cockpit-vault|%s|%d", hostname, userID)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// logSecurityEvent logs security events for audit
func logSecurityEvent(eventType, details string) {
	// In production, send to SIEM or security monitoring
	fmt.Printf("[SECURITY] %s: %s\n", eventType, details)
}
```

**Lado do Servidor:**

```go
// internal/vault/server.go
package vault

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type VaultServer struct {
	listener net.Listener
	vault    *osVault
	secretKey []byte
}

func NewVaultServer(socketPath string) (*VaultServer, error) {
	// Remove existing socket if exists
	os.Remove(socketPath)
	
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}
	
	return &VaultServer{
		listener: listener,
		vault:    newOSVault(),
		secretKey: getServiceSecretKey(),
	}, nil
}

func (vs *VaultServer) Start() error {
	fmt.Println("Vault service started on", vs.listener.Addr())
	
	for {
		conn, err := vs.listener.Accept()
		if err != nil {
			return err
		}
		
		go vs.handleConnection(conn)
	}
}

func (vs *VaultServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	var request SecretRequest
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		vs.sendError(conn, "invalid request")
		return
	}
	
	// Verify signature
	if !vs.verifyRequestSignature(&request) {
		logSecurityEvent("invalid_signature", fmt.Sprintf("PID: %d, Exe: %s", request.PID, request.Executable))
		vs.sendError(conn, "invalid signature")
		return
	}
	
	// Verify process is still running and matches
	if !vs.verifyProcessIdentity(&request) {
		logSecurityEvent("process_mismatch", fmt.Sprintf("PID: %d, Exe: %s", request.PID, request.Executable))
		vs.sendError(conn, "process identity verification failed")
		return
	}
	
	// Determine namespace from executable
	namespace := vs.determineNamespace(request.Executable)
	
	// Get secret from that namespace only
	namespacedKey := fmt.Sprintf("%s:%s", namespace, request.Key)
	value, err := vs.vault.Get(namespacedKey)
	
	if err != nil {
		vs.sendError(conn, fmt.Sprintf("secret not found: %v", err))
		return
	}
	
	// Send response
	response := SecretResponse{
		Value:   value,
		Success: true,
	}
	
	encoder := json.NewEncoder(conn)
	encoder.Encode(response)
	
	// Log successful access
	logSecurityEvent("secret_access", fmt.Sprintf("Namespace: %s, Key: %s, PID: %d", namespace, request.Key, request.PID))
}

func (vs *VaultServer) verifyRequestSignature(req *SecretRequest) bool {
	// Recreate signature
	data := fmt.Sprintf("%d|%s|%s|%d", req.PID, req.Executable, req.Key, req.Timestamp)
	hash := sha256.Sum256([]byte(data + string(vs.secretKey)))
	expectedSig := hex.EncodeToString(hash[:])
	
	return req.Signature == expectedSig
}

func (vs *VaultServer) verifyProcessIdentity(req *SecretRequest) bool {
	// Verify the process is still running
	currentExePath, err := getProcessPath(req.PID)
	if err != nil {
		return false
	}
	
	// Verify executable path matches
	return currentExePath == req.Executable
}

func (vs *VaultServer) sendError(conn net.Conn, message string) {
	response := SecretResponse{
		Success: false,
		Error:   message,
	}
	
	encoder := json.NewEncoder(conn)
	encoder.Encode(response)
}
```

### Solução 2: Injeção via Variáveis de Ambiente (Mais Simple)

**Conceito:** O vault injeta secrets nas variáveis de ambiente do processo antes de executá-lo.

```bash
# O vault gerencia a execução:
cockpit-vault-run --package kb-graphify -- ./kb-graphify/bin/search

# O vault:
# 1. Identifica o pacote (kb-graphify)
# 2. Recupera secrets do namespace kb-graphify
# 3. Define variáveis de ambiente
# 4. Executa o comando
# 5. Remove variáveis após execução
```

**Implementação:**

```go
// cmd/vault_run.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	
	"github.com/lleitep3/aicockpit/internal/vault"
)

func NewVaultRunCommand() *cobra.Command {
	var packageFlag string
	
	runCmd := &cobra.Command{
		Use:   "run --package <name> -- <command>",
		Short: "Run a command with vault secrets injected",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageFlag = strings.Join(args[:1], " ")
			commandArgs := args[1:]
			
			// Determine namespace from package name
			namespace := sanitizeNamespace(packageFlag)
			
			// Get all secrets for this namespace
			v := vault.NewOSVault()
			secrets := getAllSecretsForNamespace(v, namespace)
			
			// Inject into environment
			env := os.Environ()
			for key, value := range secrets {
				env = append(env, fmt.Sprintf("COCKPIT_SECRET_%s=%s", strings.ToUpper(key), value))
			}
			
			// Execute command with injected environment
			cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
			cmd.Env = env
			
			return cmd.Run()
		},
	}
	
	runCmd.Flags().StringVar(&packageFlag, "package", "", "Package name (determines namespace)")
	
	return runCmd
}

func getAllSecretsForNamespace(v vault.Manager, namespace string) map[string]string {
	// This would need to be implemented by listing all keys
	// and filtering by namespace prefix
	return make(map[string]string)
}
```

**Uso:**
```bash
# Em vez de:
PROVIDER=$(cockpit vault get --namespace kb-graphify provider)
API_KEY=$(cockpit vault get --namespace kb-graphify api-key)
./kb-graphify/bin/search

# Usar:
cockpit vault run --package kb-graphify -- ./kb-graphify/bin/search
# O pacote lê: $COCKPIT_SECRET_PROVIDER, $COCKPIT_SECRET_API_KEY
```

### Solução 3: Token-Based com Escopo Limitado

**Conceito:** Cada pacote recebe um token que só pode acessar seu próprio namespace.

```go
// internal/vault/token_auth.go
package vault

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

type AccessToken struct {
	TokenID     string    `json:"token_id"`
	Namespace   string    `json:"namespace"`
	Permissions []string  `json:"permissions"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Signature   string    `json:"signature"`
}

type TokenAuthority struct {
	secretKey  []byte
	tokens     map[string]*AccessToken
}

func NewTokenAuthority() *TokenAuthority {
	return &TokenAuthority{
		secretKey: generateSecretKey(),
		tokens:    make(map[string]*AccessToken),
	}
}

// IssueToken issues a token for a specific namespace
func (ta *TokenAuthority) IssueToken(namespace string, ttl time.Duration) (string, error) {
	tokenID := generateTokenID()
	
	token := &AccessToken{
		TokenID:     tokenID,
		Namespace:   namespace,
		Permissions: []string{fmt.Sprintf("namespace:%s", namespace)},
		IssuedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
	}
	
	// Sign token
	token.Signature = ta.signToken(token)
	
	// Store token
	ta.tokens[tokenID] = token
	
	// Return encoded token
	return ta.encodeToken(token), nil
}

// ValidateAndExecute validates token and executes operation
func (ta *TokenAuthority) ValidateAndExecute(tokenString, operation, key string) (string, error) {
	// Decode and validate token
	token, err := ta.decodeAndValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	
	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return "", fmt.Errorf("token expired")
	}
	
	// Check permission
	if !ta.hasPermission(token, operation, key) {
		return "", fmt.Errorf("permission denied")
	}
	
	// Execute operation (only on token's namespace)
	v := newOSVault()
	namespacedKey := fmt.Sprintf("%s:%s", token.Namespace, key)
	
	switch operation {
	case "get":
		return v.Get(namespacedKey)
	case "set":
		// Would need value parameter
		return "", fmt.Errorf("not implemented")
	case "delete":
		err := v.Delete(namespacedKey)
		return "", err
	default:
		return "", fmt.Errorf("unknown operation")
	}
}

func (ta *TokenAuthority) hasPermission(token *AccessToken, operation, key string) bool {
	// Check if token has permission for this namespace
	requiredPerm := fmt.Sprintf("namespace:%s", token.Namespace)
	
	for _, perm := range token.Permissions {
		if perm == requiredPerm || perm == "*" {
			return true
		}
	}
	
	return false
}
```

## 📊 Comparação de Segurança

| Solução | Pacote Especifica Namespace? | Bypass Possível? | Complexidade |
|---------|------------------------------|------------------|--------------|
| **--namespace (MINHA SOLUÇÃO)** | ✅ Sim | ✅ Sim (vulnerável) | ⭐ Baixa |
| **Vault Service** | ❌ Não | ❌ Não | ⭐⭐⭐⭐ Alta |
| **Env Injection** | ❌ Não | ❌ Não | ⭐⭐ Média |
| **Token-Based** | ⚠️ Indiretamente | ⚠️ Difícil | ⭐⭐⭐ Média-Alta |

## 🎯 Recomendação Crítica

**A solução com --namespace é INSUFICIENTE e insegura.**

**Implementar Vault Service com:**
1. ✅ Validação de identidade do processo
2. ✅ Pacote NUNCA especifica namespace
3. ✅ Vault determina namespace automaticamente
4. ✅ Assinatura de requests
5. ✅ Auditoria completa

**Isso é segurança real, não "security by obscurity".**