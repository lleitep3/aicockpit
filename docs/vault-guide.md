# Vault System - Guia Completo

## Visão Geral

O Vault System do AICockpit é um sistema seguro de gerenciamento de segredos que utiliza o keyring nativo do sistema operacional para armazenar credenciais sensíveis como chaves de API, tokens e outros segredos.

## Arquitetura

### Componentes Principais

1. **Interface Manager** (`internal/vault/vault.go`)
   - Define a interface `Manager` com métodos: `Set`, `Get`, `Delete`
   - `OSVault` implementa a interface usando o keyring do SO

2. **CLI Commands** (`cmd/vault.go`)
   - `vault set` - Armazena um segredo
   - `vault get` - Recupera um segredo
   - `vault remove` - Remove um segredo

3. **Dependência Externa**
   - `github.com/zalando/go-keyring` - Biblioteca para acesso ao keyring do SO

### Como Funciona

```
┌─────────────┐
│   CLI User  │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  vault set/get  │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│  OSVault        │
│  (serviceName:  │
│   "aicockpit")  │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│  OS Keyring     │
│  - Keychain     │
│  - Credential   │
│    Manager      │
│  - Secret       │
│    Service      │
└─────────────────┘
```

## Comandos Disponíveis

### vault set

Armazena um segredo de forma segura.

```bash
# Método interativo (seguro, não deixa rastros no histórico)
cockpit vault set <chave>

# Método direto (deixa rastros no histórico do shell)
cockpit vault set <chave> --value "valor_secreto"
```

**Exemplos:**
```bash
# Armazenar chave de API da OpenAI
cockpit vault set openai_api_key --value "sk-proj-..."

# Armazenar token do GitHub
cockpit vault set github_token --value "ghp_..."

# Armazenar de forma interativa
cockpit vault set database_password
Enter secret for 'database_password': [senha não aparece]
```

### vault get

Recupera um segredo armazenado.

```bash
cockpit vault get <chave>
```

**Exemplos:**
```bash
# Recuperar chave de API
cockpit vault get openai_api_key
# Output: sk-proj-...

# Usar em scripts
API_KEY=$(cockpit vault get openai_api_key)
```

### vault remove

Remove um segredo armazenado.

```bash
cockpit vault remove <chave>
```

**Exemplos:**
```bash
# Remover chave expirada
cockpit vault get old_api_key
```

## Integração com Sistema Operacional

O vault utiliza o keyring nativo de cada sistema operacional:

### macOS
- **Keychain Access**
- Segredos armazenados em: `aicockpit` service
- Acesso via: Keychain Access.app

### Windows
- **Credential Manager**
- Segredos armazenados em: Windows Credential Manager
- Acesso via: `control keymgr.dll`

### Linux
- **Secret Service API** (GNOME, etc.)
- **KWallet** (KDE)
- Segredos armazenados em: `aicockpit` service
- Acesso via: `secret-tool` (para Secret Service)

## Segurança

### Características de Segurança

1. **Namespace Isolado**: Todos os segredos usam o serviceName `aicockpit`, isolando-os de outras aplicações
2. **Criptografia Nativa**: Utiliza a criptografia fornecida pelo sistema operacional
3. **Input Invisível**: Modo interativo esconde a senha durante a digitação
4. **Sem Logs de Segredos**: O vault nunca loga os valores dos segredos
5. **Error Handling**: Erros não vazam informações sensíveis

### Boas Práticas

1. **Prefira modo interativo**: Evite `--value` para não deixar segredos no histórico do shell
2. **Use nomes descritivos**: `openai_api_key` em vez de `key1`
3. **Remova segredos não usados**: Mantenha o vault limpo
4. **Teste antes de produção**: Use segredos de teste durante desenvolvimento

## Testes

### Testes Unitários

Os testes utilizam `keyring.MockInit()` para simular o keyring em ambiente de teste:

```go
func TestOSVault(t *testing.T) {
    keyring.MockInit()
    v := NewOSVault()
    // Testes de Set, Get, Delete
}
```

### Executar Testes

```bash
# Testes do pacote vault
go test ./internal/vault/... -v

# Testes dos comandos CLI
go test ./cmd/vault_test.go ./cmd/vault.go -v
```

### Cobertura de Testes

- ✅ Teste de Set/Get/Delete básico
- ✅ Teste de erro ao buscar chave inexistente
- ✅ Teste de erro ao deletar chave inexistente
- ✅ Teste de integração CLI
- ✅ Teste de modo interativo (via flag --value)

## Troubleshooting

### Erro: "failed to retrieve secret: secret not found in keyring"

**Causa**: A chave não existe no vault

**Solução**: Verifique se a chave foi armazenada corretamente
```bash
# Liste chaves disponíveis (depende do SO)
# No Linux com secret-tool:
secret-tool search service aicockpit
```

### Erro: "inappropriate ioctl for device" (modo interativo)

**Causa**: O terminal não suporta input interativo (ex: em scripts ou CI/CD)

**Solução**: Use a flag `--value` em vez do modo interativo
```bash
cockpit vault set my_key --value "my_value"
```

### Erro: "failed to save secret to vault"

**Causa**: Problemas com o keyring do sistema operacional

**Solução**: 
- Verifique se o serviço de keyring está rodando
- No Linux: verifique se `gnome-keyring` ou `kwallet` está instalado
- No macOS: verifique as permissões do Keychain Access

## Exemplos de Uso Avançado

### Script de Backup

```bash
#!/bin/bash
# Backup de segredos importantes
keys=("openai_api_key" "github_token" "database_url")

for key in "${keys[@]}"; do
    value=$(cockpit vault get "$key")
    echo "$key=$value" >> backup.txt
done

# Criptografar backup
gpg --encrypt backup.txt
shred backup.txt
```

### Integração com Deploy

```bash
#!/bin/bash
# Script de deploy usando segredos do vault
DB_PASSWORD=$(cockpit vault get database_password)
API_KEY=$(cockpit vault get api_key)

# Usar no deploy
deploy.sh --db-password "$DB_PASSWORD" --api-key "$API_KEY"
```

### Rotação de Segredos

```bash
#!/bin/bash
# Rotacionar chave de API
old_key=$(cockpit vault get openai_api_key)
new_key="sk-proj-new-..."

# Atualizar no vault
cockpit vault set openai_api_key --value "$new_key"

# Invalidar chave antiga no serviço externo
curl -X POST https://api.openai.com/v1/keys/revoke \
  -H "Authorization: Bearer $old_key"
```

## Comparação com Documentação Existente

### Documentação Atual (docs/architecture/05-vault-system.md)

A documentação existente está **correta e completa**, cobrindo:

✅ **Conceitos Básicos**
- Explica o propósito do vault
- Descreve a integração com keyring do SO
- Mostra o diagrama de sequência

✅ **Comandos**
- Lista os comandos principais (set, get, remove)
- Descreve o propósito de cada comando

✅ **Segurança**
- Menciona o namespace fixo "aicockpit"
- Explica a integração sem eco
- Menciona o mock para CI/CD

### Diferenças e Adições

Este guia adiciona:

📋 **Exemplos Práticos**
- Exemplos de uso real com comandos completos
- Scripts de uso avançado
- Casos de uso específicos

🔧 **Troubleshooting**
- Soluções para erros comuns
- Diagnóstico de problemas
- Comandos para verificação

🧪 **Detalhes de Testes**
- Como executar os testes
- Estrutura dos testes
- Cobertura de testes

📚 **Boas Práticas**
- Recomendações de segurança
- Padrões de nomenclatura
- Integração com workflows

## Conclusão

O Vault System do AICockpit está **funcionando corretamente** e bem implementado. A documentação existente está precisa e este guia complementa com exemplos práticos e detalhes operacionais.

**Status**: ✅ Operacional e bem documentado