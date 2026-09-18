# Skill resources during deployment

Skills may link supporting files under `references/`, `scripts/`, `assets/`,
and `agents/` (including `agents/openai.yaml`). The canonical parser loads these
directories recursively, preserving relative paths and file bytes. Providers
that emit skill resources, including Codex, receive the complete resource map.
The internal `ScriptFiles` field retains its existing name for compatibility.

Symbolic links and non-regular resource files are rejected with an explicit
error. Keep resources inside the skill directory; do not link external files.
This change preserves content, not executable permission bits.

For a local package registry, register a Git repository containing
`package-index.yaml`, select its branch, and install with `--source`. A staging
directory alone is not a registered registry. The index `path` identifies the
package directory; it need not match the package name. Current main already
supports this resolution, so a cache lookup failure on an older installed CLI
must also be checked against the current version.

Regression coverage exercises cold-cache installation and forced reinstallation
from a local Git registry, through canonical installation and real Codex
deployment into an isolated `.agents/skills` directory. It does not modify the
user's global configuration or install an updated CLI.
