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

## Inconsistências Resolvidas

As seguintes questões abertas dos refinamentos originais foram fechadas:

| # | Inconsistência Original | Decisão / Solução | Documento Atualizado |
| :--- | :--- | :--- | :--- |
| 1 | Stack não decidida (React vs Headless?) | Manter SvelteKit 5 + Tailwind CSS 3. | `refinement-layout-overview.md`, `dashboard-refinement-roadmap.md` |
| 2 | Tecnologia do grafo não definida | D3.js v7 + SVG renderizado em Svelte. | `refinement-kb-miniapps.md`, `dashboard-refinement-roadmap.md` |
| 3 | Execução de comandos do cockpit sem segurança | `CommandExecutor` whitelistado; leitura via backend, ações sensíveis com confirmação. | `refinement-layout-overview.md`, `dashboard-refinement-roadmap.md` |
| 4 | Quick-fix automático de alto risco | Exige confirmação explícita e audit log. | `refinement-layout-overview.md` |
| 5 | Vault "só em memória" no React | Chave de descriptografia mantida em memória do backend Python. | `refinement-packages-vault.md`, `dashboard-refinement-roadmap.md` |
| 6 | Protocolo de logs não definido | Server-Sent Events (SSE) para logs de mini-apps. | `refinement-kb-miniapps.md`, `dashboard-refinement-roadmap.md` |
| 7 | Métricas de recursos sem resposta | `psutil` no backend para CPU/RAM dos processos filhos. | `refinement-kb-miniapps.md`, `dashboard-refinement-roadmap.md` |
| 8 | Múltiplas instâncias sem gestão de portas | Mantido como evolução futura. | `refinement-kb-miniapps.md` |
| 9 | SSE vs polling para jobs longos | SSE com `job_id` para progresso contínuo. | `refinement-packages-vault.md`, `dashboard-refinement-roadmap.md` |

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
