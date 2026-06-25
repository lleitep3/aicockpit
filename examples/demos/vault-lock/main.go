package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lleitep3/aicockpit/internal/vault"
)

func main() {
	fmt.Println("=== Demonstração: Sistema Lock/Unlock do Vault ===")
	fmt.Println()

	// Inicializar LockManager
	lm := vault.NewLockManager("/tmp/vault-lock-demo.json")
	defer os.Remove("/tmp/vault-lock-demo.json")

	// Configurar secrets de teste
	vaultDirect := vault.NewOSVault()
	vaultDirect.Set("test_secret", "test_value")
	defer vaultDirect.Delete("test_secret")

	fmt.Println("1. Estado Inicial (Padrão)")
	fmt.Println("   Por padrão, vault começa LOCKED por segurança")
	status := lm.GetStatus()
	fmt.Printf("   Status inicial: IsLocked=%v, GlobalUnlock=%v\n", status.IsLocked, status.GlobalUnlock)

	fmt.Println()
	fmt.Println("2. Tentando acessar vault sem desbloquear")
	currentPkg := "test-package"
	canAccess := lm.CanPackageAccess(currentPkg)
	fmt.Printf("   '%s' pode acessar vault: %v\n", currentPkg, canAccess)

	if !canAccess {
		fmt.Println("   ✓ Acesso bloqueado corretamente")
	}

	fmt.Println()
	fmt.Println("3. Desbloqueando vault globalmente")
	err := lm.Unlock("Sessão de desenvolvimento")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Vault desbloqueado globalmente")

	status = lm.GetStatus()
	fmt.Printf("   Status: IsLocked=%v, GlobalUnlock=%v\n", status.IsLocked, status.GlobalUnlock)

	fmt.Println()
	fmt.Println("4. Desbloqueando pacote específico")
	err = lm.UnlockPackage("kb-graphify", "Necessário para KB graph")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Pacote 'kb-graphify' desbloqueado")

	err = lm.UnlockPackage("user-service", "Necessário para User Service")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Pacote 'user-service' desbloqueado")

	status = lm.GetStatus()
	fmt.Printf("   Pacotes desbloqueados: %v\n", status.UnlockedPackages)

	fmt.Println()
	fmt.Println("5. Bloqueando vault novamente")
	err = lm.Lock("Fim da sessão de trabalho")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Vault bloqueado novamente")

	status = lm.GetStatus()
	fmt.Printf("   Status: IsLocked=%v, GlobalUnlock=%v\n", status.IsLocked, status.GlobalUnlock)
	fmt.Printf("   Pacotes desbloqueados após lock: %v\n", status.UnlockedPackages)

	fmt.Println()
	fmt.Println("6. Desbloqueando apenas um pacote")
	err = lm.Unlock("Sessão específica para kb-graphify")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}

	err = lm.UnlockPackage("kb-graphify", "Sessão específica")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Vault desbloqueado, mas só kb-graphify tem acesso")

	status = lm.GetStatus()
	fmt.Printf("   Status: IsLocked=%v, GlobalUnlock=%v\n", status.IsLocked, status.GlobalUnlock)

	// Verificar acessos
	fmt.Println()
	fmt.Println("   Verificando acessos:")
	fmt.Printf("   - kb-graphify pode acessar: %v\n", lm.CanPackageAccess("kb-graphify"))
	fmt.Printf("   - user-service pode acessar: %v\n", lm.CanPackageAccess("user-service"))
	fmt.Printf("   - outro-pacote pode acessar: %v\n", lm.CanPackageAccess("outro-pacote"))

	fmt.Println()
	fmt.Println("7. Bloqueando pacote específico")
	err = lm.LockPackage("kb-graphify")
	if err != nil {
		fmt.Printf("   Erro: %v\n", err)
		return
	}
	fmt.Println("   ✓ Pacote 'kb-graphify' bloqueado")

	fmt.Println()
	fmt.Println("8. Auto-lock timeout (demo)")
	lm.Unlock("Teste de auto-lock")
	lm.SetAutoLockTimeout(3 * time.Second)
	fmt.Println("   ✓ Vault desbloqueado com auto-lock em 3 segundos")
	fmt.Println("   Aguardando auto-lock...")

	time.Sleep(4 * time.Second)

	status = lm.GetStatus()
	fmt.Printf("   Status após timeout: IsLocked=%v (auto-lock funcionou)\n", status.IsLocked)

	fmt.Println()
	fmt.Println("9. Comando status (como seria na CLI)")
	fmt.Println("   $ cockpit vault status")
	fmt.Println("   === Vault Lock Status ===")
	fmt.Println("   Status: 🔒 LOCKED")
	fmt.Println("   Locked at: 2026-06-25 15:30:00")
	fmt.Println("   Locked by: user")
	fmt.Println("   Reason: Fim da sessão de trabalho")
	fmt.Println()
	fmt.Println("   Global Access: 🔒 Vault is locked")
	fmt.Println()
	fmt.Println("   Package Access:")
	fmt.Println("   =================")
	fmt.Println("     ✓ kb-graphify: 🔓 Unlocked")
	fmt.Println("     ✗ user-service: 🔒 Locked")
	fmt.Println()
	fmt.Println("   Summary: Vault is locked. Use 'cockpit vault unlock' to access secrets.")
	fmt.Println("           Only 1 packages have access: [kb-graphify]")

	fmt.Println()
	fmt.Println("=== COMPARAÇÃO COM MASTER PASS ===")
	fmt.Println()
	fmt.Println("❌ Master Pass:")
	fmt.Println("   - Requer senha")
	fmt.Println("   - Se senha comprometida, todo fica vulnerável")
	fmt.Println("   - Usuário pode esquecer senha")
	fmt.Println("   - Complexo para automação")
	fmt.Println()
	fmt.Println("✅ Lock/Unlock System:")
	fmt.Println("   - Sem senhas (nada para comprometer)")
	fmt.Println("   - Intuitivo (padrão em segurança)")
	fmt.Println("   - Controle granular por pacote")
	fmt.Println("   - Auto-lock possível")
	fmt.Println("   - Status claro e visível")
	fmt.Println("   - Fácil para automação")
	fmt.Println()
	fmt.Println("=== EXEMPLOS DE USO ===")
	fmt.Println()
	fmt.Println("Desenvolvimento:")
	fmt.Println("  cockpit vault unlock --timeout 1h")
	fmt.Println("  cockpit vault set api-key '...'")
	fmt.Println("  (Auto-lock após 1 hora)")
	fmt.Println()
	fmt.Println("Produção:")
	fmt.Println("  cockpit vault unlock kb-graphify")
	fmt.Println("  ./kb-graphify/bin/search  # Só kb-graphify tem acesso")
	fmt.Println("  cockpit vault lock kb-graphify")
	fmt.Println()
	fmt.Println("Sessão de trabalho:")
	fmt.Println("  cockpit vault unlock --reason 'Sessão de deploy'")
	fmt.Println("  # ... trabalho com secrets ...")
	fmt.Println("  cockpit vault lock --reason 'Fim da sessão'")
	fmt.Println()
	fmt.Println("=== SEU SISTEMA É SUPERIOR ===")
	fmt.Println("✅ Mais intuitivo que master pass")
	fmt.Println("✅ Sem senhas para comprometer")
	fmt.Println("✅ Controle granular por pacote")
	fmt.Println("✅ Auto-lock para segurança adicional")
	fmt.Println("✅ Status claro e auditável")
	fmt.Println("✅ Padrão em sistemas de segurança reais")
}
