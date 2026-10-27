---
title: "Telemetria Centralizada de Comandos"
description: "Como o AICockpit registra métricas e logs uma vez por comando Cobra"
tags: ["telemetria", "logging", "metrics", "cobra", "packages"]
created: "2026-07-17T00:00:00Z"
modified: "2026-07-17T00:00:00Z"
author: "AICockpit Team"
version: "1.0"
related: ["metrics-collection", "dashboard-logs-insights"]
---

# Telemetria Centralizada de Comandos

## Decisão

A telemetria de execução é centralizada em `decorateCommands`, chamada por `NewRootCommand` depois que os comandos nativos e os comandos de pacotes instalados são registrados. O decorator envolve cada subcomando Cobra executável e produz exatamente uma métrica e uma entrada de log quando o handler termina.

Comandos individuais não devem chamar `logging.Manager.LogCommand` diretamente. Essa regra elimina registros duplicados e mantém o comportamento uniforme entre comandos nativos, subcomandos de `pkg` e comandos fornecidos por pacotes.

## Dados Registrados

Cada execução registra:

- caminho Cobra sem o prefixo `cockpit`;
- argumentos recebidos;
- status de sucesso ou erro;
- código de saída;
- duração em milissegundos;
- detalhes e tipo do erro, quando houver.

As métricas são persistidas em `~/.cockpit/metrics.json`; as entradas JSON rotacionadas diariamente ficam em `~/.cockpit/logs/`.

## Erros e Códigos de Saída

Para handlers `RunE`, erros resultam em `status: error` e exit code `1`. Quando o erro é um `exec.ExitError`, o código de saída do processo filho é preservado. Handlers `Run` que não retornam erro podem definir a annotation Cobra `telemetry_status` como `error` para registrar uma falha de domínio.

## Comandos de Pacote

Os comandos de `pkg` e os wrappers dos pacotes instalados são decorados automaticamente, porque são adicionados à raiz antes de `decorateCommands` ser executado. Scripts de pacotes devem retornar seu resultado para o wrapper Cobra e não registrar telemetria própria; assim, a execução do usuário permanece representada por uma única métrica.

Comandos adicionados após a criação da raiz devem ser decorados antes de serem executados. Ao introduzir carregamento tardio de comandos, a API de registro deve aplicar o mesmo decorator.

## Testes de Regressão

A cobertura do decorator verifica:

- sucesso, argumentos e caminho de subcomando aninhado;
- erro retornado por `RunE`;
- preservação de `exec.ExitError`;
- status lógico de erro para `Run`.

Os testes ficam em `cmd/telemetry_test.go`.
