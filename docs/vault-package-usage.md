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

### 2. Acessar Secrets em Go

Pacotes devem usar PackageVault helper:

```go
import "github.com/lleitep3/aicockpit/internal/vault"

// Criar vault para este pacote
vault := vault.NewPackageVault("meu-pacote")

// Acessar secrets (funciona mesmo com vault locked)
apiKey, err := vault.Get("api-key")
if err != nil {
    // ⚠️ IMPORTANTE: NUNCA logar o valor da chave em texto plano
    log.Printf("Failed to get api-key: %v", err)
    return err
}

// Usar a chave
// ⚠️ NUNCA logar secrets sensíveis
fmt.Printf("API Key: %s\n", apiKey) // ❌ INSEGURO - não faça isso
```

**✅ FORMA SEGURA de usar secrets:**

```go
import "github.com/lleitep3/aicockpit/internal/vault"

vault := vault.NewPackageVault("meu-pacote")

// Função segura para mascarar secrets
func maskSecret(secret string) string {
    if len(secret) <= 8 {
        return "***"
    }
    return secret[:4] + "..." + secret[len(secret)-4:]
}

// Acessar secrets
apiKey, err := vault.Get("api-key")
if err != nil {
    log.Printf("Failed to get api-key: %v", err) // ✅ Seguro - só loga erro
    return err
}

// Log de forma segura
log.Printf("Using API key: %s", maskSecret(apiKey)) // ✅ Seguro - mascarado

// Usar em código
makeAPICall(apiKey)
```

### 3. Acessar Secrets em Shell Scripts

Pacotes devem usar `--namespace` em todas as operações:

```bash
# INCORRETO - Sem namespace (afetado por lock)
PROVIDER=$(cockpit vault get kb-graphify.provider)

# CORRETO - Com namespace (não afetado por lock)
NAMESPACE="kb-graphify"
PROVIDER=$(cockpit vault get --namespace $NAMESPACE provider)
```

**⚠️ IMPORTANTE em shell scripts:**

```bash
# ❌ INSEGURO - Deixa rastros no histórico do shell
API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key)
echo "API Key: $API_KEY"  # ❌ NUNCA imprimir secrets

# ✅ SEGURO - Use diretamente, não armazene em variáveis se possível
cockpit vault get --namespace $NAMESPACE api-key | xargs -I {} curl -H "Authorization: Bearer {}" https://api.example.com
```

### 4. Configurar Secrets

```bash
# Configure secrets do pacote
NAMESPACE="meu-pacote"
cockpit vault set --namespace $NAMESPACE api-key --value "sk-12345"
cockpit vault set --namespace $NAMESPACE database-url --value "postgres://..."
```

**⚠️ Use modo interativo quando possível:**

```bash
# ✅ Mais seguro (modo interativo - input invisível)
cockpit vault set --namespace $NAMESPACE api-key
# [prompt interativo - input invisível]

# ⚠️ Menos seguro (deixa rastros no histórico do shell)
cockpit vault set --namespace $NAMESPACE api-key --value "sk-12345"
```

### 5. Usar em Variáveis de Ambiente

```bash
# Exportar secrets para variáveis de ambiente
NAMESPACE="meu-pacote"
export API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key)
export DB_URL=$(cockpit vault get --namespace $NAMESPACE database-url)

# Usar na aplicação
./my-application
```

**⚠️ Cuidado:** Variáveis de ambiente podem ser visíveis para outros processos. Use apenas quando necessário.

## Helper Go para Pacotes

### PackageVault

PackageVault fornece acesso simplificado ao vault com namespace automático:

```go
type PackageVault struct {
    namespace string
}

// NewPackageVault cria uma vault instance para um pacote
func NewPackageVault(packageName string) *PackageVault

// Get recupera um secret do namespace do pacote
func (pv *PackageVault) Get(key string) (string, error)

// Set armazena um secret no namespace do pacote
func (pv *PackageVault) Set(key, value string) error

// SetInteractive armazena um secret com input interativo (mais seguro)
func (pv *PackageVault) SetInteractive(key string) error

// Remove remove um secret do namespace do pacote
func (pv *PackageVault) Remove(key string) error

// GetWithDefault recupera um secret, retornando default se não encontrado
func (pv *PackageVault) GetWithDefault(key, defaultValue string) string
```

## Exemplos de Uso Seguro

### Pacote Go - Padrão Completo

```go
package main

import (
    "fmt"
    "log"
    "github.com/lleitep3/aicockpit/internal/vault"
)

func main() {
    // Criar vault para este pacote
    vault := vault.NewPackageVault("meu-pacote")
    
    // Configurar secrets (normalmente feito uma vez, não no código)
    // Use modo interativo em produção:
    // vault.SetInteractive("api-key")
    
    // Recuperar secrets
    apiKey, err := vault.Get("api-key")
    if err != nil {
        // ✅ Seguro - só loga o erro, não o valor
        log.Printf("Failed to get api-key: %v", err)
        return
    }
    
    // ✅ Seguro - log mascarado
    log.Printf("Using API key: %s", maskSecret(apiKey))
    
    // Usar o secret
    err = makeAPICall(apiKey)
    if err != nil {
        log.Printf("API call failed: %v", err)
        return
    }
    
    // Limpar secret quando não for mais necessário
    defer vault.Remove("api-key")
}

func makeAPICall(apiKey string) error {
    // Implementação da chamada de API
    // ❌ NUNCA logar o apiKey aqui
    return nil
}

func maskSecret(secret string) string {
    if len(secret) <= 8 {
        return "***"
    }
    return secret[:4] + "..." + secret[len(secret)-4:]
}
```

### Script Shell - Padrão Completo

```bash
#!/bin/bash
set -e

NAMESPACE="meu-pacote"

# Configurar secrets (normalmente feito uma vez)
# Use modo interativo em produção:
# cockpit vault set --namespace $NAMESPACE api-key
# [prompt interativo]

# Recuperar e usar secrets diretamente (sem armazenar em variáveis)
# ✅ Mais seguro - não armazena em variável
cockpit vault get --namespace $NAMESPACE api-key | \
    xargs -I {} curl -H "Authorization: Bearer {}" https://api.example.com

# Se precisar armazenar, use com cuidado
# ✅ Aceitável - variável de ambiente
export API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key)
./my-application

# Limpar a variável após uso
unset API_KEY

# Remover secrets quando não forem mais necessários
cockpit vault remove --namespace $NAMESPACE api-key
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
# Configurar secrets do pacote (use modo interativo)
cockpit vault set --namespace meu-pacote api-key
# [prompt interativo]

cockpit vault set --namespace meu-pacote database-url
# [prompt interativo]

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

## Melhores Práticas de Segurança

### 1. ❌ NUNCA Logar Secrets em Texto Plano

```go
// ❌ INSEGURO
log.Printf("API Key: %s", apiKey)
fmt.Printf("Debug: %s\n", apiKey)
```

```go
// ✅ SEGURO
log.Printf("API Key: %s", maskSecret(apiKey))
log.Printf("Failed to get api-key: %v", err) // só loga erro
```

### 2. ⚠️ Use Modo Interativo Quando Possível

```bash
# ⚠️ Menos seguro (deixa rastros no histórico do shell)
cockpit vault set --namespace $NAMESPACE api-key --value "sk-12345"

# ✅ Mais seguro (input invisível)
cockpit vault set --namespace $NAMESPACE api-key
```

### 3. ✅ Sempre Use Namespace

```go
// ❌ INCORRETO (afetado por lock)
v := vault.NewOSVault()
apiKey, _ := v.Get("api-key")

// ✅ CORRETO (bypassa lock)
v := vault.NewPackageVault("meu-pacote")
apiKey, _ := v.Get("api-key")
```

### 4. ✅ Remover Secrets Quando Não Forem Necessários

```go
defer vault.Remove("api-key")
```

```bash
cockpit vault remove --namespace $NAMESPACE api-key
```

### 5. ⚠️ Evite Armazenar Secrets em Variáveis de Ambiente

```bash
# ⚠️ Aceitável, mas use com cuidado
export API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key)

# ✅ Melhor - use diretamente
cockpit vault get --namespace $NAMESPACE api-key | xargs -I {} comando
```

## Resumo

| Cenário | Com Namespace | Sem Namespace |
|---------|---------------|----------------|
| **Vault locked** | ✅ Funciona | 🔒 Bloqueado |
| **Vault unlocked** | ✅ Funciona | ✅ Funciona |
| **Cross-namespace** | 🔒 Bloqueado | 🔒 Bloqueado |
| **Master password** | ❌ Não exige | ✅ Exige (se habilitado) |

**Regra para Pacotes:** Sempre use `--namespace` com o nome do pacote!

**Regra de Segurança:** NUNCA logar secrets em texto plano. Sempre use masking ou logue apenas erros.