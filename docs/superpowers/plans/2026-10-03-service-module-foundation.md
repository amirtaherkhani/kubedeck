# Service Module Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a portable, versioned service-module and installation-profile contract with validated examples and Go/TypeScript types, without changing installed workloads.

**Architecture:** JSON Schema is the shared contract for YAML module/profile files. A small Node validator checks schemas plus cross-file profile references. Go agent types and UI TypeScript types mirror the same versioned fields; this foundation does not yet serve a catalog at runtime. Templates and fixtures are non-installed examples.

**Tech Stack:** Node.js, Ajv, js-yaml, JSON Schema Draft 7, Go, `sigs.k8s.io/yaml`, TypeScript, Helm-compatible relative paths.

**Spec:** `docs/superpowers/specs/2026-10-03-service-module-foundation-design.md`

## Global Constraints

- V1 describes one local single-node Kubernetes environment.
- Keep existing homelab chart locations, vendor values, Helm release names, namespaces, PVCs, and live workload ownership unchanged.
- Do not add a real service catalog entry or migrate existing app modules in this foundation phase.
- External modules describe observed workloads only and do not include install/deploy instructions.
- Metadata contains secret references only; never raw credentials.
- Do not store absolute local filesystem paths or a developer's cluster context in shared files.
- Keep service-module declarations distinct from executable plugins and agent write permissions.

## Review Focus

- An external module accidentally receives install state: profile-validation test rejects it.
- Duplicate module/component identifiers make UI mappings ambiguous: validator tests reject duplicates.
- Unknown or raw-secret fields leak into shared metadata: strict schema tests reject them.
- Machine-specific paths reduce portability: profile tests reject absolute and parent-relative paths.
- Go/TypeScript drift from the shared contract version: fixture tests assert matching version and core identifiers.

---

### Task 1: Define versioned schemas, validator, template, and fixtures

**Files:**
- Create: `contracts/service-module/v1alpha1/schema.json`
- Create: `contracts/installation-profile/v1alpha1/schema.json`
- Create: `scripts/service-modules/validate.mjs`
- Create: `templates/service-module/service.yaml`
- Create: `templates/installation-profile/local-single-node.yaml`
- Create: `examples/service-modules/managed.example.yaml`
- Create: `examples/service-modules/external.example.yaml`
- Create: `examples/installation-profiles/local-single-node.example.yaml`
- Create: `tests/service-module.test.mjs`
- Modify: `package.json`, `package-lock.json`, `package.json` test script

**Interfaces:**
- `validateServiceModule(value) -> string[]`
- `validateInstallationProfile(profile, modules) -> string[]`
- A managed module has deployment/install data only in a separate InstallationProfile; an external module has observation metadata only.

- [ ] Add Ajv and js-yaml as direct development dependencies using versions already present in the lockfile.
- [ ] Define schemas with strict additional properties, versioned IDs, ownership, components, `provides`, `requires`, external connections, workload refs, endpoints, health, observability refs, and Infisical reference-only bindings.
- [ ] Define profile installations with stable module refs and repository-relative Helm chart/values references.
- [ ] Implement validator errors for duplicate IDs, unknown module refs, external-module installs, absolute/parent-relative paths, and schema violations.
- [ ] Add generic fixtures and templates only; use placeholders and do not register actual deployed services.
- [ ] Run `node --test tests/service-module.test.mjs` and verify the existing KubeDeck unit suite.

### Task 2: Add agent and UI contract types

**Files:**
- Create: `kubedeck-agent/internal/catalog/v1alpha1/types.go`
- Create: `kubedeck-agent/internal/catalog/v1alpha1/types_test.go`
- Create: `src/lib/service-catalog/types.ts`
- Modify: `kubedeck-agent/go.mod` only if the YAML package becomes a direct import

**Interfaces:**
- Go `ServiceModule`, `Component`, `WorkloadRef`, `SecretRef`, and `InstallationProfile` structs use JSON field names aligned with the schemas.
- TypeScript exports corresponding versioned types from `src/lib/service-catalog/types.ts`.

- [ ] Add Go test that parses both module examples and the profile fixture and checks the shared API version and ownership semantics.
- [ ] Add TypeScript compile-time fixture assignments for managed, external, and profile contracts.
- [ ] Implement only the data types and fixture parsing; do not add an HTTP endpoint or change agent RBAC/discovery behavior.
- [ ] Run `cd kubedeck-agent && go test -race ./... && go vet ./...`.
- [ ] Run `npm run lint`, `npm run build`, and `npm run test:unit` from repository root.

### Task 3: Document authoring and compatibility workflow

**Files:**
- Create: `docs/roadmap/service-module-authoring.md`
- Modify: `docs/roadmap/requirements.md`

- [ ] Document how to create a module, validate it, add a separate environment profile installation, use vendor charts without flattening values, and store only secret references.
- [ ] Document that real service registration, chart migration, runtime catalog serving, and plugin execution require separate user-reviewed phases for the current Docker Desktop system.
- [ ] Verify the documented commands exactly match repository scripts and fixtures.

### Task 4: Full verification and review

- [ ] Run root lint/build/unit tests and Go race tests/vet.
- [ ] Run `git diff --check` and review changed-file list for live workload config, credentials, absolute paths, and accidental actual service entries.
- [ ] Confirm no cluster commands, deployment, publication, or external notifications occurred.
