# Exemplos de Uso do Vault AICockpit

Este diretório contém exemplos práticos de como diferentes tipos de aplicações podem usar os secrets armazenados no vault do AICockpit.

## 📁 Exemplos Disponíveis

### 1. basic/vault_example.go
**Linguagem**: Go  
**Tipo**: Integração direta com o pacote `internal/vault`

**Como executar:**
```bash
cd /home/lleite/projects/aicockpit
go run examples/basic/vault_example.go
```

**O que demonstra:**
- Uso direto do pacote `vault` em código Go
- Recuperação de secrets
- Tratamento de erros
- Uso em simulação de chamadas de API
- Limpeza de secrets de teste

**Quando usar:**
- Sua aplicação é escrita em Go
- Você tem acesso ao código do AICockpit
- Quer integração direta sem overhead de CLI

### 2. basic/vault_example.sh
**Linguagem**: Shell Script  
**Tipo**: Uso via CLI

**Como executar:**
```bash
cd /home/lleite/projects/aicockpit
chmod +x examples/basic/vault_example.sh
./examples/basic/vault_example.sh
```

**O que demonstra:**
- Uso do vault via linha de comando
- Captura de saída para variáveis
- Exportação para variáveis de ambiente
- Geração de arquivos de configuração
- Limpeza automática

**Quando usar:**
- Scripts de deploy
- Scripts de CI/CD
- Integração com ferramentas Unix/Linux
- Automação em geral

### 3. basic/vault_example.py
**Linguagem**: Python  
**Tipo**: Uso via CLI (cross-language)

**Como executar:**
```bash
cd /home/lleite/projects/aicockpit
python3 examples/basic/vault_example.py
```

**O que demonstra:**
- Integração Python com vault via subprocess
- Tratamento de erros robusto
- Geração de configurações JSON
- Mascaramento de secrets para logs
- Limpeza de recursos

**Quando usar:**
- Aplicações Python
- Scripts de automação em Python
- Data pipelines
- Aplicações web/microserviços

### 4. security/namespaced/security_namespaced.go
**Linguagem**: Go  
**Tipo**: NamespacedVault (Isolamento por Aplicação)

**Como executar:**
```bash
cd /home/lleite/projects/aicockpit
go run examples/security/namespaced/security_namespaced.go
```

**O que demonstra:**
- Isolamento de secrets por aplicação/pacote
- Apps só acessam seus próprios secrets
- Mesma chave pode ter valores diferentes em apps diferentes
- Detecção automática de namespace
- Prevenção de acesso cross-namespace

**Quando usar:**
- Múltiplas aplicações no mesmo ambiente
- Pacotes que precisam de isolamento de secrets
- Ambientes multi-tenant
- Quando uma app comprometida não deve afetar outras

### 5. security/command_handler/security_command_handler.go
**Linguagem**: Go  
**Tipo**: CommandHandler (Injeção Segura)

**Como executar:**
```bash
cd /home/lleite/projects/aicockpit
go run examples/security/command_handler/security_command_handler.go
```

**O que demonstra:**
- Aplicações não acessam secrets diretamente
- Injeção controlada de secrets em comandos
- Whitelist de comandos permitidos
- Auditoria automática de uso
- Sanitização de output para evitar vazamento
- Bloqueio de comandos perigosos

**Quando usar:**
- Operações críticas de segurança
- Execução de comandos com secrets sensíveis
- Ambientes com requisitos de auditoria estritos
- Quando preciso controlar exatamente como secrets são usados

## 🔄 Padrões de Uso Comuns

### Padrão 1: Inicialização de Aplicação
```go
// Go
v := vault.NewOSVault()
apiKey, _ := v.Get("api_key")
app.Init(apiKey)
```

```python
# Python
api_key = get_secret("api_key")
app.init(api_key)
```

```bash
# Shell
API_KEY=$(cockpit vault get api_key)
./app --api-key="$API_KEY"
```

### Padrão 2: Configuração de Ambiente
```bash
# Exportar secrets para variáveis de ambiente
export DB_URL=$(cockpit vault get database_url)
export API_KEY=$(cockpit vault get api_key)

# Iniciar aplicação
./my-application
```

### Padrão 3: Deploy Automatizado
```bash
#!/bin/bash
# deploy.sh

# Carregar secrets
DB_PASSWORD=$(cockpit vault get db_password)
API_KEY=$(cockpit vault get api_key)

# Deploy
./deploy-app --db-password="$DB_PASSWORD" --api-key="$API_KEY"
```

### Padrão 4: Configuração Dinâmica
```python
# Python
config = {
    "database": {
        "password": get_secret("db_password")
    },
    "api": {
        "key": get_secret("api_key")
    }
}
```

### Padrão 5: Pacotes Go (Recomendado)
```go
// Usar PackageVault helper para isolamento automático
import "github.com/lleitep3/aicockpit/internal/vault"

vault := vault.NewPackageVault("meu-pacote")
apiKey, _ := vault.Get("api-key")
```

**Benefícios do PackageVault:**
- ✅ Namespace automático baseado no nome do pacote
- ✅ Bypassa lock/unlock (namespace isola)
- ✅ Sem necessidade de master password
- ✅ Cross-namespace bloqueado automaticamente

**Documentação completa:** Veja `docs/vault-package-usage.md` para exemplos detalhados de uso seguro do vault por pacotes.

## 🛠️ Pré-requisitos

### Para executar os exemplos:
- ✅ AICockpit instalado e configurado
- ✅ Go 1.26+ (para exemplo em Go)
- ✅ Python 3 (para exemplo em Python)
- ✅ Bash (para exemplo em Shell)
- ✅ Permissões para acessar o keyring do sistema

### Verificar instalação:
```bash
# Verificar AICockpit
cockpit --version

# Verificar Go (opcional)
go version

# Verificar Python (opcional)
python3 --version

# Verificar keyring (Linux)
secret-tool --version
```

## 🧪 Testar os Exemplos

Todos os exemplos são **auto-contidos** e **auto-limpantes**:
- Criam seus próprios secrets de teste
- Demonstram o uso
- Removem os secrets de teste automaticamente
- Não deixam rastros

Você pode executá-los com segurança sem afetar seus secrets reais.

## 🔐 Segurança Avançada

### Problema de Segurança Atual

No modelo atual, qualquer aplicação com acesso ao vault pode ler todos os secrets:

```go
// ❌ PROBLEMA: Qualquer app pode acessar qualquer secret
v := vault.NewOSVault()
apiKey := v.Get("any_secret_key") // Acessa qualquer secret!
```

### Soluções Implementadas

#### 1. NamespacedVault (Isolamento por Aplicação)

Cada aplicação tem seu próprio namespace:

```go
// ✅ SOLUÇÃO: Apps só acessam seus próprios secrets
app1Vault := vault.NewNamespacedVault("payment-service")
apiKey := app1Vault.Get("api_key") // Só acessa "payment-service:api_key"

app2Vault := vault.NewNamespacedVault("user-service")
apiKey := app2Vault.Get("api_key") // Só acessa "user-service:api_key"
```

**Vantagens:**
- ✅ Isolamento completo entre aplicações
- ✅ Apps comprometidas não afetam outras apps
- ✅ Fácil de implementar e usar

#### 2. CommandHandler (Injeção Controlada)

Apps não acessam secrets diretamente; solicitam ao handler:

```go
// ✅ SOLUÇÃO: Apps solicitam execução com secrets injetados
handler := vault.NewCommandHandler()
output, err := handler.ExecuteWithSecret(
    "curl",
    []string{"-H", "Authorization: Bearer {{API_KEY}}", "https://api.example.com"},
    []vault.SecretInjection{{SecretKey: "api_key", Placeholder: "{{API_KEY}}"}},
)
```

**Vantagens:**
- ✅ Apps nunca acessam secrets diretamente
- ✅ Controle total sobre comandos permitidos
- ✅ Auditoria automática de uso
- ✅ Sanitização de output

### Comparação de Abordagens

| Abordagem | Segurança | Complexidade | Quando Usar |
|-----------|-----------|--------------|-------------|
| **Vault Atual** | ⭐⭐ | ⭐ | Desenvolvimento, testes |
| **NamespacedVault** | ⭐⭐⭐⭐ | ⭐⭐ | Múltiplas apps, isolamento básico |
| **CommandHandler** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Operações críticas, auditoria estrita |

### Recomendação de Uso

**Fase 1 (Imediato):** Implementar NamespacedVault
- Melhoria imediata de segurança
- Baixa complexidade
- Compatível com código existente

**Fase 2 (Curto Prazo):** Adicionar CommandHandler para operações críticas
- Deploy, operações de banco de dados
- Scripts sensíveis
- Ambientes de produção

**Fase 3 (Médio Prazo):** Implementar sistema completo de auditoria
- Logs detalhados de acesso
- Alertas de uso suspeito
- Integração com SIEM

Para mais detalhes, consulte [docs/vault-security-evolution.md](../docs/vault-security-evolution.md)

## 📚 Documentação Adicional

- **[Guia Completo do Vault](../docs/vault-guide.md)** - Documentação completa do sistema
- **[Arquitetura do Vault](../docs/architecture/05-vault-system.md)** - Detalhes técnicos da arquitetura
- **[Exemplos de Uso em Aplicações](../docs/vault-usage-examples.md)** - Mais exemplos e padrões avançados

## ⚠️ Notas de Segurança

1. **Nunca logue secrets** - Os exemplos sempre mascaram secrets para exibição
2. **Tratamento de erros** - Sempre trate erros ao recuperar secrets
3. **Permissões** - Apenas aplicações autorizadas devem acessar o vault
4. **Ambiente de teste** - Use secrets de teste durante desenvolvimento
5. **Rotação** - Implemente rotação regular de secrets em produção

## 🤝 Contribuir

Quer adicionar mais exemplos?
- Fork o projeto
- Adicione seu exemplo na pasta `examples/`
- Siga os padrões existentes (auto-contido, auto-limpante)
- Adicione documentação neste README
- Abra um PR

## 📞 Suporte

Para dúvidas ou problemas:
- Verifique a documentação em `docs/`
- Abra uma issue no GitHub
- Consulte o guia de troubleshooting