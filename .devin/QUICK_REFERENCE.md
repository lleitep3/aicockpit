# ⚡ Quick Reference - Evolução do Pacote de Projetos

Guia rápido para consulta durante implementação.

---

## 🎯 Features por Fase

### FASE 1: Tracking (6-9 dias)

```
┌─────────────────────────────────────────────┐
│ 1.1 Activity Log System                     │
├─────────────────────────────────────────────┤
│ Arquivos: activity.go, activity_test.go     │
│ Estruturas: Activity, ActivityLog            │
│ Métodos: AddActivity, GetActivities          │
│ Persistência: project-activities.yaml        │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 1.2 Automatic Tracking                      │
├─────────────────────────────────────────────┤
│ Modificar: manager.go                       │
│ Interceptar: AddTask, MoveTask, SaveProject  │
│ Registrar: Mudanças antes/depois             │
│ Detectar: User agent (CLI vs Dashboard)      │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 1.3 Activity Queries                        │
├─────────────────────────────────────────────┤
│ Métodos: GetActivities, GetActivityStats     │
│ Exportar: JSON, Markdown                     │
│ CLI: project activity list/stats             │
│ Filtros: data, tipo, task                    │
└─────────────────────────────────────────────┘
```

### FASE 2: Multi-Repo (10-14 dias)

```
┌─────────────────────────────────────────────┐
│ 2.1 Repository Configuration                │
├─────────────────────────────────────────────┤
│ Estruturas: RepositoryConfig                │
│            TaskRepositoryMapping             │
│ Modificar: model.go (Metadata)              │
│ CLI: repo configure/list                    │
│ Validação: URLs, credenciais                │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 2.2 Intelligent Sync                        │
├─────────────────────────────────────────────┤
│ Arquivos: sync.go, sync_test.go             │
│ Estratégias: push_only, pull_only, bidi     │
│ Detectar: Conflitos                         │
│ Resultado: SyncResult com status             │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 2.3 Conflict Resolution                     │
├─────────────────────────────────────────────┤
│ Arquivos: conflict_resolver.go              │
│ Métodos: ResolveSyncConflict                │
│ Estratégias: local_wins, remote_wins, manual│
│ Rollback: Última sincronização              │
└─────────────────────────────────────────────┘
```

### FASE 3: Real-time (4-6 dias)

```
┌─────────────────────────────────────────────┐
│ 3.1 Webhook Support                         │
├─────────────────────────────────────────────┤
│ Arquivos: webhook.go, webhook_test.go       │
│ Handler: WebhookHandler                     │
│ Validação: HMAC signature                   │
│ Eventos: issue opened, closed, edited       │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 3.2 Background Sync                         │
├─────────────────────────────────────────────┤
│ Arquivos: background_sync.go                │
│ Classe: SyncScheduler                       │
│ Intervalo: Configurável                     │
│ Logging: Sincronizações automáticas         │
└─────────────────────────────────────────────┘
```

### FASE 4: AI & Insights (6-9 dias)

```
┌─────────────────────────────────────────────┐
│ 4.1 AI Context Generator                    │
├─────────────────────────────────────────────┤
│ Arquivos: ai_context.go, ai_context_test.go│
│ Estrutura: AIContext                        │
│ Exportar: Markdown, JSON                    │
│ Incluir: Atividades, status, conflitos      │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 4.2 Activity Insights                       │
├─────────────────────────────────────────────┤
│ Métodos: GetProjectInsights                 │
│         GetTaskTimeline                     │
│         GetProductivityMetrics              │
│ Métricas: Duração, taxa, frequência         │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 4.3 Dashboard Integration                   │
├─────────────────────────────────────────────┤
│ API endpoints: /activities, /insights       │
│ Timeline: Visual no dashboard               │
│ Status: Sincronização em tempo real         │
│ Recomendações: Visíveis na UI               │
└─────────────────────────────────────────────┘
```

---

## 📁 Estrutura de Arquivos

### Novos Arquivos

```
internal/project/
├── activity.go                    ← NOVO (Fase 1)
├── activity_test.go               ← NOVO (Fase 1)
├── sync.go                         ← NOVO (Fase 2)
├── sync_test.go                    ← NOVO (Fase 2)
├── conflict_resolver.go            ← NOVO (Fase 2)
├── conflict_resolver_test.go       ← NOVO (Fase 2)
├── webhook.go                      ← NOVO (Fase 3)
├── webhook_test.go                 ← NOVO (Fase 3)
├── background_sync.go              ← NOVO (Fase 3)
├── background_sync_test.go         ← NOVO (Fase 3)
├── ai_context.go                   ← NOVO (Fase 4)
└── ai_context_test.go              ← NOVO (Fase 4)
```

### Arquivos Modificados

```
internal/project/
├── model.go                        ← ESTENDIDO (Fase 2)
├── model_test.go                   ← ESTENDIDO (Fase 2)
├── manager.go                      ← ESTENDIDO (Fases 1,2,4)
└── manager_test.go                 ← ESTENDIDO (Fases 1,2,4)

cmd/
└── project.go                      ← ESTENDIDO (Todas fases)
```

---

## 🔑 Tipos Principais

### Activity System

```go
type ActivityType string

const (
    ActivityTaskCreated    = "task_created"
    ActivityTaskMoved      = "task_moved"
    ActivityTaskUpdated    = "task_updated"
    ActivityTaskDeleted    = "task_deleted"
    ActivitySyncStarted    = "sync_started"
    ActivitySyncCompleted  = "sync_completed"
    ActivitySyncFailed     = "sync_failed"
    ActivityProjectUpdated = "project_updated"
    ActivityRepoAdded      = "repo_added"
    ActivityRepoRemoved    = "repo_removed"
)

type Activity struct {
    ID          string
    Type        ActivityType
    Timestamp   time.Time
    UserAgent   string
    Description string
    Changes     map[string]interface{}
    TaskID      string
    Error       string
}
```

### Sync System

```go
type SyncStrategy string

const (
    SyncStrategyPushOnly      = "push_only"
    SyncStrategyPullOnly      = "pull_only"
    SyncStrategyBidirectional = "bidirectional"
)

type SyncResult struct {
    TaskID      string
    RepoURL     string
    Success     bool
    Message     string
    Conflict    *SyncConflict
    SyncedAt    time.Time
}

type SyncConflict struct {
    TaskID      string
    RepoURL     string
    LocalState  string
    RemoteState string
    Resolution  string
}
```

### Repository Config

```go
type RepositoryConfig struct {
    URL           string
    Owner         string
    Repo          string
    DefaultLabels []string
    IsDefault     bool
    SyncEnabled   bool
}

type TaskRepositoryMapping struct {
    TaskID      string
    RepoURL     string
    IssueNumber int
    IssueURL    string
    SyncedAt    time.Time
}
```

---

## 🧪 Cobertura de Testes

### Requisito Mínimo
- ✅ >= 90% cobertura em cada novo arquivo
- ✅ Testes unitários para cada método
- ✅ Testes de integração para sync/webhooks
- ✅ Testes de erro para edge cases

### Checklist por Feature

```
1.1 Activity Log System
  [ ] TestAddActivity
  [ ] TestGetActivities
  [ ] TestGetActivitiesByType
  [ ] TestGetActivityStats
  [ ] TestExportActivityReport
  [ ] TestActivityPersistence
  [ ] TestActivityFilters

1.2 Automatic Tracking
  [ ] TestAutoTrackAddTask
  [ ] TestAutoTrackMoveTask
  [ ] TestAutoTrackSaveProject
  [ ] TestChangeDetection
  [ ] TestUserAgentDetection

1.3 Activity Queries
  [ ] TestActivityList
  [ ] TestActivityStats
  [ ] TestActivityExport
  [ ] TestActivityFiltering
  [ ] TestActivityDateRange

2.1 Repository Configuration
  [ ] TestRepositoryConfig
  [ ] TestTaskRepositoryMapping
  [ ] TestRepositoryValidation
  [ ] TestMultipleRepositories

2.2 Intelligent Sync
  [ ] TestSyncPushOnly
  [ ] TestSyncPullOnly
  [ ] TestSyncBidirectional
  [ ] TestConflictDetection
  [ ] TestSyncResult

2.3 Conflict Resolution
  [ ] TestConflictDetection
  [ ] TestLocalWinsResolution
  [ ] TestRemoteWinsResolution
  [ ] TestManualResolution
  [ ] TestRollback

3.1 Webhook Support
  [ ] TestWebhookValidation
  [ ] TestWebhookProcessing
  [ ] TestIssueEvents
  [ ] TestWebhookSecurity

3.2 Background Sync
  [ ] TestSyncScheduler
  [ ] TestSyncInterval
  [ ] TestSyncAll
  [ ] TestErrorHandling

4.1 AI Context
  [ ] TestContextGeneration
  [ ] TestContextExportJSON
  [ ] TestContextExportMarkdown
  [ ] TestContextAccuracy

4.2 Activity Insights
  [ ] TestProjectInsights
  [ ] TestTaskTimeline
  [ ] TestProductivityMetrics
  [ ] TestRecommendations

4.3 Dashboard Integration
  [ ] TestAPIEndpoints
  [ ] TestTimelineData
  [ ] TestSyncStatus
  [ ] TestInsightsData
```

---

## 📋 Comandos CLI

### Activity Commands

```bash
cockpit project activity list <slug>
cockpit project activity stats <slug>
cockpit project activity export <slug> --format json
cockpit project activity export <slug> --format markdown
```

### Repository Commands

```bash
cockpit project repo configure <slug> <url> --default
cockpit project repo list <slug>
cockpit project repo remove <slug> <url>
```

### Sync Commands

```bash
cockpit project sync <slug> --strategy bidirectional
cockpit project sync status <slug>
cockpit project sync resolve <slug> <task-id> --resolution local_wins
cockpit project sync rollback <slug>
cockpit project sync auto <slug> --interval 5m
cockpit project sync auto status
cockpit project sync auto <slug> --disable
```

### Context Commands

```bash
cockpit project context <slug> --format json
cockpit project context <slug> --format markdown
cockpit project insights <slug>
cockpit project timeline <slug> <task-id>
```

### Webhook Commands

```bash
cockpit project webhook setup <slug>
cockpit project webhook list <slug>
cockpit project webhook remove <slug> <repo-url>
```

---

## 🔍 Debugging

### Logs Importantes

```bash
# Ver logs de sincronização
tail -f ~/.cockpit/logs/project-sync.log

# Ver logs de webhooks
tail -f ~/.cockpit/logs/webhooks.log

# Ver logs de atividades
tail -f ~/.cockpit/logs/activities.log

# Ver arquivo de atividades
cat ~/.cockpit/workspace/projects/my-project-activities.yaml
```

### Variáveis de Ambiente

```bash
# GitHub token (obrigatório)
export GITHUB_TOKEN=ghp_...

# Debug mode
export COCKPIT_DEBUG=1

# Log level
export COCKPIT_LOG_LEVEL=debug

# Webhook secret
export WEBHOOK_SECRET=...
```

---

## 🚨 Erros Comuns

| Erro | Causa | Solução |
|------|-------|---------|
| `GITHUB_TOKEN not found` | Token não configurado | `export GITHUB_TOKEN=...` |
| `invalid repository format` | URL malformada | Usar `owner/repo` ou URL completa |
| `conflict detected` | Mudanças simultâneas | Usar `sync resolve` |
| `rate limit exceeded` | Muitas requisições | Implementar backoff |
| `webhook timeout` | Processamento lento | Usar queue assíncrona |

---

## 📊 Performance Targets

| Operação | Target | Método |
|----------|--------|--------|
| Sync 100 tasks | < 5s | Batch requests |
| Webhook processing | < 1s | Async queue |
| Activity queries | < 500ms | Indexing |
| Context generation | < 2s | Caching |

---

## 🔐 Security Checklist

- [ ] GITHUB_TOKEN em variável de ambiente
- [ ] Webhook secret validado (HMAC)
- [ ] Sem secrets em logs
- [ ] Sem dados sensíveis em activity log
- [ ] Rate limiting implementado
- [ ] Validação de URLs
- [ ] Tratamento de erros seguro

---

## 📚 Referências Rápidas

### GitHub API
- Issues: `GET /repos/{owner}/{repo}/issues`
- Update Issue: `PATCH /repos/{owner}/{repo}/issues/{issue_number}`
- Create Issue: `POST /repos/{owner}/{repo}/issues`

### Go Patterns
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Table-driven tests: `for _, tt := range tests { ... }`
- Dependency injection: `func New(dep Dependency) *Type`

### YAML Marshaling
- Use tags: `yaml:"field_name"`
- Omit empty: `yaml:"field,omitempty"`
- Inline: `yaml:",inline"`

---

## ✅ Pre-Implementation Checklist

- [ ] Revisar PROJECT_EVOLUTION_PLAN.md
- [ ] Revisar IMPLEMENTATION_ROADMAP.md
- [ ] Revisar USAGE_EXAMPLES.md
- [ ] Preparar branch `feature/project-evolution`
- [ ] Configurar CI/CD
- [ ] Preparar ambiente local
- [ ] Revisar código existente
- [ ] Entender GitHub API
- [ ] Validar GITHUB_TOKEN
- [ ] Preparar testes

---

## 🎯 Fase 1 Checklist

### Implementação
- [ ] activity.go criado
- [ ] Activity struct definida
- [ ] ActivityLog struct definida
- [ ] AddActivity implementado
- [ ] GetActivities implementado
- [ ] GetActivityStats implementado
- [ ] Persistência em YAML
- [ ] Testes >= 90%

### Integração
- [ ] manager.go modificado
- [ ] Auto-tracking em AddTask
- [ ] Auto-tracking em MoveTask
- [ ] Auto-tracking em SaveProject
- [ ] CLI commands adicionados
- [ ] Documentação atualizada

### Validação
- [ ] Testes passando
- [ ] Cobertura >= 90%
- [ ] Linting OK
- [ ] Sem breaking changes
- [ ] Exemplos funcionando

---

**Versão**: 1.0  
**Última Atualização**: 25 de Julho de 2026  
**Status**: Pronto para Referência
