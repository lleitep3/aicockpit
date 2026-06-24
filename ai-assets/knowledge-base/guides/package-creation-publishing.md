---
title: "Guia: Criação e Publicação de Pacotes no AICockpit"
description: "Documentação de boas práticas para criar, empacotar e publicar extensões de pacotes para o Cockpit com base nas lições de desenvolvimento."
tags: ["packages", "publishing", "development", "cli", "plugins"]
author: "AICockpit Team"
version: "1.0"
---

# Criação e Publicação de Pacotes no AICockpit

Este guia descreve as boas práticas para criar, testar localmente e publicar pacotes modulares (plugins) no AICockpit, com base nas lições aprendidas e falhas corrigidas durante o desenvolvimento.

## Estrutura do Pacote

Cada pacote é um diretório independente com a seguinte estrutura sugerida:

```
packages/nome-do-pacote/
├── cockpit-package.yml       # Manifesto contendo metadados e mapeamento
├── bin/
│   └── nome-do-pacote        # Script wrapper executável (módulo CLI principal)
├── rules/
│   └── regra-especifica.md   # Regras de prompt para a IA
└── skills/
    └── habilidade/
        └── SKILL.md          # Habilidade que ensina a IA a usar o comando
```

---

## 1. O Manifesto (`cockpit-package.yml`)

O arquivo de configuração do pacote deve mapear as dependências, as features injetadas nos workspaces e o local dos executáveis:

```yaml
name: video
version: 0.1.0
description: "Processamento de vídeo para otimização de tokens"
author: "AICockpit"
license: "MIT"
type: plugin

requirements:
  cockpit: ">=0.1.0"

features:
  modules:
    - path: bin/video          # O script executável
      name: video              # Nome do comando raiz exposto no cockpit CLI
      description: "Comandos para processar e fatiar arquivos de vídeo"
  skills:
    - path: skills/video-slice
      name: video-slice
  rules:
    - path: rules/video-processing.md
      name: video-processing

installation:
  type: local
  method: copy
  supported_providers:
    - antigravity
    - devin
    - goose
  provider_features:
    antigravity:
      - skills/video-slice
      - rules/video-processing
```

---

## 2. Implementação do Script Wrapper (`bin/`)

Se o pacote expõe um comando CLI (mapeado em `modules`), ele deve apontar para um script ou binário na pasta `bin/`. 

### Mapeamento de Subcomandos
Para comandos multi-nível (ex: `cockpit video slice <video>`), a CLI do Cockpit passa o subcomando como o primeiro argumento ao script wrapper (`bin/video`). O script deve tratar este argumento:

```bash
#!/bin/bash
SUBCOMMAND="$1"
shift # remove o subcomando da lista de argumentos

case "$SUBCOMMAND" in
    slice)
        # Parse dos argumentos internos do slice (-i, -o, etc.)
        # Execução da ferramenta necessária (ex: ffmpeg)
        ;;
    *)
        echo "Comando desconhecido"
        exit 1
        ;;
esac
```

> [!IMPORTANT]
> Certifique-se de dar permissão de execução ao script wrapper no repositório antes de testar ou publicar:
> `chmod +x packages/nome-do-pacote/bin/nome-do-pacote`

---

## 3. Desenvolvimento e Testes Locais (Local Registry Staging)

### Onde desenvolver novos pacotes?
Sempre que for criar ou desenvolver um pacote novo para o Cockpit, você **DEVE** criá-lo dentro da pasta padrão de staging do local registry:
`~/.cockpit/local-registry/<nome-do-pacote>/`

Esta pasta centraliza todo o desenvolvimento de novas extensões.

### Por que `cockpit pkg install` local falha?
O comando `cockpit pkg install` busca o pacote no índice cache de registries registradas (como o GitHub). Ele **não** resolve caminhos locais diretamente para instalação direta, por isso desenvolvemos no `local-registry`.

### Protocolo de Teste Local:
Para desenvolver, testar e rodar o pacote localmente antes da publicação:
1. Crie e edite os arquivos do seu pacote diretamente em:
   `~/.cockpit/local-registry/nome-do-pacote/`
2. Para que o Cockpit carregue o pacote no CLI local e sincronize seus assets:
   * Copie a pasta para o diretório de pacotes instalados:
     `cp -r ~/.cockpit/local-registry/nome-do-pacote ~/.cockpit/packages/`
   * Copie os assets (skills/rules) para as pastas canônicas correspondentes:
     `cp -r ~/.cockpit/local-registry/nome-do-pacote/skills/* ~/.cockpit/skills/`
     `cp ~/.cockpit/local-registry/nome-do-pacote/rules/* ~/.cockpit/rules/`
3. Execute o comando de compilação dos provedores para atualizar as regras nos workspaces locais:
   `cockpit deploy`
4. Teste a execução do comando CLI (ex: `cockpit video slice`).

---

## 4. Publicação no Registry

Todo pacote desenvolvido no `local-registry` (`~/.cockpit/local-registry/`) pode ser publicado na registry de sua preferência. No nosso caso, publicamos no repositório oficial `cockpit-registry` (localizado em `/home/lleite/projects/cockpit-registry`).

O processo de publicação a partir do `local-registry` segue o seguinte fluxo:

1. **Nova Feature Branch**: Nunca envie commits diretamente para a branch `main`. Crie uma branch de trabalho no repositório `cockpit-registry`:
   `git checkout -b feature/pkg-nome-do-pacote`
2. **Copiar os arquivos**: Copie o diretório do seu pacote de `~/.cockpit/local-registry/nome-do-pacote` para a raiz do repositório da registry.
3. **Atualizar o Índice (`package-index.yaml`)**: Insira a entrada de metadados do seu pacote no final da lista:
   ```yaml
     - name: "video"
       version: "0.1.0"
       description: "Video processing utilities including token-saving frame slicing with ffmpeg"
       author: "AICockpit"
       license: "MIT"
       category: "productivity"
       tags:
         - video
         - token-optimization
       path: "video"
       url: "https://github.com/lleitep3/cockpit-registry/tree/main/video"
       supported_providers:
         - antigravity
       features:
         - modules
         - skills
         - rules
       requirements:
         cockpit: ">=0.1.0"
       installation_method: "copy"
       status: "stable"
       released_at: "2026-06-24T09:00:00Z"
   ```
4. **Commit e Push**: Faça commit usando conventional commits e envie a branch:
   `git commit -m "feat(nome-do-pacote): register new package"`
   `git push origin feature/pkg-nome-do-pacote`
5. **PR**: Crie o Pull Request no GitHub para mesclagem na branch `main`.
