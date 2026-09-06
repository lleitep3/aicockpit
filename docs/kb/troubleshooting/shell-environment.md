---
title: "Troubleshooting: ambiente do shell"
description: "Diagnostica comandos que funcionam no terminal mas falham em automações."
tags: ["troubleshooting", "shell", "zsh", "bash", "environment", "automation"]
created: "2026-09-05"
modified: "2026-09-05"
author: "AICockpit"
version: "1.0"
---

# Ambiente do shell em automações

Consulte a [referência completa de variáveis de ambiente do Cockpit](../../reference/environment-variables.md).

## Sintoma

Um comando funciona em um terminal, mas falha em uma automação ou sessão não interativa.

## Causa comum

O terminal interativo carrega variáveis e `PATH` de arquivos como `.zshrc` ou `.bashrc`. A automação pode iniciar outro shell, sem esses arquivos, ou iniciar um shell não interativo. Configurações específicas do Zsh, como `fpath`, `compinit` e Oh My Zsh, não devem ser copiadas para o Bash.

## Diagnóstico

```bash
command -v cockpit
printf '%s\n' "$SHELL"
printf '%s\n' "$PATH"
```

Compare com um shell de login interativo:

```bash
zsh -lic 'command -v cockpit; printf "%s\\n" "$PATH"'
```

## Correção recomendada

Para escolher explicitamente onde os logs serão gravados, defina `COCKPIT_LOG_DIR`:

```bash
export COCKPIT_LOG_DIR="$HOME/.local/state/aicockpit/logs"
```

O Cockpit valida se o diretório é gravável. Se não for, usa uma pasta organizada dentro do diretório temporário do sistema (`$TMPDIR/aicockpit/logs` em Unix ou `%TEMP%\\AICockpit\\logs` no Windows).

Mantenha apenas exports portáveis em um arquivo compartilhado, por exemplo `~/.config/aicockpit/env.sh`, e carregue-o explicitamente no `.zshrc` e `.bashrc`:

```bash
if [ -f "$HOME/.config/aicockpit/env.sh" ]; then
  . "$HOME/.config/aicockpit/env.sh"
fi
```

Para executar um comando com o ambiente completo do Zsh:

```bash
zsh -lic 'cockpit kb search "logging"'
```

As opções significam login (`-l`), interativo (`-i`) e comando (`-c`). Isso não é login no Cockpit; apenas carrega o ambiente do shell.

## Erro de log somente leitura

Se o `.zshrc` tentar atualizar cache de completions em um filesystem somente leitura, desative essa manutenção para a sessão automatizada ou garanta um diretório gravável. O diretório de logs do Cockpit também precisa ser gravável; sem isso, consultas e comandos podem falhar antes de executar.

## Regra de segurança

Não copie o `.zshrc` inteiro para o `.bashrc`. Extraia somente exports e alterações de `PATH` portáveis, sem tokens, senhas ou comandos interativos.
