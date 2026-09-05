# P0-02 / UH-02 — Plano de implementação: proteção do estado de bloqueio

Status: pronto para implementação em outra sessão, preferencialmente Luna.
Data da análise: 2026-09-04. Nenhuma mudança de código foi feita nesta tarefa de planejamento.

Contexto: [visão de produto](../PRODUCT_VISION.md) e [backlog](user-harness-roadmap.md).

## 1. Objetivo e limite da entrega

Impedir que alguém com acesso apenas ao arquivo de estado consiga reconstruir a chave, adulterar permissões e produzir uma assinatura válida usando hostname, UID e salt público. Falhas de chave, leitura, autenticação ou persistência não podem conceder acesso nem ser ocultadas.

A entrega protege o estado de bloqueio do cockpit. Não recriptografa as credenciais existentes, não implementa um novo gerenciador de senhas e não transforma o cockpit numa barreira contra processos que executam como o próprio usuário e conseguem acessar seu keyring. Documentar também a ausência de proteção contra rollback de um ciphertext válido antigo; isso exige mecanismo adicional de estado confiável.

Não implementar UH-01, demais tasks do roadmap, mudanças de providers, adapter Codex ou reformulação completa do vault. Correções adjacentes de erro/autorização necessárias a esta entrega entram no escopo, conforme seção 7.

## 2. Preparação e convivência com outras sessões

1. Ler `AGENTS.md` e instruções aplicáveis aos arquivos tocados, inclusive `cmd/AGENTS.md` se existir. Usar Go 1.26+ e RTK; nesta máquina ele está em `/home/lleite/.local/bin/rtk` quando ausente do PATH.
2. Consultar KB com `cockpit kb search "vault lock state encryption migration"`. Se falhar por logging/permissões, registrar a limitação e continuar pela leitura do código; não alterar dados pessoais para destravar o planejamento.
3. Examinar `git status` e diffs. Existem mudanças locais de outra sessão, incluindo adapter Codex. Não usar checkout/reset/stash sobre elas, não trocar a branch compartilhada e não executar formatação global.
4. Preferir checkout/worktree isolado. Se partir de HEAD, conferir se o código de vault corresponde ao baseline deste plano. Nunca incluir alterações alheias no diff ou commit.
5. Não executar comandos reais de lock/unlock/reset, não migrar o vault pessoal, não instalar/deployar globalmente. Todo experimento usa fixtures e backend falso.

## 3. Evidências a reconfirmar no código

| Arquivo | Problema observado |
|---|---|
| `internal/vault/state_encryptor.go` | `getEncryptionKey` sempre usa `deriveFixedKey`; HMAC também usa dados públicos; versão não é validada; caminho pessoal hardcoded |
| `internal/vault/lock_manager.go` | Construtor ignora erro de load; load mascara corrupção; mutações ocorrem antes de save; `GetStatus` expõe mapa interno e usa RLock aninhado |
| `internal/vault/master_password.go` | Caminho hardcoded; load pode converter corrupção em senha desabilitada; verificador é SHA-256 simples protegido por chave pública derivável |
| `cmd/vault_lock.go` | Uso de modo dev como bypass; comandos de senha/reset precisam ser examinados; alguns erros de persistência são ignorados |
| `internal/vault/vault.go` | Já usa `github.com/zalando/go-keyring`; índice de credenciais do usuário não deve armazenar nem remover a chave interna de estado |

Consultar call sites com `rg 'NewLockManager|NewStateEncryptor|NewMasterPassword|CanPackageAccess|UnlockPackage|IsVaultLocked'`. Atualizar testes de contrato, não apenas testes das funções criptográficas.

## 4. Decisão técnica para esta implementação

### 4.1 Chave aleatória gerenciada pelo sistema

- Gerar 32 bytes com `crypto/rand` para AES-256 e persistir no keyring usando a dependência já presente.
- Serviço interno dedicado, por exemplo `aicockpit-internal`; entrada de chave versionada e identificada pelo envelope. Não usar `NewOSVault`, não inserir a chave no índice público de secrets, não expô-la pela CLI de leitura de credenciais.
- Injetar uma interface pequena de key store; permitir fake em memória e falhas programáveis. Evitar globals mutáveis nos testes.
- Separar leitura de chave de criação. Abrir estado existente jamais cria ou substitui chave ausente.
- Criar chave somente na inicialização/migração explícita e controlada. `ErrNotFound` é diferente de keyring bloqueado ou indisponível. Não há fallback para arquivo plaintext, hostname, UID, variável de ambiente ou senha hardcoded.
- Validar comprimento/base64 de chave recuperada. Erros não podem incluir segredo, senha, ciphertext completo ou material derivado.
- Chave própria de infraestrutura e senha mestra têm papéis distintos: a primeira autentica o estado; a segunda autoriza operações pela interface do cockpit. Não afirmar que esta mudança deriva a criptografia da senha.

### 4.2 Envelope v2

Usar estrutura explícita com `version`, `key_id` e `data` codificada em base64. `data` contém nonce e ciphertext autenticado. Eliminar HMAC e nonce duplicado no formato novo.

Usar AES-GCM da biblioteca padrão. Com Go 1.26, preferir `cipher.NewGCMWithRandomNonce` e seguir seu contrato de `Seal/Open`; não implementar nonce manual ou cifra própria. Associar versão e key_id como AAD usando serialização determinística e sem ambiguidades. Rejeitar versões desconhecidas e combinações de campos inválidas. AAD não fornece proteção contra replay.

Limitar tamanho de arquivo antes de desserializar e tratar base64/JSON truncados sem panic. Validar semântica do plaintext: mapa não nulo, booleanos coerentes e expiração. Nunca interpretar ausência de dados como autorização global.

### 4.3 Persistência e concorrência

- Diretório privado (0700 em POSIX), arquivo temporário exclusivo com 0600 na mesma pasta, escrita completa/close/sync conforme suporte, depois rename atômico.
- Não usar destino fixo `.tmp`; limpar somente temporários criados pela operação. Preservar o arquivo anterior em falhas.
- Serializar read-modify-write entre processos, não apenas entre goroutines, para evitar unlock obsoleto sobrescrever lock. Escolher mecanismo pequeno compatível com plataformas suportadas; se exigir dependência, justificar no ADR e testar a implementação adotada. Não improvisar lockfile com remoção de lock supostamente antigo.
- Inicialização/migração e criação de chave também precisam dessa exclusão. Recarregar estado dentro da seção crítica antes de aplicar transição.
- Em mutações que ampliam acesso, publicar estado em memória só após persistência bem-sucedida. Em falha de lock/revogação, negar acesso localmente e retornar erro; não declarar revogação persistida para outros processos.

## 5. Estados e erros: contrato esperado

| Situação | Resultado |
|---|---|
| Instalação nova, arquivo ausente | Estado lógico bloqueado; leitura não cria chave; inicialização explícita persiste v2 |
| v2 válido e chave disponível | Carregar estado validado, respeitando expiração |
| v2 com chave ausente/inacessível | Negar acesso e retornar erro específico; não gerar outra chave |
| JSON/base64/versão/tag inválida | Negar acesso e retornar erro; não sobrescrever o arquivo |
| v1 legado | Negar acesso e indicar migração necessária; não honrar desbloqueios antigos |
| Falha de save em unlock | Erro e nenhum novo acesso concedido |
| Prazo expirado | Nenhum grant abrangido pelo prazo concede acesso, inclusive grant por pacote |
| Erro de migração | Credenciais e arquivos legados preservados; acesso bloqueado |

Preferir construtores que retornem erro, com dependências injetáveis; atualizar todos os call sites. Se preservar wrapper por compatibilidade, ele deve reter erro de inicialização, negar acesso e impedir mutações até recuperação explícita. Não permitir que wrapper silencioso abra um segundo caminho inseguro.

Centralizar avaliação de acesso/expiração. `IsVaultLocked`, `IsPackageUnlocked`, `CanPackageAccess` e status precisam concordar. Retornar cópias de mapas, remover RLock recursivo e testar concorrência. Não ampliar escopo para daemon de sessões; cada operação deve usar estado atual sob o contrato estabelecido.

## 6. Migração conservadora v1 → v2

Implementar operação explícita, proposta `cockpit vault migrate-state`; preservar UX dos comandos existentes, que devem apontar para ela quando necessário.

1. Detectar v1 sem confiar no seu conteúdo: a chave antiga permite falsificação.
2. Explicar que grants anteriores serão descartados e o estado ficará bloqueado. Exigir confirmação explícita por flag para automação ou confirmação interativa; sem entrada interativa disponível, retornar orientação e código não zero.
3. Reutilizar exclusão entre processos. Criar backup exclusivo dos bytes originais com 0600; nunca sobrescrever backup anterior.
4. Criar chave interna nova e gravar envelope v2 contendo apenas estado seguro bloqueado, sem grants ou desbloqueio global herdado.
5. Confirmar leitura/autenticação do novo arquivo. Não apagar o backup nem alterar credenciais, índice público, permissões de pacotes ou senha mestra.
6. Executar novamente deve ser no-op verificável quando v2 já for válido. Corrupção/versão desconhecida não entra automaticamente no caminho legado.

Se senha mestra existente estiver inválida, não interpretar como proteção desabilitada nem fazer recuperação automática. A operação de migração somente bloqueia e não deve configurar/desabilitar uma senha. Recuperação de senha é outra operação.

Perder a chave de v2 também exige recuperação explícita que preserva credenciais e reinicializa apenas grants para bloqueado; nunca gerar nova chave silenciosamente durante leitura. Registrar claramente o tratamento adotado e testar interrupções entre criação da chave e escrita do envelope. Não apagar chaves que possam estar referenciadas por outro estado.

## 7. Senha mestra: correções necessárias e limite

Esta tarefa não redesenha o verificador nem migra todos os registros de senha. Porém, não pode deixar estes caminhos abrirem acesso:

- Propagar erros de carregamento de senha mestra; corrupção não equivale a senha desabilitada.
- Remover caminhos pessoais hardcoded nos fluxos tocados e injetar paths para testes.
- Se já existe senha, configurar/substituir/desabilitar proteção exige autenticação atual no fluxo normal. Não permitir que `set-master-password` sobrescreva a existente sem validar a anterior.
- `COCKPIT_DEV_MODE` não desativa verificação do estado v2 nem autenticação de produção. Testes usam dependências falsas, não bypass ativável por ambiente.
- `ForceSet`/factory reset não viram fallback de migração. Não executar reset real; preservar confirmação destrutiva e tratar erros. Definir se a chave interna deve ser mantida ou removida em reset explícito e cobrir por fake, sem excluir credenciais durante migração.

Documentar como dívida separada o verificador SHA-256 e armazenamento legado da senha mestra, bem como acesso direto ao keyring e replay de estado. Não anunciar “vault seguro contra agentes locais” ao concluir. Se a implementação exigir redesenhar o verificador para cumprir o contrato, explicar essa dependência antes de ampliar a tarefa.

## 8. Etapas pequenas para Luna

1. **Baseline e testes de regressão:** mapear callers; fixtures temporárias; testes de adulteração e save com falha. Registrar falhas preexistentes separadamente.
2. **Key store e envelope v2:** criar interface, fake e implementação keyring; round-trip e falhas de autenticação. Sem tocar na CLI ainda.
3. **Manager e armazenamento:** propagar erros, transições seguras, expiração, snapshots e concorrência entre processos. Atualizar callers.
4. **Migração explícita:** comando, confirmação, backups, idempotência e falhas injetadas; preservar secrets.
5. **Integração de senha/CLI:** fechar bypasses citados na seção 7, manter limites do escopo.
6. **Verificação e documentação:** testes completos isolados, ADR curto, guia de migração e limitações; relatório de resultado. Não rodar migração no ambiente pessoal para demonstrar.

Concluir cada etapa com os testes pertinentes antes de avançar. Não criar subagentes; este plano foi delimitado para uma única sessão.

## 9. Matriz mínima de testes

- Mesma chave permite round-trip em duas instâncias; estados iguais produzem ciphertexts distintos.
- Chave errada, key_id/AAD adulterado, tag alterada, base64 inválido, truncamento, arquivo grande e versão desconhecida falham sem panic.
- Hostname/UID/salt antigos não abrem nem autenticam v2.
- Keyring ausente, bloqueado, erro ao criar e chave de tamanho inválido não concedem acesso; nenhuma leitura cria chave.
- Falhas em cada etapa de write/rename não publicam unlock em memória e preservam arquivo anterior.
- v1 originalmente desbloqueado migra para bloqueado; backup preservado; interrupções e repetição da migração são seguras; chamadas ao backend de credenciais permanecem zeradas.
- Erro/corrupção de senha não desabilita autenticação; dev mode não contorna v2; troca/desativação exigem autenticação quando habilitada.
- Autolock global e por pacote, status coerente, snapshot sem alias, goroutines e processos concorrentes sem perda de revogação por estado obsoleto.
- CLI retorna não zero em falha, não imprime sucesso prematuro, não inclui segredos em logs e não solicita senha inesperadamente em leitura de status.
- Nenhum teste usa HOME real, keyring real, D-Bus pessoal, índice de secrets real ou factory reset real. Fake obrigatório inclusive em testes antigos executados pela suíte.

## 10. Comandos de validação

Executar no checkout isolado após corrigir isolamento dos testes tocados. Ajustar caminho de `rtk`/Go conforme ambiente.

```sh
rtk go test -race ./internal/vault
rtk go test -race ./cmd -run 'Vault|Lock|MasterPassword'
rtk go test -race -coverprofile=/tmp/uh02-vault-coverage.out ./internal/vault
rtk proxy go tool cover -func=/tmp/uh02-vault-coverage.out
rtk go vet ./...
rtk go test -race ./...
rtk git diff --check
```

Usar timeouts finitos para testes de processos/locks. Só executar suíte completa depois de assegurar que ela não toca dados pessoais. Se houver falha alheia ou restrição de ambiente, reportar exatamente; não declarar tudo aprovado. Meta do projeto: pelo menos 90% em vault após a mudança, além dos cenários críticos acima. Não alterar testes para simplesmente aceitar falhas nem aumentar cobertura com testes que espelham implementação.

## 11. Critério de conclusão e entrega

- Estado v2 usa chave aleatória no keyring e autenticação AEAD padrão.
- Falhas não concedem acesso e têm erro observável.
- Migração preserva credenciais, guarda legado e descarta grants não confiáveis.
- Concorrência/persistência e expiração seguem o contrato e possuem testes.
- Caminhos tocados não dependem do nome do usuário; testes não usam recursos pessoais.
- Documentação distingue proteção de estado, autenticação da CLI e limites do keyring.
- Entregar arquivos alterados, resumo da decisão, testes/cobertura, instrução de migração e riscos restantes. Sugerir registro das lições na KB, sem inserir segredos e sem afirmar que `kb add` já funciona.
- Não fazer deploy, release, push ou migração real automaticamente. Não misturar a feature Codex ou outras mudanças do worktree.

## 12. Referências técnicas

- [Go crypto/cipher](https://pkg.go.dev/crypto/cipher): contrato de AEAD e GCM com nonce aleatório.
- [go-keyring](https://github.com/zalando/go-keyring): backend já adotado pelo projeto; tratar erros da API explicitamente.
- Código local listado na seção 3 é a referência do comportamento atual; reconfirmar antes de editar pois há outras sessões ativas.

## 13. Prompt para iniciar a outra sessão

> Implemente o P0-02/UH-02 do AICockpit seguindo `docs/plans/p0-02-vault-state-implementation-plan.md`. Leia primeiro as instruções aplicáveis e reconfirme o código atual. Trabalhe em etapas pequenas, com testes isolados e backend de keyring falso; preserve mudanças de outras sessões, especialmente o adapter Codex. A entrega é proteção do estado v2, erros que negam acesso, migração explícita sem perda de credenciais e documentação dos limites. Não execute migração/reset no meu vault real, não faça deploy ou push e não amplie o escopo para outros itens do roadmap. Reporte validações, cobertura e pendências com precisão.
