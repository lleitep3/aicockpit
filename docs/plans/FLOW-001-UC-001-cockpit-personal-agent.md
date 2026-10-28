# FLOW-001 — Conversar com um agente CLI local

**Caso de uso:** UC-001 — Iniciar conversa com um CLI instalado  
**Status:** ready_for_review  
**Escopo:** app_and_api

## 1. Ator e objetivo

O usuário do Cockpit quer abrir uma conversa no navegador, ver providers e
modelos disponíveis, escolher um deles, enviar mensagens e acompanhar as
respostas. Durante a mesma conversa, pode trocar provider ou modelo sem perder
o histórico anterior.

**Resultado observável:** uma linha do tempo persistida mostra mensagens,
respostas e qual provider/modelo processou cada turno. Trocar a seleção afeta os
próximos turnos e mantém os anteriores para revisão.

## 2. Evidências e limites atuais

- `internal/providers/manager.go` compila e publica instruções, skills,
  permissões e agentes nos formatos de cada provider. Não gerencia processos de
  conversa.
- `internal/providers/providers.go` detecta executáveis e diretórios; detecção
  não prova suporte a uma interface de execução não interativa.
- `docs/architecture/02-provider-compilers.md` descreve a fronteira atual como
  compilador canônico.
- Nesta máquina, `codex`, `devin`, `claude`, `copilot`, `goose` e `antigravity`
  aparecem no PATH. Codex foi escolhido para a primeira fatia por fornecer
  `app-server` com catálogo de modelos, threads, eventos, retomada e interrupção.
  Codex, Antigravity, GitHub Copilot CLI e Devin têm adapters de conversa; Claude
  e Goose ainda são apenas detectados. O adapter Devin consulta
  `devin models list --format json` e executa turnos via `devin --print`, sem
  streaming nem ID de sessão nativa; o histórico canônico é transferido a cada
  turno.
- O comando instalado `cockpit providers detect` respondeu `unknown command
  "providers"`; confirmar comandos disponíveis no binário deste checkout antes
  de escolher o ponto de entrada.

## 3. Pré-condições

1. O Cockpit local inicia um serviço para a interface web.
2. Há ao menos um adapter de runtime compatível com um CLI instalado e
   autenticado pelo próprio CLI.
3. O adapter informa catálogo de modelos e se consegue selecionar cada modelo.
   A origem do catálogo pode ser introspecção do CLI, configuração local ou
   lista administrada; Cockpit não deve inventar modelos como se estivessem
   disponíveis.
4. O usuário escolheu ou autorizou uma pasta de trabalho para a sessão.
5. O serviço e a interface rodam na mesma máquina e sob a conta local do
   usuário.

## 4. Fluxo principal proposto

1. Usuário inicia a interface Cockpit local.
2. Cockpit mostra providers e modelos, com disponibilidade, suporte à seleção e
   origem/horário do catálogo.
3. Usuário cria conversa e escolhe provider, modelo e pasta de trabalho.
4. Usuário envia uma mensagem.
5. O runtime valida provider, modelo, pasta, estado de autorização e tamanho da
   mensagem.
6. O adapter inicia ou retoma a sessão nativa do CLI com argumentos fixos e sem
   shell intermediário. Ao trocar de provider, cria sessão nativa nova e prepara
   o contexto a partir da conversa canônica.
7. Eventos do CLI são normalizados e enviados à interface em streaming.
8. Cockpit persiste mensagem, resposta e metadados do turno na conversa
   canônica.
9. Usuário pode trocar provider/modelo. A linha do tempo não é apagada; o
   próximo turno registra a nova seleção e a forma de contexto transferido.
10. Usuário revê a conversa e os metadados de cada turno.

## 5. Alternativas e exceções

- **CLI ausente:** não iniciar processo; mostrar instalação/configuração
  necessária.
- **CLI interativo ou protocolo incompatível:** marcar provider como
  incompatível; não tentar automatizar entrada de terminal.
- **CLI sem autenticação:** informar que a autenticação deve ser concluída no
  próprio CLI.
- **Catálogo não fornecido pelo CLI:** apresentar apenas modelos configurados,
  marcados como catálogo local/configurado, ou informar que não há modelos
  detectáveis. Não declarar modelo selecionável sem confirmação do adapter.
- **Modelo removido ou indisponível:** preservar conversa e turnos anteriores;
  negar somente o novo turno e atualizar o estado do catálogo.
- **Troca provider/modelo sem retomada nativa:** preservar transcript canônico;
  criar novo contexto nativo a partir do transcript dentro dos limites do
  modelo. Se precisar resumir ou não puder transferir contexto, registrar as
  mensagens de origem e informar o usuário antes de continuar sem contexto.
- **Pasta não permitida:** negar a sessão e pedir seleção de pasta autorizada.
- **Falha ao iniciar ou erro do processo:** marcar sessão `failed`, preservar
  diagnóstico sem incluir credenciais.
- **Cancelamento:** encerrar processo e descendentes; marcar sessão
  `cancelled`.
- **Desconexão da página:** sessão mantém estado no runtime; interface pode
  reconectar e buscar eventos recentes.

## 6. Estados

Runtime: `stopped -> starting -> ready`; falha de startup vai para `failed`.

Sessão: `created -> starting -> streaming -> completed`. De `starting` ou
`streaming`, pode ir para `failed`; de `streaming`, pode ir para
`cancelling -> cancelled`.

Não permitir novo envio enquanto a sessão estiver `starting` ou `streaming`, até
o adapter declarar suporte a turnos concorrentes.

## 7. Autorização e fronteiras de execução — propostas

- Serviço web escuta apenas em `127.0.0.1`; a porta zero escolhe uma porta livre.
- A página recebe um token aleatório da sessão; a API exige Bearer token e
  valida o host e `Origin`. CORS não é habilitado.
- O workspace padrão é `~/.cockpit/workspace`; `cockpit agent --workspace <diretório>`
  permite escolher outro root. Cada thread Codex recebe somente esse root,
  sandbox `workspace-write` e política `on-request`.
- Adapter monta argumentos como array de processo; nunca aceitar comando ou
  shell arbitrário enviado pela interface.
- Reutilizar a autenticação e o modo de permissão configurados no CLI. Cockpit
  não deve conceder permissões mais amplas nem expor tokens do CLI ao navegador.
- Registrar metadados de início, provider, pasta, conclusão, falha e
  cancelamento. Não gravar conteúdo integral da conversa em logs de auditoria.
- Persistir transcript e metadados em armazenamento local para revisão. Não
  persistir tokens, secrets, ambiente completo ou conteúdo sensível de comandos
  sem política explícita.
- Separar transcript canônico dos IDs de sessão nativos do provider. Um ID
  nativo pode ser associado a um ou mais trechos de execução, nunca ser a única
  cópia do histórico.

### Metadados de proveniência por turno

Cada turno deve preservar, quando fornecido pelo provider:

- IDs da conversa, mensagem e turno; papel, sequência e horário UTC.
- Provider e ID/nome exato do modelo; origem e horário do catálogo usado na
  seleção.
- Versões do CLI, adapter e runtime Cockpit; ID opaco da sessão nativa, se
  houver.
- Workspace/projeto e diretório de execução autorizado.
- Início, fim, duração, estado, exit code e código/resumo de erro sanitizado.
- Perfil de permissões usado; modo de transferência de contexto (por exemplo,
  `provider_native_session`, `full`, `summary`, `none`) e IDs das mensagens
  incluídas.
- Eventos normalizados de ferramentas/ações e aprovações, sem secrets nem
  transcript bruto de depuração.
- Uso/custo somente quando o CLI informar; valor ausente fica `null`, nunca
  estimado.

O transcript visível fica associado à conversa. A tela de revisão deve mostrar
proveniência e limitações por turno; os campos não disponíveis no provider
aparecem como indisponíveis. A política de exportação e exclusão permanece
aberta.

## 8. Contrato inicial — proposta

API local orientada a conversas e turnos, sem acoplar o contrato principal a uma API de
modelo:

- `GET /api/v1/providers`: lista providers, capacidades e estado de detecção.
- `GET /api/v1/providers/{id}/models`: lista modelos, origem, instante da
  descoberta, disponibilidade e suporte à seleção.
- `GET /api/v1/workspace`: informa o único workspace permitido nesta sessão.
- `POST /api/v1/conversations`: cria conversa com seleção inicial de provider,
  modelo e workspace.
- `POST /api/v1/conversations/{id}/messages`: inicia um turno e retorna seu ID.
- `GET /api/v1/conversations/{id}/turns/{turn_id}/events`: transmite eventos
  normalizados via SSE; cada turno salva provider/modelo efetivamente usados.
- `PATCH /api/v1/conversations/{id}/selection`: muda provider/modelo padrão
  para os próximos turnos, sem reescrever a linha do tempo.
- `POST /api/v1/conversations/{id}/turns/{turn_id}/cancel`: solicita
  cancelamento do turno ativo.
- `GET /api/v1/conversations/{id}`: lê transcript, estado e metadados dos
  turnos.

O comando `agent` não registra os argumentos do workspace nos logs genéricos de
execução. Transcript, eventos normalizados e metadados ficam no armazenamento
da conversa; credenciais e ambiente do processo não são persistidos.

O runtime usa JSON sobre HTTP para comandos e SSE para eventos de turno. O
histórico canônico fica em um arquivo JSON versionado por conversa, sob
`~/.cockpit/conversations`, com escrita atômica e permissões `0600`; o diretório
é criado com `0700`. Os adapters mantêm IDs nativos separados do transcript.
Para uma prova com Open WebUI, pode existir um gateway temporário compatível
com OpenAI Chat Completions. Ele não substitui o contrato de runtime próprio
do Cockpit. A documentação do Open WebUI lista Chat Completions ou Open
Responses como protocolos de conexão; portanto, CLIs precisam de um
adapter/gateway para essa prova:
[Open WebUI — OpenAI-Compatible](https://docs.openwebui.com/getting-started/quick-start/connect-a-provider/starting-with-openai-compatible/).

## 9. Dados fixos para cenários

- Provider A: `fixture-codex`, modelo `model-alpha`, executável falso.
- Provider B: `fixture-claude`, modelo `model-beta`, executável falso.
- Os dois executáveis emitem eventos conhecidos e não executam comandos do
  usuário.
- Workspace: `ws-demo`, diretório temporário criado pelo teste.
- Conversa: `conv-001`; turnos `turn-001` e `turn-002`.
- Mensagens: `Resuma o README deste projeto.` e `Agora explique o ponto 2.`
- Saídas: Provider A responde `Resumo.`; Provider B responde `Ponto 2.`.
- Metadados esperados: provider/modelo, versão do CLI e adapter, timestamps,
  workspace, estado/exit code, modo de transferência de contexto e referência à
  sessão nativa, quando disponível.
- Relógio: congelado em `2026-10-25T12:00:00-03:00`, timezone
  `America/Sao_Paulo`.

## 10. Cenários e critérios de aceite

```gherkin
Scenario: listar catálogo de modelos
  Given fixture-codex retorna model-alpha como selecionável
  When o usuário abre a lista de providers
  Then Cockpit mostra model-alpha com origem e horário da descoberta

Scenario: conversa concluída com provider/modelo selecionado
  Given fixture-codex, model-alpha e ws-demo autorizados
  When o usuário envia a primeira mensagem da fixture
  Then a resposta é exibida e persistida como turno de conv-001
  And os metadados registram fixture-codex e model-alpha

Scenario: trocar provider sem perder histórico
  Given conv-001 tem turn-001 respondido por fixture-codex/model-alpha
  When o usuário escolhe fixture-claude/model-beta e envia a segunda mensagem
  Then turn-001 continua visível e inalterado
  And turn-002 registra fixture-claude/model-beta e modo de contexto transferido
  And o adapter registra quais mensagens anteriores forneceu ao provider

Scenario: revisar proveniência de turno
  Given conv-001 tem turnos de providers diferentes
  When o usuário abre detalhes de turn-002
  Then a interface mostra metadados disponíveis e marca campos indisponíveis
  And não inventa versão, usage ou identificadores que o CLI não forneceu

Scenario: provider indisponível
  Given o executável do provider não existe
  When o usuário tenta criar uma sessão
  Then nenhum processo é iniciado e a interface mostra estado indisponível
  And o histórico existente permanece acessível

Scenario: pasta fora do escopo
  Given ws-forbidden não está na lista autorizada
  When o usuário tenta criar uma sessão com ws-forbidden
  Then o runtime nega a operação e não inicia o provider

Scenario: cancelar execução
  Given fixture-codex está em streaming
  When o usuário solicita cancelamento
  Then o processo e seus descendentes são encerrados e o estado vira cancelled

Scenario: origem web não autorizada
  Given uma página envia uma mutação com Origin não permitido
  When o runtime recebe a requisição
  Then a requisição é negada e nenhuma sessão é criada
```

### Composer inspirado no Codex — incremento

#### Decisões

- Provider e modelo ficam em botões do composer; menus mostram catálogo, origem e
  disponibilidade. Apenas providers com adapter de conversa podem ser
  selecionados.
- Esforço de raciocínio é opcional, validado contra o catálogo do modelo e
  registrado na seleção e no turno. Sem valor explícito, vale o padrão do
  provider.
- Política de aprovação pertence à seleção da conversa e é copiada para cada
  turno. `on-request` é o padrão; `never` exige escolha explícita. O sandbox
  permanece `workspace-write` em ambos os casos.
- Anexos aceitos nesta fatia são arquivos de texto pequenos, persistidos no
  histórico local e apresentados ao modelo como conteúdo de usuário, marcados
  como dados não confiáveis. Imagens e arquivos binários não entram no contrato.
- Ditado por voz usa a API de reconhecimento do navegador, somente após clique
  no microfone. O texto transcrito permanece no campo até ser enviado.
- O roteamento automático usa regras locais determinísticas, sem gastar um
  turno extra com classificador de IA. A ordem de prioridade é código, pesquisa,
  raciocínio e, por fim, pedido rápido. Cada regra procura sinais em português
  e inglês; anexo de código vai para `coding`. O motivo e o sinal encontrado
  ficam registrados no turno para revisão.
- A tabela global de rotas configura provider, modelo e esforço por classe. A
  configuração inicial aponta as quatro classes ao modelo padrão disponível;
  o usuário pode personalizar cada destino. O modo automático/manual é salvo
  por conversa. No modo manual, os botões do composer selecionam provider,
  modelo e esforço sem alterar turnos anteriores.
- O roteador valida o destino contra o catálogo e adapter no momento da
  execução. Não promete menor custo ou menor latência, pois os CLIs não
  fornecem esses dados de forma uniforme.
- Conversas nos schemas v1 e v2 ganham defaults (`on-request`, modo manual,
  sem esforço e sem anexos) por migração compatível; a próxima gravação salva o
  schema atual.

#### Cenários para validação

1. Trocar modelo, esforço ou política altera apenas a seleção dos próximos
   turnos; mensagens e metadados anteriores continuam idênticos.
2. Um provider sem adapter aparece com motivo de indisponibilidade e não pode
   ser selecionado nem executar turnos.
3. Anexo fora da lista de tipos, acima do limite, inválido em UTF-8 ou acima da
   quantidade permitida é recusado antes de persistir a mensagem.
4. Anexos de texto selecionados são guardados no registro da mensagem e
   enviados ao adapter como dados, sem substituir o texto escrito pelo usuário.
5. `on-request` continua exibindo solicitações de aprovação; `never` é gravado
   no turno e aplicado pelo app-server Codex sem ampliar o sandbox.
6. Sem suporte a reconhecimento de voz, o controle de microfone informa a
   limitação e não quebra o envio de texto.
7. O modo automático classifica a mensagem com a regra e o sinal visíveis,
   aplica a rota da classe e guarda provider/modelo/esforço efetivamente usados
   nos metadados do turno. O modo manual segue a seleção do composer.
8. Alterar uma regra global só afeta novas mensagens; históricos e decisões de
   rota anteriores permanecem revisáveis.
9. Ao trocar para provider sem sessão nativa, o Cockpit transfere o transcript
   canônico e registra os IDs de origem. Acima de 1 MiB, bloqueia o turno com
   mensagem explícita em vez de iniciar sem contexto. Ao voltar a uma sessão
   nativa anterior depois de usar outro provider, envia também o transcript para
   incluir os turnos que essa sessão não viu.

Critérios verificáveis: listar providers/modelos com a origem do catálogo;
enviar mensagem; mudar provider/modelo mantendo o mesmo transcript; mostrar
proveniência por turno; mostrar streaming, falha e cancelamento; negar workspace
não autorizado; recuperar histórico após recarga; rejeitar origem/token
inválidos. Metadados não fornecidos pelo CLI ficam explicitamente ausentes. Os
testes usam providers falsos, nunca iniciam CLIs reais.

## 11. Telas e estados

| Tela | Conteúdo mínimo |
|---|---|
| Início/conversas | sessões recentes, criar conversa |
| Nova conversa | provider/modelo detectados, origem do catálogo, capacidades, workspace permitido |
| Conversa | linha do tempo, seleção provider/modelo para próximo turno, streaming, cancelar |
| Detalhes do turno | provider/modelo, versões, timestamps, workspace, estado, eventos e limitações da proveniência |
| Providers | detectado, catálogo, modelos selecionáveis, autenticação pendente, capacidades |

## Decisões aceitas

- Conversa e histórico pertencem ao Cockpit e persistem independentemente da
  sessão nativa de um CLI.
- A interface lista providers/modelos e permite mudar a seleção dentro da
  conversa.
- A troca afeta turnos futuros; mensagens e metadados dos turnos anteriores
  permanecem preservados para revisão.
- Ao trocar provider, Cockpit prepara o contexto do próximo turno a partir do
  transcript canônico. Se usar resumo ou não transferir mensagens, a interface
  informa esse limite e os metadados registram a origem do contexto.
- Transcript e metadados de execução devem ser salvos localmente. Nunca salvar
  credenciais nos metadados.
- Cada turno deve indicar provider/modelo reais usados; campos que o CLI não
  expõe devem ficar ausentes, não estimados.

## Decisões adotadas para a primeira fatia

- UI própria do Cockpit, sem gateway Open WebUI.
- Primeiro adapter de runtime: Codex via `codex app-server`.
- Catálogo de modelos obtido no `model/list`, com origem e instante da
  descoberta guardados na seleção e em cada turno.
- O catálogo descreve modelos conhecidos pelo app-server; não comprova
  entitlement/acesso à conta. A seleção pode falhar ao iniciar o turno, e o
  estado real é registrado na conclusão da inferência.
- A troca de modelo do Codex vale para o próximo turno e mantém a thread nativa
  e o transcript canônico. Providers sem adapter aparecem como instalados ou
  ausentes, mas indisponíveis para execução.
- Persistência inicial em JSON por conversa, com `schema_version`, IDs de
  mensagem/turno e gravação atômica; não requer serviço ou dependência de banco.
- API local usa HTTP e SSE. A página mostra proveniência por turno e retoma
  eventos enquanto o processo estiver ativo.
- O runtime pede sandbox `workspace-write` e aprovação `on-request`. Pedidos
  de comando e alteração de arquivo aguardam escolha explícita na UI; tipos de
  aprovação ainda não tratados pelo adapter são recusados.
- Integração Hindsight fica para uma etapa posterior, atrás de uma porta de
  memória separada do runtime e do transcript.

## Implementação experimental nesta branch

O comando `cockpit agent --workspace <diretório>` inicia a interface no
loopback. O histórico é criado em `~/.cockpit/conversations`; o runtime lista
modelos dos adapters disponíveis, cria conversas, envia turnos, troca provider
ou modelo com transferência do transcript canônico, transmite respostas,
permite cancelar e pede aprovação para comandos/alterações quando o Codex envia
essas solicitações. Providers sem adapter de conversa aparecem com estado
`adapter pendente`.

Exportação, exclusão, busca full-text e transferência do conteúdo visual para
outro provider ainda não fazem parte dessa fatia. O composer aceita arquivos
de texto e imagens coladas pelo clipboard ou selecionadas do disco; imagens
são encaminhadas ao Codex, único adapter multimodal nesta implementação. Se
uma conversa crescer além do contexto, o comportamento de resumo ainda precisa
ser decidido; o primeiro adapter mantém uma thread Codex e não fabrica um
resumo.

O desktop fica em `internal/personalagent/desktop`: Electron inicia o mesmo
binário Go, carrega apenas a URL loopback anunciada pelo serviço e encerra o
processo ao fechar a janela. O empacotamento Linux gera AppImage e `.deb`; o
histórico e o roteamento continuam em `~/.cockpit/conversations`, enquanto o
workspace padrão permanece em `~/.cockpit/workspace`.
As respostas do agente são renderizadas como Markdown (incluindo código,
listas e tabelas) sem interpretar HTML arbitrário. O histórico mantém os
conteúdos originais das mensagens.

O composer também recebe arquivos por arrastar e soltar ou Ctrl+V para
capturas de tela. A lista de conversas consulta resumos leves periodicamente,
destaca chats atualizados até serem abertos e permite notificações do sistema
por opt-in; estados de carregamento ficam visíveis durante operações de chat e
anexos.

O caso de uso `FLOW-002-UC-002` adiciona a sessão separada **Assistente
Cockpit**, inicialmente apenas com Codex e ferramentas estruturadas para
pesquisar/ler conversas e propor envio de mensagem. A UI mostra o destino, a
seleção e o texto exatos; cada envio exige aprovação explícita e única.

## Propostas

1. Fazer o runtime local em Go, separado do compilador de providers existente.
2. Criar adapters um a um; cada adapter declara `ListModels`, `SelectModel`,
   streaming, cancelamento, retomada e transferência de contexto como
   capacidades independentes.
3. Fazer UI própria com identidade Cockpit como produto final. Usar Open WebUI
   apenas como prova descartável do gateway, se isso acelerar a validação.
4. Guardar catálogo com origem e `discovered_at`; atualizar sem apagar a última
   conversa nem assumir que nomes de modelo são portáveis entre providers.
5. Tratar memória pessoal como uma porta separada (`MemoryPort`). Validar
   primeiro conversa/sessões; avaliar integração Hindsight depois para
   `retain`, `recall` e `reflect`, sem misturar memória com execução de CLI.
6. Evoluir o armazenamento JSON versionado para SQLite quando busca,
   concorrência e migrações locais justificarem a dependência; preservar uma
   migração explícita do schema e os arquivos originais durante a transição.

## Perguntas abertas e próximo passo

| Pergunta | Responsável | Próximo passo |
|---|---|---|
| Quais adapters seguem Codex? | Engenharia | Claude Code, Copilot e Goose precisam contratos independentes e fixtures |
| Qual política para exportar/apagar histórico e anexos? | Produto | Definir antes de implementar gestão de dados |
| Como transferir histórico ao trocar provider? | Engenharia | Preparar contexto canônico e registrar IDs/origem e modo usado |
| Como resumir conversas que excedem o contexto do modelo? | Engenharia | Preservar original e registrar IDs de origem do resumo |
| Como obter catálogo quando CLI não lista modelos? | Engenharia | Usar configuração explícita e mostrar origem/limite |
| SQLite ou outro armazenamento local? | Engenharia | Reavaliar após requisitos de busca, backup e multi-device |

Próxima ação: executar o protótipo com Codex, revisar o fluxo de aprovação e
conversar com turnos em modelos diferentes. Antes de adicionar cada novo CLI,
documentar seu protocolo de modelos, execução, streaming, retomada e aprovação.
