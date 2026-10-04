---
name: kubedeck-versioning-release
description: Use when choosing a KubeDeck, kubedeck-agent, Helm chart, or KubeDeck environment MCP version; updating a changelog; or preparing a versioned tag or GitHub release.
---

# KubeDeck Versioning and Releases

Use this skill to keep each KubeDeck component's version, changelog, tag, release channel, and published artifact aligned. A release is identified by its component and source commit, not by the newest version-looking number anywhere in the monorepo. Before naming a release, record the product/package version, every relevant chart package version and `appVersion`, image tag, tag name, and channel. Do not report one of these numbers as if it were another.

## Identify the release unit first

Inspect the target branch, working-tree state, release history, tags, manifests, workflows, and package metadata before choosing a version. Versions belong to independently shipped components:

| Release unit | Authoritative version sources | Notes |
| --- | --- | --- |
| KubeDeck application | Root `package.json` and lockfile; `charts/kubedeck/Chart.yaml` | Keep dashboard package and chart `appVersion` aligned. The chart's `version` is the Helm chart package version and can change independently when chart packaging changes. |
| KubeDeck agent | `kubedeck-agent` release metadata and `charts/kubedeck-agent/Chart.yaml` | Check agent image/version metadata and chart `appVersion`; align them with the application release when the release process ships both together. |
| `platform-storage` Helm release | Its own chart `Chart.yaml` and deployment configuration under `my-home-lab/apps/platform/storage/` | Independent platform release. Do not bump it just because KubeDeck changes. |
| KubeDeck environment MCP tool | `my-home-lab/tools/kubedeck-env-mcp/package.json` and lockfile | Independent local tool version. It is currently marked `private`; do not publish it to npm unless its package policy and publication workflow are deliberately changed. Record its changes in `my-home-lab/CHANGELOG.md`. |

Use current repository evidence for every release. For example, KubeDeck `0.1.2`, the proposed KubeDeck `0.1.3` in PR #15, `platform-storage` `0.1.8`, and the environment MCP tool `1.0.0` describe separate version lines; these are examples, not defaults or current-state assertions.

When the dashboard and agent ship together, name the application release once and verify both immutable image tags and both chart `appVersion` values. A chart-only packaging change increments that chart's `version` while its `appVersion` continues to identify the application image. Never assume that equal chart and app versions are required for all future changes.

## Choose a SemVer version and channel

Use `MAJOR.MINOR.PATCH` for stable versions. For a `0.x` component, treat compatibility as pre-stable: use a patch increment for compatible fixes, a minor increment for new functionality and breaking changes, and reserve `1.0.0` for a deliberate stability and compatibility commitment. Once at `1.x`, use major for incompatible public changes, minor for compatible features, and patch for compatible fixes.

Use SemVer prerelease identifiers for beta or release-candidate builds, such as `0.1.3-beta.1` or `0.1.3-rc.1`. Do not relabel a `0.x` build as `1.x` merely to make it appear stable. A prerelease must be published with the GitHub prerelease flag and must not be promoted as the latest stable release. Promote only a separately verified stable version. Build metadata (`+...`) does not distinguish release precedence and is not a replacement for an incremented version.

Keep names consistent:

- Stable tag: `v0.1.3`; release title: `KubeDeck v0.1.3`.
- Beta tag: `v0.1.3-beta.1`; release title: `KubeDeck v0.1.3-beta.1 (Beta 1)`; mark as prerelease.
- Use component-qualified titles/tags for independently released components when the repository's established tag scheme requires it. Existing examples include `v0.1.2` for KubeDeck and `platform-storage-v0.1.8` for platform-storage. Inspect tags first; never invent a second naming scheme midstream.

Words such as “beta,” “stable,” “latest,” and “production” in a UI or request do not determine the version channel. Confirm the requested release unit and intended channel when they are ambiguous.

## Changelog is part of every version

Before creating **any version tag or release, including a beta tag**, add and review a dated entry for that exact version in the component's changelog. For KubeDeck, use the root `CHANGELOG.md`; for platform-storage and the local MCP tool, use `my-home-lab/CHANGELOG.md`. Follow the existing Keep a Changelog headings. Include user-visible changes, fixes, compatibility or migration notes, and known limitations. Keep `[Unreleased]` separate; retain all prior entries and links. A stable release after prereleases summarizes all changes since the previous stable release.

Example heading:

```markdown
## [0.1.3-beta.1] - YYYY-MM-DD
```

Never postpone the changelog until after tagging or publishing. If a tag already exists without its matching dated entry, stop and reconcile the release history transparently; do not silently move or force-update a tag.

## Prepare, verify, publish

1. Confirm the exact repository, release unit, target branch/commit, prior release for that unit, and whether the request is stable or prerelease. Name each app/package, Helm chart, and image version separately. Preserve unrelated dirty work; do not release from uncommitted or unreviewed changes.
2. Inspect release scripts and CI triggers to identify the single publishing owner. Check the resulting diff and ensure authoritative version fields, lockfiles, Helm `version`/`appVersion`, image tags, and release notes agree where applicable.
3. Add the dated changelog entry before any tag or publication action. Review it against the full diff since the component's previous release.
4. Run the affected component's documented lint, tests, build, package/chart validation, and required CI. A missing CI result is not a passing result; if no workflow or status checks exist, report that gap and the exact local checks performed. For deployment releases, use the repository's build/deploy policy and verify readiness, endpoints, logs, and persistent-volume health where applicable.
5. Create a tag only at the verified intended source commit and follow the existing tag format. Before a GitHub release, verify the tag, commit, release title, notes, prerelease/latest flags, and artifacts all refer to the same release unit and version.
6. After publishing, read back the remote tag and release metadata and verify the artifact/version. Report the exact component, version/channel, tag, source SHA, changelog, checks, and publication state. Distinguish prepared, tagged, draft, published, and deployed states.

Creating this skill, updating a changelog, or preparing version files does not itself authorize a tag, GitHub release, push, or deployment. Follow the user's current request and repository workflow for each external action.

## Common mistakes

- Bumping `platform-storage` because the KubeDeck app version changed, or vice versa.
- Treating a Helm chart's `version` and `appVersion` as interchangeable.
- Using `v1.0.0` for a `0.x` beta to make it look stable.
- Calling a beta “latest” or omitting the prerelease flag.
- Tagging first and promising to write the changelog later.
- Treating a successful local build as remote publication or deployment health.
