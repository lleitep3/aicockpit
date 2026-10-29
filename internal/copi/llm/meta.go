package llm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// SystemMeta contains configurable operating system and environment metadata.
type SystemMeta struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Hostname    string `json:"hostname,omitempty"`
	Workspace   string `json:"workspace"`
	SandboxDir  string `json:"sandbox_dir"`
	ProjectName string `json:"project_name"`
	RepoURL     string `json:"repo_url"`
}

// DefaultSystemMeta detects real host OS metadata.
func DefaultSystemMeta(sandboxDir string) SystemMeta {
	hostname, _ := os.Hostname()
	return SystemMeta{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Hostname:    hostname,
		Workspace:   sandboxDir,
		SandboxDir:  sandboxDir,
		ProjectName: "AICockpit (github.com/lleitep3/aicockpit)",
		RepoURL:     "https://github.com/lleitep3/aicockpit",
	}
}

// ChatTurn represents a historical message in the current conversation.
type ChatTurn struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// BuildCopiSystemPrompt generates the system prompt with host metadata, tools definitions, and identity.
func BuildCopiSystemPrompt(meta SystemMeta) string {
	var b strings.Builder
	b.WriteString("Você é o Copi, o bot mascote e orquestrador autônomo do AICockpit.\n")
	b.WriteString("IMPORTANTE: Você atua no projeto AICockpit (harness engineering para IA em Go, github.com/lleitep3/aicockpit). Você NÃO É o Cockpit Project da Red Hat / OpenShift.\n\n")

	b.WriteString("METAINFORMAÇÕES DO AMBIENTE (Host OS & Sandbox):\n")
	b.WriteString(fmt.Sprintf("- Sistema Operacional: %s (%s)\n", meta.OS, meta.Arch))
	if meta.Hostname != "" {
		b.WriteString(fmt.Sprintf("- Hostname: %s\n", meta.Hostname))
	}
	b.WriteString(fmt.Sprintf("- Sandbox / Workspace: %s\n", meta.SandboxDir))
	b.WriteString(fmt.Sprintf("- Repositório: %s\n\n", meta.RepoURL))

	b.WriteString("DIRETRIZES FUNDAMENTAIS:\n")
	b.WriteString("1. AUTOMATION FIRST: Resolva tarefas com código/scripts determinísticos (Go/Bash/CLI do Cockpit/gh). Não invente comandos inexistentes.\n")
	b.WriteString("2. RESPOSTAS OBJETIVAS: Estilo Caveman inteligente / TL;DR. Direto ao ponto, sem enrolação.\n")
	b.WriteString("3. CHECAGEM DE REPOSITÓRIO: Para verificar pipelines/CI do GitHub do projeto, use ferramentas do GitHub (ex: `gh run list`, `gh pr checks`).\n")
	b.WriteString("4. FERRAMENTAS DISPONÍVEIS VIA TOOL CALL:\n")
	b.WriteString("   Se precisar pesquisar histórico antigo de mensagens ou sessões, invoque a tool search_history com o termo desejado:\n")
	b.WriteString("   TOOL_CALL: search_history {\"query\": \"termo de busca\"}\n")

	return b.String()
}

// OpenAIChatMessage represents a message in OpenAI/OpenRouter format.
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PrepareChatMessages prepares the system prompt + last 2 historical messages + current message.
func PrepareChatMessages(systemPrompt string, history []ChatTurn, currentMessage string) []OpenAIChatMessage {
	var msgs []OpenAIChatMessage
	msgs = append(msgs, OpenAIChatMessage{Role: "system", Content: systemPrompt})

	// Add up to 2 most recent turns
	if len(history) > 2 {
		history = history[len(history)-2:]
	}
	for _, h := range history {
		msgs = append(msgs, OpenAIChatMessage{Role: h.Role, Content: h.Content})
	}

	msgs = append(msgs, OpenAIChatMessage{Role: "user", Content: currentMessage})
	return msgs
}
