# Antigravity Workflows

No **Google Antigravity**, fluxos de trabalho (Workflows) são tratados de forma muito granular e modular. Eles se alinham fortemente ao conceito de criação de skills baseadas em interações.

## Como Funcionam
O Antigravity se destaca na extração de rotinas. Em vez de escrever fluxos longos do zero, ele possui ferramentas como a skill nativa `workflow-skill-creator`. Essa skill permite destilar um fluxo de trabalho (ou uma interação completa com o usuário) em uma nova habilidade reutilizável de agente.

## Mapeamento no AICockpit
O adaptador do AICockpit gerencia fluxos de trabalho no Antigravity da seguinte forma:
- Compila definições abstratas do Cockpit para scripts passo a passo.
- Salva essas definições dentro das *customization roots* (ex: `~/.gemini/config/workflows/` ou embutidos como sub-passos de uma Skill global).
- Orienta o agente sobre quando evocar um fluxo de trabalho baseado no contexto ou nos arquivos em aberto pelo desenvolvedor.

## Integração Contínua
Graças ao uso intensivo de *Reactive Wakeup* e subagentes, um Workflow no Antigravity não é apenas texto lido passivamente; ele pode engatilhar uma orquestração inteira de background tasks e delegação para outros subagentes até que a meta (goal) seja concluída.
