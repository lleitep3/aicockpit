# Antigravity Provider

## Introdução
Antigravity (AGY) é um poderoso assistente e agente de codificação autônomo desenvolvido pela equipe do Google DeepMind. Ele foi projetado para atuar tanto globalmente (na máquina do desenvolvedor) quanto em escopos de workspace específicos. O AICockpit se integra ao Antigravity para fornecer subagentes, skills especializadas com parsing rigoroso de frontmatter YAML e regras dinâmicas que alteram o comportamento do modelo.

## Documentação Oficial
- **Principal:** Uso interno / Documentação da Google DeepMind (Google Antigravity SDK).

## Features Suportadas Nativamente pelo Provider
O Antigravity possui suporte a um vasto ecossistema de personalizações através das suas `customization roots`:
- **[Skills](SKILLS.md) (com Frontmatter YAML):** Diretórios contendo um arquivo `SKILL.md` cuja primeira seção deve obrigatoriamente ser um bloco de metadados YAML (`name` e `description`) para indexação semântica correta.
- **[Rules](RULES.md) (AGENTS.md):** Um arquivo markdown que consolida as diretivas, restrições e diretrizes de estilo ou de projeto para o comportamento do agente.
- **Plugins & Subagentes:** Agrupamentos lógicos que empacotam configurações (`plugin.json`), múltiplas skills e subagentes prontos para serem delegados.
- **Global Config / Workspace Config:** O agente carrega suas personalizações de um caminho global (como `~/.gemini/config`) ou local (`.agents`).
- **[Permissions](PERMISSIONS.md):** O sistema tem capacidades de leitura, escrita e execução restritas por permissões que podem ser cedidas previamente num `config.json`.

## Integrações Atuais no AICockpit Adapter
O adapter do Antigravity no AICockpit foca numa abordagem de injeção **global** na máquina do usuário, compilando para `~/.gemini/config/`:
- `skills`: Compila skills (mantendo rigorosamente o YAML Frontmatter intacto no topo do arquivo) para o diretório `~/.gemini/config/skills/`.
- `rules` / `gold_rules`: Mescla todas as regras compiladas num arquivo canônico `AGENTS.md` localizado em `~/.gemini/config/rules/`.
- `hooks`: Implanta ganchos de ciclo de vida em `~/.gemini/config/hooks/`.
- `workflows`: Transporta definições de fluxos de trabalho para `~/.gemini/config/workflows/`.
- `agents`: Transporta definições de agentes e subagentes.
- `permissions`: Mantém as concessões de execução de terminais ou de acesso de sistema no `config.json`.

## Metadados
- **Versão da Integração:** 1.0.0
- **Última Atualização:** 21 de Junho de 2026
