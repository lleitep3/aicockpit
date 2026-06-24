---
title: "Package: pkg-playwright"
description: "Documentation on the Playwright package integrated into AICockpit for browser automation."
tags: ["package", "playwright", "browser", "automation"]
author: "AICockpit"
version: "1.0"
---

# Package: pkg-playwright

`pkg-playwright` é um pacote nativo embutido no AICockpit para permitir a automação de navegadores utilizando a engine do Chromium via `playwright-go`. Diferente de scripts isolados, este pacote herda a sessão e os cookies do seu navegador (caso desejado) e levanta um servidor HTTP em background para receber comandos da CLI de forma contínua, permitindo interação com páginas logadas sem ser interrompido por verificações de 2FA.

## Como Funciona

A arquitetura do `pkg-playwright` divide-se em:
1. **Server (Driver/Background)**: Quando iniciado, instancia um navegador persistente (`UserDataDir` local) usando o Chrome do sistema, abrindo uma página inicial fornecida. Ao mesmo tempo, levanta um servidor na porta `9091` que fica escutando as ações de controle.
2. **Client (Comandos CLI)**: Comandos avulsos da CLI são enviados como requisições JSON POST para a porta `9091`. O servidor interpreta o comando (ex: `click`, `type`, `eval`), executa no Chromium via protocolo de debugging e retorna o resultado para o terminal.

## Comandos Disponíveis

### 1. Iniciar o Servidor (Obrigatório)
Antes de executar qualquer interação, o servidor precisa estar rodando (normalmente em um terminal dedicado ou em background).
```bash
cockpit playwright start --url "https://github.com"
```
* O comando procurará automaticamente pelo Chrome local em vez de tentar baixar binários do Playwright.
* Se omitido, o perfil de usuário (`--profile`) usará o diretório padrão `~/.cockpit/browser_profile`.

### 2. Ações de Interação

- **Clicar em um elemento**:
  ```bash
  cockpit playwright click "selector-css-aqui"
  ```
  Exemplo: `cockpit playwright click "#login-button"`

- **Digitar em um campo de texto**:
  ```bash
  cockpit playwright type "selector-css" "meu texto"
  ```
  Exemplo: `cockpit playwright type "input[name='q']" "AICockpit"`

- **Avaliar código JavaScript na página (Eval)**:
  Retorna o resultado de um script JS sendo executado no contexto da página.
  ```bash
  cockpit playwright eval "document.title"
  ```

## Boas Práticas e Casos de Uso
- **Bypass de Login**: Em vez de codificar rotinas de login no bot, inicie o `cockpit playwright start`, faça login manualmente na janela do navegador que se abre, e a partir desse ponto instrua a IA a usar os comandos `eval`/`click` na sessão já autenticada.
- **Leitura de Dom/Views**: Utilize `cockpit playwright eval "..."` para fazer scraping da estrutura atual de um kanban board, roadmap ou dashboard e trazer os dados em formato JSON para a IA interpretar no terminal.
