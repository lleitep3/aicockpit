---
title: "Guia: Benefícios e Uso do GitHub Projects no Cockpit"
description: "Documentação sobre os benefícios de utilizar o GitHub Projects (V2) de forma centralizada para o ecossistema Cockpit, incluindo features como Kanban, Roadmap e Custom Fields."
tags: ["github", "projects", "management", "kanban", "roadmap", "agile"]
author: "AICockpit"
version: "1.0"
---

# GitHub Projects (V2) para o Cockpit

Este guia documenta os principais benefícios e os recursos disponíveis ao gerenciar o ecossistema do Cockpit utilizando o GitHub Projects (V2).

## 1. Visão Centralizada de Múltiplos Repositórios
O ecossistema Cockpit é distribuído em múltiplos repositórios (como o CLI `aicockpit` e o `cockpit-registry`). O principal benefício do GitHub Projects é a possibilidade de vincular todos esses repositórios a um único quadro. Dessa forma, é possível agregar _Issues_ e _Pull Requests_ de projetos diferentes em um só lugar, evitando a alternância constante de abas e contexto.

## 2. Layouts e Visões Dinâmicas (Views)
O Projects V2 não é apenas um Kanban; ele permite múltiplas visões sobre os mesmos dados:
- **Board (Kanban):** Ótimo para o fluxo diário. As colunas típicas incluem Todo, In Progress, In Review e Done.
- **Table (Tabela):** Uma visão que se assemelha a planilhas ou bancos de dados (como o Notion), ideal para realizar edições em massa, agrupar, ordenar e filtrar dados massivos de forma muito rápida.
- **Roadmap (Linha do Tempo):** Oferece um gráfico de Gantt nativo. Perfeito para planejar e visualizar o lançamento de novas features (milestones, epics) baseando-se em datas de início e término.

## 3. Campos Customizados (Custom Fields)
Para evitar a poluição de dezenas de `labels` nos repositórios, o Projects permite a criação de campos nativos no nível do quadro:
- **Prioridade:** (Alta, Média, Baixa)
- **Tamanho/Estimativa:** (Story Points ou tamanhos de camisa P/M/G)
- **Sprints/Iterações:** Permite agrupar o trabalho em ciclos de desenvolvimento definidos, facilitando o gerenciamento ágil.
- **Status:** Estados customizados que fazem sentido para a equipe.

## 4. Automações Integradas (Workflows)
O GitHub Projects permite configurar workflows embutidos (ou via GitHub Actions) para atualizar automaticamente o status dos itens:
- Quando um Pull Request (ex: `fixes #39`) é aberto e atrelado a uma issue, o cartão correspondente pode ser movido automaticamente para "In Review" ou "In Progress".
- Ao realizar o *merge* do PR na branch principal (`main`), o cartão correspondente é movido imediatamente para "Done", sem intervenção manual.

## 5. Gráficos e Métricas (Insights)
Na aba **Insights**, o GitHub gera gráficos detalhados em tempo real sobre o board:
- Gráficos históricos de *Burndown*.
- Status de progresso, acúmulo de trabalho e velocidade da equipe.
- Ajudam o gestor/desenvolvedor a localizar gargalos no pipeline de desenvolvimento.

---

**Dica Prática:** A interface do GitHub Projects permite transitar rapidamente entre essas visões. Para criar uma visão de "Roadmap" (caso a visualização principal seja Board ou Table), basta criar uma nova `View` na interface web do GitHub e selecionar a disposição `Roadmap`, definindo um campo de data para organizar a linha do tempo.
