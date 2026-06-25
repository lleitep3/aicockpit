package main

import (
	"fmt"
	"log"

	"github.com/lleitep3/aicockpit/internal/vault"
)

// Exemplo de como uma aplicação Go usaria o vault com PackageVault
// PackageVault fornece isolamento automático via namespace
func main() {
	fmt.Println("=== Exemplo de Uso do Vault AICockpit com PackageVault ===")

	// Criar vault para este pacote (namespace automático)
	// O nome do pacote é sanitizado automaticamente
	v := vault.NewPackageVault("meu-app-exemplo")

	// Exemplo 1: Recuperar um secret existente
	fmt.Println("1. Recuperando secret do vault...")
	apiKey, err := v.Get("test_api_key")
	if err != nil {
		log.Printf("Aviso: test_api_key não encontrado (isso é normal se não existir): %v", err)
		fmt.Println("   Vamos criar um secret de teste...")

		// Criar secret de teste
		err = v.Set("test_api_key", "sk-test-1234567890abcdef")
		if err != nil {
			log.Fatalf("Erro ao criar secret de teste: %v", err)
		}
		fmt.Println("   ✓ Secret de teste criado no namespace 'meu-app-exemplo'")

		// Tentar recuperar novamente
		apiKey, err = v.Get("test_api_key")
		if err != nil {
			log.Fatalf("Erro ao recuperar secret: %v", err)
		}
	}

	fmt.Printf("   ✓ API Key recuperada: %s...%s\n", apiKey[:10], apiKey[len(apiKey)-4:])
	fmt.Println("   ✓ (Namespace: meu-app-exemplo)")

	// Exemplo 2: Usar o secret em uma "chamada de API" simulada
	fmt.Println("\n2. Simulando chamada de API com o secret...")
	callAPISimulation(apiKey)

	// Exemplo 3: Recuperar múltiplos secrets
	fmt.Println("\n3. Recuperando múltiplos secrets...")
	secrets := []string{"test_api_key", "database_url", "secret_key"}

	for _, key := range secrets {
		value, err := v.Get(key)
		if err != nil {
			fmt.Printf("   ✗ %s: não encontrado\n", key)
		} else {
			masked := maskSecret(value)
			fmt.Printf("   ✓ %s: %s\n", key, masked)
		}
	}

	// Limpar: remover secret de teste
	fmt.Println("\n4. Limpando secret de teste...")
	err = v.Remove("test_api_key")
	if err != nil {
		log.Printf("Aviso: Erro ao remover secret de teste: %v", err)
	} else {
		fmt.Println("   ✓ Secret de teste removido do namespace 'meu-app-exemplo'")
	}

	fmt.Println("\n=== Benefícios do PackageVault ===")
	fmt.Println("✓ Namespace automático baseado no nome do pacote")
	fmt.Println("✓ Isolamento entre pacotes (cross-namespace bloqueado)")
	fmt.Println("✓ Bypassa lock/unlock (namespace fornece isolamento)")
	fmt.Println("✓ Sem necessidade de master password para pacotes")
	fmt.Println("\n=== Exemplo concluído ===")
}

// Simulação de chamada de API
func callAPISimulation(apiKey string) {
	// Em uma aplicação real, aqui seria a chamada real à API
	fmt.Printf("   Chamando API com key: %s...%s\n", apiKey[:10], apiKey[len(apiKey)-4:])
	fmt.Println("   ✓ Resposta da API simulada: {\"status\": \"success\", \"data\": [...]}")
}

// Mascara secret para exibição segura
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
