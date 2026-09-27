# Package configuration and profiles

## Objective and scope
Add shared configuration for Cockpit packages, using the existing OS vault for secrets. No cloud changes, new credentials, or New Relic browser automation are part of this work.

## Findings
The current namespace is a storage prefix, not authorization. CLI namespace flags skip the lock check; package identity is not propagated by the dispatcher. No cross-package grants exist. Builder assets describe nonexistent `cockpit agent` commands and duplicate package guidance. Correct these facts instead of replicating assumptions.

## Delivery plan
1. Core profile service: namespace-owned public values and vault references, explicit read grants, validation, private atomic storage and concurrent-write protection.
2. CLI integration: profile set/show/list, secret input via stdin/hidden terminal, grant/revoke, environment injection into a child process. Namespace vault operations must check lock state and access. Dispatcher propagates package context. Never put secrets in argv or telemetry.
3. Consumer and process guidance: document New Relic adoption and update builder agent, package skill, reusable workflow and KB guide. Keep existing profiles/keys intact; migration is explicit.
4. Verify denial, revocation, lock failures, collisions, traversal, secret-free metadata and child-only injection. Run baseline/full tests, race, lint, build and platform compilation where supported. Publish reviewable PRs; no auto-merge.

## Security boundary
Packages are trusted code executing as the same OS user. Namespace and package-context checks prevent accidental cross-package access through supported commands; they are NOT an OS sandbox and environment identity is not tamper-proof. A malicious package can call the OS keyring or edit user files. Do not claim protection against such code. True hostile-package isolation needs a broker plus OS-enforced process boundaries, separately designed.

Owner writes only; explicit namespace readers may read all profiles and associated secrets but cannot mutate or delegate. Operator CLI may manage all namespaces. Vault access always also requires its existing unlock check for the caller. A reference alone never grants access. Missing/corrupt policy fails closed. No implicit grants from dependencies.

## Storage and compatibility
Use ~/.cockpit/package-config/<namespace>.json (0600; directory 0700), versioned with public values, references and readers. Secret values remain under namespace in OS keyring. Writes serialize and replace atomically. Preserve existing legacy key locations; never auto-delete/migrate credentials. Existing --namespace callers now need an unlocked vault. Child environment inherits normal process configuration but receives only selected profile bindings; Cockpit never prints resolved secrets. The child itself is trusted and can disclose its environment.

## Validation, risk and rollback
Key risks: false security claims, secrets in errors/logs, grants bypassing locks, env collisions, concurrent updates, platform filesystem differences. Test these directly. Rollback: restore prior binary; metadata additive and ignored by old versions, original keys untouched. Revoke grants without deleting stored data. Back up installed binary before make install-local.

## Components
Four logical components planned: profile service, CLI/vault integration, consumer contract, builder process assets. Zero IaC resources. Record final counts/evidence after verification.


## Builder audit results
- Core builder agent advertised nonexistent agent install/run commands: replaced
  with actual skill/workflow routing, preserving its role as an instruction asset.
- Core/package creation guides disagreed on registry layout: aligned examples to
  indexed packages/<name> and actual registry requirements.
- Builder feature-path failures and shell-loop failures were swallowed: corrected
  and tested with missing, escaping and invalid-syntax fixtures.
- Builder existed only in local staging, not the current registry: prepare its
  complete package as a separate PR, including required configure/validate entrypoints.
- Added explicit configuration, grants, platform limitations and evidence categories
  to canonical skills, workflow, KB reference and generated scaffold.

Final scope: four logical components as planned; zero IaC resources. No live
credentials were migrated or created. Existing New Relic consumer migration remains
explicit: use the documented config exec contract rather than silently switching
account profiles. Native macOS/Windows keyring validation needs those hosts.
