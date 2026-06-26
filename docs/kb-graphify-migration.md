# Migração do kb-graphify para Vault Seguro

## Problema Atual

O kb-graphify usa o vault de forma insegura:

```bash
# kb-graphify/bin/validate (ATUAL - INSEGURO)
PROVIDER=$(cockpit vault get kb-graphify.provider 2>/dev/null || true)
API_KEY=$(cockpit vault get kb-graphify.api-key 2>/dev/null || true)
```

**Problemas:**
- ❌ Pode acessar QUALQUER secret do vault
- ❌ Não há isolamento por namespace
- ❌ Não há verificação de quem está chamando
- ❌ Não há criptografia adicional

## Solução 1: Uso de Namespaces (Imediato)

### Atualização dos Scripts

```bash
# kb-graphify/bin/validate (NOVO - SEGURO)
NAMESPACE="kb-graphify"
PROVIDER=$(cockpit vault get --namespace $NAMESPACE provider 2>/dev/null || true)
API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key 2>/dev/null || true)

# kb-graphify/bin/configure (NOVO - SEGURO)
NAMESPACE="kb-graphify"
cockpit vault set --namespace $NAMESPACE provider --value "$PROVIDER"
cockpit vault set --namespace $NAMESPACE api-key --value "$API_KEY"
```

### Implementação na CLI

Adicionar flag `--namespace` ao comando vault:

```go
// cmd/vault.go
var namespaceFlag string

func NewVaultGetCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Retrieve a secret",
		Long:  "Retrieve a secret from the vault and print it to standard output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			namespace := namespaceFlag

			// Se não especificado, detectar automaticamente
			if namespace == "" {
				namespace = detectCallerNamespace()
			}

			// Validar caller
			if !validateCallerAccess(namespace) {
				return fmt.Errorf("unauthorized access to namespace: %s", namespace)
			}

			// Usar NamespacedVault
			v := vault.NewNamespacedVault(namespace)
			value, err := v.Get(key)
			if err != nil {
				return fmt.Errorf("failed to retrieve secret: %w", err)
			}

			fmt.Fprint(cmd.OutOrStdout(), value)
			return nil
		},
	}

	getCmd.Flags().StringVar(&namespaceFlag, "namespace", "", "Vault namespace (default: auto-detect)")
	return getCmd
}

func detectCallerNamespace() string {
	// Detectar baseado no processo que está chamando
	callerPID := os.Getppid()
	exePath := getProcessPath(callerPID)
	return extractAppIDFromPath(exePath)
}

func validateCallerAccess(requestedNamespace string) bool {
	// Verificar se o caller tem permissão para acessar o namespace
	callerNamespace := detectCallerNamespace()
	
	// Em dev mode, permitir qualquer acesso
	if os.Getenv("COCKPIT_DEV_MODE") == "true" {
		return true
	}
	
	// Verificar se o namespace do caller corresponde ao solicitado
	return callerNamespace == requestedNamespace
}
```

## Solução 2: Integração com SecureVault (Recomendado)

### Criar Wrapper em Go para kb-graphify

```go
// kb-graphify/internal/vault/client.go
package vault

import (
	"fmt"
	"os"
	
	"github.com/lleitep3/aicockpit/internal/vault"
)

type VaultClient struct {
	secureVault *vault.SecureVault
}

func NewVaultClient() (*VaultClient, error) {
	// Usar "kb-graphify" como namespace fixo
	sv, err := vault.NewSecureVault("kb-graphify")
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}
	
	return &VaultClient{
		secureVault: sv,
	}, nil
}

func (vc *VaultClient) GetProvider() (string, error) {
	provider, err := vc.secureVault.Get("provider")
	if err != nil {
		return "", fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

func (vc *VaultClient) GetAPIKey() (string, error) {
	apiKey, err := vc.secureVault.Get("api-key")
	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}
	return apiKey, nil
}

func (vc *VaultClient) SetProvider(provider string) error {
	return vc.secureVault.Set("provider", provider)
}

func (vc *VaultClient) SetAPIKey(apiKey string) error {
	return vc.secureVault.Set("api-key", apiKey)
}
```

### Atualizar Scripts para Usar o Cliente Go

```bash
# kb-graphify/bin/validate (NOVO - COM CLIENTE GO)
# Em vez de chamar cockpit vault diretamente, chamar um pequeno utilitário Go
PROVIDER=$(kb-graphify-vault get provider)
API_KEY=$(kb-graphify-vault get api-key)
```

### Criar Utilitário CLI em Go

```go
// cmd/kb-graphify-vault/main.go
package main

import (
	"fmt"
	"os"
	
	"github.com/lleitep3/aicockpit/kb-graphify/internal/vault"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: kb-graphify-vault <get|set> <key> [value]")
		os.Exit(1)
	}
	
	command := os.Args[1]
	
	client, err := vault.NewVaultClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	switch command {
	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: kb-graphify-vault get <key>")
			os.Exit(1)
		}
		key := os.Args[2]
		
		var value string
		var err error
		
		switch key {
		case "provider":
			value, err = client.GetProvider()
		case "api-key":
			value, err = client.GetAPIKey()
		default:
			fmt.Printf("Unknown key: %s\n", key)
			os.Exit(1)
		}
		
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Print(value)
		
	case "set":
		if len(os.Args) < 4 {
			fmt.Println("Usage: kb-graphify-vault set <key> <value>")
			os.Exit(1)
		}
		key := os.Args[2]
		value := os.Args[3]
		
		switch key {
		case "provider":
			err = client.SetProvider(value)
		case "api-key":
			err = client.SetAPIKey(value)
		default:
			fmt.Printf("Unknown key: %s\n", key)
			os.Exit(1)
		}
		
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
```

## Solução 3: Integração Direta no Código Go (Melhor)

Se o kb-graphify tiver código Go, integrar diretamente:

```go
// kb-graphify/internal/config/vault.go
package config

import (
	"fmt"
	
	"github.com/lleitep3/aicockpit/internal/vault"
)

type VaultConfig struct {
	provider string
	apiKey   string
}

func LoadFromVault() (*VaultConfig, error) {
	sv, err := vault.NewSecureVault("kb-graphify")
	if err != nil {
		return nil, fmt.Errorf("failed to create secure vault: %w", err)
	}
	
	provider, err := sv.Get("provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	apiKey, err := sv.Get("api-key")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	
	return &VaultConfig{
		provider: provider,
		apiKey:   apiKey,
	}, nil
}

func (vc *VaultConfig) GetProvider() string {
	return vc.provider
}

func (vc *VaultConfig) GetAPIKey() string {
	return vc.apiKey
}
```

## Comparação das Soluções

| Solução | Segurança | Complexidade | Compatibilidade | Tempo de Implementação |
|---------|-----------|--------------|-----------------|----------------------|
| **Namespace CLI** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | Imediato |
| **Cliente Go** | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | Curto prazo |
| **Integração Direta** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | Médio prazo |

## Plano de Migração Recomendado

### Fase 1: Imediato (1-2 horas)
1. Adicionar flag `--namespace` à CLI do vault
2. Implementar `detectCallerNamespace()` e `validateCallerAccess()`
3. Atualizar scripts do kb-graphify para usar `--namespace kb-graphify`

### Fase 2: Curto Prazo (1-2 dias)
1. Criar cliente Go simples para kb-graphify
2. Compilar como binário estático `kb-graphify-vault`
3. Atualizar scripts para usar o cliente em vez da CLI direta

### Fase 3: Médio Prazo (1 semana)
1. Integrar SecureVault diretamente no código Go do kb-graphify
2. Remover dependência de scripts shell
3. Implementar tratamento de erros robusto

## Exemplo de Script Atualizado

```bash
#!/bin/bash
# kb-graphify/bin/validate (VERSÃO FINAL)

set -e

# Carregar configuração do vault de forma segura
NAMESPACE="kb-graphify"

# Método 1: Usar CLI com namespace (se disponível)
if cockpit vault get --help | grep -q -- --namespace; then
    PROVIDER=$(cockpit vault get --namespace $NAMESPACE provider 2>/dev/null || echo "")
    API_KEY=$(cockpit vault get --namespace $NAMESPACE api-key 2>/dev/null || echo "")
else
    # Método 2: Usar cliente Go dedicado (fallback)
    PROVIDER=$(kb-graphify-vault get provider 2>/dev/null || echo "")
    API_KEY=$(kb-graphify-vault get api-key 2>/dev/null || echo "")
fi

# Validar que obtivemos as credenciais
if [ -z "$PROVIDER" ] || [ -z "$API_KEY" ]; then
    echo "Error: Vault credentials not found. Run 'kb-graphify configure' first."
    exit 1
fi

# Continuar com a validação...
echo "Provider: $PROVIDER"
echo "API Key: ${API_KEY:0:10}..."
```

## Benefícios da Migração

✅ **Segurança**: kb-graphify só acessa seus próprios secrets
✅ **Isolamento**: Namespace "kb-graphify" separado
✅ **Auditoria**: Todos os acessos são logados
✅ **Criptografia**: Secrets são criptografados com SecureVault
✅ **Verificação**: Identidade do caller é validada
✅ **Compatibilidade**: Funciona com código existente