# Goose Subagents

No ecossistema do Goose, os subagentes são uma forma de *Context Engineering*. Eles representam instâncias independentes focadas em executar tarefas isoladas para manter a conversa principal (e a janela de contexto) enxuta e focada.

## Como Funcionam
Pense neles como assistentes temporários que realizam um trabalho específico. O Goose adota subagentes para lidar com "isolation" e "context preservation", ou seja, garantir que a execução de um script gigante que emite milhares de linhas de log não afogue a memória do agente principal.

- **Isolamento de Processo:** Eles operam em sua própria thread/sessão.
- **Delegação:** O Goose principal pode "chamar" um subagente, passar o prompt/tarefa e aguardar a resposta sumarizada.
- **Gerenciamento de Contexto:** Como o contexto LLM é limitado, o uso de subagentes evita o problema de "perda de foco" em conversas muito longas.

## Implementação
A integração com o Goose pode ser feita via MCPs (Model Context Protocol) que expõem ferramentas de "delegação", ou ativando rotinas de sub-receitas (*subrecipes*). O desenvolvedor também pode usar o CLI do Goose para instanciar tarefas secundárias atreladas a uma sessão pai.

## Casos de Uso
- Compilação e análise de erros extensos.
- Busca e raspagem de dados web pesados.
- Testes end-to-end automatizados onde apenas o relatório de falha/sucesso importa.

## Referência Oficial
- [Goose Subagents Guide](https://goose-docs.ai/docs/guides/context-engineering/subagents)
