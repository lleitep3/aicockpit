# Safe Cockpit updates

## Goal
An explicit `cockpit update --all` updates the official binary and installed
registry packages from any working directory, preserves user state, and reports
independent outcomes. No merge or live package replacement is implied by this plan.

## Evidence
v0.4.41 updates source in the current directory and has no binary release assets.
Package upgrade passes a package name rather than the index path; stale/divergent
registry caches can hide merged packages. The upgrader removes old assets/package
before validating the candidate and lacks automatic rollback. Deploy errors are
reported as warnings. --version reads stale configuration instead of build metadata.

## Ordered delivery
1. **This PR — package transaction.** Resolve index paths; validate identity,
   candidate files and version before replacement; stage candidate; take a complete
   package/affected-canonical-assets snapshot; restore on hook/sync failure. Retain
   recovery data on failure. Propagate deploy errors instead of reporting success.
   Serialize updates sharing canonical assets. Test real temporary package trees.
2. **Registry and package selection.** Refresh registries into validated snapshots
   without merging stale histories or deleting local work. Respect configured origin,
   disabled registries and pinned versions. Reject silent downgrade. Add installed
   inventory and structured plan/outcomes; avoid ambiguous duplicate-package origins.
3. **Binary update.** Fetch official release into isolated staging, verify checksums
   and platform, atomically replace with rollback/backup. Until release artifacts
   exist, build a verified official tag in a separate checkout with the required Go
   toolchain; never checkout/pull inside the caller's repository. Windows replacement
   needs a helper because a running executable cannot be replaced normally.
4. **Unified UX and release validation.** Add explicit --all/--binary-only/
   --packages-only, --yes and --check/structured status. No hidden prompts in CI.
   Continue package checks when the binary is current; execute post-update work with
   the new binary. Use build metadata for --version. Report partial failures and
   correct exit status. Verify on native Linux/macOS/Windows before portability claims.

## Invariants and limitations
Credentials, profiles and user configuration outside managed package/asset paths
are not rewritten. Backups include manifests. Never infer atomicity across remote
APIs, arbitrary hooks or multiple packages; rollback covers managed local files.
Hooks may cause external effects that cannot be undone: report that limitation.
Process interruption retains backup/staging and a lock requiring reconciliation;
this first transaction does not promise automatic recovery from power loss.
Provider deployment failure is distinct from package activation: return failure
and an actionable retry while retaining the package backup.

## Validation and rollback
Baseline package+CLI tests; failing candidate, name mismatch, missing assets,
path traversal/symlinks, failing pre/post hooks, restore canonical contents and
manifest, stale assets removed after success, preserve unrelated files, complete
backup, busy lock, explicit deployment failure, nested registry paths. Full suite,
race detection, vet, secret scan, CI. Use temporary fixtures before any live update.
Rollback this code through the prior release; recover packages from the recorded
complete snapshots. Do not auto-delete recovery material after failures.

## Scope
First delivery: three logical pieces (transaction/recovery, CLI integration, tests
and operational documentation). Zero cloud/IaC resources; no new recurring costs.
Future phases are intentionally separate reviewable PRs, not claimed implemented.

## First implementation status
The transaction and CLI integration are implemented on the feature branch.
The full Linux suite passed (1,255 tests before two additional snapshot tests),
and `go vet ./...` passed. New fixtures cover rollback after post-install failure,
complete manifest backups, canonical customization preservation, absent assets,
nested index paths, legacy index compatibility and explicit deployment failures.
Follow-up validation expanded rollback coverage to 90.7% across `upgrader.go`
and `upgrade_transaction.go` (individual files: 84.1% and 94.4%). The new registry
refresh code reaches 93.8%; the upgrade CLI reaches 93.8%. These scoped figures
are not the repository-wide coverage. No live installation was replaced.

Package trees containing symbolic links or special files are rejected before
activation in this first implementation. Snapshots are retained under backups;
operators must reconcile a stale lock after confirming no updater is running.
A failed restore retains the lock and reports the recovery snapshot location.
This lock serializes upgrade commands only, not install/uninstall commands.

## Registry refresh implementation plan
Use a fresh shallow clone of the configured URL/branch, validate its index, then
swap directories under a per-registry writer lock. Retain the previous cache,
including local commits and untracked files. Clone/validation failures preserve
the active cache. No cloud resources or recurring cost; retained clones consume
local disk until explicitly cleaned by the operator. Test divergent histories,
changed origin, invalid index, busy lock and preserved user files with local Git.
Also prevent accidental package downgrades; explicit version requests remain
available. Binary replacement remains a separate delivery.

## Registry delivery behavior and validation
A failed refresh now stops package selection; it does not install from stale
cache or silently switch to a lower-priority registry. Browsing/listing retain
existing warning-and-cache behavior. Automatic downgrade is rejected, including
with `--force`; `pkg upgrade name@version` permits an explicitly requested version
when that exact version is present in the selected index. No persistent version
pin or installed-origin metadata is introduced by this delivery.

Fresh clones are staged beside the active cache. Prior directories and failed
staging directories are retained and their recovery paths are reported where
applicable. A failed restoration retains the writer lock. This does not serialize
readers with writers or promise automatic recovery from process termination;
operators must reconcile retained directories/locks before retrying after a crash.

Validated using real temporary Git repositories on Linux: divergent histories,
untracked files, changed configured origin, invalid/symbolic indexes, lock
contention, clone failure and injected directory-rename failures. Full suite:
1,296 passing tests; the subsequent validation-centralization change also passed
57 focused tests. macOS/Windows execution and the binary updater remain pending.
