# 05. Sistema de Cofre (Vault System)

> [!NOTE]
> **Fase de Desenvolvimento:** A arquitetura do Sistema de Cofre faz parte da **Fase 5** do *roadmap*. Esta estrutura lida com a segurança nativa de segredos na máquina host.

> [!IMPORTANT]
> **Contrato atual do estado de lock:** consulte [Vault lock state v2](../vault-state-v2.md) para o comportamento implementado. O estado usa AES-GCM com uma chave aleatória no keyring do sistema, falha fechada quando não pode ser lido e exige `cockpit vault migrate-state --confirm` para converter um estado legado. As credenciais armazenadas no keyring não são migradas nem apagadas por esse comando.

O `Vault System` (`internal/vault`) é responsável pelo armazenamento seguro de chaves de API, tokens e segredos em geral que o AICockpit e seus Agentes precisam usar (como tokens da OpenAI, GitHub PAT, etc.).

Em vez de armazenar segredos em arquivos de configuração estáticos (`config.yaml`) de forma não segura, o Vault se integra diretamente ao **Gerenciador de Credenciais do Sistema Operacional**, com recursos avançados de segurança incluindo lock/unlock, master password, isolamento de namespace e criptografia de estado.

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Commands                              │
│  vault set, get, remove, lock, unlock, status, etc.            │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    checkVaultAccess                              │
│  Valida estado de lock antes de operações (sem namespace)       │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    LockManager                                  │
│  - Lock/unlock global e por pacote                             │
│  - Auto-lock com expiração por timestamp                       │
│  - Persistência de estado criptografado (AES-256-GCM)           │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                Master Password (Opcional)                        │
│  - Set, change, disable master password                         │
│  - Requerido para lock/unlock quando habilitada                 │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              Vault Implementations                               │
│  - NamespacedVault (isolamento de namespace)                    │
│  - OSVault (acesso direto, descontinuado)                       │
│  - Service (verificação baseada em processo)                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                  OS Keyring (go-keyring)                         │
│  - macOS: Keychain                                              │
│  - Windows: Credential Manager                                  │
│  - Linux: gnome-keyring / KWallet                               │
└─────────────────────────────────────────────────────────────────┘
```

## Componentes

### 1. LockManager

Gerencia estados de lock do vault com persistência criptografada:

```go
type LockManager struct {
    state        *LockState
    storagePath  string
    mu           sync.RWMutex
    encryptor    *StateEncryptor
}

type LockState struct {
    IsLocked          bool              `json:"is_locked"`
    LockedAt          time.Time         `json:"locked_at,omitempty"`
    LockedBy          string            `json:"locked_by,omitempty"`
    PackageLocks      map[string]bool    `json:"package_locks"`
    GlobalUnlock      bool              `json:"global_unlock"`
    UnlockReason      string            `json:"unlock_reason,omitempty"`
    UnlockTime        time.Time         `json:"unlock_time,omitempty"`
    AutoLockExpireAt  time.Time         `json:"auto_lock_expire_at,omitempty"`
}
```

**Funcionalidades:**
- Lock/unlock global
- Lock/unlock específico por pacote
- Auto-lock com expiração por timestamp
- Persistência de estado criptografado
- Controle de acesso via `CanPackageAccess()`

### 2. Master Password

Master password opcional para segurança adicional:

```go
type MasterPassword struct {
    enabled     bool
    passwordHash string
    storagePath string
}
```

**Funcionalidades:**
- Definir master password (mínimo 8 caracteres)
- Mudar master password (requer senha antiga)
- Desabilitar master password (operação administrativa explícita)
- Armazenamento criptografado com chave específica do sistema

**Comandos:**
```bash
cockpit vault set-master-password
cockpit vault change-master-password
cockpit vault disable-master-password
```

### 3. Criptografia de Estado

Estado de lock é um envelope v2 autenticado com AES-256-GCM. A chave de
32 bytes é aleatória e fica no keyring do sistema; o arquivo contém apenas o
identificador da chave, o nonce/ciphertext e a versão:

```go
type EncryptedState struct {
    Version string `json:"version"`
    KeyID   string `json:"key_id"`
    Data    string `json:"data"`
}
```

**Recursos de Segurança:**
- Criptografia AES-256-GCM
- Chave aleatória protegida pelo keyring nativo do SO
- Detecção de adulteração e falhas de keyring
- Falha fechada: estado inválido ou indisponível nega acesso
- Persistência atômica e lock de arquivo para concorrência entre processos

### 4. Isolamento de Namespace

NamespacedVault fornece isolamento de segredos por pacote:

```go
type NamespacedVault struct {
    namespace string
    osVault   *osVault
}
```

**Funcionalidades:**
- Prefixo de namespace para todas as chaves
- Acesso cross-namespace bloqueado
- Sanitização automática de namespace
- Detecção de namespace baseada em processo

## Uso

### Gerenciamento Básico de Segredos

```bash
# Definir um segredo
cockpit vault set api-key --value "sk-12345"

# Definir um segredo em um namespace (recomendado)
cockpit vault set --namespace meu-pacote api-key --value "sk-12345"

# Obter um segredo
cockpit vault get api-key

# Obter um segredo de um namespace
cockpit vault get --namespace meu-pacote api-key

# Remover um segredo
cockpit vault remove api-key

# Remover de um namespace
cockpit vault remove --namespace meu-pacote api-key
```

### Operações de Lock/Unlock

```bash
# Bloquear vault globalmente
cockpit vault lock

# Bloquear pacote específico
cockpit vault lock meu-pacote

# Desbloquear vault globalmente
cockpit vault unlock

# Desbloquear pacote específico
cockpit vault unlock meu-pacote

# Desbloquear com auto-lock timeout
cockpit vault unlock --timeout 1h

# Verificar status de lock
cockpit vault status
```

**Nota:** Lock/unlock requer master password quando ela está habilitada; falhas
de leitura da senha ou do estado são reportadas e negam a operação.

### Gerenciamento de Master Password

```bash
# Definir master password (interativo)
cockpit vault set-master-password

# Mudar master password (requer senha antiga)
cockpit vault change-master-password

# Desabilitar master password (operação explícita, não recomendado)
cockpit vault disable-master-password
```

### Factory Reset

```bash
# Resetar vault - deleta todos os segredos e configurações
cockpit vault factory-reset
```

**Aviso:** Esta ação não pode ser desfeita. Use se esqueceu sua master password.

### Migração do estado de lock

Estados legados não são sobrescritos automaticamente. A migração deve ser
explícita e cria um backup do arquivo antigo:

```bash
cockpit vault migrate-state --confirm
```

Os grants de pacotes e desbloqueios globais do estado legado são descartados;
as credenciais do keyring são preservadas. O estado v2 novo começa bloqueado.
`COCKPIT_DEV_MODE` não desativa a verificação do estado nem a autenticação de
produção; testes usam keyring falso e diretórios temporários.

## Modelo de Segurança

### Controle de Acesso

1. **Com Namespace (--namespace):**
   - Namespace fornece isolamento
   - Sem verificação de lock necessária
   - Pacotes só podem acessar seus próprios segredos

2. **Sem Namespace:**
   - `checkVaultAccess()` valida estado de lock
   - Requer vault desbloqueado
   - Usa identidade de processo para controle de acesso

### Mecanismo de Auto-Lock

Auto-lock é implementado usando expiração por timestamp:

1. Ao desbloquear com `--timeout 5s`, salva `AutoLockExpireAt = agora + 5s`
2. Em cada acesso via `CanPackageAccess()`, verifica se `agora > AutoLockExpireAt`
3. Se expirado, bloqueia automaticamente o vault
4. Funciona perfeitamente no contexto CLI sem processos de background

### Segurança de Estado

1. **Criptografia:**
   - Estado criptografado com AES-256-GCM
   - Chave aleatória armazenada no keyring do sistema
   - `key_id` identifica a chave sem expô-la no arquivo

2. **Integridade e disponibilidade:**
   - AES-GCM detecta modificações manuais do envelope
   - Ausência da chave, corrupção ou erro do keyring falha fechada
   - O arquivo é gravado atomicamente sob lock entre processos

## Estrutura de Arquivos

```
~/.cockpit/vault/
├── lock_state.json          # Estado de lock criptografado
└── master_password.dat      # Master password criptografada
```

Ambos arquivos são criptografados com chaves específicas do sistema e não podem ser modificados manualmente sem detecção.

## Testes

Execute testes do vault:

```bash
# Executar todos os testes do vault
go test ./internal/vault/...

# Executar teste específico
go test ./internal/vault/ -run TestNamespacedVault
```

## Exemplos

Veja o diretório `examples/` para exemplos completos:

- `examples/basic/vault_example.go` - Uso básico do vault
- `examples/demos/vault-lock/main.go` - Demo de lock/unlock
- `examples/demos/namespaced/main.go` - Demo de isolamento de namespace

## Considerações de Segurança

1. **Master Password:**
   - Opcional mas recomendado para produção
   - Mínimo 8 caracteres
   - Hashed com SHA-256 antes do armazenamento
   - Criptografado com chave específica do sistema

2. **Isolamento de Namespace:**
   - Use `--namespace` para todos os segredos de pacotes
   - Previne acesso cross-pacote
   - Bypassa verificações de lock (namespace fornece isolamento)

3. **Auto-Lock:**
   - Use `--timeout` para acesso temporário
   - Bloqueia automaticamente após expiração
   - Verificado em cada tentativa de acesso

4. **Criptografia de Estado:**
   - Não pode ser modificado manualmente sem detecção
   - Erros de corrupção não viram defaults desbloqueados
   - Usa criptografia padrão da indústria

## Como Funciona

A integração utiliza o ecossistema subjacente de cada SO para garantir criptografia e controle de acesso:

- **macOS:** Keychain Access
- **Windows:** Credential Manager
- **Linux:** Secret Service API / KWallet

```mermaid
sequenceDiagram
    participant User as Usuário (CLI)
    participant Vault as VaultManager
    participant Lock as LockManager
    participant OS as OS Keyring
    participant Config as config.yaml

    Note over User, Lock: Definindo Segredo com Namespace
    User->>Vault: cockpit vault set --namespace pkg key
    Vault->>OS: Save("aicockpit", "pkg:key", value)
    OS-->>Vault: Sucesso
    Vault-->>User: Guardado com Segurança
    
    Note over Config, Lock: Durante Execução da IA
    Config->>Vault: Get("pkg:key")
    Vault->>Lock: CheckAccess()
    Lock->>Lock: checkAutoLock()
    Lock-->>Vault: Acesso Permitido
    Vault->>OS: Retrieve("aicockpit", "pkg:key")
    OS-->>Vault: (Valor Descriptografado)
    Vault-->>Config: Injeta na Sessão
```

## Padrões de Segurança

1. **Namespace Fixo:** O serviço é registrado sob o namespace estrito `aicockpit`, isolando os tokens de outras credenciais do sistema.
2. **Integração sem Eco:** A CLI previne que chaves longas e sensíveis apareçam no terminal durante o momento da inserção.
3. **Mock em CI/CD:** A arquitetura suporta um modo simulado (`keyring.MockInit`) para executar testes automatizados transparentes no GitHub Actions.
4. **Validação de Input:** O sistema valida que valores vazios não são aceitos, evitando armazenamento incorreto.
5. **Tratamento de Erros:** Erros são encapsulados com informações contextuais sem vazar dados sensíveis.
6. **Lock por Padrão:** O vault inicia locked por segurança, requerendo unlock explícito.
7. **Criptografia de Estado:** Estado v2 é autenticado por AES-GCM com chave no keyring.
8. **Auto-Lock:** Timeout automático previne acesso não autorizado após período de inatividade.

## Troubleshooting

### Erro: "secret not found in keyring"

**Causa:** A chave não existe no vault.

**Solução:** Verifique se a chave foi armazenada corretamente e se está usando o namespace correto.
```bash
# Tente recuperar a chave sem namespace
cockpit vault get sua_chave

# Ou com namespace
cockpit vault get --namespace seu-pacote sua_chave
```

### Erro: "inappropriate ioctl for device"

**Causa:** O terminal não suporta input interativo (ex: scripts ou CI/CD).

**Solução:** Use a flag `--value` em vez do modo interativo.
```bash
cockpit vault set sua_chave --value "seu_valor"
```

### Erro: "Vault is locked. Access denied"

**Causa:** O vault está locked e o acesso foi bloqueado.

**Solução:** Desbloqueie o vault primeiro.
```bash
# Desbloquear globalmente
cockpit vault unlock

# Ou desbloquear pacote específico
cockpit vault unlock seu-pacote
```

### Erro: "failed to save secret to vault"

**Causa:** Problemas com o keyring do sistema operacional.

**Solução:**
- **Linux:** Verifique se `gnome-keyring` ou `kwallet` está instalado e rodando
- **macOS:** Verifique as permissões do Keychain Access
- **Windows:** Verifique se o Credential Manager está funcionando

### Erro: "vault lock state unavailable" ou "vault lock state migration required"

**Causa:** O estado v2 não pôde ser lido/descriptografado, a chave desapareceu do
keyring ou o arquivo ainda usa o formato legado.

**Solução:** Verifique o keyring do sistema. Para um estado legado, faça backup
e execute a migração explicitamente:

```bash
cockpit vault migrate-state --confirm
```

Essa operação preserva as credenciais, descarta grants/desbloqueios legados e
inicia o novo estado bloqueado.

### Erro: "master password not set"

**Causa:** Operação de lock/unlock requer master password.

**Solução:**
```bash
# Definir master password
cockpit vault set-master-password

# Configure a senha antes de lock/unlock
cockpit vault set-master-password
cockpit vault unlock
```

## Detalhes de Implementação

### Estrutura do Código

```
internal/vault/
├── vault.go                  - Interface Manager e implementação OSVault
├── vault_test.go             - Testes unitários com mock
├── namespaced.go             - NamespacedVault para isolamento
├── namespaced_test.go        - Testes de namespace
├── lock_manager.go           - LockManager com estado criptografado
├── master_password.go        - Master password management
├── state_encryptor.go        - Criptografia e assinatura de estado
├── command_handler.go        - CommandHandler para execução segura
└── secure_vault.go           - SecureVault com criptografia AES-256

cmd/
├── vault.go                  - Comandos CLI (set, get, remove)
├── vault_lock.go             - Comandos CLI (lock, unlock, status)
└── vault_test.go             - Testes de integração CLI
```

### Interface Manager

```go
type Manager interface {
    Set(key string, value string) error
    Get(key string) (string, error)
    Delete(key string) error
    ClearAllSecrets() error  // Factory reset
}
```

### Dependências

- `github.com/zalando/go-keyring`: Biblioteca para acesso ao keyring do sistema operacional
- `golang.org/x/term`: Para input de senha invisível no modo interativo
- `crypto/aes`: Criptografia AES-256
- `crypto/sha256`: Hash SHA-256

> **Próximo Passo:** Entenda como as informações são guardadas e interligadas no AICockpit lendo o [06. Base de Conhecimento (Knowledge Base)](06-knowledge-base.md).
