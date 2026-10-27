# Package profiles and secrets

`cockpit config` is the shared profile service. Public settings are stored in
`~/.cockpit/package-config/<namespace>.json`; secret values stay in the OS vault.
Profiles are explicit: there is no implicit production/default account.

```sh
cockpit config --namespace newrelic --profile partilhar-dev set account_id 8554578
cockpit config --namespace newrelic --profile partilhar-dev set region US
cockpit config --namespace newrelic --profile partilhar-dev secret api_key
cockpit config --namespace newrelic --profile partilhar-dev show
cockpit config --namespace newrelic list
```

`secret` uses hidden terminal input. Automation can pipe a value to `secret api_key
--stdin`; never include credentials in arguments, tracked files or shell tracing.
`show` returns JSON public values and references, never resolves a secret. A
reference means configured metadata, not a verified or available credential.

## Package ownership and sharing

The dispatcher supplies `COCKPIT_PACKAGE_CONTEXT` for installed package commands.
Packages omit `--namespace` to use their own namespace. Other namespaces are denied
unless the operator explicitly grants read access:

```sh
cockpit config --namespace newrelic grant incident-tools
cockpit config --namespace newrelic revoke incident-tools
```

A grant covers **all profiles and vault keys in the owner namespace**, including
future entries. Readers cannot write or delegate. Grants are never inferred from
package dependencies. Grant/revoke requires operator vault unlock; reading secret
values also requires the calling package to be unlocked. Revocation affects new
reads, not credentials already delivered to a running child. Stop such processes
and rotate credentials if access must be withdrawn retroactively.

`cockpit vault get/set/remove/list --namespace ...` now checks both namespace
policy and lock state. Existing callers that depended on bypassing the lock must
unlock explicitly. Legacy unnamespaced access remains operator-only. Low-level
Go vault types are storage primitives; they do not enforce CLI policy on their own.

## Environment injection

```sh
cockpit config --namespace newrelic --profile partilhar-dev exec \
  --env TF_VAR_newrelic_api_key=api_key \
  --env TF_VAR_newrelic_account_id=account_id -- terraform plan
```

Only selected fields are resolved. Values are passed via the child's environment,
never shell interpolation or argv. No global exports or .env files are produced.
The child inherits the current environment; matching variables are replaced.
COCKPIT_* bindings and duplicate bindings are rejected. Use trusted commands:
the child receives secrets and can print/transmit them. Cockpit cannot redact
arbitrary transformed secrets emitted by another program. Telemetry omits config
arguments and stores no resolved environment. Exit failures remain failures.

## Actual security boundary

These checks enforce cooperative CLI access, **not isolation from hostile code
running as the same OS user**. Such code can alter environment context, user-owned
files or directly call the OS keyring. Do not advertise namespace names as secure
identities. Strong isolation requires an OS-enforced broker/sandbox design.
Private file modes protect Unix files; Windows protection follows user-directory
ACLs. Platform compilation does not prove native keyring integration.

## Failure and recovery

Corrupt/unsupported metadata fails closed. Concurrent writers receive a busy
error. After a crash, verify no writer is running and reconcile the named .lock
file before removing it manually. Failed vault writes retain a reference; retry
secret input after correcting vault access. Field types cannot change from secret
to public. Names reject traversal/collisions. Storage is versioned and bounded.

## New Relic adoption

The existing New Relic package has independent profiles. It must explicitly adopt
this service; the core does not silently move/delete its keys. First confirm the
account and region, create public fields, and enter a valid User API key through
`secret`. Run API/Terraform commands through explicit bindings. Compare the same
account with the existing profile before switching consumers. Preserve legacy
profiles until the package migration is verified. User keys and ingest keys need
separate fields. Do not pass named legacy profiles together with injected legacy
environment variables: the current New Relic client rejects that combination.
