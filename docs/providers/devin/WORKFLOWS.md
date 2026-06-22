# Devin Workflows

No Devin, não existe um artefato puramente rotulado como "Workflow" em sua base nativa, mas o conceito é amplamente suportado através da composição de ferramentas e regras.

## Como o AICockpit Mapeia Workflows no Devin
Para orquestrar passos complexos e reprodutíveis no Devin, o adaptador do AICockpit compila fluxos de trabalho usando duas estratégias principais:

1. **Skills como Workflows:** A melhor forma de definir um fluxo de trabalho passo-a-passo no Devin é empacotá-lo como uma [Skill](SKILLS.md). O `SKILL.md` atua como o manual do workflow, descrevendo os gatilhos (quando rodar) e os passos sequenciais rigorosos que o agente deve tomar.
2. **Regras de Repositório (Project Rules):** Para workflows comportamentais contínuos (ex: "sempre que for testar, primeiro rode lint, depois build, depois teste"), o cockpit pode injetar essas instruções diretamente no `AGENTS.md` ou arquivo de prompt de projeto do Devin.

## Casos de Uso
- **Deployment Automático:** Um workflow/skill que dita como gerar a build, autenticar nos servidores, transferir arquivos e invalidar o cache.
- **Troubleshooting Padrão:** Passos sequenciais de como investigar falhas (olhar logs do docker, checar portas, validar variáveis de ambiente).
