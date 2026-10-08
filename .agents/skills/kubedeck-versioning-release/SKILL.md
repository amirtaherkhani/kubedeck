---
name: kubedeck-versioning-release
description: Use when versioning the KubeDeck agents, their Helm charts, or the independent local environment MCP tool.
---

# KubeDeck agent versions and releases

Identify the release unit before choosing a version. The web application and its Helm chart have been removed.

| Unit | Version source | Changelog |
| --- | --- | --- |
| Repository release | Annotated `vX.Y.Z` tag on merged `main` | Root `CHANGELOG.md` |
| Cluster agent chart | `charts/kubedeck-agent/Chart.yaml` or `lab/apps/dev/kubedeck-agent/Chart.yaml`, each independently versioned | Root `CHANGELOG.md` for agent changes; `lab/CHANGELOG.md` for lab deployment changes |
| Platform storage chart | `lab/apps/platform/storage/Chart.yaml` | `lab/CHANGELOG.md` |
| Local environment MCP tool | `lab/tools/kubedeck-env-mcp/package.json` and lockfile | `lab/CHANGELOG.md` |

For `0.x` versions, increment the minor version for breaking changes and the patch version for compatible fixes. A chart `version` identifies the chart package; `appVersion` identifies the agent image and changes only when that binary changes. Do not infer one unit's version from another.

Before tagging, compare the full merged diff with the preceding tag for the same unit. Add a dated matching changelog entry, keeping older entries and `[Unreleased]`. Note operator-visible behavior, compatibility, and any required configuration changes. Run the affected Go tests, chart lint/template, and relevant lab validation. A local pass does not prove live deployment health.

Create an annotated tag only on the verified merged source commit. Push it and read back the remote tag. A GitHub release must use the same tag, version, source SHA, and notes. Report source, chart, and image versions separately, and distinguish published source from deployed runtime.
