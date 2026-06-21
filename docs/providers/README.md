# AICockpit Providers

## Introdução
O AICockpit atua como uma camada agnóstica de orquestração (harness) para múltiplos assistentes e agentes de inteligência artificial. Esta pasta contém as especificações técnicas de como cada provider lida com capacidades fundamentais de IA autônoma.

O objetivo principal desta documentação é consolidar o formato **Canônico** do AICockpit: o desenvolvedor escreve regras e habilidades uma única vez, e o Cockpit as "compila" (unrolls) perfeitamente para a linguagem, estrutura e locais que cada provider específico espera.

## Providers Implementados
* [Devin (Cognition AI)](devin/README.md)
* [Goose (Block)](goose/README.md)
* [Antigravity (Google DeepMind)](antigravity/README.md)

---

## Feature Matrix (Cruzamento de Capacidades)

Abaixo mapeamos os 5 pilares de *Context Engineering* do AICockpit e como eles se traduzem fisicamente para cada provedor.

| Feature (AICockpit) | Conceito Canônico | Devin | Goose | Antigravity |
| --- | --- | --- | --- | --- |
| **Skills** | Capacidades modulares e ferramentas locais com dependências determinísticas. | **Skills**: `[project]/.devin/skills/` com `SKILL.md` (YAML Frontmatter). | **Extensions / MCP**: `~/.config/goose/config.yaml` mapeando servidores MCP. | **Skills**: `~/.gemini/config/skills/` com `SKILL.md` (YAML Frontmatter). |
| **Rules** | Diretrizes de comportamento, regras de projeto e restrições absolutas. | **Project Context**: `AGENTS.md` ou regras injetadas em repositório. | **Hints**: Compiladas para o arquivo de contexto `.goosehints`. | **Rules**: Arquivo unificado `AGENTS.md` em `~/.gemini/config/rules/`. |
| **Permissions** | Restrições de segurança do sistema local (Shell, File System). | **Allowed/Blocked**: Grants inseridos em `.devin/config.local.json`. | **Extension Config**: Mapeamento seguro no `config.yaml`. | **Grants JSON**: Liberadas via `grantedPermissions` no `config.json`. |
| **Subagents** | Delegação e isolamento de processos extensos/repetitivos em background. | **Background Tasks**: Evocados por comandos internos para isolamento. | **Context Preservation**: Tarefas isoladas nativas do motor. | **Agents Registry**: `define_subagent` armazenado em `.agents/`. |
| **Workflows** | Roteiros passo-a-passo e checklists repetitivos (Receitas). | **Workflow Skills**: Roteiros empacotados como Skills no `.devin/skills/`. | **Recipes**: Exportados e evocados como Goose Recipes. | **Workflow Skills**: Empacotados via `workflow-skill-creator`. |

---

## Proposta da Implementação Canônica (Adapter Pattern)

Para que o AICockpit cumpra a promessa de ser *"Escreva uma vez, rode em qualquer agente"*, propomos a seguinte arquitetura de compilação no core do sistema:

### 1. O Formato Universal (AICockpit Assets)
O usuário mantém tudo num formato agnóstico dentro de `~/.cockpit/` (Global) ou `.cockpit/` (Projeto).
- `.cockpit/skills/`: Diretórios com scripts e um `manifest.yaml` agnóstico.
- `.cockpit/rules/`: Arquivos Markdown curtos (ex: `lint_rules.md`).
- `.cockpit/workflows/`: Checklists estruturados em YAML/Markdown.

### 2. A Engine de Compilação (Providers Core)
O `internal/providers` deixará de fazer cópias burras (Copy/Paste) de arquivos. Em vez disso, ele usará **Adapters** implementando uma interface comum:

```go
type Compiler interface {
    CompileSkills(skills []CanonicalSkill) error
    CompileRules(rules []CanonicalRule) error
    CompileWorkflows(workflows []CanonicalWorkflow) error
    CompilePermissions(perms CanonicalPermissions) error
}
```

### 3. Como se Desenrola por Provider (O "Unroll")

#### Para o Devin:
- A interface `CompileRules` vai juntar todos os Markdowns de `rules/` do Cockpit e concatenar em um único `AGENTS.md` (ou `.devinrules`) na raiz do repositório.
- A interface `CompileWorkflows` vai transformar o checklist universal em uma estrutura de pasta compatível com o Devin (`.devin/skills/<workflow_name>/SKILL.md`), pois o Devin entende workflows melhor como Skills.

#### Para o Goose:
- A interface `CompileRules` vai gerar o arquivo `.goosehints`.
- A interface `CompileWorkflows` vai transformar o workflow em uma **Recipe** compatível com o Goose.
- A interface `CompileSkills` vai abstrair os scripts e empacotá-los num mini-servidor MCP local se necessário, ou inseri-los nas configurações suportadas pelo motor.

#### Para o Antigravity:
- A interface unirá os conceitos e escreverá globalmente na pasta `~/.gemini/config/`, injetando os cabeçalhos YAML rigorosos exigidos pelo motor do Google para roteamento semântico.

### Conclusão do Redesign
Em vez da propagação ser guiada pela "pasta de origem", ela passa a ser guiada pela **Feature Canônica**. O adapter recebe a entidade (`Skill`, `Rule`, `Workflow`) do AICockpit e toma a decisão técnica de onde, como e em que formato gravar no ambiente específico daquele LLM.
