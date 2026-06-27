# Devin Workflows

No Devin, não existe um artefato puramente rotulado como "Workflow" em sua base nativa, mas o conceito é amplamente suportado através da composição de ferramentas, skills e regras.

## Como o AICockpit Mapeia Workflows no Devin
Para orquestrar passos complexos e reprodutíveis no Devin, o adaptador do AICockpit compila fluxos de trabalho usando duas estratégias principais:

1. **Skills como Workflows:** A melhor forma de definir um fluxo de trabalho passo-a-passo no Devin é empacotá-lo como uma [Skill](SKILLS.md). O `SKILL.md` atua como o manual do workflow, descrevendo os gatilhos (quando rodar) e os passos sequenciais rigorosos que o agente deve tomar.
2. **Regras de Repositório (Project Rules):** Para workflows comportamentais contínuos (ex: "sempre que for testar, primeiro rode lint, depois build, depois teste"), o cockpit pode injetar essas instruções diretamente no `AGENTS.md` ou arquivo de prompt de projeto do Devin.
3. **Skills como Orquestradores de Subagentes:** Skills podem orquestrar múltiplos subagentes, cada um executando uma parte do workflow em paralelo, com a skill principal sintetizando os resultados.

## Implementando Workflows como Skills

### Workflow Sequencial Simples

```markdown
---
name: deploy-staging
description: Deploy the current branch to staging environment
allowed-tools:
  - read
  - exec
  - grep
---

Deploy the current branch to staging:

1. Ensure all tests pass: `npm test`
2. Build the project: `npm run build`
3. Run linter: `npm run lint`
4. Check for uncommitted changes: `git status`
5. Deploy to staging: `npm run deploy:staging`
6. Verify deployment health checks

Report the status of each step. If anything fails, stop and explain the issue.
```

### Workflow com Orquestração de Subagentes

```markdown
---
name: full-health-check
description: Comprehensive project health check
---

Perform a full health check on this project:

1. First, use the /research-changes skill to understand recent changes
2. Then, use the /validate-tests skill to verify the test suite
3. Finally, use the /security-audit skill to check for vulnerabilities
4. Synthesize all findings into a comprehensive report
```

## Casos de Uso

### Deployment Automático
Um workflow/skill que dita como gerar a build, autenticar nos servidores, transferir arquivos e invalidar o cache.

### Troubleshooting Padrão
Passos sequenciais de como investigar falhas (olhar logs do docker, checar portas, validar variáveis de ambiente).

### Code Review Systemático
Workflow que executa múltiplas verificações: lint, testes, segurança, performance e estilo, gerando um relatório consolidado.

### Onboarding de Novos Desenvolvedores
Skill que guia o agente através da configuração do ambiente local, instalação de dependências e setup de ferramentas de desenvolvimento.

### Release Process
Workflow orquestrado que verifica branch protection, roda testes, gera changelog, cria tag e executa deploy em múltiplos ambientes.

## Melhores Práticas para Workflows

1. **Passos Claros e Verificáveis:** Cada passo deve ter um critério de sucesso claro e verificável.
2. **Tratamento de Erros:** Defina o que fazer em caso de falha em cada passo (continuar, parar, rollback).
3. **Logging Adequado:** Cada passo deve reportar seu status claramente.
4. **Idempotência:** Workflows devem ser seguros para rodar múltiplas vezes.
5. **Argumentos:** Use argumentos para parametrizar workflows (ex: `/deploy staging` vs `/deploy production`).
6. **Subagentes para Tarefas Paralelas:** Use subagentes para executar tarefas independentes em paralelo quando possível.
