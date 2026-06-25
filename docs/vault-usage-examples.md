# Como Aplicações Usam Secrets do Vault AICockpit

## Visão Geral

Existem várias formas de uma aplicação acessar secrets armazenados no vault do AICockpit:

1. **Integração direta via código Go** (usando o pacote internal/vault)
2. **Via CLI** (chamando `cockpit vault get`)
3. **Via variáveis de ambiente** (exportando para o processo)
4. **Em scripts de deploy/CI/CD**
5. **Em configurações de aplicações**

## 1. Integração Direta via Código Go

### Exemplo Básico

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/lleitep3/aicockpit/internal/vault"
)

func main() {
    // Criar instância do vault
    v := vault.NewOSVault()
    
    // Recuperar secret
    apiKey, err := v.Get("openai_api_key")
    if err != nil {
        log.Fatalf("Falha ao recuperar API key: %v", err)
    }
    
    // Usar o secret
    fmt.Printf("API Key recuperada: %s...%s\n", 
        apiKey[:10], apiKey[len(apiKey)-4:])
    
    // Fazer chamada à API usando o secret
    // callOpenAI(apiKey)
}
```

### Exemplo com Configuração de Aplicação

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/lleitep3/aicockpit/internal/vault"
)

type Config struct {
    DatabaseURL string
    APIKey      string
    SecretKey   string
}

func LoadConfigFromVault() (*Config, error) {
    v := vault.NewOSVault()
    
    config := &Config{}
    
    // Recuperar múltiplos secrets
    var err error
    config.DatabaseURL, err = v.Get("database_url")
    if err != nil {
        return nil, fmt.Errorf("database_url: %w", err)
    }
    
    config.APIKey, err = v.Get("api_key")
    if err != nil {
        return nil, fmt.Errorf("api_key: %w", err)
    }
    
    config.SecretKey, err = v.Get("secret_key")
    if err != nil {
        return nil, fmt.Errorf("secret_key: %w", err)
    }
    
    return config, nil
}

func main() {
    config, err := LoadConfigFromVault()
    if err != nil {
        log.Fatalf("Erro ao carregar configuração: %v", err)
    }
    
    fmt.Printf("Configuração carregada com sucesso\n")
    fmt.Printf("Database URL: %s\n", maskSecret(config.DatabaseURL))
    // Usar configuração...
}

func maskSecret(secret string) string {
    if len(secret) <= 8 {
        return "***"
    }
    return secret[:4] + "..." + secret[len(secret)-4:]
}
```

### Exemplo com Fallback

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/lleitep3/aicockpit/internal/vault"
)

// GetSecret com fallback para variável de ambiente
func GetSecret(key string) (string, error) {
    // Tentar obter do vault primeiro
    v := vault.NewOSVault()
    secret, err := v.Get(key)
    if err == nil {
        return secret, nil
    }
    
    // Fallback para variável de ambiente
    envKey := fmt.Sprintf("%s_%s", "COCKPIT", key)
    secret = os.Getenv(envKey)
    if secret != "" {
        return secret, nil
    }
    
    return "", fmt.Errorf("secret não encontrado no vault ou variável de ambiente")
}

func main() {
    apiKey, err := GetSecret("openai_api_key")
    if err != nil {
        fmt.Printf("Aviso: %v\n", err)
        apiKey = "default_key" // Fallback final
    }
    
    fmt.Printf("Usando API key: %s...\n", apiKey[:10])
}
```

## 2. Via CLI (Cross-Language)

### Shell Script

```bash
#!/bin/bash

# Recuperar secret do vault
DB_PASSWORD=$(cockpit vault get database_password)

# Usar em comando
psql -h localhost -U user -d mydb -c "SELECT 1" -P password="$DB_PASSWORD"

# Ou passar como argumento
./my_app --db-password="$DB_PASSWORD"
```

### Python

```python
import subprocess
import os

def get_secret(key: str) -> str:
    """Recupera secret do vault AICockpit"""
    try:
        result = subprocess.run(
            ['cockpit', 'vault', 'get', key],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        raise Exception(f"Erro ao recuperar secret '{key}': {e}")

# Uso
api_key = get_secret("openai_api_key")
print(f"API Key: {api_key[:10]}...")

# Usar em requests
import requests
headers = {"Authorization": f"Bearer {api_key}"}
response = requests.get("https://api.example.com/endpoint", headers=headers)
```

### Node.js

```javascript
const { execSync } = require('child_process');

function getSecret(key) {
    try {
        const secret = execSync(`cockpit vault get ${key}`, {
            encoding: 'utf-8'
        }).trim();
        return secret;
    } catch (error) {
        throw new Error(`Erro ao recuperar secret '${key}': ${error.message}`);
    }
}

// Uso
const apiKey = getSecret('openai_api_key');
console.log(`API Key: ${apiKey.substring(0, 10)}...`);

// Usar em fetch
fetch('https://api.example.com/endpoint', {
    headers: {
        'Authorization': `Bearer ${apiKey}`
    }
});
```

## 3. Via Variáveis de Ambiente

### Script de Inicialização

```bash
#!/bin/bash
# start_app.sh - Script para iniciar aplicação com secrets do vault

set -e

# Carregar secrets do vault para variáveis de ambiente
export DATABASE_URL=$(cockpit vault get database_url)
export API_KEY=$(cockpit vault get api_key)
export SECRET_KEY=$(cockpit vault get secret_key)

# Iniciar aplicação
./my-application
```

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    image: my-application
    env_file:
      - .env.vault  # Arquivo gerado pelo vault
    command: ["./start-app"]

# Gerar .env.vault antes do docker-compose up
# cockpit vault get database_url > .env.vault
# cockpit vault get api_key >> .env.vault
```

### Kubernetes ConfigMap/Secret

```bash
#!/bin/bash
# Criar Kubernetes secret a partir do vault

DB_PASSWORD=$(cockpit vault get database_password)
kubectl create secret generic db-secret \
  --from-literal=password="$DB_PASSWORD"
```

## 4. Em Scripts de Deploy/CI/CD

### GitHub Actions

```yaml
name: Deploy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Install AICockpit
        run: |
          go install github.com/lleitep3/aicockpit@latest
          export PATH=$PATH:$(go env GOPATH)/bin
          
      - name: Load secrets from vault
        run: |
          export API_KEY=$(cockpit vault get api_key)
          export DB_PASSWORD=$(cockpit vault get database_password)
          
      - name: Deploy application
        env:
          API_KEY: ${{ steps.secrets.outputs.API_KEY }}
          DB_PASSWORD: ${{ steps.secrets.outputs.DB_PASSWORD }}
        run: |
          ./deploy.sh
```

### Script de Deploy

```bash
#!/bin/bash
# deploy.sh

echo "Carregando secrets do vault..."

# Recuperar secrets
API_KEY=$(cockpit vault get api_key)
DB_PASSWORD=$(cockpit vault get database_password)
DEPLOY_KEY=$(cockpit vault get deploy_key)

# Validar que os secrets existem
if [ -z "$API_KEY" ]; then
    echo "Erro: API key não encontrada no vault"
    exit 1
fi

echo "Secrets carregados com sucesso"
echo "Iniciando deploy..."

# Deploy com secrets
./deploy-application \
  --api-key="$API_KEY" \
  --db-password="$DB_PASSWORD" \
  --deploy-key="$DEPLOY_KEY"
```

## 5. Em Configurações de Aplicações

### Arquivo de Configuração Dinâmico

```bash
#!/bin/bash
# generate_config.sh

# Template de configuração
cat > config.template.json <<EOF
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "admin",
    "password": "{{DB_PASSWORD}}"
  },
  "api": {
    "key": "{{API_KEY}}",
    "endpoint": "https://api.example.com"
  }
}
EOF

# Substituir placeholders com secrets do vault
DB_PASSWORD=$(cockpit vault get database_password)
API_KEY=$(cockpit vault get api_key)

sed "s/{{DB_PASSWORD}}/$DB_PASSWORD/g" config.template.json | \
sed "s/{{API_KEY}}/$API_KEY/g" > config.json

echo "Configuração gerada: config.json"
```

### Nginx Configuration

```bash
#!/bin/bash
# Gerar configuração do Nginx com secrets

API_KEY=$(cockpit vault get api_key)

cat > /etc/nginx/conf.d/api.conf <<EOF
server {
    listen 80;
    
    location /api {
        proxy_pass http://backend;
        proxy_set_header X-API-Key $API_KEY;
    }
}
EOF

nginx -t && nginx -s reload
```

## 6. Padrões Avançados

### Cache de Secrets em Memória

```go
package main

import (
    "sync"
    "time"
    
    "github.com/lleitep3/aicockpit/internal/vault"
)

type SecretCache struct {
    vault     *vault.OSVault
    cache     map[string]string
    mutex     sync.RWMutex
    ttl       time.Duration
    lastUpdate map[string]time.Time
}

func NewSecretCache(ttl time.Duration) *SecretCache {
    return &SecretCache{
        vault:     vault.NewOSVault(),
        cache:     make(map[string]string),
        ttl:       ttl,
        lastUpdate: make(map[string]time.Time),
    }
}

func (sc *SecretCache) Get(key string) (string, error) {
    sc.mutex.RLock()
    
    // Verificar se está em cache e válido
    if secret, exists := sc.cache[key]; exists {
        if time.Since(sc.lastUpdate[key]) < sc.ttl {
            sc.mutex.RUnlock()
            return secret, nil
        }
    }
    sc.mutex.RUnlock()
    
    // Recuperar do vault
    sc.mutex.Lock()
    defer sc.mutex.Unlock()
    
    secret, err := sc.vault.Get(key)
    if err != nil {
        return "", err
    }
    
    // Atualizar cache
    sc.cache[key] = secret
    sc.lastUpdate[key] = time.Now()
    
    return secret, nil
}
```

### Rotação Automática de Secrets

```go
package main

import (
    "log"
    "time"
    
    "github.com/lleitep3/aicockpit/internal/vault"
)

type SecretRotator struct {
    vault        *vault.OSVault
    rotationKeys map[string]time.Duration
}

func NewSecretRotator() *SecretRotator {
    return &SecretRotator{
        vault:        vault.NewOSVault(),
        rotationKeys: make(map[string]time.Duration),
    }
}

func (sr *SecretRotator) AddKey(key string, interval time.Duration) {
    sr.rotationKeys[key] = interval
}

func (sr *SecretRotator) Start() {
    for key, interval := range sr.rotationKeys {
        go sr.rotateKey(key, interval)
    }
}

func (sr *SecretRotator) rotateKey(key string, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        // Gerar nova chave
        newSecret := generateNewSecret()
        
        // Atualizar no vault
        if err := sr.vault.Set(key, newSecret); err != nil {
            log.Printf("Erro ao rotacionar chave %s: %v", key, err)
            continue
        }
        
        // Notificar aplicação para usar nova chave
        notifyApplication(key, newSecret)
        
        log.Printf("Chave %s rotacionada com sucesso", key)
    }
}

func generateNewSecret() string {
    // Lógica para gerar novo secret
    return "new_secret_" + time.Now().Format("20060102150405")
}

func notifyApplication(key, newSecret string) {
    // Notificar aplicação sobre nova chave
    // Pode ser via HTTP, message queue, etc.
}
```

## Boas Práticas

1. **Tratamento de Erros**: Sempre trate erros ao recuperar secrets
2. **Fallbacks**: Tenha fallbacks para desenvolvimento/testes
3. **Cache**: Considere cache para evitar chamadas frequentes ao keyring
4. **Rotação**: Implemente rotação regular de secrets
5. **Logging**: Não logue secrets, apenas sucesso/fracasso das operações
6. **Permissões**: Apenas aplicações autorizadas devem acessar o vault
7. **Validação**: Valide que os secrets existem antes de usar

## Conclusão

O vault do AICockpit pode ser integrado de várias formas:
- **Diretamente em Go** para aplicações que usam o código do AICockpit
- **Via CLI** para qualquer linguagem/ambiente
- **Via variáveis de ambiente** para compatibilidade com qualquer aplicação
- **Em scripts de CI/CD** para automação de deploy

A escolha depende do seu ambiente, linguagem e requisitos de segurança.