# Refinamento Parte 3: Base de Conhecimento (KB) & Mini-Apps

Este documento detalha os componentes, ações de usuário e possíveis evoluções para as telas de **Base de Conhecimento (KB)** e **Console de Mini-Apps**.

---

## 5. Base de Conhecimento (KB Explorer)

Esta tela é focada na visualização e gerenciamento das notas e documentações locais do desenvolvedor.

### A. Componentes Necessários
1.  **Visualizador de Grafo Interativo (Graph Canvas):**
    *   Renderização 2D/3D interativa de nós (notas/artigos) e linhas (links de referência).
    *   *Destaques visuais:* Nós coloridos conforme o status (ex: vermelho para notas órfãs ou notas com links quebrados).
2.  **Lista e Tabela de Notas:**
    *   Visualização tabulada com colunas: Título do Artigo, Data de Modificação, Tags, Quantidade de Conexões (In/Out) e Status de Compilação (Compilado/Erro).
3.  **Filtros & Busca Fuzzy:**
    *   Barra de busca para consulta em tempo real no conteúdo e título das notas.
    *   Seletor de tags para filtragem em lote.
4.  **Preview Drawer (Painel de Pré-visualização):**
    *   Gaveta lateral que exibe o artigo formatado em HTML estático (com visualização idêntica à final) sem fechar o painel de busca.

### B. Ações Disponíveis
*   **Criar Novo Artigo:** Formulário simplificado para criar rapidamente um arquivo Markdown com metadados/front-matter corretos.
*   **Auditar Base de Conhecimento:** Ação para varrer os links internos e relatar referências quebradas.
*   **Recompilar Artigos:** Botão para forçar a regeneração estática dos arquivos HTML de saída.

### C. Decisão de Tecnologia do Grafo
*   **Biblioteca:** D3.js v7 + SVG renderizado em Svelte.
*   **Motivação:** leve, controle total de layout, sem dependência de React.
*   **Layout:** force-directed simples para < 500 nós; virtualização se ultrapassar.

### D. API do Backend (Contrato)

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/v1/kb` | Lista documentos com metadados. |
| `GET` | `/api/v1/kb/graph` | Retorna nós e arestas para o grafo. |
| `GET` | `/api/v1/kb/search?q={query}` | Busca fuzzy em título e conteúdo. |
| `GET` | `/api/v1/kb/{id}/preview` | Retorna HTML preview do artigo. |
| `POST` | `/api/v1/kb/audit` | Varre links internos e retorna quebrados/órfãos. |
| `POST` | `/api/v1/kb/rebuild` | Recompila HTML estático dos artigos. |
| `POST` | `/api/v1/kb` | Cria novo artigo Markdown. |

### E. Evoluções Futuras
*   **Editor Markdown WYSIWYG:** Editor visual completo integrado no próprio dashboard com suporte a pré-visualização lado-a-lado.
*   **Recomendações Inteligentes:** Sugestões de links internos utilizando similaridade textual local para enriquecer o grafo de conhecimento.

### F. Critérios de Aceitação
- [ ] Grafo renderiza nós e arestas interativamente.
- [ ] Busca fuzzy retorna resultados em < 200ms para < 1000 docs.
- [ ] Preview drawer abre sem sair da tela de busca.
- [ ] Auditoria detecta links quebrados e notas órfãs.

---

## 6. Console de Mini-Apps (Background Processes)

Monitoramento e controle de execução dos mini-apps locais do desenvolvedor.

### A. Componentes Necessários
1.  **Cards de Mini-Apps (Process Grid):**
    *   Cartões individuais contendo: Nome do mini-app, Uptime (tempo ativo), Porta local (ex: `:8080`), Status (`Running`, `Stopped`, `Failing`) e Consumo Estimado de CPU/RAM.
2.  **Terminal de Logs Integrado:**
    *   Fundo simulando terminal monousuário (`bg-black text-green-400 font-mono`) com rolagem automática para visualização dos fluxos `stdout` e `stderr` do processo selecionado.
3.  **Indicador de Conectividade/Porta:**
    *   Sinalizador de status HTTP (Health Check) indicando se o serviço na porta mapeada está respondendo a requisições com código 200 OK.

### B. Ações Disponíveis
*   **Controle de Ciclo de Vida:** Botões rápidos de `Start`, `Stop` e `Restart`.
*   **Navegação Rápida:** Botão para abrir o mini-app ativo na porta local correspondente em uma nova aba do navegador.
*   **Limpar Histórico de Logs:** Ação visual para limpar a visualização de logs do terminal embutido.

### C. Decisão de Tecnologia
*   **Monitoramento:** backend lê `.pids` e logs do workspace (`~/.cockpit/workspace/mini-apps/<name>/`).
*   **Stream de logs:** Server-Sent Events (SSE) endpoint `/api/v1/mini-apps/{name}/logs` lê `frontend.log` e `backend.log` em tempo real.
*   **Métricas:** `psutil` no backend para CPU/RAM dos processos filhos (PID do `.pids`).
*   **Ações:** backend executa subprocessos controlados para start/stop/restart; nomes whitelistados.

### D. API do Backend (Contrato)

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/v1/mini-apps` | Lista mini-apps e status. |
| `GET` | `/api/v1/mini-apps/{name}` | Detalhes de um mini-app. |
| `POST` | `/api/v1/mini-apps/{name}/start` | Inicia mini-app. |
| `POST` | `/api/v1/mini-apps/{name}/stop` | Para mini-app. |
| `POST` | `/api/v1/mini-apps/{name}/restart` | Reinicia mini-app. |
| `GET` | `/api/v1/mini-apps/{name}/logs/stream` | SSE de logs. |
| `GET` | `/api/v1/mini-apps/{name}/metrics` | CPU/RAM do processo. |

### E. Evoluções Futuras
*   **Orquestrador de Múltiplas Instâncias:** Permitir subir mais de uma cópia do mesmo mini-app rodando em portas distintas simultaneamente.
*   **Limitação de Recursos na UI:** Interface de controle para limitar uso de CPU ou alocação máxima de memória por sub-processo executado.

### F. Critérios de Aceitação
- [ ] Lista mostra status, porta, uptime e CPU/RAM.
- [ ] Logs atualizam em tempo real via SSE.
- [ ] Start/stop/restart funcionam sem travar a UI.
- [ ] Health check reflete se a porta responde HTTP 200.
