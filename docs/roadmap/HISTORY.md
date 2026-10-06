# Roadmap decision and execution history

Append new entries. Record observation, implementation, target verification, merge, and release as separate events. Do not put credentials, secret values, private environment exports, or raw logs here.

| Date (Asia/Tehran) | ID | Transition | Evidence | Open item |
| --- | --- | --- | --- | --- |
| 2026-10-06 | Baseline | GitHub v0.2.0 selected as latest released version; `main` has four newer commits | [GitHub release](https://github.com/amirtaherkhani/kubedeck/releases/tag/v0.2.0), `git rev-list --count v0.2.0..origin/main` = 4 at inspection | Later commits must be included in the next release comparison |
| 2026-10-06 | V1-01 | Partial read-only Docker Desktop and Kubernetes inventory captured | [Baseline snapshot](baseline-2026-10-06.md) | Service functionality, DNS access, data restore, and consumer connectivity remain unverified |

## Entry template

```markdown
| YYYY-MM-DD | V1-XX | In progress / Implemented / Verified / Blocked / Released | PR or commit; target context; redacted evidence summary | Remaining action or none |
```

On a failed attempt, append a second row showing the failure and a later row for the repair. Keep the original history.
