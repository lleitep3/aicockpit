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

Consulte a referência completa de variáveis de ambiente do Cockpit na documentação instalada.

Comandos podem funcionar no terminal porque `.zshrc` ou `.bashrc` carregam PATH e variáveis ausentes na automação. Use `zsh -lic 'comando'` para carregar o ambiente do Zsh; isso significa login, interativo e comando, não login no Cockpit.

Não copie `.zshrc` para `.bashrc`: mantenha exports portáveis em `~/.config/aicockpit/env.sh` e carregue-o nos dois arquivos. Verifique também se o diretório de logs e caches usados pelo shell são graváveis.
