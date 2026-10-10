# Host helper inventory — 2026-10-10

Verified from tracked command sources, launchd executable paths and the project's installed binary directory.

| File / command | Purpose | Disposition |
|---|---|---|
| `infisical-login` | Optional checkout convenience wrapper | Now calls installed `kuchdesk-host-agent admin-login`; contains no authentication implementation. |
| `.kuchdesk/bin/kuchdesk-enrollment`, `kuchdesk-enrollment-easy`, `kuchdesk-enrollment-fallback` | Existing separately installed enrollment/login builds | No longer needed for normal login. Retained, not deleted; optional legacy status/maintenance remains available. |
| `host-agent/cmd/kuchdesk-enrollment` | Compatibility enrollment CLI | Thin wrapper around `internal/enrollmentcli`, shared with Host `admin-login`; no duplicated login code. |
| `host-agent/cmd/kuchdesk-host-bridge` | Persistent HTTPS bridge and opt-in enrollment controller | Active runtime. Keep separate service lifecycle for now; consolidation would require a service migration, not merely moving a script. |
| `host-agent/cmd/kuchdesk-infisical-mcp` | Stdio MCP adapter | Keep as an integration entry point; it uses shared Go packages. |
| `host-agent/cmd/kuchdesk-infisical` | Operator access/secret metadata and explicit management CLI | Candidate for future Host subcommands, not required for login or continuous reconciliation. |
| `host-agent/cmd/kuchdesk-doctor` | Standalone read-only diagnostic CLI | Shared Doctor package is already used by the Host bridge; standalone frontend is optional. |
| `host-agent/cmd/kuchdesk-deploy` | Explicit developer/operator deployment CLI | Keep as developer tooling; not part of ordinary Host daemon startup. |
| `host-agent/install-macos.sh` | Build/install and launchd provisioning | Keep as installation tooling. |
| `lab/cmd/render-site` | Generate site configuration | Keep as developer/configuration tooling. |
| Task-local `.activation-helper` and `/tmp/kuchdesk-k8-auth-activation` | Bounded live activation checks | Temporary verification tooling, not a runtime dependency; untracked source retained in the active worktree. |
| `/tmp/kuchdesk-grafana-auth-only.py` and process-scoped Helm plugin | Preserve unrelated release resources during the scoped auth cutover | One-time deployment tooling; no running service depends on them. |

Installed Host command: `/Users/mac/.local/bin/kuchdesk-host-agent admin-login --project-root /Users/mac/Documents/GitHub/kuchdesk`.
It was built, installed atomically with a private rollback copy, and its help verified from `/tmp`. No new human login was started. DNS service startup remains non-interactive.
