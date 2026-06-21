# Goose Provider

## Introdução
Goose é um agente de software de IA open-source focado no desenvolvedor, criado pelo Block. Ele opera via linha de comando, estendendo capacidades locais por meio de um sistema de extensões baseado no protocolo MCP (Model Context Protocol). O AICockpit interage com o Goose configurando regras globais de comportamento e orquestrando suas permissões e extensões de forma autônoma.

## Documentação Oficial
- **Principal:** [https://goose-docs.ai/](https://goose-docs.ai/)
- **Guia de Contexto/Skills:** [https://goose-docs.ai/docs/guides/context-engineering/using-skills/](https://goose-docs.ai/docs/guides/context-engineering/using-skills/)

## Features Suportadas Nativamente pelo Provider
O Goose, em sua arquitetura nativa, oferece suporte para:
- **[Extensions / Skills](SKILLS.md) (MCP):** Ferramentas modulares que seguem o padrão Model Context Protocol, permitindo que o Goose acesse utilitários de sistema e interaja com APIs e CLIs (configuradas no diretório global).
- **[Hints](RULES.md) (.goosehints):** Um arquivo de contexto injetado a cada interação que orienta o comportamento padrão do agente durante a sessão.
- **[Global Config / Permissions](PERMISSIONS.md):** Configuração global do agente, incluindo definições de provedores de LLM e mapeamento de extensões via `~/.config/goose/config.yaml`.

## Integrações Atuais no AICockpit Adapter
O adapter do Goose no AICockpit atualmente compila e gerencia os seguintes artefatos:
- `skills`: Compila definições e scripts de habilidades para o diretório global de uso do Goose (`~/.config/goose/skills/`).
- `rules` / `gold_rules`: Agrupa regras do projeto e regras globais do cockpit e as converte no arquivo de contexto persistente do Goose, normalmente referenciado como `rtk-hints.md` (no formato compatível com `.goosehints`).
- `permissions` / `config`: Manipula o arquivo global `~/.config/goose/config.yaml` para configurar as chamadas de extensões ou habilitar integrações de ferramentas requeridas pelo cockpit.

## Metadados
- **Versão da Integração:** 1.0.0
- **Última Atualização:** 21 de Junho de 2026
