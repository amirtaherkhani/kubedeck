# Service module foundation design

## Purpose and scope

Establish a portable, versioned metadata contract and reusable starter for describing services and tools in the combined KubeDeck/homelab project. The immediate milestone is local and additive: the existing homelab Helm charts, release ordering, secrets, workloads, and UI/agent behavior stay unchanged. It is not a bulk catalog migration or a new runtime plugin loader.

V1 targets one local single-node Kubernetes environment. The contract must distinguish infrastructure managed by the homelab from external applications registered for observation. Vero Finance remains an external client workload, not a KubeDeck-owned service.

## Existing architecture

- Homelab modules currently live under `lab/apps/{platform,observability,dev,ai}/<module>` and vary by local Helm chart, upstream Helm chart, values, manifests, and templates.
- `lab/core/helm/releases.conf` defines deployment order, release name, namespace, chart reference/version, values path, and timeout; `repositories.conf` maps remote chart repositories. These remain the current deployment source of truth.
- Some current modules bundle multiple components in one chart/release. `apps/platform/storage` is the relevant example; a module therefore cannot be assumed to represent exactly one Kubernetes workload or one chart component.
- The Go agent exposes versioned live cluster snapshots (`kubedeck.io/v1alpha1`) over its API/SSE and discovers Kubernetes resources. It has no declarative service metadata catalog today. Its discovery remains read-only.
- The UI consumes the agent's current model. This milestone adds stable shared declarations and types but does not yet add a catalog-serving endpoint or change UI inventory behavior.

## Proposed contract

Use two distinct versioned file contracts rather than one object mixing durable module identity with environment-specific installation settings:

1. **ServiceModule** describes a logical module and its components. It contains stable identity, category, ownership (`managed` or `external`), purpose, capabilities provided and required, external connections, workload references, endpoints, health/observability references, and Infisical reference-only secret bindings.
2. **InstallationProfile** describes one deployment target/environment and references ServiceModule IDs. Its managed installation entries carry Helm source/chart/version, release name, namespace, and relative values-file overrides. V1 profile target is local single-node Kubernetes; machine-specific context and filesystem paths are supplied by the developer at runtime, not committed.

Use a separate API group/version for declarations (proposed `catalog.kubedeck.io/v1alpha1`); do not conflate it with the current live snapshot version `kubedeck.io/v1alpha1`. JSON Schema is the language-neutral structural contract. Go and TypeScript types reflect that schema; fixtures validate in both runtimes. Additional unknown fields fail validation in this first version. Additive schema changes must increment or explicitly revise the contract version and compatibility tests.

Secret metadata contains only references (provider, project/environment reference, and secret path/name). Raw tokens, passwords, keys, or secret values are forbidden. A module's declared capabilities and metadata never grant agent permissions or authorize executable plugin code.

## File and directory layout

```text
contracts/service-module/v1alpha1/schema.json
contracts/installation-profile/v1alpha1/schema.json
kubedeck-agent/internal/catalog/v1alpha1/types.go
src/lib/service-catalog/types.ts
templates/service-module/service.yaml
templates/installation-profile/local-single-node.yaml
examples/service-modules/managed.example.yaml
examples/service-modules/external.example.yaml
examples/installation-profiles/local-single-node.example.yaml
lab/apps/<existing-domain>/<existing-module>/  # existing vendor/chart layout unchanged
lab/core/helm/{releases,repositories}.conf     # existing deployment truth unchanged in this phase
```

Templates and examples are non-installed reference files, not catalog entries. Real service registration or migration requires separate explicit review. Existing modules can adopt descriptors incrementally after approval; the foundation does not move charts or redefine the existing storage release.

## UI and agent relationship

This phase provides versioned Go and TypeScript value types and schema-valid examples. Later work may load module declarations into the agent as read-only metadata, combine them with live Kubernetes observations using explicit workload references, and expose a normalized view for the UI. The exact transport and reconciliation rules are deferred. Discovery continues through `client-go`; use Helm lifecycle APIs for future managed release operations rather than treating render/apply as a complete Helm lifecycle.

## Validation and failure behavior

- Validate each example/profile against its JSON Schema; reject wrong API versions, duplicate identifiers, invalid ownership/deployment combinations, unknown fields, invalid references, absolute machine-specific paths, and secret values.
- Go and TypeScript tests parse the same fixtures and verify the same contract version and core fields.
- Preserve vendor `values.schema.json` and chart-specific values; do not impose identical chart configuration on different upstreams.
- Profile validation checks module references and relative paths. It does not install or alter a cluster.

## Portability and security

- Shared files contain relative repository paths only. Local overrides and cluster context live outside committed profile/module files.
- Keep environment data in profiles and secret values in Infisical-backed Kubernetes Secrets; module metadata records references only.
- V1 remains scoped to the current Docker Desktop cluster; internet scanning, exposed Kubernetes API, arbitrary executable modules, and staging/production actions are outside its local development workflow.
- Future remote Linux/Mac hosts require a separate reviewed design for enrollment, authentication, topology, and failure isolation.

## Acceptance criteria

1. A generic managed module, a generic external registration, and a local single-node profile example validate against versioned schemas without referring to a real service or containing secret values.
2. Go agent and TypeScript UI types represent the same API version and core contract fields; tests parse the common fixtures.
3. Example and template paths are documented, portable, and reference existing Helm module locations without moving or changing those charts.
4. No live Kubernetes resources, Helm releases, existing service catalog entries, secrets, or production/staging environments are changed.

## References

- [Helm chart format](https://helm.sh/docs/topics/charts/) and [Helm SDK examples](https://helm.sh/docs/sdk/examples/)
- [Kubernetes common labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/) and [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
- [Helm dependency build](https://helm.sh/docs/helm/helm_dependency_build/) and [CRD chart guidance](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)
- [Infisical Kubernetes integration](https://infisical.com/docs/integrations/platforms/kubernetes/overview); do not migrate operator APIs without first identifying the installed version.
