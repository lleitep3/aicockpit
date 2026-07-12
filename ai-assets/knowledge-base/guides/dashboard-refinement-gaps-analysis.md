# Análise de Gaps: Cockpit Dashboard Refinement

Este documento compara os refinamentos planejados com a implementação atual do `cockpit-dashboard` e do `cockpit-mini-apps`, identificando gaps, riscos e incrementos sugeridos.

## Estado Atual (Baseline)

### Dashboard
- Página única com 5 cards: Status Geral, Providers, Packages, Registries, KB.
- Backend FastAPI lê arquivos de `~/.cockpit/` (config, packages, registries, kb).
- Sem navegação, sem múltiplas telas, sem command palette.
- Sem integração com `cockpit doctor`, vault ou console de mini-apps.
- Stack: SvelteKit + Tailwind CSS 3 + FastAPI.

### Mini-Apps
- Script shell nativo (`bin/mini-app`) cria e gerencia mini-apps sem Docker (without-db) ou com Docker (with-db).
- Processos gerenciados por PID files em `~/.cockpit/workspace/mini-apps/<name>/.pids/`.
- Logs escritos em `backend.log` e `frontend.log`.

## Gaps por Módulo

| # | Módulo | Implementação Atual | Refinamento Planejado | Gap |
| :--- | :--- | :--- | :--- | :--- |
| 1 | Layout & Design System | Header simples com título | Sidebar, navbar, command palette, breadcrumbs, multi-tema | Grande: sem navegação para as telas planejadas. |
| 2 | Overview | Status Geral (version/env/active) | KPIs, Doctor, feed de atividade, quick-fix | Médio: faltam doctor, KPIs e feed. |
| 3 | Packages | Lista simples de pacotes instalados | Abas Instalados/Registry, busca fuzzy, ações install/uninstall/upgrade | Grande: sem registry, ações ou busca. |
| 4 | Vault | Não existe | Lock screen, credenciais, auto-lock, copy seguro | Módulo inteiro ausente. |
| 5 | KB | Lista de `.md` com nome/categoria | Grafo interativo, busca fuzzy, preview, auditoria, criar artigo | Grande: sem grafo, busca ou ações. |
| 6 | Mini-Apps Console | Não existe | Cards de processos, terminal de logs, health, start/stop/restart | Módulo inteiro ausente. |

## Inconsistências e Riscos nos Refinamentos

1. **Stack não decidida:** documentos perguntavam "React ou Headless?". Dashboard já usa SvelteKit. **Decisão fechada: manter SvelteKit.**
2. **Grafo sem tecnologia:** perguntavam "D3, React Flow ou Vis.js?". **Decisão fechada: D3.js + SVG.**
3. **Execução de comandos do cockpit:** proposta de endpoint do `cockpit doctor` sem detalhes de segurança. **Solução: `CommandExecutor` whitelistado, leitura via backend, ações sensíveis com confirmação.**
4. **Quick-fix automático:** alto risco de segurança. **Mitigação: confirmação explícita e audit log.**
5. **Vault "só em memória":** chave no estado React não resolve backend Python. **Solução: chave mantida em memória do backend, nunca persistida.**
6. **WebSocket de logs:** sem protocolo definido. **Solução: usar SSE (Server-Sent Events) mais simples.**
7. **Métricas de recursos:** pergunta sem resposta. **Solução: `psutil` no backend para ler PIDs dos mini-apps.**
8. **Múltiplas instâncias:** sem gestão de portas. **Marcado como evolução futura.**
9. **SSE vs polling:** pergunta aberta. **Decisão fechada: SSE para jobs e logs.**

## Decisões de Arquitetura Fechadas

| Decisão | Escolha | Motivação |
| :--- | :--- | :--- |
| Stack UI | SvelteKit 5 + Tailwind CSS 3 | Evita reescrita; dashboard já implementado. |
| Execução de comandos | Híbrida | Leitura via backend whitelist; ações sensíveis com confirmação. |
| Logs de mini-apps | SSE | Mais simples que WebSocket para stream unidirecional. |
| Grafo de KB | D3.js + SVG | Controle total, leve, funciona com Svelte. |
| Atualizações longas | SSE com job_id | Progresso contínuo sem polling. |

## Modelo de Segurança

1. **Whitelist:** apenas comandos `cockpit` autorizados no backend.
2. **Sanitização:** regex `^[a-z0-9_.-]+$` para nomes e chaves.
3. **Timeout:** 30s leitura, 5min ação; kill automático ao exceder.
4. **Audit log:** timestamp, usuário, comando, args, status.
5. **Ações sensíveis:** exigem token de confirmação ou execução local.
6. **Vault:** chave de descriptografia só em memória do backend.

## Sequência de Implementação Sugerida

1. Layout + Sidebar + navegação (não depende de backend)
2. Overview com `cockpit doctor` (leitura JSON)
3. Packages Manager com busca fuzzy e ações seguras
4. Mini-Apps Console (monitor + SSE de logs)
5. KB Explorer (grafo + busca + preview)
6. Vault Manager (segurança crítica; por último)

## Contratos de API Comuns

### Jobs Assíncronos
- `POST /api/v1/jobs` inicia ação; retorna `{ job_id }`.
- `GET /api/v1/jobs/{job_id}` retorna status.
- `GET /api/v1/jobs/{job_id}/stream` retorna SSE com progresso.

### Audit Log
- `GET /api/v1/audit` lista ações executadas.

## Validações Técnicas Necessárias

- [ ] `cockpit doctor` suporta saída JSON (`--json` ou similar).
- [ ] `cockpit pkg list` suporta saída JSON.
- [ ] KB possui front-matter parseável em todos os `.md`.
- [ ] Backend consegue ler PIDs e logs dos mini-apps.
- [ ] `psutil` disponível para métricas de processos.
- [ ] Prototipar grafo D3 com 100+ nós para validar performance.

## Recomendação Imediata

Antes de implementar qualquer módulo, fechar três pontos:
1. Criar `CommandExecutor` no backend com whitelist e audit.
2. Definir estrutura de rotas do SvelteKit para as 6 telas.
3. Prototipar grafo de KB com D3 + SVG.
