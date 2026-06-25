# Análise de Segurança do Vault - Vulnerabilidades e Soluções

## 🚨 Descoberta Crítica

### Pacotes que Usam o Vault Atualmente

**kb-graphify** - Já usa o vault via CLI:
```bash
# kb-graphify/bin/validate
PROVIDER=$(cockpit vault get kb-graphify.provider 2>/dev/null || true)
API_KEY=$(cockpit vault get kb-graphify.api-key 2>/dev/null || true)

# kb-graphify/bin/configure
cockpit vault set kb-graphify.provider --value "$PROVIDER"
cockpit vault set kb-graphify.api-key --value "$API_KEY"

# kb-graphify/bin/kb-search e kb-index
API_KEY=$(cockpit vault get kb-graphify.api-key 2>/dev/null)
PROVIDER=$(cockpit vault get kb-graphify.provider 2>/dev/null)
```

**Problema Identificado:**
- O pacote usa `cockpit vault get` diretamente via CLI
- Pode acessar QUALQUER secret, não apenas os seus
- Não há isolamento por namespace

## 🔓 Formas de Contornar as Proteções Implementadas

### 1. Acesso Direto ao OSVault (Bypass do NamespacedVault)

**Vulnerabilidade:**
```go
// ❌ Bypass: Pacote malicioso pode usar OSVault diretamente
import "github.com/lleitep3/aicockpit/internal/vault"

v := vault.NewOSVault() // Em vez de NewNamespacedVault
allSecrets := v.Get("any_key") // Acessa qualquer secret!
```

**Por que funciona:**
- OSVault ainda é público
- Não há restrição de acesso no nível de código
- Qualquer pacote pode importar e usar OSVault

**Solução:**
```go
// Tornar OSVault privado ou remover acesso direto
// Forçar uso apenas através de interfaces controladas

// Opção 1: Tornar OSVault privado
type osVault struct { ... } // minúscula = privado

// Opção 2: Remover exportação
// Não exportar NewOSVault()
```

### 2. Acesso via CLI (Bypass Completo)

**Vulnerabilidade:**
```bash
# ❌ Bypass: Qualquer pacote pode chamar a CLI diretamente
cockpit vault get any_secret_key
cockpit vault set any_key --value "any_value"
```

**Por que funciona:**
- CLI não tem verificação de quem está chamando
- Não há autenticação de processo
- Qualquer script/pacote pode executar

**Solução:**
```bash
# Adicionar verificação de caller
# Opção 1: Verificar UID/GID do processo
# Opção 2: Exigir token de autenticação
# Opção 3: Whitelist de binários permitidos

# Exemplo:
cockpit vault get kb-graphify.api-key --caller-pid $$ --verify-signature
```

### 3. Acesso Direto ao Keyring (Bypass Total)

**Vulnerabilidade:**
```go
// ❌ Bypass: Acessar keyring diretamente
import "github.com/zalando/go-keyring"

keyring.Set("aicockpit", "any_key", "any_value")
secret := keyring.Get("aicockpit", "any_key")
```

**Por que funciona:**
- Dependência go-keyring é pública
- Qualquer um pode importar e usar
- Não há proteção no nível de dependência

**Solução:**
```go
// Opção 1: Wrapper com autenticação
// Opção 2: Usar keyring com credenciais específicas
// Opção 3: Implementar própria camada de criptografia

// Exemplo:
type SecureKeyring struct {
    authToken string
    appID     string
}

func (sk *SecureKeyring) Get(key string) (string, error) {
    if !sk.verifyAccess() {
        return "", errors.New("unauthorized")
    }
    return keyring.Get("aicockpit", sk.namespacedKey(key))
}
```

### 4. Leitura de Arquivos do Keyring (Bypass Físico)

**Vulnerabilidade:**
```bash
# ❌ Bypass: Ler arquivos do keyring diretamente
cat ~/.local/share/keyrings/login.keyring
# Ou copiar o arquivo para outro lugar e atacar offline
```

**Por que funciona:**
- Arquivos do keyring são acessíveis
- Se o usuário tiver permissão, pode ler
- Criptografia pode ser quebrada offline

**Solução:**
```bash
# Opção 1: Permissões mais restritivas
chmod 600 ~/.local/share/keyrings/login.keyring

# Opção 2: Usar keyring com proteção adicional
# Opção 3: Implementar camada de criptografia adicional

# Exemplo:
# Criptografar secrets com chave específica do app
encrypted := encrypt(secret, appSpecificKey)
keyring.Set("aicockpit", key, encrypted)
```

### 5. Environment Variable Poisoning

**Vulnerabilidade:**
```bash
# ❌ Bypass: Manipular variáveis de ambiente
export COCKPIT_APP_ID="other_app"
# Agora o NamespacedVault usa namespace do outro app
```

**Por que funciona:**
- NamespacedVaultFromEnv usa variáveis de ambiente
- Fácil de manipular
- Não há validação da identidade real

**Solução:**
```go
// Opção 1: Validar identidade do processo
// Opção 2: Usar múltiplos fatores de autenticação
// Opção 3: Assinar digitalmente o namespace

// Exemplo:
func NewSecureNamespacedVault() *NamespacedVault {
    appID := os.Getenv("COCKPIT_APP_ID")
    realAppID := verifyProcessIdentity() // Verificar exePath, UID, etc.
    
    if appID != realAppID {
        log.Fatalf("Namespace mismatch: env=%s, real=%s", appID, realAppID)
    }
    
    return NewNamespacedVault(appID)
}
```

## 🛡️ Soluções Completas

### Solução 1: Vault como Serviço (Melhor Segurança)

**Arquitetura:**
```
App → Auth → Vault Service → Keyring
```

**Implementação:**
```go
// Vault Service (separado, com autenticação)
type VaultService struct {
    server      *http.Server
    authTokens  map[string]string // appID -> token
    accessLog   []AccessRecord
}

type AccessRequest struct {
    AppID     string `json:"app_id"`
    Token     string `json:"token"`
    Key       string `json:"key"`
    Operation string `json:"operation"` // get/set/delete
}

func (vs *VaultService) handleRequest(w http.ResponseWriter, r *http.Request) {
    var req AccessRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Validar token
    if !vs.validateToken(req.AppID, req.Token) {
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    // Validar permissão
    if !vs.hasPermission(req.AppID, req.Key) {
        http.Error(w, "Forbidden", 403)
        return
    }
    
    // Executar operação
    // Log de auditoria
    vs.logAccess(req)
}
```

**Vantagens:**
- ✅ Controle total de autenticação
- ✅ Auditoria centralizada
- ✅ Pode implementar rate limiting
- ✅ Tokens com expiração
- ✅ Revogação de acesso

**Desvantagens:**
- ❌ Requer serviço adicional
- ❌ Single point of failure
- ❌ Mais complexidade operacional

### Solução 2: Hardening do Vault Atual

**Mudanças necessárias:**

1. **Tornar OSVault privado**
```go
// internal/vault/vault.go
type osVault struct { ... } // minúsculo = privado

// Só exportar interface controlada
func NewVault(appID string) Manager {
    return &namespacedVault{
        namespace: appID,
        osVault:   &osVault{}, // uso interno
    }
}
```

2. **Adicionar autenticação na CLI**
```bash
# cmd/vault.go
// Adicionar verificação de caller
func verifyCaller() error {
    callerPID := os.Getppid()
    exePath := getProcessPath(callerPID)
    
    if !isWhitelisted(exePath) {
        return fmt.Errorf("caller not whitelisted: %s", exePath)
    }
    return nil
}
```

3. **Wrapper seguro para keyring**
```go
// internal/vault/secure_keyring.go
type SecureKeyring struct {
    appID      string
    encryptionKey []byte
}

func (sk *SecureKeyring) Set(key string, value string) error {
    // Criptografar antes de armazenar
    encrypted := encrypt(value, sk.encryptionKey)
    namespacedKey := fmt.Sprintf("%s:%s", sk.appID, key)
    return keyring.Set("aicockpit", namespacedKey, encrypted)
}
```

4. **Validação de identidade**
```go
// internal/vault/identity.go
func verifyProcessIdentity() (string, error) {
    exePath, _ := os.Executable()
    uid := os.Getuid()
    gid := os.Getgid()
    
    // Verificar assinatura do executável (opcional)
    if !verifySignature(exePath) {
        return "", fmt.Errorf("unsigned executable")
    }
    
    // Gerar appID baseado em identidade verificada
    return generateAppID(exePath, uid, gid), nil
}
```

**Vantagens:**
- ✅ Melhora significativa sem mudança de arquitetura
- ✅ Mantém compatibilidade parcial
- ✅ Não requer serviço adicional

**Desvantagens:**
- ❌ Ainda vulnerável a ataques no nível de SO
- ❌ Difícil de implementar perfeitamente
- ❌ Pode quebrar compatibilidade

### Solução 3: Híbrida (Recomendada)

**Fase 1: Hardening Imediato**
- Tornar OSVault privado
- Adicionar validação básica na CLI
- Implementar SecureKeyring

**Fase 2: Namespaces Obrigatórios**
- Forçar uso de NamespacedVault
- Remover acesso direto ao OSVault
- Adicionar validação de identidade

**Fase 3: Vault Service (Produção)**
- Implementar Vault Service para ambientes críticos
- Migrar pacotes sensíveis para usar o serviço
- Manter CLI compatível para desenvolvimento

## 📊 Análise de Risco Atual

| Vulnerabilidade | Probabilidade | Impacto | Risco |
|-----------------|---------------|---------|-------|
| OSVault bypass | Alta | Alto | 🔴 Crítico |
| CLI bypass | Alta | Alto | 🔴 Crítico |
| Keyring direto | Média | Alto | 🟡 Alto |
| Arquivo keyring | Baixa | Alto | 🟡 Alto |
| Env poisoning | Média | Médio | 🟡 Médio |

## 🎯 Plano de Ação Imediato

### 1. Auditoria de Pacotes Existentes
```bash
# Verificar todos os pacotes instalados
for pkg in ~/.cockpit/packages/*/; do
    echo "Checking $pkg for vault usage..."
    grep -r "vault" "$pkg" || echo "No vault usage found"
done
```

### 2. Atualizar kb-graphify
```bash
# Migrar para usar NamespacedVault
# Atualmente usa: cockpit vault get kb-graphify.api-key
# Deveria usar: cockpit vault get --namespace kb-graphify api-key
```

### 3. Implementar Hardening Básico
- Tornar OSVault privado
- Adicionar verificação na CLI
- Documentar migração para pacotes

### 4. Adicionar Avisos de Segurança
- Documentar vulnerabilidades conhecidas
- Fornecer guidelines para desenvolvedores
- Adicionar warnings na CLI

## 📝 Conclusão

**Status Atual:** 🔴 **Vulnerável**

O vault implementado tem boas ideias (namespaces, command handler) mas ainda é **vulnerável** a bypass porque:

1. OSVault ainda é público
2. CLI não tem autenticação de caller
3. Qualquer pacote pode acessar keyring diretamente
4. Não há validação real de identidade

**Recomendação:** Implementar **Solução Híbrida** começando pelo hardening básico e evoluindo para Vault Service em produção.

**Prioridade:** 🔴 **Alta** - kb-graphify já usa o vault e está vulnerável.