# Vault Master Pass - Análise e Solução Híbrida

## 🎯 Sua Proposta: Master Pass para CLI

**Ideia:**
```bash
# Modo seguro (padrão)
cockpit vault set key value
# → Requer master pass
# → Usa VaultService automaticamente

# Modo unsafe (requer master pass + confirmação)
cockpit vault set --unsafe key value
# → Requer master pass + confirmação explícita
# → Usa OSVault direto (para compatibilidade)
```

## ✅ VANTAGENS DA SUA SOLUÇÃO

1. **Evita uso acidental** de comandos inseguros
2. **Permite modo "unsafe" quando necessário** (desenvolvimento, debug)
3. **Adiciona camada de autenticação** simples
4. **Fácil de entender** para usuários
5. **Pode ser desabilitada** se usuário quiser (dev mode)

## ❌ PROBLEMAS DA SOLUÇÃO APENAS COM MASTER PASS

### Problema 1: Ainda permite bypass se usuário tiver a senha
```bash
# Se usuário tem master pass, pode fazer:
cockpit vault --unsafe get --namespace other-package secret
# Ainda vulnerável se usuário decidir usar modo unsafe
```

### Problema 2: Não resolve o problema fundamental
```bash
# Mesmo com master pass, pacote pode:
export COCKPIT_MASTER_PASS="senha"
cockpit vault --unsafe get --namespace kb-graphify secret
# Se pacote obtiver a senha de alguma forma, está vulnerável
```

### Problema 3: Complexidade para automação
```bash
# Scripts precisam da senha
export COCKPIT_MASTER_PASS="senha"
cockpit vault set key value
# Senha em variável de ambiente = vulnerável
```

## 🎯 SOLUÇÃO HÍBRIDA RECOMENDADA

### Combinação: Master Pass + VaultService + Modo Unsafe Controlado

```go
// cmd/vault.go
var masterPassRequired = true  // Padrão
var unsafeModeEnabled = false

func NewVaultSetCommand() *cobra.Command {
	var masterPassFlag string
	var unsafeFlag bool
	
	setCmd := &cobra.Command{
		Use:   "set <key>",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Modo unsafe requer +master pass + confirmação
			if unsafeFlag {
				if masterPassFlag == "" {
					masterPassFlag = promptMasterPassword()
				}
				
				if !validateMasterPassword(masterPassFlag) {
					return fmt.Errorf("invalid master password")
				}
				
				// Confirmação explícita
				fmt.Print("UNSAFE MODE: This bypasses security. Continue? (type 'UNSAFE'): ")
				var confirmation string
				fmt.Scanln(&confirmation)
				if confirmation != "UNSAFE" {
					return fmt.Errorf("unsafe mode cancelled")
				}
				
				logSecurityEvent("unsafe_mode_used", "vault set")
				// Usar OSVault (compatibilidade)
				v := vault.NewOSVault()
				return v.Set(key, value)
			}
			
			// Modo seguro padrão
			if masterPassRequired {
				if masterPassFlag == "" {
					masterPassFlag = promptMasterPassword()
				}
				
				if !validateMasterPassword(masterPassFlag) {
					return fmt.Errorf("invalid master password")
				}
			}
			
			// Usar VaultService (seguro)
			client := vault.NewVaultServiceClient()
			return client.SetSecret(key, value)
		},
	}
	
	setCmd.Flags().StringVar(&masterPassFlag, "master-password", "", "Master password for vault operations")
	setCmd.Flags().BoolVar(&unsafeFlag, "unsafe", false, "UNSAFE mode: bypass security (not recommended)")
	
	return setCmd
}
```

### Configuração Global

```yaml
# ~/.cockpit/config.yaml
vault:
  master_pass_required: true  # Requerer master pass (padrão)
  master_pass_hash: "..."    # Hash da master pass
  allow_unsafe_mode: true    # Permitir modo unsafe (requer confirmação)
  default_mode: "service"   # "service" (seguro) ou "direct" (inseguro)
```

### Comandos de Configuração

```bash
# Configurar master password
cockpit vault set-master-password
# Enter new master password: *****
# Confirm master password: *****
# ✓ Master password configured

# Desabilitar master password (dev mode)
cockpit vault disable-master-password
# WARNING: This disables security protection. Continue? (type DISABLE): DISABLE
# ✓ Master password disabled

# Habilitar master password novamente
cockpit vault enable-master-password
```

## 📊 COMPARAÇÃO DE SOLUÇÕES

| Solução | Segurança | Usabilidade | Proteção contra bypass |
|---------|-----------|--------------|----------------------|
| **Sem master pass** | ❌ Baixa | ✅ Alta | ❌ Nenhuma |
| **Master pass apenas** | ⚠️ Média | ⚠️ Média | ⚠️ Parcial (se senha comprometida) |
| **Master pass + VaultService** | ✅ Alta | ⚠️ Média | ✅ Boa |
| **Master pass + VaultService + Unsafe controlado** | ✅✅ Muito Alta | ✅ Boa | ✅ Excelente |

## 🎯 FLUXO RECOMENDADO

### Modo Seguro (Padrão)
```bash
$ cockpit vault set api-key "sk-..."
Master password: *****
✓ Secret stored securely via VaultService
```

### Modo Unsafe (Quando necessário)
```bash
$ cockpit vault set --unsafe api-key "sk-..."
Master password: *****
UNSAFE MODE: This bypasses security. Continue? (type 'UNSAFE'): UNSAFE
⚠️ WARNING: Using unsafe mode - secrets accessible without namespace isolation
✓ Secret stored directly (for compatibility/debugging only)
```

### Modo Dev (Sem master pass)
```bash
$ cockpit vault disable-master-password
WARNING: This disables security protection. Continue? (type DISABLE): DISABLE
✓ Master password disabled (dev mode)

$ cockpit vault set api-key "sk-..."
⚠️ Running in dev mode - no master password required
✓ Secret stored via VaultService (still uses service, just no password)
```

## 🔐 IMPLEMENTAÇÃO DE MASTER PASS

```go
// internal/vault/master_auth.go
package vault

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"golang.org/x/term"
)

type MasterAuth struct {
	enabled     bool
	passwordHash string
	storagePath string
}

func NewMasterAuth() *MasterAuth {
	return &MasterAuth{
		enabled:     true,
		storagePath: "/home/lleite/.cockpit/vault/master.dat",
	}
}

func (ma *MasterAuth) SetPassword(password string) error {
	// Hash the password
	hash := sha256.Sum256([]byte(password))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])
	
	ma.passwordHash = hashStr
	ma.enabled = true
	
	return ma.save()
}

func (ma *MasterAuth) Validate(password string) bool {
	if !ma.enabled {
		return true // Se desabilitado, sempre válido
	}
	
	hash := sha256.Sum256([]byte(password))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])
	
	return hashStr == ma.passwordHash
}

func (ma *MasterAuth) Disable() error {
	ma.enabled = false
	return ma.save()
}

func (ma *MasterAuth) Enable() error {
	ma.enabled = true
	return ma.save()
}

func (ma *MasterAuth) save() error {
	data := fmt.Sprintf("%v|%s", ma.enabled, ma.passwordHash)
	return os.WriteFile(ma.storagePath, []byte(data), 0600)
}

func (ma *MasterAuth) load() error {
	data, err := os.ReadFile(ma.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Arquivo não existe ainda
		}
		return err
	}
	
	// Parse: enabled|hash
	parts := strings.Split(string(data), "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid master auth file format")
	}
	
	enabled := parts[0] == "true"
	ma.enabled = enabled
	ma.passwordHash = parts[1]
	
	return nil
}

func promptMasterPassword() string {
	fmt.Print("Master password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}
	
	// Confirmar
	fmt.Print("Confirm master password: ")
	byteConfirm, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}
	
	if string(bytePassword) != string(byteConfirm) {
		return ""
	}
	
	return string(bytePassword)
}
```

## 🎯 RESPOSTA DIRETA

**Sua ideia de master pass é BOA e resolve parte do problema, mas sozinha não é suficiente.**

**Solução recomendada:**
1. ✅ **Master pass** (sua ideia) - Evita uso acidental
2. ✅ **VaultService padrão** - Segurança real por padrão
3. ✅ **Modo --unsafe controlado** - Requer master pass + confirmação "UNSAFE"
4. ✅ **Dev mode sem master pass** - Para desenvolvimento
5. ✅ **Logging de operações inseguras** - Auditoria

**Assim você tem:**
- Segurança real no uso normal
- Flexibilidade para casos especiais
- Proteção contra uso acidental
- Auditoria de operações de risco
- Conforto para desenvolvimento

**Master pass sozinho não é suficiente porque ainda permite bypass se usuário tiver a senha. Combinado com VaultService + controle de unsafe mode, você tem segurança real.**