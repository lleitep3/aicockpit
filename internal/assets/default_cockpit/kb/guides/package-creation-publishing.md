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

## 3. Desenvolvimento e Testes Locais (Análise de Falhas)

### Por que `cockpit pkg install` local falha?
O comando `cockpit pkg install` busca o pacote no índice cache da registry remota (GitHub). Ele **não** resolve caminhos locais de pacotes em desenvolvimento diretamente, a menos que o pacote já esteja indexado e atualizado na registry remota.

### Protocolo de Teste Local:
Para desenvolver e testar um pacote localmente sem publicá-lo:
1. Copie a pasta do pacote diretamente para o diretório de pacotes instalados:
   `cp -r packages/nome-do-pacote ~/.cockpit/packages/`
2. Copie manualmente os assets do pacote (skills, rules) para as pastas canônicas correspondentes para simular o comportamento de instalação:
   `cp -r packages/nome-do-pacote/skills/* ~/.cockpit/skills/`
   `cp packages/nome-do-pacote/rules/* ~/.cockpit/rules/`
3. Execute o comando de compilação dos provedores para atualizar as regras nos workspaces locais:
   `cockpit deploy`
4. Teste a execução do comando CLI (ex: `cockpit video slice`).

---

## 4. Publicação no Registry

A registry oficial do cockpit fica em `/home/lleite/projects/cockpit-registry`. O processo de publicação segue o seguinte fluxo:

1. **Nova Feature Branch**: Nunca envie commits diretamente para a branch `main`. Crie uma branch de trabalho no repositório `cockpit-registry`:
   `git checkout -b feature/pkg-nome-do-pacote`
2. **Copiar os arquivos**: Copie o diretório do seu pacote para a raiz do repositório da registry.
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
