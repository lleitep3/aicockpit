---
name: kb-search
description: "MANDATORY: Execute FIRST before starting any task to search the Knowledge Base for context. Also use at the end of a task to document lessons learned."
---

# Habilidade: kb-search

Esta habilidade garante que a IA utilize a Base de Conhecimento (KB) local do desenvolvedor como fonte central de contexto e registro de aprendizados.

## Regras de Comportamento (MANDATORY)

1. **Fase de Pesquisa (Antes de tudo)**: Você DEVE pesquisar os termos principais relacionados ao pedido do usuário usando `cockpit kb search` ANTES de agir ou escrever códigos. Entenda o contexto e se há decisões arquiteturais ou problemas conhecidos.
2. **Fase de Documentação (Após concluir)**: Ao finalizar a tarefa com sucesso, você DEVE avaliar as dificuldades superadas e sugerir proativamente ao usuário a criação de uma nova entrada no KB (Lições Aprendidas) detalhando a solução.

## Como usar a Busca

Execute o seguinte comando de CLI no terminal para realizar uma busca por termos específicos:

```bash
cockpit kb search "seu termo de busca"
```

### Exemplo de Uso (Busca):

Se o desenvolvedor solicitar para depurar as configurações de logs:
1. Execute: `cockpit kb search "logging"`
2. Analise os resultados retornados.
3. Se necessário, abra e leia o arquivo retornado utilizando a ferramenta de leitura de arquivos adequada.

## Como documentar Lições Aprendidas

Ao final da interação, sugira ao usuário que você documente os achados. Sugira a criação de um documento em markdown (para ser indexado com `cockpit kb add`) com o seguinte formato:
- Título do problema/tarefa (ex: 'Como debugar vazamento de memória')
- Resumo do contexto e solução encontrada
- Comandos ou snippets de código úteis
