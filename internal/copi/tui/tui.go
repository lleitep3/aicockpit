package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lleitep3/aicockpit/internal/copi"
)

// UI handles the interactive terminal session for Copi.
type UI struct {
	engine *copi.Engine
	in     io.Reader
	out    io.Writer
}

// NewUI creates a new TUI instance.
func NewUI(engine *copi.Engine, in io.Reader, out io.Writer) *UI {
	return &UI{
		engine: engine,
		in:     in,
		out:    out,
	}
}

// Run launches the interactive loop.
func (u *UI) Run(ctx context.Context) error {
	u.printBanner()
	u.printMenu()

	reader := bufio.NewReader(u.in)
	inputChan := make(chan string)
	errChan := make(chan error)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			inputChan <- strings.TrimSpace(line)
		}
	}()

	for {
		u.printPrompt()

		select {
		case <-ctx.Done():
			fmt.Fprintln(u.out, "\nEncerrando Copi. Até logo!")
			return nil

		case err := <-errChan:
			if err == io.EOF {
				return nil
			}
			return err

		case input := <-inputChan:
			if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") || strings.EqualFold(input, "sair") {
				fmt.Fprintln(u.out, "Até logo!")
				return nil
			}

			u.dispatchInput(ctx, input)
		}
	}
}

func (u *UI) printBanner() {
	fmt.Fprintln(u.out, "╭─────────────────────────────────────────────────────────────╮")
	fmt.Fprintln(u.out, "│ 🤖 Copi — Cockpit Bot Assistant                             │")
	fmt.Fprintf(u.out, "│ Sandbox: %-51s│\n", u.engine.SandboxDir())
	fmt.Fprintln(u.out, "╰─────────────────────────────────────────────────────────────╯")
}

func (u *UI) printMenu() {
	fmt.Fprintln(u.out, "\nAtalhos Rápidos (0 Tokens):")
	fmt.Fprintln(u.out, "  [1] Atualizações e Diffs")
	fmt.Fprintln(u.out, "  [2] Sessões & Ações Pendentes")
	fmt.Fprintln(u.out, "  [3] Aprovar ação pendente")
	fmt.Fprintln(u.out, "  [4] Enviar mensagem para uma sessão")
	fmt.Fprintln(u.out, "  [5] Relatório Consolidado")
	fmt.Fprintln(u.out, "  [6] Limpar sessões vazias (idle)")
}

func (u *UI) printPrompt() {
	fmt.Fprint(u.out, "\ncopi> ")
}

func (u *UI) dispatchInput(ctx context.Context, input string) {
	switch input {
	case "1":
		u.handleUpdates(ctx)
	case "2":
		u.handleListSessions(ctx)
	case "3":
		u.handleApproveAction(ctx)
	case "4":
		u.handleSendMessage(ctx)
	case "5":
		u.handleReport(ctx)
	case "6":
		u.handleCleanEmpty(ctx)
	case "":
		// empty line
	default:
		// Send prompt to Copi AI
		fmt.Fprintf(u.out, "[Copi pensando...]:\n")
		resp, err := u.engine.AskCopi(ctx, input)
		if err != nil {
			fmt.Fprintf(u.out, "Erro ao processar: %v\n", err)
		} else {
			fmt.Fprintf(u.out, "%s\n", resp)
		}
	}
}

func (u *UI) handleCleanEmpty(ctx context.Context) {
	fmt.Fprintln(u.out, "🧹 Limpando sessões vazias em idle (0 tokens)...")
	count, err := u.engine.CleanIdleEmptySessions(ctx)
	if err != nil {
		fmt.Fprintf(u.out, "Erro ao limpar sessões: %v\n", err)
		return
	}
	fmt.Fprintf(u.out, "✅ %d sessões vazias arquivadas com sucesso!\n", count)
}

func (u *UI) handleUpdates(ctx context.Context) {
	report, err := u.engine.ScanUpdates(ctx)
	if err != nil {
		fmt.Fprintf(u.out, "Erro ao verificar atualizações: %v\n", err)
		return
	}
	fmt.Fprintln(u.out, report.FormattedText)
}

func (u *UI) handleListSessions(ctx context.Context) {
	snapshots := u.engine.ListSnapshots()
	if len(snapshots) == 0 {
		// Run a scan if empty
		u.handleUpdates(ctx)
		snapshots = u.engine.ListSnapshots()
	}

	if len(snapshots) == 0 {
		fmt.Fprintln(u.out, "Nenhuma sessão ativa encontrada.")
		return
	}

	fmt.Fprintf(u.out, "📋 Sessões Monitoradas (%d):\n", len(snapshots))
	for _, s := range snapshots {
		fmt.Fprintf(u.out, "  • [%s] \"%s\" | Status: %s | Msgs: %d | Tokens: %d\n",
			s.ID, s.Title, s.Status, s.MessageCount, s.InputTokens+s.OutputTokens)
		for _, p := range s.PendingActions {
			fmt.Fprintf(u.out, "      ⚠️ Pendente [%s]: %s\n", p.ID, p.Description)
		}
	}
}

func (u *UI) handleApproveAction(ctx context.Context) {
	pending := u.engine.PendingActions()
	if len(pending) == 0 {
		fmt.Fprintln(u.out, "Nenhuma ação pendente de aprovação no momento.")
		return
	}

	for _, p := range pending {
		err := u.engine.ApproveAction(ctx, p.ConversationID, p.ID)
		if err != nil {
			fmt.Fprintf(u.out, "Erro ao aprovar [%s]: %v\n", p.ID, err)
		} else {
			fmt.Fprintf(u.out, "✅ Ação [%s] aprovada com sucesso na sessão [%s]!\n", p.ID, p.ConversationID)
		}
	}
}

func (u *UI) handleSendMessage(ctx context.Context) {
	snapshots := u.engine.ListSnapshots()
	if len(snapshots) == 0 {
		fmt.Fprintln(u.out, "Nenhuma sessão disponível para envio.")
		return
	}
	target := snapshots[0]
	fmt.Fprintf(u.out, "Para enviar mensagem, utilize: /send <id> <mensagem>\n(Exemplo na sessão [%s])\n", target.ID)
}

func (u *UI) handleReport(ctx context.Context) {
	report := u.engine.LastReport()
	if report.FormattedText == "" {
		u.handleUpdates(ctx)
		return
	}
	fmt.Fprintln(u.out, report.FormattedText)
}
