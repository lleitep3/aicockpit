# Diretriz de Arquitetura: Copi Bot (Personal Agent Autônomo & Token-Aware)

## 1. Visão Geral e Propósito
O **Copi** (`cockpit copi`) é o mascote e bot orquestrador do Cockpit. Ele atua como um assistente central proativo responsável por:
- Monitorar sessões ativas e filhas sem intervenção humana contínua.
- Alertar o usuário quando houver ações bloqueantes ou pendências de decisão (`AssistantAction`).
- Gerar relatórios executivos (digests) sob demanda ou periodicamente.
- Encaminhar instruções do usuário diretamente para sessões filhas.

---

## 2. Princípio "Automation First" e Eficiência de Tokens (Zero-Token Guardrails)

A IA não deve ser usada para tarefas determinísticas mecânicas:
1. **Detecção de Estado & Diffs em Go (0 Tokens)**:
   - Toda checagem de progresso de sessão, contagem de mensagens, tokens consumidos e pendências é calculada via snapshots em memória/disco (`internal/copi/snapshot.go`).
   - Não se pergunta para uma LLM se uma sessão terminou ou se tem erro se o exit code ou o status do Turn já é conhecido.
2. **Atalhos Rápidos no Terminal**:
   - Menu inicial com teclas `[1]` a `[5]` para consulta direta sem passar por prompts.
   - Timer de inatividade de 10s: se o usuário entrar e não digitar nada, roda a automação de verificação de diffs (0 tokens).
3. **Compressão Pré-LLM com RTK**:
   - Saídas de terminal ou logs de sessões filhas passam por filtros determinísticos antes de serem entregues ao bot.
4. **Criação de Sub-Agentes/Especialistas**:
   - Quando o Copi sugerir ou instanciar novos bots, ele deve aplicar as mesmas regras de restrição de tokens e foco em automação.

---

## 3. Topologia e Isolamento (Headless Daemon / Notebook Servidor)

O Copi é projetado em camadas desacopladas (Clean Architecture):
- **Core Engine (`internal/copi`)**:
  - Independente de terminal e UI.
  - Capaz de rodar como processo headless/daemon 24/7 em um servidor dedicado da rede local (ex: notebook servidor doméstico).
  - Executa tarefas contínuas de watchdog, cron e persistência em sandboxes isoladas.
- **Camada de Apresentação TUI (`internal/copi/tui`)**:
  - Cliente de terminal interativo para uso diário.
  - Pode conectar no Engine localmente no mesmo binário ou remotamente via rede/socket no daemon do servidor.
- **Integração Futura (Web & Push Notifications)**:
  - O mesmo Core Engine alimentará dashboards web (SSE/REST) e enviará notificações ativas (Telegram/Desktop).

---

## 4. Estado da Implementação (2026-10-29)
- **Branch**: `feature/copi-bot`
- **Módulos Core**:
  - `internal/copi/types.go`: Estruturas de Snapshot, SessionDiff, PendingAction e UpdateReport.
  - `internal/copi/snapshot.go`: Extrator de snapshot, comparador diferencial e gerador de relatórios (95% de cobertura de testes).
  - `internal/copi/snapshot_test.go`: Validações de transição de estado e 0 tokens.
- **Próximos Passos**:
  - `internal/copi/engine.go`: Implementação do serviço de orquestração.
  - `internal/copi/tui`: Interface interativa com menu numérico e timer de 10s.
  - `cmd/copi.go`: Comando CLI `cockpit copi`.
