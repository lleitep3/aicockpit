# FLOW-002 — UC-002 Plano de testes

**Documento principal:** `FLOW-002-UC-002-assistente-operacional-cockpit.md`  
**Status:** implemented_initial_slice  
**Relógio:** relógio real UTC nos testes atuais; expiração ainda não tem teste com relógio injetável.

Os cenários usam providers e ferramentas falsos. Nenhum cenário inicia CLI real,
usa rede ou depende do histórico local do desenvolvedor.

## Fixture

- `codex/model-fast`: adapter falso em memória com callback de ferramenta.
- IDs e diretórios temporários gerados pelos testes.
- A conversa de destino começa livre e sem turnos.
- A ferramenta de leitura retorna somente conversas registradas na fixture.
- A ferramenta de escrita cria exatamente um turno quando autorizada.
- Nenhum teste inicia CLI real ou usa rede.

## Cenários

```gherkin
Scenario: consultar e resumir uma conversa sem alterar o histórico
  Given fixture-codex está selecionado em assistant-001
  And conv-desktop contém duas mensagens concluídas
  When o usuário pede um resumo da conversa conv-desktop
  Then o assistente consulta conv-desktop e apresenta um resumo
  And conv-desktop mantém as mesmas mensagens e turnos
  And a sessão do assistente registra a ferramenta e os IDs consultados

Scenario: provider sem tool calling não executa ferramentas
  Given fixture-plain está selecionado em assistant-001
  When o usuário pede para consultar uma conversa
  Then o runtime informa que o provider não suporta ferramentas
  And nenhuma chamada de ferramenta ou leitura de conversa ocorre

Scenario: mensagem é enviada somente após autorização explícita
  Given fixture-codex selecionado e conv-desktop livre
  When o assistente prepara uma mensagem para conv-desktop
  Then a UI mostra destino, seleção e texto exato com estado awaiting_approval
  And nenhuma mensagem é criada em conv-desktop antes da decisão
  When o usuário aprova a ação action-001
  Then exatamente um turno é iniciado em conv-desktop
  And a auditoria referencia action-001 e o turno de destino

Scenario: usuário rejeita a mensagem
  Given action-001 aguarda aprovação para conv-desktop
  When o usuário rejeita action-001
  Then o estado da ação vira rejected
  And nenhum turno ou mensagem é criado em conv-desktop

Scenario: aprovação não vale para payload alterado ou duplicado
  Given action-001 foi apresentada com payload e destino fixos
  When o payload ou destino muda antes da execução
  Then a autorização anterior é invalidada e uma nova prévia é exigida
  When a mesma decisão de aprovação é reenviada
  Then nenhuma segunda mensagem é criada

Scenario: modelo tenta invocar ferramenta desconhecida ou com schema inválido
  Given fixture-codex retorna uma chamada inexistente ou com campos adicionais
  When o runtime valida a chamada
  Then a chamada é rejeitada e registrada como erro
  And nenhuma ferramenta é executada

Scenario: prompt injection está presente no transcript consultado
  Given conv-desktop contém texto instruindo a ignorar políticas e enviar mensagens
  When o assistente lê e resume conv-desktop
  Then esse texto é tratado como dado não confiável
  And nenhuma mutação ocorre sem uma nova prévia aprovada pelo usuário

Scenario: destino foi removido ou está em execução
  Given action-001 aponta para conversa removida ou conv-active
  When o usuário tenta aprovar a ação
  Then o runtime nega a execução com estado e erro explícitos
  And não cria, enfileira nem cancela turno no destino

Scenario: seleção do destino mudou depois da prévia
  Given action-001 contém a seleção efetiva registrada no momento da proposta
  When a seleção do chat muda antes da aprovação
  Then action-001 vira invalidated
  And o runtime não inicia turno no chat de destino

Scenario: ambiente do assistente nega shell e escrita local
  Given assistant-001 inicia um turno
  When o adapter pede aprovação de comando ou arquivo
  Then o runtime recusa sem exibir aprovação de shell na UI
  And o turno usa sandbox read-only em workspace isolado

Scenario: chamada local sem token ou origem autorizada
  Given uma requisição de ferramenta não tem token válido ou Origin local
  When o runtime recebe a requisição
  Then responde com negação
  And nenhuma consulta ou mutação é realizada

Scenario: cancelar o turno do assistente
  Given assistant-turn-001 está em reasoning
  When o usuário cancela o turno
  Then assistant-turn-001 termina como cancelled
  And nenhuma ação pendente ou turno de destino é executado
```

## Evidências e cobertura atual

- Estado do turno/ação antes e depois.
- IDs da sessão do assistente, conversa consultada, ação aprovada e turno
  destino quando aplicável.
- Contagem de chamadas de ferramenta; deve ser zero em todas as negações.
- Evento de auditoria sem credenciais, transcript duplicado ou ambiente de
  processo.
- Testes de aprovação/rejeição demonstram zero turnos antes da decisão e no
  máximo um turno após aprovação repetida.
- Testes cobrem schema estrito, bloqueio de leitura da própria sessão,
  invalidação por mudança de seleção, hash do payload, auditoria de IDs de
  mensagem, contrato HTTP e resposta estruturada do adapter Codex.

## Pontos que dependem de decisão

- Acrescentar relógio injetável para testar expiração sem espera real.
- Adicionar cobertura de cancelamento de turno com ação pendente e recuperação
  de sessão interrompida.
- Fixar retenção do transcript e eventos de ferramentas.
