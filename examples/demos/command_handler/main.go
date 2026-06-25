package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/lleitep3/aicockpit/internal/vault"
)

// Exemplo demonstrando o uso de CommandHandler para injeção segura de secrets
// A aplicação NÃO acessa os secrets diretamente, solicita ao handler que execute comandos
func main() {
	fmt.Println("=== Exemplo de CommandHandler (Injeção Segura de Secrets) ===")
	fmt.Println()

	// Criar handler
	handler := vault.NewCommandHandler()

	// Setup secrets de teste
	fmt.Println("1. Configurando secrets de teste...")
	testSecrets := map[string]string{
		"api_key":          "sk-test-1234567890abcdef",
		"database_url":     "postgresql://user:pass@localhost/mydb",
		"deploy_token":     "deploy_secret_token_xyz789",
		"service_password": "service_pass_123456",
	}

	for key, value := range testSecrets {
		// Usar OSVault diretamente para setup
		osVault := vault.NewOSVault()
		err := osVault.Set(key, value)
		if err != nil {
			log.Fatalf("Erro ao configurar secret %s: %v", key, err)
		}
		fmt.Printf("   ✓ %s configurado\n", key)
	}

	// Cleanup
	defer func() {
		fmt.Println("\n6. Limpando secrets de teste...")
		osVault := vault.NewOSVault()
		for key := range testSecrets {
			osVault.Delete(key)
		}
		fmt.Println("   ✓ Todos os secrets removidos")
	}()

	fmt.Println("\n2. Executando comando com injeção de secret...")

	// Exemplo 1: Simular chamada de API com curl
	output, err := handler.ExecuteWithSecret(
		"echo",
		[]string{"curl -H 'Authorization: Bearer {{API_KEY}}' https://api.example.com/endpoint"},
		[]vault.SecretInjection{
			{SecretKey: "api_key", Placeholder: "{{API_KEY}}"},
		},
	)

	if err != nil {
		log.Printf("Erro ao executar comando: %v", err)
	} else {
		fmt.Println("   ✓ Comando executado com sucesso")
		fmt.Printf("   Output: %s\n", strings.TrimSpace(output))

		// Verificar que o secret foi injetado
		if strings.Contains(output, "sk-test-1234567890abcdef") {
			fmt.Println("   ✓ Secret injetado corretamente no comando")
		}
	}

	fmt.Println("\n3. Testando sanitarização de output...")

	// Exemplo com sanitarização (remove secrets do output)
	sanitizedOutput, err := handler.ExecuteWithSecretForOutput(
		"echo",
		[]string{"The database URL is postgresql://user:pass@localhost/mydb and API key is sk-test-1234567890abcdef"},
		[]vault.SecretInjection{
			{SecretKey: "database_url", Placeholder: "postgresql://user:pass@localhost/mydb"},
			{SecretKey: "api_key", Placeholder: "sk-test-1234567890abcdef"},
		},
	)

	if err != nil {
		log.Printf("Erro ao executar comando: %v", err)
	} else {
		fmt.Println("   ✓ Output sanitizado gerado")
		fmt.Printf("   Output: %s\n", strings.TrimSpace(sanitizedOutput))

		// Verificar que os secrets foram removidos
		if strings.Contains(sanitizedOutput, "sk-test-1234567890abcdef") {
			fmt.Println("   ✗ AVISO: Secret ainda presente no output!")
		} else if strings.Contains(sanitizedOutput, "***REDACTED***") {
			fmt.Println("   ✓ Secrets removidos do output (redacted)")
		}
	}

	fmt.Println("\n4. Testando múltiplas injeções de secrets...")

	// Exemplo com múltiplos secrets
	output, err = handler.ExecuteWithSecret(
		"echo",
		[]string{"Deploy with token {{DEPLOY_TOKEN}} to {{DATABASE_URL}}"},
		[]vault.SecretInjection{
			{SecretKey: "deploy_token", Placeholder: "{{DEPLOY_TOKEN}}"},
			{SecretKey: "database_url", Placeholder: "{{DATABASE_URL}}"},
		},
	)

	if err != nil {
		log.Printf("Erro ao executar comando: %v", err)
	} else {
		fmt.Println("   ✓ Múltiplos secrets injetados com sucesso")
		fmt.Printf("   Output: %s\n", strings.TrimSpace(output))
	}

	fmt.Println("\n5. Testando controle de comandos permitidos...")

	// Tentar executar comando não permitido
	_, err = handler.ExecuteWithSecret(
		"rm",
		[]string{"-rf", "/"},
		[]vault.SecretInjection{},
	)

	if err != nil {
		fmt.Println("   ✓ Comando perigoso BLOQUEADO: rm")
		fmt.Printf("   Erro: %v\n", err)
	} else {
		fmt.Println("   ✗ ERRO: Comando perigoso foi permitido!")
	}

	// Tentar executar comando não permitido com secret
	_, err = handler.ExecuteWithSecret(
		"malicious_command",
		[]string{"--steal-secrets"},
		[]vault.SecretInjection{
			{SecretKey: "api_key", Placeholder: "{{API_KEY}}"},
		},
	)

	if err != nil {
		fmt.Println("   ✓ Comando malicioso BLOQUEADO: malicious_command")
		fmt.Printf("   Erro: %v\n", err)
	} else {
		fmt.Println("   ✗ ERRO: Comando malicioso foi permitido!")
	}

	fmt.Println("\n7. Testando configuração customizada...")

	// Criar handler com configuração customizada
	customHandler := vault.NewCommandHandlerWithConfig(vault.CommandHandlerConfig{
		AllowedCommands: []string{"echo", "cat"}, // Apenas echo e cat permitidos
		EnableAudit:     true,
	})

	fmt.Println("   Handler customizado criado com comandos permitidos: echo, cat")

	// Testar comando permitido
	_, err = customHandler.ExecuteWithSecret(
		"echo",
		[]string{"test"},
		[]vault.SecretInjection{},
	)

	if err == nil {
		fmt.Println("   ✓ Comando permitido (echo) executado com sucesso")
	}

	// Testar comando não permitido
	_, err = customHandler.ExecuteWithSecret(
		"curl",
		[]string{"test"},
		[]vault.SecretInjection{},
	)

	if err != nil {
		fmt.Println("   ✓ Comando não permitido (curl) bloqueado corretamente")
	}

	fmt.Println("\n=== Vantagens do CommandHandler ===")
	fmt.Println("✓ Aplicações NÃO acessam secrets diretamente")
	fmt.Println("✓ Controle total sobre quais comandos podem usar secrets")
	fmt.Println("✓ Auditoria automática de uso de secrets")
	fmt.Println("✓ Sanitização de output para evitar vazamento")
	fmt.Println("✓ Whitelist de comandos permite controle granular")
	fmt.Println("✓ Secrets têm tempo de vida limitado ao comando")
	fmt.Println("\n=== Exemplo concluído ===")
}
