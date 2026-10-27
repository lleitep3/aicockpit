---
title: "Guia: Resolução de Autenticação do GitHub CLI (gh)"
description: "Como contornar erros 401 Bad Credentials no GitHub CLI causados por variáveis de ambiente expiradas ou incorretas."
tags: ["github", "cli", "auth", "troubleshooting", "git"]
author: "AICockpit Team"
version: "1.0"
---

# Resolução de Autenticação do GitHub CLI (`gh`)

Este guia documenta o diagnóstico e a solução para falhas de autenticação com o código de erro `HTTP 401: Bad credentials` ao usar o utilitário GitHub CLI (`gh`).

## O Problema

O GitHub CLI (`gh`) possui uma ordem de prioridade estrita para decidir qual método de autenticação utilizar. A ordem é:
1. Variável de ambiente `GH_TOKEN` ou `GITHUB_TOKEN`.
2. Arquivos de configuração locais (ex: `~/.config/gh/hosts.yml` configurados com `gh auth login`).
3. Chaves SSH ou credenciais de chaveamento do sistema.

Se uma variável de ambiente global como `GITHUB_TOKEN` estiver definida com um token inválido, expirado ou incorreto (como em ambientes de CI ou máquinas de desenvolvimento que reutilizam scripts), o `gh` tentará usá-la **obrigatoriamente**, ignorando as credenciais nativas configuradas e corretas da máquina. Isso causa o erro:
`HTTP 401: Bad credentials (https://api.github.com/graphql)`

---

## A Solução

Para contornar o problema e forçar o GitHub CLI a cair de volta nos arquivos de configuração locais (ou chaveamento nativo que está correto), é necessário limpar temporariamente as variáveis de ambiente de token do contexto do comando.

### 1. Comando Temporário (Recomendado)
Você pode usar o comando `env` com o parâmetro `-u` (unset) antes do comando `gh` para executar sem as variáveis no ambiente:
```bash
env -u GITHUB_TOKEN -u GH_TOKEN gh pr list
```

### 2. Desativar na Sessão Atual do Terminal
Para remover as variáveis da sessão do terminal corrente:
```bash
unset GITHUB_TOKEN GH_TOKEN
```

Após desativar ou omitir as variáveis de ambiente incorretas, o comando do GitHub CLI funcionará normalmente utilizando a autenticação local já estabelecida (SSH, Hosts config, etc.).
