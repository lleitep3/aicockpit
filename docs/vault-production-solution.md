# Vault Production Solution - Permissões e Compartilhamento

## 🎯 Seus Pontos Excelentes

### 1. **Aplicação precisa apontar para pasta do vault?**
**Problema:** Com VaultService, o cliente precisa saber onde está o socket Unix.

**Soluções:**
- ✅ Local padrão: `/tmp/cockpit-vault.sock` (não precisa configurar)
- ✅ Variável de ambiente: `COCKPIT_VAULT_SOCKET` (para casos especiais)
- ✅ Auto-descoberta via ambiente

### 2. **Compartilhar secrets entre pacotes?**
**Problema:** Alguns secrets precisam ser compartilhados (ex: certificado SSL, DB de produção).

**Soluções:**
- ✅ Namespace especial "shared"
- ✅ Sistema de permissões/ACL
- ✅ Hierarquia de namespaces

### 3. **Comando de permissão na instalação?**
**Ideia BRILHANTE:** Durante instalação, definir quais secrets o pacote pode acessar.

**Solução:**
- ✅ Manifesto de permissões no pacote
- ✅ Comando `cockpit vault grant`
- ✅ Prompt interativo durante instalação

## 🏗️ Solução Completa de Produção

### 1. Sistema de Permissões (ACL)

```go
// internal/vault/permissions.go
package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Permission struct {
	Package    string   `json:"package"`
	Namespace  string   `json:"namespace"`
	Secrets    []string `json:"secrets"`      // Secrets específicos
	Wildcards  []string `json:"wildcards"`    // Padrões como "db_*"
	Shared     []string `json:"shared"`       // Secrets compartilhados
	GrantedAt  time.Time `json:"granted_at"`
	GrantedBy  string   `json:"granted_by"`   // Quem concedeu (user/process)
}

type PermissionManager struct {
	permissions map[string]*Permission  // package -> permissions
	mu          sync.RWMutex
	storagePath string
}

func NewPermissionManager(storagePath string) *PermissionManager {
	if storagePath == "" {
		storagePath = "/home/lleite/.cockpit/vault/permissions.json"
	}
	
	pm := &PermissionManager{
		permissions: make(map[string]*Permission),
		storagePath: storagePath,
	}
	
	pm.load()
	return pm
}

// GrantPermission grants permission for a package to access specific secrets
func (pm *PermissionManager) GrantPermission(packageName, namespace string, secrets []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	perm := &Permission{
		Package:   packageName,
		Namespace: namespace,
		Secrets:   secrets,
		GrantedAt: time.Now(),
		GrantedBy: "user", // ou identidade do processo
	}
	
	pm.permissions[packageName] = perm
	return pm.save()
}

// RevokePermission revokes all permissions for a package
func (pm *PermissionManager) RevokePermission(packageName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	delete(pm.permissions, packageName)
	return pm.save()
}

// CheckPermission checks if a package has permission to access a secret
func (pm *PermissionManager) CheckPermission(packageName, secret string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	perm, exists := pm.permissions[packageName]
	if !exists {
		return false
	}
	
	// Check direct secret access
	for _, s := range perm.Secrets {
		if s == secret {
			return true
		}
	}
	
	// Check wildcards
	for _, wildcard := range perm.Wildcards {
		if matchWildcard(secret, wildcard) {
			return true
		}
	}
	
	// Check shared secrets
	for _, shared := range perm.Shared {
		if shared == secret {
			return true
		}
	}
	
	return false
}

// GetAuthorizedSecrets returns all secrets a package can access
func (pm *PermissionManager) GetAuthorizedSecrets(packageName string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	perm, exists := pm.permissions[packageName]
	if !exists {
		return []string{}
	}
	
	return perm.Secrets
}

func matchWildcard(secret, pattern string) bool {
	// Implement wildcard matching (ex: "db_*" matches "db_password")
	// Por simplicidade, implementação básica
	if pattern == "*" {
		return true
	}
	if pattern == secret {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(secret) >= len(prefix) && secret[:len(prefix)] == prefix
	}
	return false
}

func (pm *PermissionManager) save() error {
	data, err := json.MarshalIndent(pm.permissions, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(pm.storagePath, data, 0600)
}

func (pm *PermissionManager) load() error {
	data, err := os.ReadFile(pm.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Arquivo não existe ainda
		}
		return err
	}
	
	return json.Unmarshal(data, &pm.permissions)
}
```

### 2. Vault Service com Permissões

```go
// internal/vault/vault_service_with_permissions.go
package vault

type VaultServiceWithPermissions struct {
	*VaultService
	permManager *PermissionManager
}

func NewVaultServiceWithPermissions(socketPath string) *VaultServiceWithPermissions {
	return &VaultServiceWithPermissions{
		VaultService: NewVaultService(socketPath),
		permManager:  NewPermissionManager(""),
	}
}

func (vs *VaultServiceWithPermissions) handleConnection(conn net.Conn) {
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
		vs.sendError(encoder, fmt.Sprintf("identity verification failed: %w", err))
		return
	}
	
	// Check permissions
	packageName := vs.determinePackageName(request.Executable)
	if !vs.permManager.CheckPermission(packageName, request.Key) {
		vs.logSecurityEvent("permission_denied", fmt.Sprintf("Package: %s, Key: %s", packageName, request.Key))
		vs.sendError(encoder, fmt.Sprintf("permission denied for key: %s", request.Key))
		return
	}
	
	// Get secret
	namespacedKey := fmt.Sprintf("%s:%s", namespace, request.Key)
	value, err := vs.vault.Get(namespacedKey)
	
	if err != nil {
		vs.sendError(encoder, fmt.Sprintf("secret not found: %v", err))
		return
	}
	
	// Send success
	response := SecretResponse{
		Value:   value,
		Success: true,
	}
	
	encoder.Encode(response)
	vs.logSecurityEvent("secret_access_granted", fmt.Sprintf("Package: %s, Key: %s, Namespace: %s", packageName, request.Key, namespace))
}

func (vs *VaultServiceWithPermissions) determinePackageName(exePath string) string {
	// Similar a determineNamespace mas retorna nome do pacote
	if strings.Contains(exePath, "/.cockpit/packages/") {
		parts := strings.Split(exePath, "/.cockpit/packages/")
		if len(parts) > 1 {
			packagePart := parts[1]
			packageParts := strings.Split(packagePart, "/")
			if len(packageParts) > 0 {
				return packageParts[0]
			}
		}
	}
	
	exeName := filepath.Base(exePath)
	exeName = strings.TrimSuffix(exeName, filepath.Ext(exeName))
	return exeName
}
```

### 3. Comandos de Permissão

```go
// cmd/vault_grant.go
package cmd

import (
	"fmt"
	
	"github.com/lleitep3/aicockpit/internal/vault"
)

func NewVaultGrantCommand() *cobra.Command {
	var packageFlag string
	var namespaceFlag string
	var secretsFlag []string
	
	grantCmd := &cobra.Command{
		Use:   "grant --package <name> --namespace <ns> --secret <secret>",
		Short: "Grant permission for a package to access secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if packageFlag == "" {
				return fmt.Errorf("--package is required")
			}
			if namespaceFlag == "" {
				return fmt.Errorf("--namespace is required")
			}
			if len(secretsFlag) == 0 {
				return fmt.Errorf("--secret is required (can be specified multiple times)")
			}
			
			pm := vault.NewPermissionManager("")
			err := pm.GrantPermission(packageFlag, namespaceFlag, secretsFlag)
			if err != nil {
				return fmt.Errorf("failed to grant permission: %w", err)
			}
			
			fmt.Printf("Granted permission for package '%s' to access secrets: %v in namespace '%s'\n", 
				packageFlag, secretsFlag, namespaceFlag)
			return nil
		},
	}
	
	grantCmd.Flags().StringVar(&packageFlag, "package", "", "Package name")
	grantCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Namespace")
	grantCmd.Flags().StringSliceVar(&secretsFlag, "secret", []string{}, "Secret to grant access (can be repeated)")
	
	return grantCmd
}

func NewVaultRevokeCommand() *cobra.Command {
	var packageFlag string
	
	revokeCmd := &cobra.Command{
		Use:   "revoke --package <name>",
		Short: "Revoke all permissions for a package",
		RunE: func(cmd *cobra.Command, args []string) error {
			if packageFlag == "" {
				return fmt.Errorf("--package is required")
			}
			
			pm := vault.NewPermissionManager("")
			err := pm.RevokePermission(packageFlag)
			if err != nil {
				return fmt.Errorf("failed to revoke permission: %w", err)
			}
			
			fmt.Printf("Revoked all permissions for package '%s'\n", packageFlag)
			return nil
		},
	}
	
	revokeCmd.Flags().StringVar(&packageFlag, "package", "", "Package name")
	
	return revokeCmd
}

func NewVaultListPermissionsCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list-permissions",
		Short: "List all granted permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm := vault.NewPermissionManager("")
			
			fmt.Println("Package Permissions:")
			fmt.Println("=====================")
			
			for pkg, perm := range pm.permissions {
				fmt.Printf("Package: %s\n", pkg)
				fmt.Printf("  Namespace: %s\n", perm.Namespace)
				fmt.Printf("  Secrets: %v\n", perm.Secrets)
				fmt.Printf("  Wildcards: %v\n", perm.Wildcards)
				fmt.Printf("  Shared: %v\n", perm.Shared)
				fmt.Printf("  Granted at: %s\n", perm.GrantedAt.Format("2006-01-02 15:04:05"))
				fmt.Println()
			}
			
			return nil
		},
	}
	
	return listCmd
}
```

### 4. Manifesto de Pacote

```yaml
# cockpit-package.yaml (no pacote)
name: kb-graphify
version: 1.0.0
description: Knowledge base graph integration

# Permissões de vault necessárias
vault_permissions:
  namespace: kb-graphify
  secrets:
    - api-key
    - provider
  shared:
    - shared-db-connection  # Secret compartilhado entre pacotes
  wildcards:
    - kb_*  # Pode acessar qualquer secret começando com kb_
```

### 5. Integração com Instalação de Pacote

```go
// cmd/pkg_install.go
func NewPkgInstallCommand() *cobra.Command {
	var autoGrantFlag bool
	
	installCmd := &cobra.Command{
		Use:   "install <package>",
		Short: "Install a package",
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			
			// 1. Baixar e instalar pacote
			// ... código existente ...
			
			// 2. Ler manifesto do pacote
			manifest, err := readPackageManifest(packageName)
			if err != nil {
				return fmt.Errorf("failed to read manifest: %w", err)
			}
			
			// 3. Verificar se o pacote precisa de permissões de vault
			if manifest.VaultPermissions != nil {
				if autoGrantFlag {
					// Auto-conceder permissões (não recomendado em produção)
					grantPermissions(manifest.VaultPermissions)
				} else {
					// Perguntar ao usuário
					fmt.Printf("\nPackage '%s' requests access to the following secrets:\n", packageName)
					fmt.Printf("  Namespace: %s\n", manifest.VaultPermissions.Namespace)
					fmt.Printf("  Secrets: %v\n", manifest.VaultPermissions.Secrets)
					fmt.Printf("  Shared: %v\n", manifest.VaultPermissions.Shared)
					fmt.Print("\nGrant these permissions? (y/n): ")
					
					var response string
					fmt.Scanln(&response)
					
					if response == "y" || response == "Y" {
						grantPermissions(manifest.VaultPermissions)
						fmt.Println("✓ Permissions granted")
					} else {
						fmt.Println("✗ Permissions denied")
						fmt.Println("  You can grant later using: cockpit vault grant")
					}
				}
			}
			
			return nil
		},
	}
	
	installCmd.Flags().BoolVar(&autoGrantFlag, "auto-grant", false, "Auto-grant vault permissions (not recommended)")
	
	return installCmd
}

func grantPermissions(perm *VaultPermissions) error {
	pm := vault.NewPermissionManager("")
	
	// Combinar secrets, shared e wildcards
	allSecrets := append(perm.Secrets, perm.Shared...)
	allSecrets = append(allSecrets, perm.Wildcards...)
	
	return pm.GrantPermission(perm.Namespace, perm.Namespace, allSecrets)
}
```

### 6. Secrets Compartilhados

```go
// internal/vault/shared.go
package vault

type SharedSecretManager struct {
	vault *osVault
}

func NewSharedSecretManager() *SharedSecretManager {
	return &SharedSecretManager{
		vault: newOSVault(),
	}
}

// SetSharedSecret stores a secret in the shared namespace
func (ssm *SharedSecretManager) SetSharedSecret(key string, value string) error {
	sharedKey := fmt.Sprintf("shared:%s", key)
	return ssm.vault.Set(sharedKey, value)
}

// GetSharedSecret retrieves a shared secret
func (ssm *SharedSecretManager) GetSharedSecret(key string) (string, error) {
	sharedKey := fmt.Sprintf("shared:%s", key)
	return ssm.vault.Get(sharedKey)
}

// GrantSharedAccess grants a package access to a shared secret
func (ssm *SharedSecretManager) GrantSharedAccess(packageName, sharedSecret string) error {
	pm := NewPermissionManager("")
	
	perm, exists := pm.permissions[packageName]
	if !exists {
		perm = &Permission{
			Package:   packageName,
			Namespace: packageName,
			Secrets:   []string{},
		}
	}
	
	// Add to shared list
	perm.Shared = append(perm.Shared, sharedSecret)
	
	pm.permissions[packageName] = perm
	return pm.save()
}
```

## 📋 Fluxo Completo de Instalação com Permissões

```bash
# Usuário instala pacote
cockpit pkg install kb-graphify

# Sistema:
# 1. Baixa pacote
# 2. Lê manifesto: cockpit-package.yaml
# 3. Detecta que precisa de permissões:
#    - namespace: kb-graphify
#    - secrets: api-key, provider
#    - shared: shared-db-connection
# 4. Pergunta ao usuário:
#    "Package kb-graphify requests access to:
#     - kb-graphify:api-key
#     - kb-graphify:provider
#     - shared:shared-db-connection
#     Grant? (y/n)"
# 5. Se usuário aprovar:
#    cockpit vault grant --package kb-graphify --namespace kb-graphify --secret api-key --secret provider --secret shared-db-connection
# 6. Pacote instalado com permissões concedidas
```

## 🎯 SOLUÇÃO PARA SEUS 3 PONTOS

### 1. **Aplicação apontar para pasta do vault?**

```go
// Cliente usa local padrão ou variável de ambiente
socketPath := os.Getenv("COCKPIT_VAULT_SOCKET")
if socketPath == "" {
    socketPath = "/tmp/cockpit-vault.sock"  // Padrão
}

client := vault.NewVaultServiceClient(socketPath)
```
**Resposta:** Não precisa configurar, usa local padrão.

### 2. **Compartilhar secrets entre pacotes?**

```bash
# Definir secret compartilhado
cockpit vault set shared:db-connection "postgres://..."

# Conceder acesso a múltiplos pacotes
cockpit vault grant --package kb-graphify --secret shared:db-connection
cockpit vault grant --package user-service --secret shared:db-connection

# Pacotes acessam via VaultService
client.GetSecret("shared:db-connection")  // Se tiver permissão
```
**Resposta:** Namespace "shared" + sistema de permissões.

### 3. **Comando de permissão na instalação?**

```bash
# Comando manual
cockpit vault grant --package kb-graphify --namespace kb-graphify --secret api-key

# Automático durante instalação
cockpit pkg install kb-graphify
# Sistema pergunta automaticamente baseado no manifesto
```
**Resposta:** Sim! Excelente ideia implementada.