# Evolução de Segurança do Vault - Propostas

## 🚨 Problema Atual

**Vulnerabilidade**: Qualquer aplicação que tenha acesso ao vault pode ler todos os secrets, não há segregação de acesso ou controle de permissões.

**Exemplo do problema**:
```go
// Qualquer app pode fazer isso:
v := vault.NewOSVault()
apiKey := v.Get("any_secret_key") // Acessa qualquer secret!
dbPassword := v.Get("any_other_secret") // Sem restrições!
```

## 🎯 Objetivos de Segurança

1. **Segregação de Acesso**: Apps só devem acessar seus próprios secrets
2. **Auditoria**: Rastrear quem acessou quais secrets e quando
3. **Princípio do Menor Privilégio**: Apps devem ter acesso mínimo necessário
4. **Controle de Identidade**: Verificar quem está solicitando os secrets
5. **Tempo de Vida**: Secrets injetados devem ter expiração

## 📋 Propostas de Solução

### Proposta 1: Namespaces por Aplicação (Isolamento)

**Conceito**: Cada aplicação/pacote tem seu próprio namespace no vault, com controle de acesso estrito.

#### Arquitetura

```
Vault Structure:
├── app1/
│   ├── api_key
│   ├── db_password
│   └── config
├── app2/
│   ├── api_key
│   ├── different_secret
└── shared/
    ├── common_cert
    └── infrastructure_key
```

#### Implementação

```go
package vault

import (
    "fmt"
    "os"
    "os/user"
    "path/filepath"
    "github.com/zalando/go-keyring"
)

type NamespacedVault struct {
    namespace string
    osVault   *OSVault
}

// NewNamespacedVault cria um vault para uma aplicação específica
func NewNamespacedVault(appID string) *NamespacedVault {
    return &NamespacedVault{
        namespace: appID,
        osVault:   NewOSVault(),
    }
}

// NewNamespacedVaultFromProcess detecta automaticamente o namespace baseado no processo
func NewNamespacedVaultFromProcess() *NamespacedVault {
    // Detectar aplicação baseado em:
    // - Nome do executável
    // - Diretório de trabalho
    // - UID/GID do processo
    // - Variáveis de ambiente
    
    exePath, _ := os.Executable()
    appName := filepath.Base(exePath)
    
    return NewNamespacedVault(appName)
}

func (nv *NamespacedVault) Set(key string, value string) error {
    namespacedKey := fmt.Sprintf("%s:%s", nv.namespace, key)
    return nv.osVault.Set(namespacedKey, value)
}

func (nv *NamespacedVault) Get(key string) (string, error) {
    namespacedKey := fmt.Sprintf("%s:%s", nv.namespace, key)
    return nv.osVault.Get(namespacedKey)
}

func (nv *NamespacedVault) Delete(key string) error {
    namespacedKey := fmt.Sprintf("%s:%s", nv.namespace, key)
    return nv.osVault.Delete(namespacedKey)
}
```

#### Uso

```go
// App 1 só acessa seus próprios secrets
app1Vault := vault.NewNamespacedVault("myapp")
apiKey, _ := app1Vault.Get("api_key") // Só acessa "myapp:api_key"

// App 2 só acessa seus próprios secrets
app2Vault := vault.NewNamespacedVault("otherapp")
apiKey, _ := app2Vault.Get("api_key") // Só acessa "otherapp:api_key"
```

#### Vantagens
- ✅ Isolamento completo entre aplicações
- ✅ Implementação relativamente simples
- ✅ Compatível com keyring existente

#### Desvantagens
- ❌ Requer registro prévio de aplicações
- ❌ Não previne acesso se app comprometido
- ❌ Difícil compartilhar secrets entre apps

---

### Proposta 2: Command Handler Proxy (Injeção Controlada)

**Conceito**: Aplicações não acessam secrets diretamente; solicitam ao vault que injete em comandos específicos.

#### Arquitetura

```
App → Command Handler → Vault → Executa Comando com Secret
```

#### Implementação

```go
package vault

import (
    "fmt"
    "os/exec"
    "strings"
)

type CommandHandler struct {
    vault *OSVault
}

type SecretInjection struct {
    SecretKey   string
    Placeholder string // ex: {{SECRET}}, $SECRET, etc.
}

func NewCommandHandler() *CommandHandler {
    return &CommandHandler{
        vault: NewOSVault(),
    }
}

// ExecuteWithSecret executa um comando injetando secrets de forma segura
func (ch *CommandHandler) ExecuteWithSecret(
    command string,
    args []string,
    injections []SecretInjection,
) (string, error) {
    
    // Validar comando (whitelist de comandos permitidos)
    if !ch.isCommandAllowed(command) {
        return "", fmt.Errorf("comando não permitido: %s", command)
    }
    
    // Recuperar secrets
    resolvedArgs := make([]string, len(args))
    for i, arg := range args {
        resolvedArg := arg
        for _, injection := range injections {
            secret, err := ch.vault.Get(injection.SecretKey)
            if err != nil {
                return "", fmt.Errorf("erro ao recuperar secret %s: %w", injection.SecretKey, err)
            }
            resolvedArg = strings.ReplaceAll(resolvedArg, injection.Placeholder, secret)
        }
        resolvedArgs[i] = resolvedArg
    }
    
    // Executar comando
    cmd := exec.Command(command, resolvedArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("erro ao executar comando: %w", err)
    }
    
    // Log de auditoria
    ch.logAccess(command, injections)
    
    return string(output), nil
}

func (ch *CommandHandler) isCommandAllowed(command string) bool {
    allowedCommands := []string{
        "psql", "mysql", "curl", "wget", "docker",
        // adicionar conforme necessário
    }
    
    for _, allowed := range allowedCommands {
        if command == allowed {
            return true
        }
    }
    return false
}

func (ch *CommandHandler) logAccess(command string, injections []SecretInjection) {
    // Log de auditoria (sem os valores dos secrets)
    keys := make([]string, len(injections))
    for i, injection := range injections {
        keys[i] = injection.SecretKey
    }
    
    fmt.Printf("[AUDIT] Command: %s, Secrets: %v\n", command, keys)
}
```

#### Uso

```go
handler := vault.NewCommandHandler()

// Em vez de:
// apiKey := vault.Get("api_key")
// exec.Command("curl", "-H", "Authorization: Bearer " + apiKey)

// A aplicação faz:
output, err := handler.ExecuteWithSecret(
    "curl",
    []string{"-H", "Authorization: Bearer {{API_KEY}}", "https://api.example.com"},
    []vault.SecretInjection{
        {SecretKey: "api_key", Placeholder: "{{API_KEY}}"},
    },
)
```

#### Vantagens
- ✅ Controle total sobre quais comandos podem usar secrets
- ✅ Auditoria completa de uso
- ✅ Secrets nunca ficam expostos para a aplicação
- ✅ Pode implementar rate limiting, TTL, etc.

#### Desvantagens
- ❌ Mudança significativa no modelo de uso
- ❌ Requer whitelist de comandos
- ❌ Mais complexo de implementar

---

### Proposta 3: Token-Based Access Control

**Conceito**: Cada aplicação recebe um token com permissões específicas e tempo de vida limitado.

#### Arquitetura

```
Registration → Token Issuance → Token Validation → Secret Access
```

#### Implementação

```go
package vault

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "time"
)

type AccessToken struct {
    AppID       string   `json:"app_id"`
    Permissions []string `json:"permissions"` // ex: ["api_key", "db_password"]
    ExpiresAt   int64    `json:"expires_at"`
    Signature   string   `json:"signature"`
}

type TokenVault struct {
    osVault     *OSVault
    secretKey   string // Chave para assinar tokens
}

func NewTokenVault(secretKey string) *TokenVault {
    return &TokenVault{
        osVault:   NewOSVault(),
        secretKey: secretKey,
    }
}

// RegisterApp registra uma aplicação e emite um token
func (tv *TokenVault) RegisterApp(appID string, permissions []string, ttl time.Duration) (string, error) {
    token := AccessToken{
        AppID:       appID,
        Permissions: permissions,
        ExpiresAt:   time.Now().Add(ttl).Unix(),
    }
    
    // Assinar token
    signature := tv.signToken(token)
    token.Signature = signature
    
    // Serializar
    data, err := json.Marshal(token)
    if err != nil {
        return "", err
    }
    
    return base64.URLEncoding.EncodeToString(data), nil
}

// ValidateToken valida um token e retorna as permissões
func (tv *TokenVault) ValidateToken(tokenString string) (*AccessToken, error) {
    // Decodificar
    data, err := base64.URLEncoding.DecodeString(tokenString)
    if err != nil {
        return nil, err
    }
    
    var token AccessToken
    err = json.Unmarshal(data, &token)
    if err != nil {
        return nil, err
    }
    
    // Validar assinatura
    expectedSig := tv.signToken(token)
    if token.Signature != expectedSig {
        return nil, fmt.Errorf("assinatura inválida")
    }
    
    // Validar expiração
    if time.Now().Unix() > token.ExpiresAt {
        return nil, fmt.Errorf("token expirado")
    }
    
    return &token, nil
}

func (tv *TokenVault) signToken(token AccessToken) string {
    h := hmac.New(sha256.New, []byte(tv.secretKey))
    data := fmt.Sprintf("%s|%v|%d", token.AppID, token.Permissions, token.ExpiresAt)
    h.Write([]byte(data))
    return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// GetWithToken recupera um secret usando um token
func (tv *TokenVault) GetWithToken(tokenString, key string) (string, error) {
    // Validar token
    token, err := tv.ValidateToken(tokenString)
    if err != nil {
        return "", fmt.Errorf("token inválido: %w", err)
    }
    
    // Verificar permissão
    if !tv.hasPermission(token, key) {
        return "", fmt.Errorf("permissão negada para key: %s", key)
    }
    
    // Recuperar secret
    secret, err := tv.osVault.Get(key)
    if err != nil {
        return "", err
    }
    
    // Log de auditoria
    tv.logAccess(token.AppID, key)
    
    return secret, nil
}

func (tv *TokenVault) hasPermission(token *AccessToken, key string) bool {
    for _, perm := range token.Permissions {
        if perm == key || perm == "*" {
            return true
        }
    }
    return false
}

func (tv *TokenVault) logAccess(appID, key string) {
    fmt.Printf("[AUDIT] App: %s accessed key: %s at %v\n", 
        appID, key, time.Now())
}
```

#### Uso

```go
// Registro (feito uma vez)
tokenVault := vault.NewTokenVault("master-secret-key")
token, _ := tokenVault.RegisterApp("myapp", []string{"api_key", "db_password"}, 24*time.Hour)

// Na aplicação
secret, err := tokenVault.GetWithToken(token, "api_key")
```

#### Vantagens
- ✅ Controle granular de permissões
- ✅ Tokens com expiração automática
- ✅ Auditoria por aplicação
- ✅ Pode revogar tokens

#### Desvantagens
- ❌ Requer sistema de gestão de tokens
- ❌ Tokens podem ser comprometidos
- ❌ Complexidade adicional

---

### Proposta 4: Process Identity Verification

**Conceito**: Verificar a identidade do processo que está solicitando o secret (UID, GID, SELinux context, etc.)

#### Implementação

```go
package vault

import (
    "fmt"
    "os"
    "os/user"
    "syscall"
)

type ProcessVault struct {
    osVault       *OSVault
    allowedUIDs   map[int]bool
    allowedGIDs   map[int]bool
    allowedPaths  map[string]bool
}

func NewProcessVault() *ProcessVault {
    return &ProcessVault{
        osVault:      NewOSVault(),
        allowedUIDs:  make(map[int]bool),
        allowedGIDs:  make(map[int]bool),
        allowedPaths: make(map[string]bool),
    }
}

// AllowProcess adiciona permissão para um processo específico
func (pv *ProcessVault) AllowProcess(uid int, gid int, exePath string) {
    pv.allowedUIDs[uid] = true
    pv.allowedGIDs[gid] = true
    pv.allowedPaths[exePath] = true
}

func (pv *ProcessVault) Get(key string) (string, error) {
    // Verificar identidade do processo
    if !pv.verifyProcessIdentity() {
        return "", fmt.Errorf("processo não autorizado")
    }
    
    // Recuperar secret
    return pv.osVault.Get(key)
}

func (pv *ProcessVault) verifyProcessIdentity() bool {
    // Obter informações do processo atual
    pid := os.Getpid()
    
    // No Linux, podemos ler /proc/[pid]/status
    // ou usar syscall para obter UID/GID
    
    uid := os.Getuid()
    gid := os.Getgid()
    
    exePath, err := os.Executable()
    if err != nil {
        return false
    }
    
    // Verificar permissões
    uidAllowed := pv.allowedUIDs[uid]
    gidAllowed := pv.allowedGIDs[gid]
    pathAllowed := pv.allowedPaths[exePath]
    
    return uidAllowed && gidAllowed && pathAllowed
}
```

#### Vantagens
- ✅ Baseado em identidade do SO
- ✅ Difícil de falsificar
- ✅ Integração com segurança do SO

#### Desvantagens
- ❌ Específico para cada SO
- ❌ Complexo de configurar
- ❌ Pode ter problemas com containers

---

### Proposta 5: Secret Injection Service (Microserviço)

**Conceito**: Um serviço separado que gerencia a injeção de secrets, as aplicações solicitam a ele via API.

#### Arquitetura

```
App → HTTP/gRPC → Secret Service → Vault → Return Secret (with TTL)
```

#### Implementação (Conceitual)

```go
// Secret Service (separado)
type SecretService struct {
    vault   *OSVault
    server  *http.Server
}

func (ss *SecretService) Start() {
    http.HandleFunc("/secret", ss.handleSecretRequest)
    ss.server.ListenAndServe(":8080")
}

func (ss *SecretService) handleSecretRequest(w http.ResponseWriter, r *http.Request) {
    // Validar API key do cliente
    clientKey := r.Header.Get("X-Client-Key")
    if !ss.validateClient(clientKey) {
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    key := r.URL.Query().Get("key")
    secret, err := ss.vault.Get(key)
    if err != nil {
        http.Error(w, "Secret not found", 404)
        return
    }
    
    // Retornar secret
    w.Write([]byte(secret))
}
```

#### Uso

```go
// Na aplicação
secret, err := http.Get("http://localhost:8080/secret?key=api_key")
```

#### Vantagens
- ✅ Separação completa de concerns
- ✅ Centraliza gestão de secrets
- ✅ Pode implementar rate limiting, cache, etc.

#### Desvantagens
- ❌ Requer serviço adicional
- ❌ Single point of failure
- ❌ Mais complexidade operacional

---

## 🎯 Recomendação Híbrida

Sugiro uma **abordagem híbrida** combinando as melhores características:

### Fase 1: Namespaces por Aplicação (Imediato)
- Implementar isolamento básico por namespace
- Cada pacote/aplicação tem seu próprio prefixo
- Fácil de implementar, melhoria imediata

### Fase 2: Token-Based Access (Curto Prazo)
- Adicionar controle de acesso baseado em tokens
- Tokens com permissões específicas
- Auditoria básica

### Fase 3: Command Handler (Médio Prazo)
- Implementar para operações críticas
- Injeção controlada em comandos específicos
- Auditoria avançada

### Fase 4: Secret Service (Longo Prazo)
- Para ambientes de produção críticos
- Serviço dedicado de gestão de secrets
- Recursos avançados de segurança

## 📊 Comparação das Propostas

| Proposta | Segurança | Complexidade | Performance | Flexibilidade |
|----------|-----------|--------------|-------------|---------------|
| Namespaces | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Command Handler | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Token-Based | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Process Identity | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| Secret Service | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |

## 🔧 Próximos Passos Sugeridos

1. **Implementar Namespaces** (Fase 1)
   - Adicionar suporte a namespaces no vault atual
   - Manter compatibilidade com código existente
   - Adicionar flags para ativar/desativar

2. **Adicionar Auditoria Básica**
   - Log de acessos aos secrets
   - Timestamp, aplicação, key acessada
   - Sem logar os valores dos secrets

3. **Implementar Token System** (Fase 2)
   - Sistema simples de tokens
   - Permissões básicas
   - Expiração de tokens

4. **Avaliar Command Handler** (Fase 3)
   - Para comandos críticos (deploy, operações DB)
   - Whitelist de comandos permitidos
   - Auditoria detalhada

Essa abordagem evolutiva permite melhorias graduais na segurança sem quebrar compatibilidade existente.