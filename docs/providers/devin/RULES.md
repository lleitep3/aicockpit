# Devin Rules

Regras são instruções persistentes que moldam como o Devin CLI se comporta em seu projeto. Elas são injetadas no contexto do agente no início de cada sessão, garantindo comportamento consistente em toda a equipe.

**Referência Oficial:** [Devin Rules & AGENTS.md](https://docs.devin.ai/cli/extensibility/rules)

> **Nota Importante:** Para melhorar a capacidade de codificação, velocidade de conclusão e reduzir custo, recomendamos fortemente **usar Skills sempre que possível**. Skills são injetadas no contexto apenas quando relevantes. **Regras e AGENTS devem ser mantidas o menor possível.**

## AGENTS.md

A forma mais simples de adicionar regras é com um arquivo `AGENTS.md` na raiz do seu projeto:

```markdown
# Project Rules

- Use TypeScript for all new files
- Follow the existing patterns in src/components/
- Always run `npm run lint` before committing
- Use pnpm, not npm or yarn
- Write tests for all new utility functions
```

O Devin CLI lê este arquivo automaticamente.

## Diretórios e Arquivos de Regras

### Regras de Projeto

| Arquivo | Notas |
|---------|-------|
| `AGENTS.md` | Recomendado |
| `AGENT.md` | Alternativa singular |
| `CLAUDE.md` | Compatível com Claude Code |

Todos estes arquivos são tratados identicamente — seus conteúdos são carregados como regras always-on.

### Regras Globais

Você também pode criar regras que se aplicam a **todos os projetos** colocando um arquivo `AGENTS.md` no diretório de config do usuário:

**Linux / macOS:**
```
~/.config/devin/AGENTS.md
```

**Windows:**
```
%APPDATA%\devin\AGENTS.md
```

Regras globais são carregadas no início de cada sessão, independentemente de qual projeto você está trabalhando. Use-as para preferências pessoais que se aplicam em todo lugar:

```markdown
# My Global Rules

- Always write commit messages in conventional commit format
- Prefer functional patterns over imperative code
- Run tests before suggesting a task is complete
```

Regras globais funcionam junto com regras de projeto — ambas são carregadas e ativas ao mesmo tempo. `AGENT.md` também é suportado neste local.

> **Nota:** Se você usa Claude Code, o Devin CLI também lê `~/.claude/CLAUDE.md` como uma regra global.

### Regras de Outras Ferramentas

O Devin CLI pode ler regras de outras ferramentas de codificação IA:

**Cursor:**
- Lê de `.cursor/rules/*.md` e `.cursor/rules/*.mdc`
- Suporta frontmatter para controlar ativação

**Windsurf:**
- Lê de `.windsurf/rules/*.md` e `.windsurf/global_rules.md`
- Suporta diretórios em múltiplos níveis no projeto
- Suporta frontmatter com diferentes triggers

**Claude Code:**
- Lê do diretório `.claude/`

Você pode habilitar ou desabilitar a leitura de formatos específicos no arquivo de config:

```json
{
  "read_config_from": {
    "cursor": true,
    "windsurf": true,
    "claude": true
  }
}
```

`AGENTS.md` é sempre lido e não pode ser desabilitado.

## Formato do Arquivo

Arquivo de texto em Markdown (`.md`). Não há schemas rígidos (como YAML frontmatter exigido nas skills), mas a clareza textual dita o quão bem o agente interpretará as restrições.

## Tipos de Ativação de Regras

Regras carregadas de formatos externos podem ter diferentes comportamentos de ativação:

| Tipo | Comportamento |
|------|---------------|
| **Always-on** | Ativo em cada sessão, nenhuma ação do usuário necessária |
| **Glob-activated** | Ativo quando o agente trabalha com arquivos que correspondem a padrões específicos |
| **Agent-decided** | O agente escolhe quando aplicar baseado na descrição da regra |
| **User-invocable** | Apenas ativo quando explicitamente acionado pelo usuário |

Regras de `AGENTS.md` são sempre "always-on".

## Boas Práticas
1. **Exemplos de "Good vs Bad":** O Devin aprende muito rápido por contrastes. Forneça exemplos práticos de como ele deve fazer algo versus como ele **não** deve fazer.
2. **Prioridade Global vs Local:** Use as regras globais estritamente para preferências do usuário (ex: "Sempre responda em PT-BR" ou "Use caveman mode"). Use o `AGENTS.md` do projeto para convenções (ex: "Aqui usamos lint no formato X" ou "A versão do Go é a 1.26").
3. **Seções Claras:** Separe suas regras por temas usando cabeçalhos markdown (ex: `## Estilo de Código`, `## Testes`, `## CI/CD`).
4. **Mantenha conciso:** Regras longas e verbosas diluem a atenção do agente. Foque no que mais importa.
5. **Seja específico:** "Use pnpm" é melhor que "use o gerenciador de pacotes certo". Instruções concretas são mais fáceis de seguir.
6. **Inclua exemplos:** Mostre o padrão que você quer, não apenas uma descrição dele.
7. **Version control:** Mantenha regras no seu repo para que toda a equipe se beneficie das mesmas diretrizes.

## Padrão Recomendado

Nosso padrão recomendado é usar uma regra para referenciar skills que o modelo deve usar em cenários particulares:

```markdown
# Project Rules

## Code Review
When reviewing code, use the /code-review skill

## Deployment
When deploying, use the /deploy-staging skill

## Testing
Always run tests before suggesting a task is complete
```

## Exemplo de Regra Completa

**Conteúdo do `AGENTS.md`:**
```markdown
# Regras do Projeto AICockpit

## 1. Commits
- Nunca realize git commit sem antes rodar o comando `make check` localmente para garantir que o CI não vai quebrar.
- Se o `make check` falhar, você é obrigado a corrigir o código antes de tentar commitar novamente.

## 2. Prefixo de Ferramentas
- Todo comando executado no terminal deve ser precedido do prefixo `rtk`.
- Exemplo CORRETO: `rtk git commit -m "feat: xxx"`
- Exemplo ERRADO: `git commit -m "feat: xxx"`

## 3. Qualidade de Código (Go)
```go
// ✓ Correto: Tratar o erro explicitamente
if err != nil {
    return fmt.Errorf("falha ao processar: %w", err)
}

// ✗ Incorreto: Ignorar o erro silenciosamente
_ = processar()
```

## 4. Skills para Usar
- Para criar novos pacotes: use /package-creation
- Para revisar código: use /code-review
- Para deploy: use /deploy-staging
```
