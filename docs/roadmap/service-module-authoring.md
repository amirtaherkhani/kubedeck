# Authoring a service module

This guide describes the first, local-only module contract. A module is declarative metadata; it is not an executable plugin, a Helm release, or permission for the agent to write to Kubernetes.

## Files and responsibilities

| File | Purpose |
| --- | --- |
| `contracts/service-module/v1alpha1/schema.json` | Shape of a versioned managed or external module. |
| `contracts/installation-profile/v1alpha1/schema.json` | Local installation settings separate from module identity. |
| `templates/service-module/service.yaml` | Starting point for a new module descriptor. |
| `templates/installation-profile/local-single-node.yaml` | Starting point for a local single-node profile. |
| `examples/service-modules/*.example.yaml` | Generic, non-installed examples for managed and external ownership. |
| `examples/installation-profiles/*.example.yaml` | Generic, non-installed profile example. |

Existing charts and upstream values remain in `lab/apps/{platform,observability,dev,ai}/<module>`. Keep their vendor-specific structure and validate their values with the chart's own `values.schema.json` and Helm tooling. `lab/core/helm/releases.conf` and `repositories.conf` remain the current deployment source of truth; adding a descriptor does not install or migrate a release. A module may describe multiple components when one current chart/release bundles several tools, as `apps/platform/storage` does.

## Add a module descriptor

1. Copy `templates/service-module/service.yaml` into the relevant module folder, then replace every example field with verified facts. Assign one stable module ID and distinct component IDs. Set `ownership: managed` for homelab-owned infrastructure, or `ownership: external` for an observed application whose source and deployment belong elsewhere.
2. Describe each component's type and purpose. Use `provides` and `requires` for capabilities between modules; use `externalConnections` for dependencies owned outside the platform. Record exact Kubernetes workload references and named endpoints where verified.
3. Record health and observability references that exist. Infisical bindings contain project/environment/path/key **references only**. Put no token, password, API key, or secret value in the descriptor.
4. Keep installation state in a separate profile. Managed entries identify the module, Helm chart source/name/version, release name, namespace, and repository-relative values files. External registrations do not appear in `installations`.
5. Compare any future managed profile entry with the current release registry before adopting it. Preserve release name, namespace, Helm ownership, chart version, PVC names, and upgrade behavior. Do not rewrite or move vendor chart values merely to fit a common metadata format.

Use `npm run catalog:validate` to validate the shipped generic fixtures. For a new descriptor and profile, run:

```sh
npm run catalog:validate -- --module path/to/service.yaml --profile path/to/local-profile.yaml
```

The validator checks schemas, duplicate IDs, unknown module references, external-module installations, and local path safety. It does not verify a running Kubernetes cluster or install anything. The current agent snapshot API is `kubedeck.io/v1alpha1`; these declarations use `catalog.kubedeck.io/v1alpha1` and need a later reviewed reconciliation adapter before they affect the UI.

Real service registration and migration require a separately reviewed entry. V1 covers the current local single-node Docker Desktop cluster. Runtime plugin loading, service operations, and incident delivery follow later designs.
