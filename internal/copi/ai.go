package copi

import (
	"context"
	"fmt"
	"strings"
)

// AskCopi sends a user request to the underlying AI service for Copi,
// injecting Copi's persona and strict token-efficiency guardrails.
func (e *Engine) AskCopi(ctx context.Context, userRequest string) (string, error) {
	if strings.TrimSpace(userRequest) == "" {
		return "", fmt.Errorf("requisição não pode ser vazia")
	}

	if e.source == nil {
		return "", fmt.Errorf("nenhum serviço de conversação configurado")
	}

	prompt := CopiSystemPrompt(userRequest, e.SandboxDir(), e.snapshots)
	return e.source.Ask(ctx, prompt)
}

// CleanIdleEmptySessions identifies all idle sessions with 0 messages and archives/removes them (0 tokens).
func (e *Engine) CleanIdleEmptySessions(ctx context.Context) (int, error) {
	if e.source == nil {
		return 0, fmt.Errorf("nenhum serviço de conversação configurado")
	}

	// Read all summaries including archived ones so dead files can be cleaned up
	summaries, err := e.source.List(ctx, true)
	if err != nil {
		return 0, fmt.Errorf("listar conversas: %w", err)
	}

	archivedCount := 0
	for _, summary := range summaries {
		conv, err := e.source.Get(ctx, summary.ID)
		if err != nil || conv == nil {
			continue
		}

		if len(conv.Messages) == 0 && len(conv.Turns) == 0 {
			if err := e.source.ArchiveSession(ctx, conv.ID); err == nil {
				archivedCount++
			}
		}
	}

	// Rescan updates to refresh memory state
	_, _ = e.ScanUpdates(ctx)
	return archivedCount, nil
}

// CopiSystemPrompt constructs the prompt with automation-first directives.
func CopiSystemPrompt(request, sandboxDir string, snapshots map[string]SessionSnapshot) string {
	var builder strings.Builder
	builder.WriteString("Você é o Copi, o bot mascote e orquestrador autônomo do Cockpit.\n")
	builder.WriteString(fmt.Sprintf("Sua sandbox de execução local é: %s\n\n", sandboxDir))
	builder.WriteString("DIRETRIZES FUNDAMENTAIS:\n")
	builder.WriteString("1. AUTOMATION FIRST: Sempre prefira criar ou executar scripts e rotinas determinísticas (Go/Bash/CLI do Cockpit) em vez de gastar tokens desnecessários.\n")
	builder.WriteString("2. RESPOSTAS OBJETIVAS: Seja direto e prático (estilo Caveman inteligente / TL;DR). Sem enrolação.\n")
	builder.WriteString("3. GESTÃO DE SESSÕES: Ajude o usuário a orquestrar tarefas, limpar sessões mortas e acompanhar tarefas ativas.\n\n")

	builder.WriteString(fmt.Sprintf("ESTADO ATUAL (%d sessões monitoradas):\n", len(snapshots)))
	count := 0
	for _, s := range snapshots {
		if count >= 10 {
			builder.WriteString("  ... (outras sessões omitidas para economizar tokens)\n")
			break
		}
		builder.WriteString(fmt.Sprintf("  • [%s] %q (status: %s, msgs: %d)\n", s.ID, s.Title, s.Status, s.MessageCount))
		count++
	}

	builder.WriteString(fmt.Sprintf("\nPEDIDO DO USUÁRIO:\n%s\n", request))
	return builder.String()
}
