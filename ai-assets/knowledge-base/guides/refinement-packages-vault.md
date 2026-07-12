# Refinamento Parte 2: Gerenciador de Pacotes & Vault/Segurança

Este documento detalha os componentes, ações de usuário e possíveis evoluções para as telas de **Gerenciador de Pacotes** e **Vault & Segurança**.

---

## 3. Gerenciador de Pacotes (Packages Manager)

Esta tela gerencia os pacotes instalados e permite a interação com o catálogo central (Registry).

### A. Componentes Necessários
1.  **Navegação de Abas (Tabs):**
    *   *Pacotes Instalados:* Visualização do inventário atual da máquina.
    *   *Registry (Catálogo):* Lista de pacotes disponíveis para instalação.
2.  **Barra de Filtros & Pesquisa:**
    *   Campo de texto dinâmico (com busca fuzzy) para filtrar pelo nome ou descrição do pacote.
    *   Filtros rápidos (dropdown ou chips): "Atualizações Disponíveis", "Categoria", "Dependências".
3.  **Grade/Tabela de Pacotes:**
    *   Exibição do Nome, Versão Local, Versão Recomendada (Registry) e badges de status (ex: `Atualizado`, `Upgrade Pendente`).
4.  **Drawer Lateral (Painel de Detalhes):**
    *   Disparado ao selecionar um pacote.
    *   Exibe a lista física de arquivos gerenciados na máquina local.
    *   Lista de dependências necessárias e arquivos de logs da última instalação/atualização.

### B. Ações Disponíveis
*   **Instalar/Desinstalar Pacote:** Botões diretos para alterar o estado do pacote local.
*   **Upgrade com Um Clique:** Ação destacada em pacotes com atualizações disponíveis, disparando o comando de atualização.
*   **Forçar Re-sincronização:** Atualizar cache local com dados mais recentes do Registry.
*   **Verificação de Integridade:** Checar se os arquivos do pacote local coincidem com o hash original (detecção de adulterações).

### C. API do Backend (Contrato)

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/v1/packages` | Lista pacotes instalados (cache do backend). |
| `GET` | `/api/v1/registry` | Lista pacotes disponíveis no registry. |
| `POST` | `/api/v1/packages/install` | Inicia instalação assíncrona; retorna `job_id`. |
| `POST` | `/api/v1/packages/uninstall` | Inicia desinstalação assíncrona; retorna `job_id`. |
| `POST` | `/api/v1/packages/upgrade` | Inicia upgrade assíncrono; retorna `job_id`. |
| `GET` | `/api/v1/jobs/{job_id}` | Consulta status do job. |
| `GET` | `/api/v1/jobs/{job_id}/stream` | SSE com progresso do job. |

### D. Segurança
*   Apenas comandos `cockpit pkg install|uninstall|upgrade` whitelistados.
*   Nome do pacote validado por regex: `^[a-z0-9_-]+$`.
*   Jobs de ação exigem token de confirmação no header `X-Action-Token`.
*   Audit log registra início, fim e erro de cada job.

### E. Evoluções Futuras
*   **Rollback de Versões:** Lista de versões históricas em cache local permitindo reverter o pacote instantaneamente caso surja um bug.
*   **Registries Múltiplos:** Configuração na interface para adicionar URLs de registries privados corporativos além do oficial.

### F. Critérios de Aceitação
- [ ] Busca fuzzy filtra nome e descrição em tempo real.
- [ ] Instalação/desinstalação exibem progresso via SSE.
- [ ] Apenas comandos whitelist podem ser executados.
- [ ] Audit log acessível via endpoint `/api/v1/audit`.

---

## 4. Vault & Segurança (Vault Manager)

Gerenciamento visual do cofre criptografado de credenciais locais.

### A. Componentes Necessários
1.  **Tela de Autenticação (Lock Screen):**
    *   Exibida de forma obstrutiva caso o Vault esteja fechado.
    *   Formulário centralizado com foco automático no campo de senha mestra.
2.  **Visualizador de Credenciais:**
    *   Tabela com as chaves cadastradas (IDs).
    *   Valores mascarados por padrão (ex: `••••••••••••`).
3.  **Formulário de Cadastro/Edição (Modal):**
    *   Campos: Chave (Unique ID), Valor e Descrição opcional.
4.  **Indicador Visual de Sessão (Timer):**
    *   Mostrador discreto no topo indicando quanto tempo resta antes do bloqueio automático (Auto-Lock).

### B. Ações Disponíveis
*   **Desbloquear/Bloquear Cofre:** Controle principal de acesso aos dados.
*   **Revelar/Ocultar Valor:** Ícone de olho em cada linha para ver o segredo na interface.
*   **Copiar Seguro:** Copia o valor para a área de transferência e o apaga automaticamente após 15 segundos.
*   **Exclusão Segura:** Botão para deletar a chave do Vault com confirmação em duas etapas.

### C. Arquitetura de Segurança
1. **Chave mestra:** inserida pelo usuário no frontend e enviada ao backend apenas para desbloqueio.
2. **Chave de descriptografia:** mantida apenas em memória do backend (RAM). Nunca persistida em disco, localStorage ou sessionStorage.
3. **Backend Python:** utiliza `cryptography.fernet` para criptografar/descriptografar valores. Vault file armazenado em `~/.cockpit/vault`.
4. **Auto-lock:** timer no backend limpa a chave da memória após período de inatividade (padrão 5 min).
5. **Comunicação:** HTTPS recomendado; em desenvolvimento, manter em localhost.

### D. API do Backend (Contrato)

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `POST` | `/api/v1/vault/unlock` | Desbloqueia vault com senha mestra. |
| `POST` | `/api/v1/vault/lock` | Bloqueia vault e limpa chave da memória. |
| `GET` | `/api/v1/vault/status` | Retorna `locked`/`unlocked`. |
| `GET` | `/api/v1/vault/items` | Lista chaves cadastradas (valores mascarados). |
| `POST` | `/api/v1/vault/items` | Adiciona nova chave. |
| `PUT` | `/api/v1/vault/items/{id}` | Edita chave. |
| `DELETE` | `/api/v1/vault/items/{id}` | Remove chave. |
| `POST` | `/api/v1/vault/items/{id}/reveal` | Retorna valor descriptografado (log audit). |

### E. Evoluções Futuras
*   **Gerador de Senhas Integrado:** Auxílio visual para criar senhas robustas no momento do cadastro de novas credenciais.
*   **Configuração de Timeout:** Opção para o usuário ajustar o tempo de Auto-Lock da sessão (ex: 2 min, 5 min, 15 min ou desativado).
*   **Backup do Vault:** Exportação del arquivo `.vault` criptografado por uma segunda senha chave para salvamento externo seguro.

### F. Critérios de Aceitação
- [ ] Vault bloqueado por padrão ao iniciar dashboard.
- [ ] Valores sempre mascarados na lista; reveal gera log audit.
- [ ] Chave mestra nunca persistida em disco.
- [ ] Auto-lock funciona após inatividade configurada.
- [ ] Copiar segredo limpa clipboard após 15 segundos.
