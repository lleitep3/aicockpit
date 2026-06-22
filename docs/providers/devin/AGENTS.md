# Devin Subagents

O Devin suporta a delegação de tarefas complexas para subagentes independentes. Subagentes permitem que o agente principal mantenha seu contexto focado enquanto descarrega tarefas secundárias, demoradas ou de pesquisa para instâncias isoladas.

## Como Funcionam
Subagentes no Devin são capazes de:
- Rodar em **background**, permitindo que o Devin principal continue interagindo com o usuário e realizando outras tarefas concorrentemente.
- Utilizar ferramentas idênticas ao agente principal (leitura de arquivos, terminal, navegação web).
- Reportar os resultados de forma resumida para o contexto do Devin principal quando concluírem a tarefa.

## Casos de Uso Comuns
- **Pesquisa Extensiva:** Delegar a leitura de uma grande documentação ou a análise de muitos arquivos de log.
- **Testes em Segundo Plano:** Rodar suítes de testes demoradas enquanto o Devin continua escrevendo código em outro módulo.
- **Refatorações Isoladas:** Pedir a um subagente para refatorar um arquivo específico baseado em um novo padrão, revisando o resultado apenas no final.

## Integração
O suporte a Subagents é nativo e pode ser ativado ou comandado via prompts específicos ou pelo uso das tools internas do Devin dedicadas ao gerenciamento do ciclo de vida de subagentes (`delegate`, `spawn_subagent`, etc).

## Referência Oficial
- [Devin Subagents](https://docs.devin.ai/cli/subagents)
