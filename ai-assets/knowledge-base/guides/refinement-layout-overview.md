# Refinamento Parte 1: Layout, Design System & Visão Geral

Este documento detalha os componentes, ações de usuário e possíveis evoluções para a fundação visual do Cockpit Dashboard e a tela de **Visão Geral (Overview)**.

---

## 1. Layout & Design System

### Decisão de Stack
*   **Framework:** SvelteKit 5.
*   **Estilização:** Tailwind CSS 3 + CSS variables para tokens de cor.
*   **Componentes:** Iniciar com classes utilitárias Tailwind; evitar dependências de UI pesadas. Usar `bits-ui` ou `headlessui` apenas se necessário para acessibilidade.

### A. Componentes Estruturais (Shell)
*   **Sidebar (Navegação Principal):**
    *   *Elementos:* Logo do Cockpit, itens de navegação com ícones (Home, Pacotes, Vault, KB, Mini-Apps), botão de colapsar sidebar (`w-64` para `w-16`), indicador visual da página ativa.
    *   *Estilo Tailwind:* `bg-slate-900 border-r border-slate-800 text-slate-400`.
*   **Top Navbar (Barra Utilitária):**
    *   *Elementos:* Breadcrumbs dinâmicos, barra de busca (com indicador visual `Ctrl + K` para command palette), botão de status rápido do Vault (ícone de cadeado colorido), indicador de conexão local (Online/Offline).
    *   *Estilo Tailwind:* `h-16 border-b border-slate-800 bg-slate-950/80 backdrop-blur-md px-6 flex items-center justify-between`.
*   **Main Wrapper (Área de Conteúdo):**
    *   *Elementos:* Área com scroll independente que renderiza a página selecionada.
    *   *Estilo Tailwind:* `flex-1 overflow-y-auto p-6 bg-slate-950`.

### B. Componentes Atômicos (UI Library)
*   **Card:** Contêiner padrão para widgets com gradiente escuro (`bg-slate-900/40 border border-slate-800 rounded-xl p-6`).
*   **Button:** Variações: Primário (violeta/indigo), Secundário (slate/gray), Perigo (rose/red), Ícone.
*   **Badge:** Rótulo de status (ex: `Atualização` em âmbar, `Ativo` em esmeralda, `Inativo` em slate).
*   **Input/Select:** Campos de texto e seletores adaptados para tema escuro.
*   **Command Palette:** Modal acionado por `Ctrl + K` para busca global por comandos/páginas.

### C. Ações do Sistema Visual
*   **Alternar Menu:** Colapsar sidebar para maximizar espaço de trabalho.
*   **Paleta de Comandos (`Ctrl + K`):** Abrir modal de busca global rápida por atalho.

### D. Evoluções Futuras
*   **Temas Customizados:** Suporte a múltiplos esquemas de cores (ex: Gruvbox, Nord, Dracula).
*   **Layout Modular (Grid Drag & Drop):** Permitir que o desenvolvedor reorganize os cards da página inicial.

### E. Critérios de Aceitação
- [ ] Sidebar navega entre as 6 telas planejadas.
- [ ] Layout responsivo: sidebar vira drawer em telas < 768px.
- [ ] Command palette acessível por `Ctrl + K` e toca.
- [ ] Tokens de cor centralizados em `app.css` usando CSS variables.

---

## 2. Visão Geral (Overview / Home)

Esta página apresenta a situação geral da máquina local em tempo real.

### A. Componentes Necessários
1.  **Grid de KPIs Rápidos (Stat Cards):**
    *   *Card Vault:* Exibe status (Bloqueado/Desbloqueado).
    *   *Card Pacotes:* Exibe total instalado e quantidade de pacotes com atualização disponível.
    *   *Card Mini-Apps:* Exibe total de mini-apps configurados e quantos estão ativos.
    *   *Card KB:* Exibe total de artigos e conexões no grafo.
2.  **Painel Doctor (Diagnósticos):**
    *   Interface limpa listando verificações do sistema (Pastas, Configurações, Permissões, Conexões).
    *   *Estilo:* Lista de itens com ícones de sucesso (`check-circle` verde), aviso (`exclamation-triangle` amarelo) ou falha (`x-circle` vermelho).
3.  **Feed de Atividade Recente:**
    *   Exibe histórico rápido das últimas ações executadas (ex: "Mini-app X iniciado há 5 min", "Pacote Y atualizado às 10:15").

### B. Ações Disponíveis
*   **Bloqueio Rápido do Vault:** Um clique para fechar o cofre e limpar chaves da memória imediatamente.
*   **Executar Diagnóstico Completo:** Botão para forçar nova varredura do ambiente local (`cockpit doctor`).
*   **Correção Automática (Quick Fix):** Em caso de falha corrigível no Doctor, botão ao lado do erro para executar a correção automaticamente.

### C. Execução de Comandos do Cockpit
Para leitura (doctor, status, listagens), o backend usará um `CommandExecutor` whitelistado:
```python
# Exemplo de contrato
allowed = {
    "doctor": ["cockpit", "doctor", "--json"],
    "pkg_list": ["cockpit", "pkg", "list", "--json"],
}
```
*   Comando e argumentos hardcoded no backend.
*   Timeout de 30 segundos.
*   Saída parseada como JSON.
*   Ações sensíveis (quick-fix) exigem token de confirmação gerado pelo backend.

### D. Evoluções Futuras
*   **Gráfico de Uso de Recursos:** Linha de tendência mostrando consumo de CPU e RAM dos processos ativos do Cockpit no último minuto.
*   **Notificações Desktop nativas:** Alertas quando um processo em background falhar ou quando um pacote importante receber update.

### E. Critérios de Aceitação
- [ ] Doctor exibe checks em lista com status ok/warning/error.
- [ ] Quick-fix só executa após confirmação do usuário.
- [ ] KPIs atualizam a cada 30 segundos.
- [ ] Feed de atividade mostra últimas 20 ações do audit log.
