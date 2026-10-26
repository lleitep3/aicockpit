# Vault — índice

O vault guarda segredos no keyring do sistema e um estado de bloqueio
autenticado. Esta pasta reúne os textos que estavam soltos em `docs/vault-*.md`.
O conteúdo de cada página foi preservado; só os caminhos e os links internos
mudaram.

A arquitetura continua na trilha numerada:
[05. O Sistema de Cofre](../architecture/05-vault-system.md).

## Como ler

Comece pelo contrato que o código implementa. As páginas históricas explicam
propostas anteriores e mantêm o aviso no topo. Um snippet dessas páginas só
vale se o aviso e o contrato v2 concordarem com ele.

| Ordem | Página | Papel |
|---|---|---|
| 1 | [Estado de lock v2](vault-state-v2.md) | Contrato atual: AES-GCM, chave no keyring, migração explícita, falha fechada, limites do modelo de ameaça. |
| 2 | [Arquitetura](../architecture/05-vault-system.md) | Componentes, fluxo e lugar do vault no cockpit. |
| 3 | [Guia de uso](vault-guide.md) | Operação de credenciais pela CLI. |
| 4 | [Lock e unlock](vault-lock-unlock-implementation.md) | Comandos `cockpit vault lock` e `unlock`. Leia junto com o contrato v2. |
| 5 | [Uso por pacotes](vault-package-usage.md) | Namespace, perfis e o que um pacote pode assumir. Namespace não prova identidade de processo. |
| 6 | [Exemplos de aplicação](vault-usage-examples.md) | Integração a partir de Go, CLI e scripts. |

A direção de produto que usa esses tokens para chamar APIs de provedores está
na [visão de produto](../PRODUCT_VISION.md).

## Propostas e histórico

Estes documentos registram análises e desenhos. Eles não substituem o
contrato v2 nem descrevem um serviço de socket, ACL ou identidade de processo
já entregue.

| Página | Assunto |
|---|---|
| [Comparação documentação e implementação](vault-comparison.md) | Diferença entre a doc antiga e o código, anterior ao P0-02. |
| [Master pass](vault-master-pass-solution.md) | Proposta híbrida de senha mestra. |
| [Solução de produção](vault-production-solution.md) | Permissões, compartilhamento e socket. |
| [Segurança real](vault-real-security.md) | Controle de namespace e identidade. |
| [Análise de segurança](vault-security-analysis.md) | Riscos e hardening anteriores ao lock v2. |
| [Resposta sobre a crítica](vault-security-answer.md) | Resposta a uma proposta de namespace. |
| [Evolução de segurança](vault-security-evolution.md) | Catálogo de propostas futuras. |
| [Três pontos](vault-three-points-answer.md) | Resposta de desenho sobre pasta, socket e permissões. |

O plano de implementação do estado v2 está em
[P0-02](../plans/p0-02-vault-state-implementation-plan.md).

## Caminhos antigos

| Antes | Agora |
|---|---|
| `docs/vault-state-v2.md` | `docs/vault/vault-state-v2.md` |
| `docs/vault-guide.md` | `docs/vault/vault-guide.md` |
| `docs/vault-lock-unlock-implementation.md` | `docs/vault/vault-lock-unlock-implementation.md` |
| `docs/vault-package-usage.md` | `docs/vault/vault-package-usage.md` |
| `docs/vault-usage-examples.md` | `docs/vault/vault-usage-examples.md` |
| `docs/vault-comparison.md` | `docs/vault/vault-comparison.md` |
| `docs/vault-master-pass-solution.md` | `docs/vault/vault-master-pass-solution.md` |
| `docs/vault-production-solution.md` | `docs/vault/vault-production-solution.md` |
| `docs/vault-real-security.md` | `docs/vault/vault-real-security.md` |
| `docs/vault-security-analysis.md` | `docs/vault/vault-security-analysis.md` |
| `docs/vault-security-answer.md` | `docs/vault/vault-security-answer.md` |
| `docs/vault-security-evolution.md` | `docs/vault/vault-security-evolution.md` |
| `docs/vault-three-points-answer.md` | `docs/vault/vault-three-points-answer.md` |
