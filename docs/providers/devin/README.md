# Devin Provider

## Introdução
O Devin é um engenheiro de software de IA autônomo desenvolvido pela Cognition AI. Ele é projetado para atuar tanto via interface web quanto via CLI (Windsurf/Devin CLI), capaz de planejar, escrever código, testar e realizar deploy. O AICockpit integra-se ao Devin para injetar contexto de projeto, regras globais e habilidades reutilizáveis.

## Documentação Oficial
- **Principal:** [https://docs.devin.ai/](https://docs.devin.ai/)
- **Guia de Skills:** [https://docs.devin.ai/product-guides/skills](https://docs.devin.ai/product-guides/skills)

## Reference Repositories
- **Exemplos Práticos:** Explore a pasta `examples/` neste diretório para ver exemplos reais de Skills, AGENTS.md, Rules, Permissions, Subagents e Workflows aplicados ao Devin.
- **Repositório Oficial de Skills:** [DevinAI/skills na comunidade ou repositórios abertos baseados no padrão SKILL.md](https://github.com/search?q=filename%3ASKILL.md+devin&type=code)
- [Documentação Oficial do Devin](https://docs.devin.ai/)
- [Visão Geral de Skills no Devin](https://docs.devin.ai/cli/extensibility/skills/overview)
- [Subagentes no Devin](https://docs.devin.ai/cli/subagents)

## Features Suportadas Nativamente pelo Provider
O Devin, em sua arquitetura nativa, oferece suporte para:
- **[Entrypoint](RULES.md):** O `AGENTS.md` (localizado na raiz ou em `.devin/`). É o primeiro arquivo lido ("IA Entrypoint"), onde o AICockpit compilará as orientações iniciais e as "Regras de Ouro".
- **[Skills](SKILLS.md):** Capacidades adicionais modulares que o agente pode invocar (localizadas no diretório do projeto ou via repositórios).
- **[Rules](RULES.md) (Contexto de Projeto):** Regras locais injetadas no contexto, frequentemente via arquivos como `AGENTS.md` ou `README.md`.
- **[Subagents](AGENTS.md):** Delegação de tarefas demoradas para subagentes independentes que rodam em background.
- **[Workflows](WORKFLOWS.md):** Scripts de passos orquestrados (frequentemente empacotados como Skills).
- **Global Memory / Gold Rules:** Memória global entre projetos e regras universais através de arquivos globais como `~/.codeium/windsurf/memories/global_rules.md`.
- **Local [Permissions](PERMISSIONS.md):** Sistema de permissões locais para limitar ou autorizar quais comandos o agente pode executar no ambiente (ex: `.devin/config.local.json`).

## Integrações Atuais no AICockpit Adapter
O adapter do Devin atualmente compila e propaga os seguintes artefatos canônicos:
- `skills`: Copia skills da base do cockpit para a pasta `.devin/skills/` no escopo do projeto.
- `rules`: Compila arquivos de regras para o `.devin/AGENTS.md` no escopo do projeto.
- `gold_rules`: Agrega regras de ouro (gold rules) de pacotes do cockpit e as injeta no arquivo global de memória do Devin (`~/.codeium/windsurf/memories/global_rules.md`).
- `workflows`: Gera as definições de tools/skills no arquivo `.devin/config.yaml`.
- `permissions`: Gerencia liberações de comandos injetando grants no arquivo de permissões local (`~/.devin/config.local.json`).

## Metadados
- **Versão da Integração:** 1.0.0
- **Última Atualização:** 21 de Junho de 2026
