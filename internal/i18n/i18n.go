package i18n

import (
	"fmt"
	"sync"
)

// Translator handles message translation for different languages.
type Translator struct {
	language string
	messages map[string]map[string]string
	mu       sync.RWMutex
}

var (
	instance *Translator
	once     sync.Once
)

// New creates a new translator instance (singleton pattern).
func New(language string) *Translator {
	once.Do(func() {
		instance = &Translator{
			language: language,
			messages: initMessages(),
		}
	})
	return instance
}

// Get returns the translator instance.
func Get() *Translator {
	if instance == nil {
		instance = New("en-us")
	}
	return instance
}

// SetLanguage changes the current language.
func (t *Translator) SetLanguage(language string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.language = language
}

// T translates a message key with optional arguments.
func (t *Translator) T(key string, args ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if msgs, ok := t.messages[t.language]; ok {
		if msg, ok := msgs[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	// Fallback to English if translation not found
	if msgs, ok := t.messages["en-us"]; ok {
		if msg, ok := msgs[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	return key
}

// initMessages initializes all message translations.
func initMessages() map[string]map[string]string {
	return map[string]map[string]string{
		"en-us": {
			// General
			"welcome":     "Welcome to AICockpit",
			"version":     "Version",
			"language":    "Language",
			"log_level":   "Log Level",
			"ai_provider": "AI Provider",
			"error":       "Error",
			"success":     "Success",
			"failed":      "Failed",
			"done":        "Done",

			// Setup
			"setup.welcome":  "Welcome to AICockpit Setup",
			"setup.language": "Select your preferred language",
			"setup.ai":       "Select your AI provider",
			"setup.vault":    "Creating vault...",
			"setup.complete": "Setup complete!",
			"setup.saved":    "Configuration saved to %s",

			// Info
			"info.title":  "AICockpit Information",
			"info.dir":    "Cockpit Directory",
			"info.config": "Config File",
			"info.log":    "Log File",

			// Caveman
			"caveman.enabled":          "Caveman mode enabled globally. Deploying changes...",
			"caveman.disabled":         "Caveman mode disabled. Deploying changes...",
			"caveman.already_disabled": "Caveman mode is already disabled.",
			"caveman.on":               "Caveman mode: ON",
			"caveman.off":              "Caveman mode: OFF",
			"caveman.invalid":          "invalid action: %s. Use on, off, or status",
			"info.packages":            "Installed Packages",
			"info.no_packages":         "No packages installed",

			// Doctor
			"doctor.title":      "AICockpit Health Check",
			"doctor.checking":   "Checking %s...",
			"doctor.ok":         "✓ %s",
			"doctor.failed":     "✗ %s",
			"doctor.config_ok":  "Configuration is valid",
			"doctor.config_bad": "Configuration file is missing",
			"doctor.vault_ok":   "Vault is accessible",
			"doctor.vault_bad":  "Vault is not initialized",
			"doctor.passed":     "All checks passed!",
			"doctor.failed_msg": "Some checks failed. Please run 'cockpit setup' to fix issues.",

			// Uninstall
			"uninstall.confirm": "Are you sure you want to uninstall AICockpit? This will delete all data in %s (y/n): ",
			"uninstall.success": "AICockpit uninstalled successfully",
			"uninstall.cancel":  "Uninstall cancelled",
		},
		"pt-br": {
			// Geral
			"welcome":     "Bem-vindo ao AICockpit",
			"version":     "Versão",
			"language":    "Idioma",
			"log_level":   "Nível de Log",
			"ai_provider": "Provedor de IA",
			"error":       "Erro",
			"success":     "Sucesso",
			"failed":      "Falhou",
			"done":        "Pronto",

			// Setup
			"setup.welcome":  "Bem-vindo à Configuração do AICockpit",
			"setup.language": "Selecione seu idioma preferido",
			"setup.ai":       "Selecione seu provedor de IA",
			"setup.vault":    "Criando vault...",
			"setup.complete": "Configuração concluída!",
			"setup.saved":    "Configuração salva em %s",

			// Info
			"info.title":       "Informações do AICockpit",
			"info.dir":         "Diretório do Cockpit",
			"info.config":      "Arquivo de Configuração",
			"info.log":         "Arquivo de Log",
			"info.packages":    "Pacotes Instalados",
			"info.no_packages": "Nenhum pacote instalado",

			// Caveman
			"caveman.enabled":          "Modo Caveman ativado globalmente. Atualizando providers...",
			"caveman.disabled":         "Modo Caveman desativado. Atualizando providers...",
			"caveman.already_disabled": "Modo Caveman já está desativado.",
			"caveman.on":               "Modo Caveman: ON",
			"caveman.off":              "Modo Caveman: OFF",
			"caveman.invalid":          "ação inválida: %s. Use on, off, ou status",

			// Doctor
			"doctor.title":      "Verificação de Saúde do AICockpit",
			"doctor.checking":   "Verificando %s...",
			"doctor.ok":         "✓ %s",
			"doctor.failed":     "✗ %s",
			"doctor.config_ok":  "Configuração é válida",
			"doctor.config_bad": "Arquivo de configuração está faltando",
			"doctor.vault_ok":   "Vault está acessível",
			"doctor.vault_bad":  "Vault não foi inicializado",
			"doctor.passed":     "Todas as verificações passaram!",
			"doctor.failed_msg": "Algumas verificações falharam. Execute 'cockpit setup' para corrigir.",

			// Uninstall
			"uninstall.confirm": "Tem certeza que deseja desinstalar o AICockpit? Isso deletará todos os dados em %s (s/n): ",
			"uninstall.success": "AICockpit desinstalado com sucesso",
			"uninstall.cancel":  "Desinstalação cancelada",
		},
	}
}
