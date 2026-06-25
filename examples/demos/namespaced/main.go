package main

import (
	"fmt"
	"log"

	"github.com/lleitep3/aicockpit/internal/vault"
)

// Exemplo demonstrando o uso de NamespacedVault para isolamento de secrets
// Cada aplicação só acessa seus próprios secrets
func main() {
	fmt.Println("=== Exemplo de NamespacedVault (Isolamento por Aplicação) ===")
	fmt.Println()

	// Simular diferentes aplicações
	app1Vault := vault.NewNamespacedVault("payment-service")
	app2Vault := vault.NewNamespacedVault("user-service")
	app3Vault := vault.NewNamespacedVault("notification-service")

	fmt.Println("1. Criando secrets para diferentes aplicações...")

	// App 1: Payment Service
	err := app1Vault.Set("stripe_api_key", "sk_live_payment_1234567890")
	if err != nil {
		log.Fatalf("Erro ao criar secret payment-service: %v", err)
	}
	fmt.Println("   ✓ payment-service: stripe_api_key")

	err = app1Vault.Set("db_password", "payment_db_secret_123")
	if err != nil {
		log.Fatalf("Erro ao criar secret payment-service: %v", err)
	}
	fmt.Println("   ✓ payment-service: db_password")

	// App 2: User Service
	err = app2Vault.Set("jwt_secret", "user_jwt_super_secret_456")
	if err != nil {
		log.Fatalf("Erro ao criar secret user-service: %v", err)
	}
	fmt.Println("   ✓ user-service: jwt_secret")

	err = app2Vault.Set("db_password", "user_db_secret_789")
	if err != nil {
		log.Fatalf("Erro ao criar secret user-service: %v", err)
	}
	fmt.Println("   ✓ user-service: db_password")

	// App 3: Notification Service
	err = app3Vault.Set("sendgrid_api_key", "SG.notification_9876543210")
	if err != nil {
		log.Fatalf("Erro ao criar secret notification-service: %v", err)
	}
	fmt.Println("   ✓ notification-service: sendgrid_api_key")

	fmt.Println("\n2. Testando isolamento entre aplicações...")

	// Payment Service acessa seus próprios secrets
	stripeKey, err := app1Vault.Get("stripe_api_key")
	if err != nil {
		log.Printf("Erro ao recuperar stripe_api_key: %v", err)
	} else {
		fmt.Printf("   ✓ payment-service acessou stripe_api_key: %s...%s\n",
			stripeKey[:10], stripeKey[len(stripeKey)-4:])
	}

	// Payment Service NÃO deve conseguir acessar secrets de outras apps
	_, err = app1Vault.Get("jwt_secret")
	if err != nil {
		fmt.Println("   ✓ payment-service BLOQUEADO ao acessar user-service:jwt_secret")
	} else {
		fmt.Println("   ✗ ERRO: payment-service CONSEGUIU acessar secret de outra app!")
	}

	_, err = app1Vault.Get("sendgrid_api_key")
	if err != nil {
		fmt.Println("   ✓ payment-service BLOQUEADO ao acessar notification-service:sendgrid_api_key")
	} else {
		fmt.Println("   ✗ ERRO: payment-service CONSEGUIU acessar secret de outra app!")
	}

	fmt.Println("\n3. Mesma chave, diferentes valores (isolamento completo)...")

	// Todas as apps têm "db_password" mas com valores diferentes
	paymentDB, _ := app1Vault.Get("db_password")
	userDB, _ := app2Vault.Get("db_password")

	fmt.Printf("   payment-service:db_password = %s\n", maskSecret(paymentDB))
	fmt.Printf("   user-service:db_password    = %s\n", maskSecret(userDB))

	if paymentDB == userDB {
		fmt.Println("   ✗ ERRO: Os secrets deveriam ser diferentes!")
	} else {
		fmt.Println("   ✓ Isolamento funcionou corretamente")
	}

	fmt.Println("\n4. Detecção automática de namespace...")

	// Simular detecção baseada em variável de ambiente
	// Nota: Em Go, não podemos setar env vars para o processo atual,
	// então este é apenas um exemplo conceitual

	fmt.Println("   Namespace atual (processo):", vault.NewNamespacedVaultFromProcess().GetNamespace())
	fmt.Println("   Namespace atual (ambiente):", vault.NewNamespacedVaultFromEnv().GetNamespace())

	fmt.Println("\n5. Limpando secrets de teste...")

	// Cleanup
	app1Vault.Delete("stripe_api_key")
	app1Vault.Delete("db_password")
	app2Vault.Delete("jwt_secret")
	app2Vault.Delete("db_password")
	app3Vault.Delete("sendgrid_api_key")

	fmt.Println("   ✓ Todos os secrets removidos")

	fmt.Println("\n=== Vantagens do NamespacedVault ===")
	fmt.Println("✓ Isolamento completo entre aplicações")
	fmt.Println("✓ Mesma chave pode ter valores diferentes em apps diferentes")
	fmt.Println("✓ Apps comprometidas não afetam secrets de outras apps")
	fmt.Println("✓ Detecção automática de namespace")
	fmt.Println("✓ Compatível com keyring existente")
	fmt.Println("\n=== Exemplo concluído ===")
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
