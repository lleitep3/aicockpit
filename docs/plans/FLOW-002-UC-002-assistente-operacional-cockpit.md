# FLOW-002 — Assistente operacional do Cockpit

**Caso de uso:** UC-002 — Consultar conversas e agir em um chat com autorização  
**Status:** implemented_initial_slice  
**Escopo:** app_and_api

## 1. Ator e objetivo

O usuário local quer conversar com um assistente dentro do Cockpit e pedir que
a IA compatível consulte informações do próprio Cockpit ou
prepare uma ação em uma conversa existente. O assistente pode resumir
conversas e preparar uma mensagem para outro chat, mas não pode executar uma
escrita sem autorização explícita do usuário.

**Resultado observável:** uma conversa dedicada ao assistente registra a
solicitação, resposta, consultas a ferramentas e decisões de autorização. Uma
mensagem só chega ao chat de destino depois de o usuário aprovar o texto exato
e o destino.

## 2. Evidências e limites existentes

- `FLOW-001-UC-001-cockpit-personal-agent.md` define conversas locais, adapters
  de provider, seleção de modelo, streaming, aprovação de ações solicitadas
  pelo CLI e histórico canônico.
- O runtime já expõe leitura de providers/modelos, projetos e conversas, além
  de criação de conversa, atualização de seleção, envio de mensagem,
  cancelamento e decisão de aprovações.
- O Personal Agent agora oferece uma sessão dedicada “Assistente Cockpit” e
  endpoints locais autenticados para ferramentas e decisões.
- Os adapters atuais apresentam respostas em texto; não se deve pressupor que
  todos suportem tool calling nem interpretar texto livre como autorização.
- O serviço local usa loopback, token de sessão e validação de origem. O
  usuário do desktop é a autoridade local; não há modelo multiusuário definido
  para este fluxo.

## 3. Pré-condições

1. O serviço local do Personal Agent está iniciado e autenticado para a sessão
   local do Cockpit.
2. Codex está instalado/autenticado e seu adapter declara suporte a ferramentas
   estruturadas.
3. O assistente está em uma conversa/sessão própria, distinta dos chats que ele
   consulta ou altera.
4. Conversas de destino são identificadas por IDs obtidos do próprio runtime;
   o modelo não recebe ferramentas de filesystem/shell do Cockpit. O Codex
   roda em workspace separado e somente leitura; pedidos de aprovação de
   comandos/alterações são sempre recusados.
5. A interface mostra qual provider/modelo está conduzindo o assistente.

## 4. Fluxo principal

1. O usuário abre “Assistente Cockpit”. A primeira versão seleciona Codex e um
   modelo selecionável; outros providers continuam indisponíveis para este modo.
2. O usuário pede, por exemplo: “Resuma a conversa sobre o desktop e prepare
   uma resposta no mesmo chat”.
3. O runtime envia a solicitação ao provider com a lista de ferramentas
   permitidas e valida cada chamada recebida contra um registro tipado.
4. O assistente pode pesquisar títulos/IDs de projeto em até 50 conversas e
   ler as últimas 20 mensagens, limitadas a 8.000 bytes por mensagem. Anexos
   não são transferidos; o resultado da ferramenta e o transcript são dados
   não confiáveis.
5. O assistente apresenta o resumo e uma prévia da ação proposta: conversa de
   destino, provider/modelo do próximo turno e texto integral da mensagem.
6. O usuário aprova ou rejeita essa ação na UI. A aprovação vale apenas para o
   payload exibido e não é inferida de texto como “sim” emitido pelo modelo.
7. Após aprovação, o runtime revalida destino, seleção e estado da conversa,
   inicia exatamente um turno no chat escolhido e mostra seu progresso.
8. A conversa do assistente registra o resultado da ferramenta e uma referência
   ao turno do chat de destino, sem duplicar ou substituir o transcript desse
   chat.

## 5. Alternativas e exceções

- **Provider sem suporte ao protocolo de ferramentas:** bloquear o modo
  operacional e explicar quais providers estão aptos; oferecer conversa
  normal, sem ferramentas.
- **Provider retorna chamada inválida/desconhecida:** rejeitar a chamada,
  registrar erro sanitizado e não executar fallback por shell/texto livre.
- **Conversa inexistente ou removida:** não criar um novo destino
  implicitamente; pedir que o usuário escolha uma conversa existente.
- **Chat de destino com turno ativo:** não enfileirar nem interromper
  silenciosamente; informar o estado e deixar o usuário decidir o próximo
  passo.
- **Ação rejeitada ou autorização expirada:** registrar a decisão, não chamar a
  API mutável e preservar a conversa do assistente.
- **Payload ou seleção alterados depois da prévia:** invalidar a autorização e
  exigir nova proposta.
- **Falha do provider ou da ferramenta:** manter o estado e o diagnóstico
  sanitizado; não declarar que a ação foi realizada.
- **Instruções maliciosas dentro de chats consultados:** tratá-las como dados
  não confiáveis; não podem alterar ferramentas, escopos ou decisões de
  autorização.
- **Contexto grande demais:** a primeira fatia limita mensagens e marca
  explicitamente cada mensagem truncada; anexos são omitidos.

## 6. Estados e transições

Turno do assistente:

`created -> reasoning -> awaiting_approval -> executing -> completed`

Leituras sem escrita podem transitar diretamente de `reasoning` para
`completed`. `reasoning`, `awaiting_approval` e `executing` também podem
terminar em `failed` ou `cancelled`. Uma autorização rejeitada termina a ação
como `rejected`, sem criar turno no chat de destino.

A autorização é de uso único, associada a um `action_id`, à conversa de
destino, à seleção efetiva e a um SHA-256 do payload exibido. Ela expira após
10 minutos; ações pendentes ou aprovadas sem resultado são invalidadas se a
sessão do assistente for interrompida. Repetir a confirmação não pode iniciar
um segundo turno.

## 7. Autorização e limites

- O assistente herda a identidade e os limites da sessão local; não ganha
  privilégios adicionais por ser IA.
- A UI consulta providers/modelos por endpoints existentes e escolhe Codex.
  Ferramentas disponíveis ao modelo: `list_conversations`,
  `read_conversation` e `propose_send_message`.
- A única mutação permitida na primeira fatia é enviar uma mensagem em uma
  conversa existente, após aprovação explícita da prévia.
- Criar/excluir projetos ou conversas, mudar configuração global, mudar
  permissões, decidir aprovações de ferramentas do provider, cancelar turnos
  de destino e executar shell ficam fora da primeira fatia. O usuário pode
  cancelar apenas o próprio turno do assistente.
- Chamadas de ferramenta usam schemas estritos, limites de resultado, timeout
  de aprovação e cancelamento. O backend valida novamente cada ID, estado e
  payload.
- Nenhum endpoint de ferramenta aceita path, shell, URL arbitrária, expressão
  de busca executável ou política de aprovação fornecida pelo modelo.
- A seleção efetiva de provider/modelo, sandbox e política de aprovação do chat
  de destino permanecem sob as regras do FLOW-001; o assistente não pode
  sobrescrevê-las silenciosamente.
- Resultado de ferramenta, texto de chat e saída de provider são conteúdo não
  confiável. Não entram em logs como instruções nem podem emitir autorização.

## 8. Eventos de auditoria

Cada turno do assistente deve registrar, com horário UTC e sem credenciais:

- provider/modelo, IDs do turno do assistente e da sessão local;
- nome/versionamento da ferramenta, operação de leitura ou escrita e IDs de
  recursos consultados/alterados;
- estado da execução, duração e erro sanitizado quando houver;
- para escrita, `action_id`, hash do payload confirmado, decisão do usuário,
  horário da decisão e ID do turno criado no destino;
- referências às mensagens consultadas, sem duplicar transcript em logs de
  execução.

O transcript do assistente é a trilha local de revisão desta primeira fatia.
Política de retenção, exportação e exclusão precisa acompanhar a política do
histórico de conversas do FLOW-001.

## 9. Contrato implementado

- `GET /api/v1/assistant/providers`: providers com suporte declarado a
  ferramentas estruturadas.
- `GET /api/v1/assistant/sessions`: lista sessões dedicadas.
- `POST /api/v1/assistant/sessions`: inicia sessão dedicada com `model_id` e
  esforço opcional; a primeira versão usa Codex.
- `POST /api/v1/assistant/sessions/{id}/messages`: envia pedido do usuário e
  retorna `turn_id`.
- `GET /api/v1/assistant/sessions/{id}/turns/{turn_id}/events`: transmite
  texto, estado, consultas de ferramenta e ações pendentes por SSE.
- `POST /api/v1/assistant/sessions/{id}/actions/{action_id}/decision` recebe
  `{ "decision": "approve" | "reject" }`. Aprovação só vale para a prévia
  persistida e o payload validado pelo backend.
- `GET /api/v1/assistant/sessions/{id}`: retorna transcript, estado e auditoria
  sem credenciais. O cancelamento usa o endpoint existente de turnos em
  `/api/v1/conversations/{id}/turns/{turn_id}/cancel`.

O adapter Codex usa chamadas estruturadas do Codex App Server (`dynamicTools`
e `item/tool/call`). Cada turno do assistente inicia uma thread com ferramentas
configuradas e transfere o histórico canônico da sessão, sem depender de
ferramentas restauradas implicitamente em threads nativas. JSON em texto livre
nunca é executado como ferramenta.

## 10. Dados fictícios reprodutíveis

- Workspace: `ws-demo`, diretório temporário.
- Provider falso: `codex`/`model-fast` com adapter estruturado em memória; os
  testes não iniciam CLI real nem usam rede.
- Sessão do assistente: `assistant-001`; turnos `assistant-turn-001` e
  `assistant-turn-002`.
- Conversas: `conv-desktop` (concluída, duas mensagens) e `conv-active`
  (turno `turn-active` em execução).
- Mensagem consultada: `Como abro a versão atualizada do Personal Agent?`
- Ação prévia: enviar `A versão atual está pronta para teste.` para `conv-desktop`.
- Decisões: aprovação em `2026-10-27T12:00:00-03:00`; rejeição em
  `2026-10-27T12:05:00-03:00`.
- Relógio de teste congelado; timezone `America/Sao_Paulo`; IDs e ordenação
  estáveis; nenhum CLI real iniciado.

## 11. Critérios de aceite verificáveis

1. O assistente só fica disponível com provider/modelo cujo adapter declare
   suporte explícito a ferramentas estruturadas.
2. O assistente lista e lê conversas por ID autorizado, sem alterar seus
   transcripts.
3. Tentativa de enviar mensagem apresenta destino, texto exato e seleção do
   próximo turno antes de qualquer mutação.
4. Sem aprovação explícita do usuário, nenhum turno é criado no chat de destino.
5. Aprovação de uma prévia não autoriza payload diferente, segundo envio nem
   conversa diferente.
6. Rejeição, expiração, destino removido e conversa ocupada não causam
   mutações e são mostrados com estado explícito.
7. Chamadas inválidas, provider sem suporte, prompt injection em conteúdo lido
   e tool desconhecida falham fechadas e deixam evento auditável.
8. Histórico do assistente referencia consultas e resultado do destino sem
   sobrescrever histórico/proveniência do chat alvo.
9. O usuário consegue cancelar um turno do assistente; cancelar não cancela
   automaticamente uma ação já aprovada nem um turno de destino sem nova
   confirmação.
10. Testes usam providers, ferramentas, relógio e conversas falsos; não
    dependem de credenciais ou rede.

## 12. Telas e estados

| Tela | Estado/resultado |
|---|---|
| Assistente Cockpit | conversa dedicada, seleção de provider/modelo compatível |
| Seleção de provider | capacidade de ferramentas e indisponibilidade visíveis |
| Linha do tempo do assistente | mensagens, consultas, plano, execução e referências |
| Prévia de ação | conversa de destino, provider/modelo, texto exato, aprovar/rejeitar |
| Resultado da ação | sucesso/falha e link para turno do chat de destino |
| Chat de destino | turno normal, com proveniência e controles já definidos no FLOW-001 |

## Decisões aceitas

- O usuário local é a única autoridade para aprovar mutações em chats.
- O assistente usa uma sessão dedicada, não transforma uma conversa normal em
  agente executor implicitamente.
- Ferramentas do Cockpit são tipadas e limitadas; não há shell arbitrário.
- Ler chats é separado de enviar mensagens; mensagens só são enviadas depois de
  aprovação da prévia exata.
- O protocolo de ferramentas precisa ser estruturado no adapter; saída em texto
  livre nunca é executada como comando.
- A primeira integração é Codex App Server, com aprovação por ação obrigatória.

## Implementado na primeira fatia

- Pesquisar títulos/IDs de projeto em até 50 conversas e ler as últimas 20
  mensagens de um chat, limitadas a 8.000 bytes por mensagem, com IDs auditados.
- Preparar uma mensagem, exibir texto/destino/modelo e enviar somente após uma
  aprovação única, válida por 10 minutos.
- Codex recebe ferramentas via `dynamicTools`; chamadas inválidas ou
  desconhecidas falham fechadas e são auditadas.

## Perguntas abertas e próximo passo

| Pergunta | Responsável | Próximo passo |
|---|---|---|
| Qual política de retenção para sessões, planos, payloads e auditoria do assistente? | Produto | Alinhar com exportação/exclusão do FLOW-001 |

**Limite desta entrega:** o assistente atua somente em conversas existentes e
só pode enviar mensagens após aprovação. Criar/editar projetos, executar
comandos e operar outros chats ficam fora deste caso de uso e exigem novos
critérios de autorização e testes.
