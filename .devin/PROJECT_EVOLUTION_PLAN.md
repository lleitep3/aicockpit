# 📋 Plano de Evolução do Pacote de Projetos - AICockpit

**Data**: 25 de Julho de 2026  
**Status**: Proposta de Implementação  
**Versão**: 1.0

---

## 🎯 Visão Geral

Evoluir o pacote de projetos do AICockpit para manter rastreamento automático de todas as atividades realizadas e sincronizar tarefas bidirecionalmente com issues do GitHub (suportando múltiplos repositórios vinculados ao projeto).

### Objetivos Principais
1. **Tracking Automático**: Registrar automaticamente todas as ações realizadas no projeto
2. **Sincronização Inteligente**: Sincronizar tarefas com múltiplos repositórios GitHub
3. **Histórico Completo**: Manter histórico detalhado de todas as mudanças
4. **Inteligência de IA**: Permitir que IAs entendam o contexto completo do projeto

---

## 📊 Estado Atual da Arquitetura

### Estrutura Existente
```
internal/project/
├── model.go           # Definições de Task, Metadata, Project
├── manager.go         # Operações CRUD e sincronização básica
├── manager_test.go    # Testes do manager
└── model_test.go      # Testes do model

cmd/project.go         # CLI commands para projeto
```

### Capacidades Atuais
- ✅ Criar/listar/gerenciar projetos
- ✅ Kanban board com colunas customizáveis
- ✅ Adicionar/mover/reordenar tasks
- ✅ Sincronização básica com GitHub (1 repo por task)
- ✅ Tracking log manual (`project track`)
- ✅ Vinculação de repositórios, workspaces, KBs

### Limitações Atuais
- ❌ Sem tracking automático de ações
- ❌ Sincronização manual apenas
- ❌ Sem suporte a múltiplos repos por projeto
- ❌ Sem histórico de mudanças estruturado
- ❌ Sem contexto de IA sobre o que foi feito
- ❌ Sem detecção de conflitos de sincronização
- ❌ Sem webhooks para sincronização em tempo real

---

## 🚀 Plano de Implementação

### **FASE 1: Infraestrutura de Tracking (Semana 1)**

#### Feature 1.1: Activity Log System
**Descrição**: Sistema estruturado para registrar todas as atividades do projeto

**Implementação**:
```go
// internal/project/activity.go - NOVO
type ActivityType string

const (
    ActivityTaskCreated    ActivityType = "task_created"
    ActivityTaskMoved      ActivityType = "task_moved"
    ActivityTaskUpdated    ActivityType = "task_updated"
    ActivityTaskDeleted    ActivityType = "task_deleted"
    ActivitySyncStarted    ActivityType = "sync_started"
    ActivitySyncCompleted  ActivityType = "sync_completed"
    ActivitySyncFailed     ActivityType = "sync_failed"
    ActivityProjectUpdated ActivityType = "project_updated"
    ActivityRepoAdded      ActivityType = "repo_added"
    ActivityRepoRemoved    ActivityType = "repo_removed"
)

type Activity struct {
    ID          string                 `yaml:"id"`
    Type        ActivityType           `yaml:"type"`
    Timestamp   time.Time              `yaml:"timestamp"`
    UserAgent   string                 `yaml:"user_agent"`      // "cli" | "dashboard" | "webhook"
    Description string                 `yaml:"description"`
    Changes     map[string]interface{} `yaml:"changes,omitempty"`
    TaskID      string                 `yaml:"task_id,omitempty"`
    Error       string                 `yaml:"error,omitempty"`
}

type ActivityLog struct {
    Activities []Activity `yaml:"activities"`
}
```

**Critérios de Aceite**:
- [ ] Estrutura `Activity` e `ActivityLog` definida
- [ ] Métodos para adicionar atividades (`AddActivity`)
- [ ] Métodos para listar atividades com filtros (`GetActivities`, `GetActivitiesByType`)
- [ ] Persistência em arquivo YAML separado (`project-activities.yaml`)
- [ ] Testes com cobertura >= 90%
- [ ] Sem breaking changes na API existente

---

#### Feature 1.2: Automatic Activity Tracking
**Descrição**: Interceptar todas as operações e registrar automaticamente

**Implementação**:
- Modificar `Manager.AddTask()` para registrar `ActivityTaskCreated`
- Modificar `Manager.MoveTask()` para registrar `ActivityTaskMoved`
- Modificar `Manager.SaveProject()` para detectar mudanças e registrar `ActivityProjectUpdated`
- Criar helper `logActivity(proj *Project, activity Activity)` reutilizável

**Critérios de Aceite**:
- [ ] Todas as operações CRUD registram atividades
- [ ] Timestamp preciso (até milissegundos)
- [ ] User agent detectado corretamente (CLI vs Dashboard)
- [ ] Mudanças capturadas em `Changes` map (antes/depois)
- [ ] Testes verificam logging automático
- [ ] Cobertura >= 90%

---

#### Feature 1.3: Activity Query & Reporting
**Descrição**: Consultar e gerar relatórios de atividades

**Implementação**:
```go
// Novos métodos em Manager
func (m *Manager) GetActivities(slug string, opts ActivityQueryOptions) ([]Activity, error)
func (m *Manager) GetActivityStats(slug string) (ActivityStats, error)
func (m *Manager) ExportActivityReport(slug string, format string) (string, error)

type ActivityQueryOptions struct {
    StartDate  time.Time
    EndDate    time.Time
    Types      []ActivityType
    TaskID     string
    Limit      int
    Offset     int
}

type ActivityStats struct {
    TotalActivities  int
    ActivitiesByType map[ActivityType]int
    LastActivity     time.Time
    MostActiveTask   string
}
```

**Critérios de Aceite**:
- [ ] Filtrar atividades por data, tipo, task
- [ ] Gerar estatísticas de atividades
- [ ] Exportar em JSON e Markdown
- [ ] Comando CLI: `cockpit project activity list <slug>`
- [ ] Comando CLI: `cockpit project activity stats <slug>`
- [ ] Testes com cobertura >= 90%

---

### **FASE 2: Multi-Repository Sync (Semana 2)**

#### Feature 2.1: Repository Configuration
**Descrição**: Suportar múltiplos repositórios com mapeamento de tasks

**Implementação**:
```go
// Estender model.go
type RepositoryConfig struct {
    URL           string   `yaml:"url"`
    Owner         string   `yaml:"owner"`
    Repo          string   `yaml:"repo"`
    DefaultLabels []string `yaml:"default_labels,omitempty"`
    IsDefault     bool     `yaml:"is_default"`
    SyncEnabled   bool     `yaml:"sync_enabled"`
}

type TaskRepositoryMapping struct {
    TaskID      string `yaml:"task_id"`
    RepoURL     string `yaml:"repo_url"`
    IssueNumber int    `yaml:"issue_number"`
    IssueURL    string `yaml:"issue_url"`
    SyncedAt    time.Time `yaml:"synced_at"`
}

// Estender Metadata
type Metadata struct {
    // ... campos existentes ...
    RepositoriesConfig []RepositoryConfig `yaml:"repositories_config"`
    TaskMappings       []TaskRepositoryMapping `yaml:"task_mappings"`
}
```

**Critérios de Aceite**:
- [ ] Estrutura `RepositoryConfig` suporta múltiplos repos
- [ ] Mapeamento 1:N entre tasks e repositórios
- [ ] Validação de URLs e credenciais GitHub
- [ ] Comando CLI: `cockpit project repo configure <slug> <url>`
- [ ] Comando CLI: `cockpit project repo list <slug>`
- [ ] Testes com cobertura >= 90%

---

#### Feature 2.2: Intelligent Sync Engine
**Descrição**: Sincronizar tarefas com múltiplos repositórios inteligentemente

**Implementação**:
```go
// internal/project/sync.go - NOVO
type SyncStrategy string

const (
    SyncStrategyPushOnly    SyncStrategy = "push_only"    // Local → GitHub
    SyncStrategyPullOnly    SyncStrategy = "pull_only"    // GitHub → Local
    SyncStrategyBidirectional SyncStrategy = "bidirectional" // Ambos
)

type SyncConflict struct {
    TaskID      string
    RepoURL     string
    LocalState  string
    RemoteState string
    Resolution  string // "local_wins" | "remote_wins" | "manual"
}

type SyncResult struct {
    TaskID         string
    RepoURL        string
    Success        bool
    Message        string
    Conflict       *SyncConflict
    SyncedAt       time.Time
}

func (m *Manager) SyncProjectWithRepositories(slug string, strategy SyncStrategy) ([]SyncResult, error)
func (m *Manager) ResolveSyncConflict(slug string, conflict SyncConflict, resolution string) error
func (m *Manager) GetSyncStatus(slug string) (SyncStatus, error)
```

**Critérios de Aceite**:
- [ ] Sincronizar task com múltiplos repos simultaneamente
- [ ] Detectar conflitos (mudanças simultâneas)
- [ ] Estratégias de resolução configuráveis
- [ ] Comando CLI: `cockpit project sync <slug> --strategy bidirectional`
- [ ] Comando CLI: `cockpit project sync status <slug>`
- [ ] Testes com cobertura >= 90%
- [ ] Logging detalhado de cada sincronização

---

#### Feature 2.3: Conflict Resolution & Rollback
**Descrição**: Gerenciar conflitos e permitir rollback de sincronizações

**Implementação**:
- Manter histórico de sincronizações antes/depois
- Detectar conflitos automaticamente
- Permitir rollback de última sincronização
- Notificar usuário sobre conflitos

**Critérios de Aceite**:
- [ ] Detectar conflitos de sincronização
- [ ] Estratégias de resolução: local_wins, remote_wins, manual
- [ ] Comando CLI: `cockpit project sync resolve <slug> <task-id>`
- [ ] Comando CLI: `cockpit project sync rollback <slug>`
- [ ] Testes com cobertura >= 90%

---

### **FASE 3: Real-time Sync & Webhooks (Semana 3)**

#### Feature 3.1: Webhook Support
**Descrição**: Receber notificações do GitHub em tempo real

**Implementação**:
```go
// internal/project/webhook.go - NOVO
type WebhookEvent struct {
    Action      string    `json:"action"`
    Issue       GitHubIssue `json:"issue"`
    Repository  GitHubRepo `json:"repository"`
    Timestamp   time.Time `json:"timestamp"`
}

type WebhookHandler struct {
    manager *Manager
    secret  string
}

func (wh *WebhookHandler) HandleGitHubWebhook(payload []byte, signature string) error
func (wh *WebhookHandler) ProcessIssueEvent(event WebhookEvent) error
```

**Critérios de Aceite**:
- [ ] Receber webhooks do GitHub
- [ ] Validar assinatura HMAC
- [ ] Processar eventos de issue (opened, closed, edited)
- [ ] Atualizar projeto automaticamente
- [ ] Registrar webhook events em activity log
- [ ] Testes com cobertura >= 90%

---

#### Feature 3.2: Background Sync Service
**Descrição**: Sincronizar periodicamente em background

**Implementação**:
```go
// internal/project/background_sync.go - NOVO
type SyncScheduler struct {
    manager    *Manager
    interval   time.Duration
    ticker     *time.Ticker
    done       chan bool
}

func NewSyncScheduler(manager *Manager, interval time.Duration) *SyncScheduler
func (ss *SyncScheduler) Start() error
func (ss *SyncScheduler) Stop() error
func (ss *SyncScheduler) SyncAll() error
```

**Critérios de Aceite**:
- [ ] Sincronizar todos os projetos periodicamente
- [ ] Configurável via config.yaml
- [ ] Logging de sincronizações automáticas
- [ ] Tratamento de erros robusto
- [ ] Testes com cobertura >= 90%

---

### **FASE 4: AI Context & Intelligence (Semana 4)**

#### Feature 4.1: Project Context Generator
**Descrição**: Gerar contexto estruturado para IAs

**Implementação**:
```go
// internal/project/ai_context.go - NOVO
type AIContext struct {
    ProjectSummary     string
    RecentActivities   []Activity
    TaskStatus         map[string]int // status -> count
    SyncStatus         string
    Conflicts          []SyncConflict
    Recommendations    []string
    LastUpdated        time.Time
}

func (m *Manager) GenerateAIContext(slug string) (AIContext, error)
func (m *Manager) ExportContextAsMarkdown(slug string) (string, error)
func (m *Manager) ExportContextAsJSON(slug string) (string, error)
```

**Critérios de Aceite**:
- [ ] Gerar contexto estruturado do projeto
- [ ] Incluir atividades recentes
- [ ] Incluir status de sincronização
- [ ] Incluir conflitos pendentes
- [ ] Gerar recomendações baseadas em padrões
- [ ] Exportar em Markdown e JSON
- [ ] Testes com cobertura >= 90%

---

#### Feature 4.2: Activity-based Insights
**Descrição**: Gerar insights baseados em histórico de atividades

**Implementação**:
```go
// Novos métodos em Manager
func (m *Manager) GetProjectInsights(slug string) (ProjectInsights, error)
func (m *Manager) GetTaskTimeline(slug string, taskID string) ([]Activity, error)
func (m *Manager) GetProductivityMetrics(slug string) (ProductivityMetrics, error)

type ProjectInsights struct {
    MostActiveTask      string
    AverageTaskDuration time.Duration
    SyncSuccessRate     float64
    ConflictFrequency   float64
    Recommendations     []string
}

type ProductivityMetrics struct {
    TasksCompletedToday    int
    TasksCompletedThisWeek int
    AverageTimeToComplete  time.Duration
    SyncFrequency          int // per day
}
```

**Critérios de Aceite**:
- [ ] Calcular métricas de produtividade
- [ ] Identificar padrões de atividade
- [ ] Gerar recomendações automáticas
- [ ] Comando CLI: `cockpit project insights <slug>`
- [ ] Testes com cobertura >= 90%

---

#### Feature 4.3: Dashboard Integration
**Descrição**: Integrar tracking com dashboard visual

**Implementação**:
- Expor endpoints de atividades via API
- Mostrar timeline de atividades no dashboard
- Mostrar status de sincronização em tempo real
- Mostrar insights e recomendações

**Critérios de Aceite**:
- [ ] API endpoints para atividades
- [ ] Timeline visual no dashboard
- [ ] Status de sincronização em tempo real
- [ ] Insights e recomendações visíveis
- [ ] Testes com cobertura >= 90%

---

## 📋 Resumo de Features

| # | Feature | Fase | Complexidade | Estimativa |
|---|---------|------|--------------|-----------|
| 1.1 | Activity Log System | 1 | Média | 2-3 dias |
| 1.2 | Automatic Tracking | 1 | Média | 2-3 dias |
| 1.3 | Activity Queries | 1 | Média | 2-3 dias |
| 2.1 | Multi-Repo Config | 2 | Alta | 3-4 dias |
| 2.2 | Intelligent Sync | 2 | Alta | 4-5 dias |
| 2.3 | Conflict Resolution | 2 | Alta | 3-4 dias |
| 3.1 | Webhook Support | 3 | Média | 2-3 dias |
| 3.2 | Background Sync | 3 | Média | 2-3 dias |
| 4.1 | AI Context | 4 | Média | 2-3 dias |
| 4.2 | Activity Insights | 4 | Média | 2-3 dias |
| 4.3 | Dashboard Integration | 4 | Média | 2-3 dias |

**Total Estimado**: 4-5 semanas

---

## 🏗️ Arquitetura Proposta

```
internal/project/
├── model.go                    # Task, Metadata, Project (ESTENDIDO)
├── manager.go                  # CRUD + Sync (ESTENDIDO)
├── activity.go                 # Activity Log (NOVO)
├── sync.go                      # Multi-repo Sync (NOVO)
├── webhook.go                   # GitHub Webhooks (NOVO)
├── background_sync.go           # Sync Scheduler (NOVO)
├── ai_context.go                # AI Context Generator (NOVO)
├── conflict_resolver.go         # Conflict Resolution (NOVO)
├── manager_test.go              # Testes (ESTENDIDO)
├── activity_test.go             # Testes (NOVO)
├── sync_test.go                 # Testes (NOVO)
└── webhook_test.go              # Testes (NOVO)

cmd/project.go                  # CLI Commands (ESTENDIDO)
├── project activity            # Subcommands de atividade
├── project sync                # Subcommands de sincronização
├── project insights            # Subcommands de insights
└── project webhook             # Subcommands de webhooks
```

---

## 🔄 Fluxo de Dados Proposto

```
┌─────────────────────────────────────────────────────────────┐
│                    AICockpit Project                        │
└─────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   ┌─────────┐      ┌──────────┐      ┌──────────────┐
   │ Activity │      │  Task    │      │ Repository   │
   │   Log    │      │ Manager  │      │  Config      │
   └─────────┘      └──────────┘      └──────────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    ┌──────▼──────┐
                    │ Sync Engine  │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   ┌─────────┐      ┌──────────┐      ┌──────────────┐
   │ GitHub   │      │ Webhooks │      │ Background   │
   │   API    │      │ Handler  │      │   Sync       │
   └─────────┘      └──────────┘      └──────────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    ┌──────▼──────┐
                    │ AI Context   │
                    │  Generator   │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   ┌─────────┐      ┌──────────┐      ┌──────────────┐
   │   CLI    │      │ Dashboard│      │   Insights   │
   │ Commands │      │   API    │      │  Generator   │
   └─────────┘      └──────────┘      └──────────────┘
```

---

## 📊 Métricas de Sucesso

### Cobertura de Testes
- ✅ Mínimo 90% de cobertura em todos os novos pacotes
- ✅ Testes unitários para cada feature
- ✅ Testes de integração para sync e webhooks

### Performance
- ✅ Sincronização de até 100 tasks em < 5 segundos
- ✅ Webhook processing em < 1 segundo
- ✅ Activity queries em < 500ms

### Usabilidade
- ✅ Todos os comandos documentados
- ✅ Mensagens de erro claras
- ✅ Exemplos de uso em CLI help

### Qualidade
- ✅ Zero breaking changes
- ✅ Backward compatible
- ✅ Sem secrets em logs

---

## 🔐 Considerações de Segurança

1. **GitHub Token**: Usar variável de ambiente `GITHUB_TOKEN`
2. **Webhook Secret**: Validar assinatura HMAC de webhooks
3. **Conflict Resolution**: Nunca sobrescrever dados sem confirmação
4. **Activity Log**: Não registrar dados sensíveis
5. **Rate Limiting**: Respeitar limites da API GitHub

---

## 📝 Próximos Passos

1. **Aprovação do Plano**: Validar com stakeholders
2. **Setup de Desenvolvimento**: Preparar branches e CI/CD
3. **Implementação Fase 1**: Começar com Activity Log System
4. **Testes Contínuos**: Validar cobertura >= 90%
5. **Documentação**: Atualizar docs conforme progride

---

## 📚 Referências

- [Arquitetura Atual](./docs/architecture/07-project-management.md)
- [Modelo de Projeto](./internal/project/model.go)
- [Manager](./internal/project/manager.go)
- [GitHub API v3](https://docs.github.com/en/rest)
- [Webhooks GitHub](https://docs.github.com/en/developers/webhooks-and-events/webhooks)

---

**Autor**: Devin AI  
**Última Atualização**: 25 de Julho de 2026  
**Status**: Pronto para Implementação
