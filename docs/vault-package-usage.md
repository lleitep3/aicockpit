# Uso de Vault por Pacotes

## Visão Geral

Com o sistema de lock/unlock implementado, os pacotes devem seguir um padrão específico para garantir segurança e usabilidade.

## Regra de Ouro: **Sempre usar Namespace**

Pacotes DEVEM sempre usar `--namespace` para acessar secrets. Isso garante:

✅ **Isolamento:** Pacotes só acessam seus próprios secrets  
✅ **Sem lock check:** Namespace bypassa `checkVaultAccess()`  
✅ **Sem master password:** Pacotes não precisam de master password  
✅ **Segurança:** Cross-namespace é bloqueado automaticamente  

## Padrão para Pacotes

### 1. Definir Namespace

O namespace deve ser o nome do pacote:

```go
// No código do pacote
const PackageNamespace = "kb-graphify"
```

### 2. Acessar Secrets

Pacotes devem usar `--namespace` em todas as operações:

```bash
# INCORRETO - Sem namespace (afetado por lock)
PROVIDER=$(cockpit vault get kb-graphify.provider)

# CORRETO - Com namespace (não afetado por lock)
PROVIDER=$(cockpit vault get --namespace kb-graphify provider)
```

### 3. Helper Go para Pacotes

Vou criar um helper para facilitar o uso por pacotes Go:

```go
// internal/vault/package_vault.go
package vault

import (
	"fmt"
	"os/exec"
	"strings"
)

// PackageVault provides vault access for packages
type PackageVault struct {
	namespace string
}

// NewPackageVault creates a vault instance for a package
func NewPackageVault(packageName string) *PackageVault {
	return &PackageVault{
		namespace: sanitizeNamespace(packageName),
	}
}

// Get retrieves a secret from the package's namespace
func (pv *PackageVault) Get(key string) (string, error) {
	cmd := exec.Command("cockpit", "vault", "get", "--namespace", pv.namespace, key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Set stores a secret in the package's namespace
func (pv *PackageVault) Set(key, value string) error {
	cmd := exec.Command("cockpit", "vault", "set", "--namespace", pv.namespace, "--value", value, key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set secret '%s': %w: %s", key, err, string(output))
	}
	return nil
}

// Remove removes a secret from the package's namespace
func (pv *PackageVault) Remove(key string) error {
	cmd := exec.Command("cockpit", "vault", "remove", "--namespace", pv.namespace, key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove secret '%s': %w: %s", key, err, string(output))
	}
	return nil
}

// SetInteractive stores a secret with interactive input (more secure)
func (pv *PackageVault) SetInteractive(key string) error {
	cmd := exec.Command("cockpit", "vault", "set", "--namespace", pv.namespace, key)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

## Exemplo de Uso por Pacote

### kb-graphify (Atualização)

**Antes (Inseguro):**
```bash
PROVIDER=$(cockpit vault get kb-graphify.provider 2>/dev/null || true)
API_KEY=$(cockpit vault get kb-graphify.api-key 2>/dev/null || true)
```

**Depois (Seguro):**
```bash
# No código Go
vault := vault.NewPackageVault("kb-graphify")
provider, _ := vault.Get("provider")
apiKey, _ := vault.Get("api-key")

# Ou via shell (com namespace)
NAMESPACE="kb-graphify"
PROVIDER=$(cockpit vault get --namespace $NAMESPACE provider 2>/dev/null || true)
API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key 2>/dev/null || true)
```

### Pacote Genérico

```go
package main

import (
    "fmt"
    "github.com/lleitep3/aicockpit/internal/vault"
)

func main() {
    // Criar vault para este pacote
    vault := vault.NewPackageVault("meu-pacote")
    
    // Configurar secrets
    vault.Set("api-key", "sk-12345")
    
    // Usar secrets
    apiKey, _ := vault.Get("api-key")
    fmt.Printf("API Key: %s\n", apiKey)
}
```

## Comportamento do Lock/Unlock para Pacotes

### Com Namespace (Recomendado para Pacotes)

```bash
# Vault pode estar locked
cockpit vault lock

# Pacote ainda consegue acessar (namespace isola)
cockpit vault get --namespace meu-pacote api-key
# ✅ Funciona mesmo com vault locked
```

### Sem Namespace (Apenas para CLI do Usuário)

```bash
# Vault locked
cockpit vault lock

# Usuário tentando acessar sem namespace
cockpit vault get api-key
# 🔒 Vault is locked. Access denied
```

## Fluxo Recomendado para Desenvolvedores

### 1. Durante Desenvolvimento

```bash
# Configurar secrets do pacote
cockpit vault set --namespace meu-pacote api-key --value "sk-12345"
cockpit vault set --namespace meu-pacote database-url --value "postgres://..."

# O vault pode ficar locked sem afetar o pacote
cockpit vault lock
```

### 2. Durante Execução do Pacote

```go
// No código do pacote
vault := vault.NewPackageVault("meu-pacote")
apiKey, err := vault.Get("api-key")
// ✅ Funciona mesmo com vault locked
```

### 3. Operações Administrativas

```bash
# Usuário quer bloquear acesso via CLI
cockpit vault lock

# Pacote continua funcionando (com namespace)
meu-pacote --comando
# ✅ Funciona

# Usuário quer bloquear pacote específico
cockpit vault lock meu-pacote

# Pacote tentando acessar
cockpit vault get --namespace meu-pacote api-key
# 🔒 Vault is locked for this package
```

## Resumo

| Cenário | Com Namespace | Sem Namespace |
|---------|---------------|----------------|
| **Vault locked** | ✅ Funciona | 🔒 Bloqueado |
| **Vault unlocked** | ✅ Funciona | ✅ Funciona |
| **Cross-namespace** | 🔒 Bloqueado | 🔒 Bloqueado |
| **Master password** | ❌ Não exige | ✅ Exige (se habilitado) |

**Regra para Pacotes:** Sempre use `--namespace` com o nome do pacote!