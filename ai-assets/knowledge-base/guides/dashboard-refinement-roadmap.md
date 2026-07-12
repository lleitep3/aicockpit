# Backlog de Refinamento: Cockpit Dashboard

Este documento é a lista de trabalho para aprofundamento e detalhamento técnico de cada ponto do Cockpit Dashboard antes da fase de implementação.

---

## status do Refinamento

| Módulo | Status | Tópicos de Discussão |
| :--- | :--- | :--- |
| **1. Layout & Design System** | 🟩 REFINADO | Paleta de cores, componentes reusáveis e responsividade. |
| **2. Visão Geral (Overview)** | 🟩 REFINADO | Integração do `cockpit doctor` e formato de payload. |
| **3. Gerenciador de Pacotes** | 🟩 REFINADO | APIs de controle, paridade CLI/UI e busca. |
| **4. Vault & Segurança** | 🟩 REFINADO | Ciclo de vida da senha, auto-lock e criptografia em trânsito. |
| **5. Base de Conhecimento (KB)** | 🟩 REFINADO | Renderização de grafo, detecção de notas órfãs e busca fuzzy. |
| **6. Console de Mini-Apps** | 🟩 REFINADO | WebSockets para logs em tempo real e controle de sub-processos. |

## Decisões de Arquitetura (Fechadas)

| Decisão | Escolha | Motivação |
| :--- | :--- | :--- |
| **Stack de UI** | SvelteKit + Tailwind CSS 3 | Dashboard já implementado em Svelte; evita reescrita. |
| **Execução de comandos do cockpit** | Híbrida | Leitura via backend (`CommandExecutor` whitelist); ações sensíveis via confirmação/CLI local. |
| **Protocolo de logs de mini-apps** | Server-Sent Events (SSE) | Mais simples que WebSocket para unidirecional; fallback para polling. |
| **Grafo de KB** | D3.js + SVG | Controle total, leve, funciona bem com Svelte. |
| **Atualizações longas** | SSE com job ID | Progresso contínuo sem polling. |

## Modelo de Segurança

1. **Whitelist de comandos**: apenas comandos `cockpit` previamente autorizados podem ser executados pelo backend.
2. **Sanitização de argumentos**: regex restrita (`^[a-z0-9_.-]+$`) para nomes de pacotes, mini-apps e chaves.
3. **Timeout e kill**: subprocess com timeout máximo (30s leitura, 5min ação). Kill automático ao exceder.
4. **Audit log**: cada comando logado com timestamp, usuário, comando, args, status, stdout truncado.
5. **Ações sensíveis**: exigem token de confirmação ou execução local pelo usuário.
6. **Vault**: chave de descriptografia mantida apenas em memória do backend; nunca persistida em disco.

## Sequência de Implementação Sugerida

1. Layout + Sidebar + navegação (não depende de backend)
2. Overview com `cockpit doctor` (leitura JSON)
3. Packages Manager com busca fuzzy e ações seguras
4. Mini-Apps Console (já temos script nativo; precisa de monitor e logs SSE)
5. KB Explorer (grafo + busca + preview)
6. Vault Manager (segurança crítica; deixar por último)

## Critérios de Aceitação Gerais

- [ ] Navegação entre telas funciona sem recarregar a página (SPA).
- [ ] Todos os comandos executados pelo backend estão na whitelist.
- [ ] Ações sensíveis exigem confirmação explícita.
- [ ] Logs de ações disponíveis em audit log.
- [ ] Cobertura de testes >= 90% no backend.
- [ ] Frontend funciona em 1280x720 sem scroll horizontal.

---

## Detalhamento dos Pontos para Refinamento

### 🟩 1. Layout & Design System (Tailwind CSS)
*   **Decisões fechadas:**
    *   Framework: SvelteKit 5.
    *   Estilização: Tailwind CSS 3 + CSS variables para tokens.
    *   Componentes: classes utilitárias Tailwind; usar `bits-ui` apenas para acessibilidade complexa.
    *   Sidebar em mobile: vira drawer acionado por botão em telas < 768px.
    *   Paleta inicial: Slate para fundos, Violet para primário, Emerald/Rose para status.

### 🟩 2. Visão Geral (Overview) & Diagnósticos
*   **Decisões fechadas:**
    *   `cockpit doctor` deve suportar saída JSON (`--json`) consumida pelo backend via `CommandExecutor`.
    *   Formato do JSON de diagnóstico:
        ```json
        {
          "check_name": "Vault Access",
          "status": "warning|ok|error",
          "message": "Vault is locked",
          "fixable": true,
          "fix_command": "cockpit vault unlock"
        }
        ```
    *   Quick-Fix exige confirmação explícita do usuário e gera audit log.

### 🟩 3. Gerenciador de Pacotes & Registry
*   **Decisões fechadas:**
    *   Rotas da API:
        *   `GET /api/v1/packages` -> Lista pacotes locais.
        *   `GET /api/v1/registry` -> Lista pacotes disponíveis.
        *   `POST /api/v1/packages/install` -> Inicia instalação; retorna `job_id`.
    *   Instalações/upgrades longos usam SSE com `job_id` para progresso.
    *   Nomes de pacotes validados por regex `^[a-z0-9_-]+$`.

### 🟩 4. Vault & Segurança Local
*   **Decisões fechadas:**
    *   Chave de descriptografia mantida apenas em memória do backend (Python). Nunca em disco, localStorage ou sessionStorage.
    *   Auto-lock padrão: 5 minutos de inatividade.
    *   Clipboard: segredo apagado automaticamente após 15 segundos.
    *   Backend utiliza `cryptography.fernet` para criptografia.

### 🟩 5. Visualizador de Base de Conhecimento (KB)
*   **Decisões fechadas:**
    *   Grafo: D3.js v7 + SVG renderizado em Svelte.
    *   Escalabilidade: layout force-directed até 500 nós; virtualização acima disso.
    *   Links e órfãs: parser local de front-matter e links Markdown em `~/.cockpit/kb/`.

### 🟩 6. Console de Processos (Mini-Apps)
*   **Decisões fechadas:**
    *   Gerenciamento: backend Python executa subprocessos controlados; PIDs armazenados em `~/.cockpit/workspace/mini-apps/<name>/.pids/`.
    *   Stream de logs: SSE (`/api/v1/mini-apps/{name}/logs/stream`) lendo arquivos de log.
    *   Métricas: `psutil` no backend para CPU/RAM dos processos filhos.
