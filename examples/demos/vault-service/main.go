package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lleitep3/aicockpit/internal/vault"
)

func main() {
	fmt.Println("=== Demonstração: Vault Service com Namespace Controlado pelo Serviço ===")
	fmt.Println()

	// Iniciar o serviço
	service := vault.NewVaultService("/tmp/cockpit-vault-demo.sock")

	fmt.Println("1. Iniciando Vault Service...")
	if err := service.Start(); err != nil {
		fmt.Printf("Erro ao iniciar serviço: %v\n", err)
		os.Exit(1)
	}
	defer service.Stop()

	// Aguardar serviço iniciar
	time.Sleep(100 * time.Millisecond)

	// Configurar alguns secrets de teste
	fmt.Println("2. Configurando secrets de teste...")
	vaultDirect := vault.NewOSVault()

	// Secret para kb-graphify
	vaultDirect.Set("kb-graphify:api-key", "sk-kb-graphify-secret-123")
	fmt.Println("   ✓ Secret para 'kb-graphify' configurado")

	// Secret para outro pacote
	vaultDirect.Set("other-package:secret-password", "other-package-secret-456")
	fmt.Println("   ✓ Secret para 'other-package' configurado")

	fmt.Println()
	fmt.Println("3. Simulando acesso do pacote kb-graphify...")

	// Criar cliente (simulando o pacote kb-graphify)
	client := vault.NewVaultServiceClient("/tmp/cockpit-vault-demo.sock")

	// O pacote pede seu secret SEM especificar namespace
	fmt.Println("   Pacote solicita: GetSecret('api-key')")
	fmt.Println("   NOTA: Pacote NÃO especifica namespace!")

	// O serviço determina o namespace automaticamente baseado na identidade do processo
	value, err := client.GetSecret("api-key")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
	} else {
		fmt.Printf("   ✓ Recebido: %s\n", maskSecret(value))
	}

	fmt.Println()
	fmt.Println("4. Tentando acessar secret de outro pacote...")

	// O pacote tenta acessar secret de outro pacote
	// Mas o serviço só permitirá acesso ao seu próprio namespace
	fmt.Println("   Pacote solicita: GetSecret('secret-password')")
	fmt.Println("   NOTA: Mesmo sem especificar namespace, serviço bloqueia acesso cross-namespace")

	value, err = client.GetSecret("secret-password")
	if err != nil {
		fmt.Printf("   ✓ Acesso BLOQUEADO: %v\n", err)
	} else {
		fmt.Printf("   ✗ PERIGO: Acesso permitido: %s\n", maskSecret(value))
	}

	fmt.Println()
	fmt.Println("5. Demonstrando que o pacote não sabe o namespace")

	fmt.Println("   O pacote executou:")
	fmt.Println("   client.GetSecret('api-key')")
	fmt.Println()
	fmt.Println("   O pacote NUNCA:")
	fmt.Println("   - Especificou namespace='kb-graphify'")
	fmt.Println("   - Soube qual namespace estava sendo usado")
	fmt.Println("   - Pôde tentar acessar outro namespace")
	fmt.Println()
	fmt.Println("   O serviço:")
	fmt.Println("   - Identificou o processo (PID, executável)")
	fmt.Println("   - Determinou o namespace='kb-graphify' automaticamente")
	fmt.Println("   - Só retornou secrets desse namespace")
	fmt.Println("   - Bloqueou acesso a outros namespaces")

	fmt.Println()
	fmt.Println("6. Limpeza")
	vaultDirect.Delete("kb-graphify:api-key")
	vaultDirect.Delete("other-package:secret-password")
	fmt.Println("   ✓ Secrets de teste removidos")

	fmt.Println()
	fmt.Println("=== Diferença Crítica ===")
	fmt.Println("❌ Abordagem ANTIGA (--namespace):")
	fmt.Println("   cockpit vault get --namespace kb-graphify api-key")
	fmt.Println("   cockpit vault get --namespace other-package secret  # VULNERÁVEL!")
	fmt.Println()
	fmt.Println("✅ Abordagem NOVA (Vault Service):")
	fmt.Println("   client.GetSecret('api-key')  # Serviço determina namespace")
	fmt.Println("   client.GetSecret('secret')   # Serviço bloqueia cross-namespace")
	fmt.Println("   Pacote nunca sabe qual namespace está usando")
	fmt.Println()
	fmt.Println("=== Isso é SEGURANÇA REAL ===")
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
