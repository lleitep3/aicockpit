# Devin Rules

O Devin, além de resolver tickets e codificar autonomamente, requer guias de estilo e diretrizes do projeto para não desviar do padrão estabelecido pela equipe. Suas regras são declaradas como contexto e memórias globais ou de projeto.

**Referência Oficial:** [Devin Documentation](https://docs.devin.ai/)

## Diretórios e Arquivos de Regras
- **Global (Windsurf / Devin Desktop):** `~/.codeium/windsurf/memories/global_rules.md`
- **Project-level:** `.devin/AGENTS.md` ou arquivo referenciado como `.devinrules`.

## Formato do Arquivo
Arquivo de texto em Markdown (`.md`). Não há schemas rígidos (como YAML frontmatter exigido nas skills), mas a clareza textual dita o quão bem o agente interpretará as restrições.

## Boas Práticas
1. **Exemplos de "Good vs Bad":** O Devin aprende muito rápido por contrastes. Forneça exemplos práticos de como ele deve fazer algo versus como ele **não** deve fazer.
2. **Prioridade Global vs Local:** Use as regras globais (`global_rules.md`) estritamente para preferências do usuário (ex: "Sempre responda em PT-BR" ou "Use caveman mode"). Use o `AGENTS.md` do projeto para convenções (ex: "Aqui usamos lint no formato X" ou "A versão do Go é a 1.26").
3. **Seções Claras:** Separe suas regras por temas usando cabeçalhos markdown (ex: `## Estilo de Código`, `## Testes`, `## CI/CD`).

## Exemplo de Regra

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
