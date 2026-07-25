# 💡 Exemplos de Uso - Evolução do Pacote de Projetos

Demonstrações práticas de como as novas features funcionarão após implementação completa.

---

## 📌 Cenário 1: Criar Projeto com Multi-Repo

### Antes (Atual)
```bash
# Criar projeto
cockpit project create my-project --title "My Project" --description "A cool project"

# Adicionar um repositório
cockpit project repo add my-project https://github.com/user/repo1

# Adicionar task
cockpit project task add my-project "Implement feature X"
```

### Depois (Proposto)
```bash
# Criar projeto
cockpit project create my-project --title "My Project" --description "A cool project"

# Configurar múltiplos repositórios
cockpit project repo configure my-project https://github.com/user/repo1 --default
cockpit project repo configure my-project https://github.com/user/repo2
cockpit project repo configure my-project https://github.com/user/repo3

# Listar repositórios configurados
cockpit project repo list my-project
# Output:
# 📦 Repositórios do projeto 'my-project':
# - https://github.com/user/repo1 (DEFAULT, sync_enabled)
# - https://github.com/user/repo2 (sync_enabled)
# - https://github.com/user/repo3 (sync_enabled)

# Adicionar task
cockpit project task add my-project "Implement feature X"

# Sincronizar com todos os repos
cockpit project sync my-project --strategy bidirectional
# Output:
# 🔄 Sincronizando projeto 'my-project' com 3 repositórios...
# ✅ repo1: 5 tasks sincronizadas
# ✅ repo2: 3 tasks sincronizadas
# ✅ repo3: 2 tasks sincronizadas
# ✅ Sincronização concluída em 2.3s
```

---

## 📊 Cenário 2: Rastreamento Automático de Atividades

### Antes (Atual)
```bash
# Adicionar task
cockpit project task add my-project "Fix bug #123"

# Mover task manualmente
cockpit project task move my-project TASK-1234567890 inProgress

# Registrar manualmente o progresso
cockpit project track my-project "Começou a trabalhar no bug"
cockpit project track my-project "Encontrou a causa raiz"
cockpit project track my-project "Implementou a solução"
```

### Depois (Proposto)
```bash
# Adicionar task (automático: registra ActivityTaskCreated)
cockpit project task add my-project "Fix bug #123"
# 🔔 Activity logged: task_created

# Mover task (automático: registra ActivityTaskMoved)
cockpit project task move my-project TASK-1234567890 inProgress
# 🔔 Activity logged: task_moved (todo → inProgress)

# Sincronizar com GitHub (automático: registra ActivitySyncStarted/Completed)
cockpit project sync my-project
# 🔔 Activity logged: sync_started
# 🔔 Activity logged: sync_completed

# Visualizar histórico de atividades
cockpit project activity list my-project --limit 10
# Output:
# 📋 Atividades do projeto 'my-project':
# 
# 2026-07-25 14:32:15 | sync_completed | Sincronização com 3 repos concluída
# 2026-07-25 14:32:10 | sync_started   | Iniciando sincronização...
# 2026-07-25 14:31:45 | task_moved     | Task TASK-1234567890: todo → inProgress
# 2026-07-25 14:31:30 | task_created   | Task TASK-1234567890: Fix bug #123
# 2026-07-25 14:30:15 | project_updated| Repositório adicionado

# Visualizar estatísticas de atividades
cockpit project activity stats my-project
# Output:
# 📊 Estatísticas de Atividades - 'my-project':
# 
# Total de atividades: 47
# 
# Por tipo:
#   task_created:     12
#   task_moved:       18
#   sync_completed:   10
#   sync_started:     5
#   project_updated:  2
# 
# Última atividade: 2026-07-25 14:32:15
# Task mais ativa: TASK-1234567890 (8 atividades)

# Exportar relatório de atividades
cockpit project activity export my-project --format markdown --output report.md
# Cria: report.md com histórico completo formatado
```

---

## 🔄 Cenário 3: Sincronização Inteligente com Detecção de Conflitos

### Situação: Mudanças Simultâneas

```bash
# Cenário: Você muda uma task localmente enquanto alguém muda no GitHub

# Local: Move task para "done"
cockpit project task move my-project TASK-123 done

# Simultaneamente no GitHub: Alguém fecha a issue como "closed"

# Tentar sincronizar
cockpit project sync my-project --strategy bidirectional
# Output:
# ⚠️  Conflito detectado!
# 
# Task: TASK-123 "Fix bug #123"
# Local state:  done
# Remote state: closed
# 
# Opções:
# 1. local_wins   - Usar estado local (done)
# 2. remote_wins  - Usar estado remoto (closed)
# 3. manual       - Resolver manualmente
# 
# Escolha uma estratégia (padrão: manual):

# Resolver conflito
cockpit project sync resolve my-project TASK-123 --resolution local_wins
# ✅ Conflito resolvido: local_wins
# 🔄 Sincronizando...
# ✅ Sincronização concluída

# Ver status de sincronização
cockpit project sync status my-project
# Output:
# 🔄 Status de Sincronização - 'my-project':
# 
# Última sincronização: 2026-07-25 14:35:22
# Status: ✅ Sucesso
# 
# Repositórios:
# - repo1: ✅ Sincronizado (5 tasks)
# - repo2: ✅ Sincronizado (3 tasks)
# - repo3: ⚠️  1 conflito pendente
# 
# Conflitos pendentes: 1
# Taxa de sucesso: 99.2%

# Rollback da última sincronização (se necessário)
cockpit project sync rollback my-project
# ✅ Sincronização revertida para estado anterior
```

---

## 🤖 Cenário 4: Contexto para IA

### Antes (Atual)
```bash
# Devin precisa entender o projeto manualmente
cockpit project info my-project
# Mostra apenas metadados básicos
```

### Depois (Proposto)
```bash
# Gerar contexto completo para IA
cockpit project context my-project --format json > project-context.json

# Conteúdo do project-context.json:
{
  "project": {
    "id": "my-project",
    "title": "My Project",
    "description": "A cool project",
    "repositories": [
      "https://github.com/user/repo1",
      "https://github.com/user/repo2",
      "https://github.com/user/repo3"
    ]
  },
  "recent_activities": [
    {
      "timestamp": "2026-07-25T14:32:15Z",
      "type": "sync_completed",
      "description": "Sincronização com 3 repos concluída",
      "changes": {
        "synced_tasks": 10,
        "conflicts_resolved": 1
      }
    },
    {
      "timestamp": "2026-07-25T14:31:45Z",
      "type": "task_moved",
      "task_id": "TASK-1234567890",
      "changes": {
        "from": "todo",
        "to": "inProgress"
      }
    }
  ],
  "task_status": {
    "todo": 5,
    "inProgress": 8,
    "done": 12
  },
  "sync_status": {
    "last_sync": "2026-07-25T14:32:15Z",
    "success_rate": 0.992,
    "pending_conflicts": 1
  },
  "recommendations": [
    "Resolver 1 conflito pendente em repo3",
    "Task TASK-1234567890 está em progresso há 3 horas",
    "Sincronizar com repo2 (última sincronização há 2 horas)"
  ]
}

# Exportar contexto em Markdown para documentação
cockpit project context my-project --format markdown > PROJECT_STATUS.md

# Conteúdo do PROJECT_STATUS.md:
# # Projeto: My Project
# 
# ## Resumo
# A cool project
# 
# ## Repositórios
# - https://github.com/user/repo1
# - https://github.com/user/repo2
# - https://github.com/user/repo3
# 
# ## Status de Tasks
# - ✅ Done: 12 tasks
# - 🔄 In Progress: 8 tasks
# - 📋 To Do: 5 tasks
# 
# ## Atividades Recentes
# - 2026-07-25 14:32:15 | Sincronização com 3 repos concluída
# - 2026-07-25 14:31:45 | Task movida para inProgress
# 
# ## Recomendações
# - Resolver 1 conflito pendente em repo3
# - Task está em progresso há 3 horas
```

---

## 📈 Cenário 5: Insights e Métricas de Produtividade

### Visualizar Insights
```bash
# Gerar insights do projeto
cockpit project insights my-project

# Output:
# 📊 Insights - 'my-project':
# 
# 📈 Produtividade
#   Tasks completadas hoje:      3
#   Tasks completadas esta semana: 12
#   Tempo médio para completar:   2.5 horas
#   Taxa de conclusão:            70.6%
# 
# 🎯 Atividade
#   Task mais ativa:              TASK-1234567890 (12 atividades)
#   Frequência de sincronização:  4 vezes por dia
#   Última sincronização:         há 5 minutos
# 
# 🔄 Sincronização
#   Taxa de sucesso:              99.2%
#   Conflitos resolvidos:         2
#   Tempo médio de sync:           2.3 segundos
# 
# 💡 Recomendações
#   ✓ Excelente taxa de sincronização!
#   ✓ Produtividade em alta
#   ⚠️  Resolver 1 conflito pendente
#   ⚠️  Revisar task TASK-1234567890 (em progresso há 3h)

# Timeline de uma task específica
cockpit project timeline my-project TASK-1234567890

# Output:
# 📅 Timeline - Task TASK-1234567890 'Fix bug #123':
# 
# 2026-07-25 14:31:30 | ✏️  Criada
#                      | Descrição: Fix bug #123
# 
# 2026-07-25 14:31:45 | 🔄 Movida para inProgress
#                      | Sincronizada com repo1 (issue #456)
# 
# 2026-07-25 14:45:20 | 📝 Atualizada
#                      | Descrição: Fix bug #123 - found root cause
# 
# 2026-07-25 15:00:00 | 🔄 Sincronizada
#                      | Conflito detectado em repo2
# 
# 2026-07-25 15:05:00 | ✅ Resolvida
#                      | Movida para done
#                      | Sincronizada com todos os repos
# 
# Duração total: 33 minutos 30 segundos
```

---

## 🔔 Cenário 6: Webhooks em Tempo Real

### Configurar Webhooks
```bash
# Configurar webhook do GitHub
cockpit project webhook setup my-project

# Output:
# 🔔 Configurando webhook para 'my-project'...
# 
# Webhook URL: https://your-domain.com/webhooks/github
# Secret: ••••••••••••••••••••••••••••••••
# 
# Adicionando webhook aos repositórios:
# ✅ repo1: Webhook adicionado
# ✅ repo2: Webhook adicionado
# ✅ repo3: Webhook adicionado
# 
# ✅ Webhooks configurados com sucesso!

# Listar webhooks ativos
cockpit project webhook list my-project

# Output:
# 🔔 Webhooks Ativos - 'my-project':
# 
# repo1 (https://github.com/user/repo1)
#   Status: ✅ Ativo
#   Última entrega: 2026-07-25 15:10:22
#   Eventos: issues, pull_requests
# 
# repo2 (https://github.com/user/repo2)
#   Status: ✅ Ativo
#   Última entrega: 2026-07-25 15:08:15
#   Eventos: issues
# 
# repo3 (https://github.com/user/repo3)
#   Status: ✅ Ativo
#   Última entrega: 2026-07-25 15:05:30
#   Eventos: issues, pull_requests

# Quando alguém fecha uma issue no GitHub:
# 🔔 Webhook recebido!
# 📥 Processando evento: issue.closed
# 🔄 Atualizando task TASK-456...
# ✅ Task sincronizada automaticamente
# 📝 Activity logged: sync_completed (via webhook)
```

---

## 🔧 Cenário 7: Sincronização em Background

### Configurar Sincronização Automática
```bash
# Ativar sincronização automática
cockpit project sync auto my-project --interval 5m

# Output:
# ✅ Sincronização automática ativada para 'my-project'
# ⏱️  Intervalo: 5 minutos
# 🔄 Próxima sincronização: 2026-07-25 15:20:00

# Ver status de sincronização automática
cockpit project sync auto status

# Output:
# 🔄 Status de Sincronização Automática:
# 
# Projetos com sync automático: 3
# 
# my-project
#   Intervalo: 5 minutos
#   Última sincronização: 2026-07-25 15:15:22
#   Próxima sincronização: 2026-07-25 15:20:22
#   Status: ✅ Ativo
# 
# other-project
#   Intervalo: 10 minutos
#   Última sincronização: 2026-07-25 15:10:15
#   Próxima sincronização: 2026-07-25 15:20:15
#   Status: ✅ Ativo

# Desativar sincronização automática
cockpit project sync auto my-project --disable

# Output:
# ✅ Sincronização automática desativada para 'my-project'
```

---

## 🎨 Cenário 8: Integração com Dashboard

### No Dashboard SvelteKit
```
┌─────────────────────────────────────────────────────┐
│ 🎯 My Project                                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│ 📊 Status                                           │
│ ├─ Tasks: 25 total                                  │
│ │  ├─ ✅ Done: 12                                   │
│ │  ├─ 🔄 In Progress: 8                             │
│ │  └─ 📋 To Do: 5                                   │
│ │                                                   │
│ ├─ Sincronização: ✅ 99.2% sucesso                  │
│ └─ Última atualização: há 5 minutos                 │
│                                                     │
│ 📈 Insights                                         │
│ ├─ Produtividade: 🟢 Excelente                      │
│ ├─ Taxa de conclusão: 70.6%                         │
│ └─ Tempo médio: 2.5 horas                           │
│                                                     │
│ 📋 Kanban Board                                     │
│ ├─ [To Do]          [In Progress]    [Done]         │
│ │  ┌──────────┐     ┌──────────┐     ┌──────────┐  │
│ │  │ Task 1   │     │ Task 5   │     │ Task 10  │  │
│ │  │ #123     │     │ #456     │     │ #789     │  │
│ │  └──────────┘     └──────────┘     └──────────┘  │
│ │  ┌──────────┐     ┌──────────┐     ┌──────────┐  │
│ │  │ Task 2   │     │ Task 6   │     │ Task 11  │  │
│ │  │ #124     │     │ #457     │     │ #790     │  │
│ │  └──────────┘     └──────────┘     └──────────┘  │
│                                                     │
│ 🔔 Atividades Recentes                              │
│ ├─ 15:10 | Task 5 movida para Done                  │
│ ├─ 15:08 | Sincronização concluída (3 repos)       │
│ ├─ 15:05 | Task 6 criada                            │
│ └─ 15:00 | Conflito resolvido em repo2              │
│                                                     │
│ ⚙️  Ações                                            │
│ ├─ [Sincronizar Agora]                              │
│ ├─ [Ver Histórico]                                  │
│ ├─ [Resolver Conflitos]                             │
│ └─ [Exportar Contexto]                              │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## 🔐 Cenário 9: Segurança e Auditoria

### Auditoria Completa
```bash
# Exportar log de auditoria
cockpit project audit my-project --since "2026-07-20" --format json

# Output (JSON):
{
  "project": "my-project",
  "period": {
    "start": "2026-07-20T00:00:00Z",
    "end": "2026-07-25T23:59:59Z"
  },
  "audit_log": [
    {
      "timestamp": "2026-07-25T14:32:15Z",
      "action": "sync_completed",
      "user_agent": "cli",
      "details": {
        "repos_synced": 3,
        "tasks_updated": 10,
        "conflicts_resolved": 1
      }
    },
    {
      "timestamp": "2026-07-25T14:31:45Z",
      "action": "task_moved",
      "user_agent": "dashboard",
      "details": {
        "task_id": "TASK-1234567890",
        "from": "todo",
        "to": "inProgress"
      }
    }
  ],
  "summary": {
    "total_actions": 47,
    "by_type": {
      "task_created": 12,
      "task_moved": 18,
      "sync_completed": 10,
      "sync_started": 5,
      "project_updated": 2
    }
  }
}

# Verificar integridade de sincronização
cockpit project verify my-project

# Output:
# ✅ Verificação de Integridade - 'my-project':
# 
# Local vs Remote:
#   repo1: ✅ Sincronizado (5 tasks)
#   repo2: ✅ Sincronizado (3 tasks)
#   repo3: ⚠️  1 diferença detectada
# 
# Histórico de Sincronização:
#   Últimas 10 sincronizações: ✅ Todas bem-sucedidas
#   Taxa de sucesso: 99.2%
#   Conflitos não resolvidos: 1
# 
# Recomendações:
#   ⚠️  Resolver conflito em repo3
#   ✓ Sincronização em bom estado
```

---

## 📚 Cenário 10: Documentação Automática

### Gerar Documentação do Projeto
```bash
# Gerar documentação completa do projeto
cockpit project docs my-project --output ./PROJECT_DOCS

# Cria estrutura:
# PROJECT_DOCS/
# ├── README.md              # Visão geral
# ├── ARCHITECTURE.md        # Arquitetura do projeto
# ├── TASKS.md              # Lista de tasks com status
# ├── TIMELINE.md           # Timeline de atividades
# ├── SYNC_STATUS.md        # Status de sincronização
# ├── INSIGHTS.md           # Insights e métricas
# ├── CONFLICTS.md          # Conflitos pendentes
# └── AUDIT_LOG.md          # Log de auditoria

# Conteúdo de TASKS.md:
# # Tasks - My Project
# 
# ## Status Overview
# - ✅ Done: 12 tasks
# - 🔄 In Progress: 8 tasks
# - 📋 To Do: 5 tasks
# 
# ## To Do (5)
# - [ ] TASK-001: Feature A
# - [ ] TASK-002: Feature B
# - [ ] TASK-003: Bug fix C
# 
# ## In Progress (8)
# - [x] TASK-004: Feature D (started 3h ago)
# - [x] TASK-005: Feature E (started 1h ago)
# 
# ## Done (12)
# - [x] TASK-006: Feature F (completed 2h ago)
# - [x] TASK-007: Feature G (completed 1h ago)
```

---

## 🎓 Resumo de Benefícios

Com as novas features implementadas:

✅ **Rastreamento Automático**: Não precisa registrar manualmente  
✅ **Multi-Repo**: Gerenciar múltiplos repositórios simultaneamente  
✅ **Sincronização Inteligente**: Detecta e resolve conflitos  
✅ **Webhooks**: Atualizações em tempo real do GitHub  
✅ **Contexto para IA**: Devin entende o projeto completamente  
✅ **Insights**: Métricas e recomendações automáticas  
✅ **Auditoria**: Histórico completo de tudo que foi feito  
✅ **Dashboard**: Visualização completa e intuitiva  

---

**Versão**: 1.0  
**Última Atualização**: 25 de Julho de 2026
