# Cockpit evolution workflow

1. Discover: search KB, inspect installed help, identify source repository and
   canonical assets, preserve dirty work, choose an isolated feature branch.
2. Plan: define user outcome, core/package boundary, data ownership, compatibility,
   supported systems, costs if external resources, validation and rollback.
3. Implement: reuse shared profile/vault services; declare dependencies and grants
   explicitly. Never infer authorization from a dependency or namespace string.
4. Verify: run baseline and changed behavior tests; include denial, lock, corrupt
   state, retries, secret handling and CLI output. Add real integration only when
   access is available and authorized. Do not substitute mocks for live evidence.
5. Distribute: core uses make install-local; packages use local-registry staging,
   copy declared assets, cockpit deploy. Preserve other work and support rollback.
6. Publish: inspect actual CI and version rules, validate manifests/index and PR
   body, push a feature branch, wait for checks. Do not auto-merge.
7. Curate: record durable verified decisions, limitations and recovery steps;
   search/update existing KB entries and verify retrieval.

Definition of done: implementation, tests, user-facing help, canonical skills/agent
routing and reference docs agree. State unresolved external/platform validation
explicitly. Do not report installation, CI success or live behavior without evidence.
