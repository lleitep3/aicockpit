# Command: `cockpit memory`

Start, stop, and check an optional local [Hindsight](https://github.com/vectorize-io/hindsight) server.

## Usage

```bash
cockpit memory up
cockpit memory down
cockpit memory status
cockpit memory status --json
```

Hindsight is off until `memory.hindsight.enabled` is true. `up` and `down` do nothing useful until then: `up` returns an error, and `down` and `status` report that it is not configured. `cockpit doctor` treats that state as skipped.

## What `up` starts

The official image `ghcr.io/vectorize-io/hindsight:latest` runs detached, with:

- API on `127.0.0.1:8888` (container port 8888)
- Control plane on `127.0.0.1:9999` (container port 9999)
- Embedded pg0. `HINDSIGHT_API_DATABASE_URL` is not set
- Named volume `aicockpit-hindsight-data` mounted at `/home/hindsight/.pg0`
- `HINDSIGHT_API_LLM_PROVIDER` (default `openai`) and `HINDSIGHT_API_WORKER_ID`

Host ports, image, provider, model, container name, volume, and vault key names are the `memory.hindsight` block in `~/.cockpit/config.yaml`.

`up` is idempotent when the container is already running. A stopped container is recreated so the environment picks up the current vault values. The named volume is kept. The first boot can take several minutes before `/health` succeeds; `up` returns once the container is running.

`down` stops and removes the container and leaves the volume in place.

## Secrets

The LLM API key is read from the vault namespace `hindsight`, key `llm_api_key`, when the container is created. Store it without putting it in shell history:

```bash
cockpit vault unlock
cockpit vault set llm_api_key --namespace hindsight --stdin
```

Additional `HINDSIGHT_*` variables can map to vault keys under `memory.hindsight.secrets`. Values are passed as container environment variables. They are not written to the config file, an env file, or logs. Providers that do not need a key set `api_key_optional: true`.

Changing a secret requires `cockpit memory down` and then `cockpit memory up`.

## Knowledge base

While `memory.hindsight.enabled` is true, `cockpit kb search` keeps the local results and appends a Hindsight recall. `cockpit kb index` (and `kb rebuild-cache`) retains each markdown document in the project bank. Recall reads that bank and a second bank for the user.

```yaml
memory:
  hindsight:
    enabled: true
    api_url: http://127.0.0.1:8888
    api_token_name: api_token
    banks:
      project: my-app
      user: ada
```

Empty `api_url` uses `http://127.0.0.1:<api_port>`. Empty bank names become `project-<name>` and `user-<name>`. `api_token_name` is an optional vault key in the `hindsight` namespace, sent as a bearer token. It is not the LLM key.

If Hindsight is disabled, misconfigured, or unreachable, search and index finish with the local knowledge base and print a warning. The Hindsight source is not part of this repository. The CLI calls its HTTP API through the official Go client.

## Doctor

`cockpit doctor` includes this status. A configured server that is stopped or whose `/health` check fails marks the doctor run unhealthy. An unconfigured server does not.
