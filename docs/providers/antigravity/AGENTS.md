# Antigravity Subagents

A abordagem do **Google Antigravity** para sistemas multi-agentes é altamente programática e flexível. O Antigravity oferece ferramentas nativas (tools) injetadas no ambiente do modelo para gerenciar o ciclo de vida e a comunicação entre agentes.

## Ferramentas de Gerenciamento

Os agentes principais possuem acesso a ferramentas específicas:
- `define_subagent`: Define um novo "tipo" de subagente para a conversa, especificando seu `name`, `system_prompt`, `description` e ativando capacidades granulares (`enable_write_tools`, `enable_mcp_tools`, etc).
- `invoke_subagent`: Lança instâncias dos subagentes definidos. Cada invocação cria uma sessão com um `conversationID` único, rodando de forma assíncrona.
- `send_message`: Permite enviar mensagens para agentes rodando em background pelo seu ID, seja para corrigir a rota, pedir status ou enviar novas tarefas a um agente ocioso.
- `manage_subagents`: Lista ou encerra (kill) subagentes ativos.

## Modos de Workspace
Ao invocar um subagente, o Antigravity permite definir o comportamento do sistema de arquivos:
- `inherit` (padrão): O subagente trabalha no mesmo diretório que o pai.
- `branch`: O subagente recebe um clone isolado (novo branch) para experimentar sem quebrar o código principal.
- `share`: Semelhante ao git worktree, compartilha o repositório base mas com capacidade de checkout independente.

## Como Funcionam
A comunicação com o modelo é reativa (*Reactive Wakeup*). Quando o subagente termina sua tarefa ou encontra um bloqueio, ele responde. O modelo principal é "acordado" com essa mensagem na sua timeline e pode continuar o trabalho sem precisar fazer "polling" em loop.

## Referência Oficial
- [Antigravity Subagents](https://antigravity.google/docs/cli-subagents)
