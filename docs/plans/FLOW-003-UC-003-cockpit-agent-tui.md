# FLOW-003 — Conversar no terminal do Cockpit

**Caso de uso:** UC-003 — Navegar, buscar e conversar em sessões no terminal
**Status:** implementation_in_progress
**Escopo:** CLI local, sem API nova

## Ator e resultado

O usuário local quer abrir o Personal Agent no terminal, encontrar conversas
anteriores, ler o histórico e continuar uma conversa sem navegador.

**Resultado observável:** a TUI lista conversas com mensagens, filtra sessões
por título, projeto ou conteúdo, abre o histórico e envia mensagens pelo mesmo
serviço e armazenamento usados pela interface web.

## Decisões aceitas pelo pedido

- O acesso inicial é um modo explícito: `cockpit agent --tui`.
- A primeira fatia prioriza navegar entre conversas, buscar e continuar chats.
- A TUI reutiliza `personalagent.Service`, seus adapters, aprovações e histórico;
  não implementa regras de conversa ou execução de provider em paralelo.
- Conversas vazias e sessões dedicadas ao Assistente Cockpit não aparecem na
  lista de chats.
- Busca é local, sem persistir consultas ou enviar texto de conversas à rede.
- Uma conversa nova permanece como rascunho em memória até o primeiro envio.
- Aprovações de ações do provider exigem decisão explícita no terminal.

## Fluxo principal

1. Usuário inicia `cockpit agent --tui` em terminal interativo.
2. Cockpit carrega resumos de conversas não vazias, não arquivadas e não
   dedicadas ao assistente; ordena por atualização mais recente.
3. Usuário navega com `j`/`k`, abre com Enter, ou digita `/` para filtrar por
   título, projeto e conteúdo de mensagens.
4. No histórico, usuário lê mensagens e escolhe `i` para escrever ou `b` para
   voltar. `n` inicia rascunho; o histórico só é criado no primeiro envio.
5. O runtime inicia turno pelo serviço existente e exibe eventos/resposta em
   streaming. Pedidos de aprovação mostram método e parâmetros e aguardam
   `y` (aceitar) ou `n` (recusar).
6. Usuário sai com `q`; nenhuma conversa existente é alterada por busca ou
   navegação.

## Alternativas e negações

- Sem terminal interativo: falhar com orientação para executar em TTY; não
  bloquear scripts nem converter entrada redirecionada em diálogo oculto.
- Falha ao carregar histórico: informar erro; não ocultar falha como lista vazia.
- Busca sem resultados: mostrar estado vazio e permitir limpar o filtro.
- Provider/modelo ausente, quota indisponível ou falha de turno: mostrar erro e
  preservar histórico existente.
- Ação que exige aprovação: não aprovar automaticamente; apenas decisão
  explícita aceita ou recusa.
- Entrada vazia no rascunho: não criar conversa.

## Estados e controles iniciais

| Estado | Controles | Resultado |
|---|---|---|
| Lista | `j`/`k`, `/`, Enter, `n`, `q` | navegar, filtrar, abrir, iniciar rascunho, sair |
| Busca | texto, Backspace, Enter, Esc | filtrar ou cancelar busca |
| Conversa | `i`, `b`, `q` | compor, voltar, sair |
| Composição | texto, Enter, Backspace, Ctrl+C | enviar ou cancelar |
| Aprovação | `y`, `n` | decisão única e explícita |

## Verificação reproduzível

Os testes usam `Store` temporário e adapters/providers falsos; nunca iniciam CLI
real nem usam rede.

```gherkin
Scenario: listar e abrir conversas existentes
  Given duas conversas com mensagens e um rascunho vazio
  When a TUI carrega a lista
  Then mostra as conversas ordenadas por atualização
  And não mostra o rascunho vazio nem sessões do assistente

Scenario: buscar em títulos, projetos e conteúdo
  Given conversas com termos distintos em seus títulos, projetos e mensagens
  When o usuário filtra por um termo, sem diferenciar maiúsculas
  Then a lista contém somente conversas correspondentes
  And busca não altera seus arquivos

Scenario: iniciar conversa sem gerar sessão vazia
  Given usuário iniciou um rascunho
  When volta ou sai sem enviar texto
  Then nenhum arquivo de conversa novo é criado

Scenario: continuar conversa e transmitir resposta
  Given conversa existente e provider falso
  When usuário envia mensagem
  Then serviço persiste mensagem e resposta e a TUI exibe eventos do turno

Scenario: aprovação exige ação explícita
  Given provider falso solicita aprovação
  When usuário recusa
  Then o serviço recebe `decline` e a TUI nunca aprova por padrão

Scenario: execução sem TTY
  Given entrada padrão não é terminal
  When usuário inicia `cockpit agent --tui`
  Then recebe erro acionável sem iniciar leitura bloqueante
```

## Limites da primeira fatia

- A composição envia uma mensagem por linha; anexos, seleção interativa de
  provider/modelo, organização por projeto e sub-sessões ficam para incrementos
  posteriores.
- Busca percorre o histórico local em memória. Índice full-text persistente e
  paginação ficam fora do escopo.
- Evidência inicial é teste local do serviço/TUI e compilação CLI (E2); não
  comprova usabilidade em todos os emuladores de terminal nem distribuição.

## Próximo passo

Implementar o modo `--tui` com `personalagent.Service`, validar testes
automatizados e fazer smoke test manual em terminal interativo.
