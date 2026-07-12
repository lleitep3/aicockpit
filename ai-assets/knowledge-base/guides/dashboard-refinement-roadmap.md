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
*   **Decisões a tomar:**
    *   Usar React + Tailwind CSS puro ou adotar componentes primitivos (ex: Radix UI / Headless UI)?
    *   Definir tokens de design exatos (paleta Slate vs. Zinc, tons de destaque como Violet ou Emerald).
    *   Comportamento da Sidebar em dispositivos móveis (drawer acionado por botão vs. ocultação completa).

### 🟩 2. Visão Geral (Overview) & Diagnósticos
*   **Decisões a tomar:**
    *   Como a CLI do Cockpit expõe os resultados do `doctor` para a interface web? (Requer endpoint HTTP retornando JSON estruturado).
    *   *Formato do JSON de Diagnóstico:*
        ```json
        {
          "check_name": "Vault Access",
          "status": "warning|ok|error",
          "message": "Vault is locked",
          "fixable": true,
          "fix_command": "cockpit vault unlock"
        }
        ```
    *   Implementação do botão "Quick-Fix" (executar comando corretivo de forma segura na máquina).

### 🟩 3. Gerenciador de Pacotes & Registry
*   **Decisões a tomar:**
    *   Mapeamento de rotas da API:
        *   `GET /api/packages` -> Lista pacotes locais.
        *   `GET /api/registry` -> Lista pacotes disponíveis.
        *   `POST /api/packages/install` -> Instala um pacote.
    *   Como lidar com tempo de execução longo de instalações/upgrades? (Usar Server-Sent Events - SSE ou polling de status).

### 🟩 4. Vault & Segurança Local
*   **Decisões a tomar:**
    *   *Armazenamento do token de descriptografia:* Como garantir que a chave do Vault nunca toque no disco ou localStorage do browser? (Salvar apenas em estado do React na memória).
    *   *Auto-lock:* Tempo máximo de inatividade antes de limpar o estado e bloquear o cofre na UI (ex: 5 minutos).
    *   Tratamento de clipboard: Apagar dado copiado após X segundos por segurança.

### 🟩 5. Visualizador de Base de Conhecimento (KB)
*   **Decisões a tomar:**
    *   Escolha da tecnologia para o grafo de conexões: SVG nativo manipulado por D3-force, ou bibliotecas dedicadas como React Flow / Vis.js?
    *   Como o grafo lida com escalabilidade (ex: +500 notas)?
    *   Lógica do analisador de links quebrados e notas órfãs (leitura local do front-matter das notas Markdown).

### 🟩 6. Console de Processos (Mini-Apps)
*   **Decisões a tomar:**
    *   Como o Cockpit gerencia os processos em background (Go `os/exec` ou gerenciador de processos dedicado)?
    *   *Streaming de logs:* Protocolo do WebSocket para transmissão contínua de stdout/stderr de cada mini-app.
    *   Medição de recursos: Como obter o consumo de CPU/RAM de um processo filho em Go sem sobrecarregar a máquina local?
