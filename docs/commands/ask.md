# Command: `cockpit ask`

Run one assistant turn on a local CLI or a provider API.

## Usage

```bash
cockpit ask "<task>"
cockpit ask --backend claude --model sonnet --workdir . --timeout 2m "<task>"
cockpit ask --backend api:anthropic --no-memory "<task>"
cockpit ask --json "<task>"
```

## Description

`cockpit ask` is one turn:

1. Recall relevant memories for the task.
2. Build a prompt that includes those memories.
3. Run one worker and stream its text to the terminal.
4. Retain a short record of the task and the result.

There is no tool loop and no router. `cockpit ask` replaces the old idea of `cockpit agent run`. There is still no `cockpit agent` install or list command.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--backend` | string | config, then auto | Worker: `claude`, `codex`, `api`, or `api:<provider>` |
| `--model` | string | worker default | Model override |
| `--workdir` | string | current directory | Working directory for the worker |
| `--timeout` | duration | none | Worker time limit, for example `2m` |
| `--no-memory` | bool | false | Skip recall and retain |
| `--json` | bool | false | Print one JSON result instead of streaming text |

`--backend api` uses `agents.api.provider` from `~/.cockpit/config.yaml`. `api:<provider>` is `anthropic`, `openai`, `ollama`, `lmstudio`, or `openrouter`.

When `--backend` and `agents.backend` are both empty, the first available worker wins, in this order: `claude`, `codex`, then the configured API if its vault token is present. A named worker that is not available returns an error. It does not fall through to the next one. If nothing is available, the error lists what was checked.

## Configuration

```yaml
agents:
  backend: claude
  api:
    provider: anthropic
    model: claude-sonnet-4-5
    vault_namespace: agents
    vault_key: anthropic_api_key
```

The token stays in the vault. It is not a field of this file.

## Memory

Recall searches the local knowledge base. When `memory.hindsight.enabled` is set, Hindsight recall is added to the prompt. If Hindsight is unreachable, the turn keeps the local hits and prints a warning. Retain writes a concise task/result record to the local knowledge base and, when the service responds, to Hindsight. `--no-memory` skips both.

## Output

Human output is the worker text as it arrives. `--json` prints the same result:

```json
{
  "backend": "claude",
  "model": "sonnet",
  "text": "answer",
  "session_id": "sess",
  "recalled": 2,
  "retained": true
}
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | The worker finished and retain succeeded, or memory was skipped |
| 1 | No worker is available, the worker failed, or local retain failed |

## Related commands

- [`cockpit memory`](memory.md) — optional local Hindsight server
- [`cockpit doctor`](doctor.md) — CLI installation and API token presence
- [Product vision, phase 7](../PRODUCT_VISION.md#fase-7--cockpit-ask)
