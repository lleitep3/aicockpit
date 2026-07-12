# Cockpit Dashboard: Logs & Insights

Página de análise dos logs e métricas de execução dos comandos do cockpit.

## Fonte de Dados

- Métricas estruturadas: `~/.cockpit/metrics.json`
- Logs JSON rotacionados: `~/.cockpit/logs/cockpit-YYYY-MM-DD.log`

Ambos são gerados pelo `internal/logging/manager.go` do AICockpit.

## Endpoints da API

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/v1/logs/insights` | Insights agregados (total, sucesso, erros, comandos, duração, timeline). |
| `GET` | `/api/v1/logs/metrics` | Métricas brutas de execução. |
| `GET` | `/api/v1/logs` | Entradas recentes dos arquivos de log. |

## Insights Gerados

1. **KPIs**: total de execuções, taxa de sucesso, erros, duração média.
2. **Timeline**: atividade diária separada por sucesso/erro.
3. **Comandos**: ranking de comandos mais executados e duração média.
4. **Erros**: tipos de erro mais frequentes.
5. **Comandos mais lentos**: ranking por duração média e máxima.

## Componentes do Frontend

- Página `/logs` em SvelteKit.
- Sidebar com navegação entre Visão Geral e Logs & Insights.
- Cards de KPI, gráfico de timeline, listas de comandos/erros e tabela de comandos lentos.

## Testes

- Backend: `app/backend/tests/`
- Cobertura atual: ~93%.
- Comando: `pytest tests/ -v --cov=app --cov-report=term-missing --cov-fail-under=90`
