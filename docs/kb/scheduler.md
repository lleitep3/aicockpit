---
title: "Scheduler do AICockpit"
description: "Como usar o pacote cockpit-scheduler para agendamento de comandos e scripts"
tags: ["scheduler", "agendamento", "cron", "ubuntu-security", "automation", "pacote"]
created: 2026-07-03
modified: 2026-07-03
author: AICockpit
version: "1.0"
---

# Scheduler do AICockpit

O scheduler e um pacote do cockpit (`cockpit-scheduler`) para agendamento de comandos e scripts.

## Instalação

```bash
cockpit pkg install cockpit-scheduler
```

Ou localmente para desenvolvimento:

```bash
cp -r ~/.cockpit/local-registry/cockpit-scheduler ~/.cockpit/packages/
cockpit deploy
```

## Comandos

### Criar agendamento com cron

```bash
cockpit scheduler add --command "echo 'Hello'" --cron "0 9 * * *"
```

### Criar agendamento com intervalo e repetições finitas

```bash
cockpit scheduler add --command "scripts/backup.sh" --interval 1h --repeat 3
```

### Agendar análise de segurança do Ubuntu diariamente

```bash
cockpit scheduler add-ubuntu-security --cron "0 2 * * *"
```

### Listar agendamentos

```bash
cockpit scheduler list
```

### Remover agendamento

```bash
cockpit scheduler remove <id>
```

### Executar agendamentos pendentes

```bash
cockpit scheduler run
```

### Executar todos os agendamentos imediatamente

```bash
cockpit scheduler run --all
```

### Instalar executor automático

O modo padrão é `systemd` com `Persistent=true`, garantindo que jobs atrasados
rodem após o boot.

```bash
# Padrão: systemd, persistent=true
cockpit scheduler install
systemctl --user daemon-reload
systemctl --user enable --now aicockpit-scheduler.timer

# Desabilitar persistência
cockpit scheduler install --persistent false

# Modo cron (sem persistência automática)
cockpit scheduler install --mode cron --interval 5
crontab ~/.cockpit/scheduler/cron.txt
```

## Persistência

Os agendamentos são salvos em `~/.cockpit/scheduler/jobs.json`.

## Padrões de cron suportados

- Expressões padrão: `0 9 * * *`
- Aliases: `@daily`, `@hourly`, `@weekly`, `@monthly`, `@yearly`
- Extensões: `daily`, `hourly`, `weekdays`, `weekends`
- Intervalos: `1h`, `30m`, `15m`, `5m`, `1m`, `1d`, `1w`, `2h`, `10s`

## Ubuntu Security com Timeline

O report HTML do `ubuntu-security` agora inclui gráfico de linha do tempo com os últimos 30 dias de:
- Quantidade de processos
- Quantidade de logs
- Quantidade de serviços falhos
- Quantidade de portas expostas
