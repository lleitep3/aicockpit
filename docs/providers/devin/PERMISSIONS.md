# Devin Permissions

O sistema de permissões controla quais ações o agente pode realizar sem solicitar sua aprovação. Você pode pré-aprovar ações seguras, bloquear perigosas e sempre solicitar confirmação para operações sensíveis.

**Referência Oficial:** [Devin Permissions](https://docs.devin.ai/cli/reference/permissions)

## Comportamento Padrão de Permissões

O Devin CLI usa um sistema de permissões em camadas para equilibrar poder e segurança. O comportamento padrão depende do modo atual:

| Tipo de ferramenta | Exemplo | Normal | Accept Edits | Bypass | Autonomous (sandbox) |
|-------------------|---------|--------|--------------|--------|----------------------|
| Read-only | Leitura de arquivos, grep, glob | Não | Não | Não | Não |
| Fetch | Requisições HTTP | Sim | Sim | Não | Não |
| Comandos Bash | Execução de shell | Sim | Sim | Não | Não |
| Edição de arquivos via `edit`/`write` | Editar/escrever arquivos | Sim | Não (no workspace) | Não | Sim |

## Como as Permissões Funcionam

Quando o agente chama uma ferramenta, o sistema de permissões verifica suas regras em ordem de prioridade:

1. **Regras Deny** — Verificadas primeiro. Se houver match, a ação é bloqueada imediatamente.
2. **Regras Ask** — Verificadas em segundo lugar. Se houver match, você é sempre solicitado (override qualquer regra allow).
3. **Regras Allow** — Verificadas por último. Se houver match, a ação prossegue sem solicitação.
4. **Padrão** — Se nenhuma regra der match, você é solicitado para aprovação.

## Diretórios e Arquivos de Permissões

O sistema de permissões usa múltiplos arquivos de configuração com diferentes escopos e precedência:

| Localização | Escopo | Compartilhado com equipe? |
|-------------|--------|---------------------------|
| `~/.config/devin/config.json` (Linux/macOS) | Global | Não |
| `%APPDATA%\devin\config.json` (Windows) | Global | Não |
| `.devin/config.json` | Projeto | Sim |
| `.devin/config.local.json` | Projeto (local) | Não |

### Precedência

Quando múltiplas fontes de permissão definem regras, elas são mescladas com esta precedência (maior primeiro):

1. Configurações de organização/time (se enterprise)
2. Grants de nível de sessão (aprovações interativas)
3. Config local de projeto (`.devin/config.local.json`)
4. Config de projeto (`.devin/config.json`)
5. Config de usuário (`~/.config/devin/config.json`; `%APPDATA%\devin\config.json` no Windows)

## Sintaxe de Permissões

Existem dois tipos de matchers de permissão: **baseados em escopo** (controlando quais paths/comandos/URLs são acessíveis) e **baseados em ferramenta** (controlando quais ferramentas podem ser usadas).

### Permissões Baseadas em Escopo

#### Read(glob)
Controla acesso de leitura de arquivos. O padrão glob faz match com caminhos de arquivo.

```json
"allow": [
  "Read(src/**)",           // Todos os arquivos sob src/
  "Read(~/.config/**)",     // Arquivos de config home
  "Read(/tmp/**)"           // Diretório temp
]
```

#### Write(glob)
Controla acesso de escrita/edição de arquivos.

```json
"allow": [
  "Write(src/**)",          // Pode escrever em qualquer lugar em src/
  "Write(tests/**)"         // Pode escrever arquivos de teste
],
"deny": [
  "Write(*.lock)",          // Não pode modificar arquivos lock
  "Write(.env*)"            // Não pode modificar arquivos env
]
```

#### Exec(prefix)
Controla execução de comandos shell. Faz match com comandos que começam com o prefixo dado.

```json
"allow": [
  "Exec(git)",              // git, git status, git commit...
  "Exec(npm run)",          // npm run test, npm run build...
  "Exec(python)"            // python, python script.py...
],
"deny": [
  "Exec(rm)",               // Bloqueia rm, rm -rf, etc.
  "Exec(sudo)"              // Bloqueia comandos sudo
]
```

#### Fetch(pattern)
Controla acesso de fetch HTTP usando padrões de URL.

```json
"allow": [
  "Fetch(https://api.github.com/*)",    // GitHub API
  "Fetch(https://*.example.com/*)",     // Todos subdomínios example.com
  "Fetch(domain:npmjs.org)"             // Qualquer URL em npmjs.org
]
```

### Permissões Baseadas em Ferramenta

Faça match por nome de ferramenta para controlar ferramentas inteiras:

```json
{
  "permissions": {
    "deny": [
      "edit",       // Bloqueia todas edições de arquivo
      "exec"        // Bloqueia toda execução de comando
    ],
    "allow": [
      "read",       // Permite todas leituras de arquivo
      "grep",       // Permite todas buscas
      "glob"        // Permite encontrar arquivos
    ]
  }
}
```

**Nomes de ferramentas disponíveis:** `read`, `edit`, `grep`, `glob`, `exec`

### Permissões de Ferramenta MCP

Controle acesso a ferramentas de servidor MCP:

```json
{
  "permissions": {
    "allow": [
      "mcp__github__list_issues",     // Ferramenta específica em servidor específico
      "mcp__github__*",               // Todas ferramentas no servidor github
      "mcp__*"                        // Todas ferramentas MCP
    ],
    "deny": [
      "mcp__github__delete_repo"      // Bloqueia ferramenta específica perigosa
    ]
  }
}
```

## Formato do Arquivo

Arquivo JSON contendo uma chave `"permissions"` com subchaves para listas de allow, deny e ask:

```json
{
  "permissions": {
    "allow": [
      "Read(src/**)",
      "Exec(npm run)",
      "Exec(git)"
    ],
    "deny": [
      "Exec(rm)",
      "Write(.env*)"
    ],
    "ask": [
      "Write(**)"
    ]
  }
}
```

## Boas Práticas
1. **Segurança por Defeito (Default Deny):** O Devin tende a bloquear a execução de ferramentas não reconhecidas, pedindo aprovação explícita do usuário. Use estes arquivos para permitir antecipadamente comandos que fazem parte do seu workflow canônico (ex: binários internos, linters, `rtk`).
2. **Separação de Escopo:** Comandos puramente específicos a um projeto devem ficar em `.devin/config.json` ou `.devin/config.local.json` do workspace. Deixe o `~/.config/devin/config.json` estritamente para ferramentas globais (como o `cockpit` e o `rtk`).
3. **Use deny para perigosos:** Bloqueie explicitamente comandos perigosos como `rm -rf`, `sudo`, e escrita em arquivos sensíveis como `.env`.
4. **Use ask para sensíveis:** Sempre solicite confirmação para operações sensíveis que podem ser seguras em contexto mas requerem atenção.
5. **Preferir escopo específico:** Use escopos específicos (ex: `Exec(npm run test)`) em vez de genéricos (ex: `Exec(npm)`) quando possível.
