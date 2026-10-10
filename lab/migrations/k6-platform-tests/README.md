# k6 Operator namespace stage

Completed on 2026-10-10: six k6 resources now run in `platform-tests`. Helm release `k6-operator` remains in `observability-tests`, chart **4.5.0**, revision **2**. The original namespace remains. Do not use an ordinary unrendered upgrade: it would place workloads back in the release namespace. Do not uninstall as a migration shortcut: this chart templates and owns both k6 CRDs without a keep policy.

`post-renderer.py` preserves the release namespace while rewriting only the reviewed six resource namespaces, exact ServiceAccount subjects in one RoleBinding and two ClusterRoleBindings, and the ServiceMonitor namespace selector. It requires Python 3 and Mike Farah `yq`. It is scoped to the reviewed chart, not a validator for arbitrary future chart changes; inspect the complete diff for every upgrade.

For Helm 4, configure a process-scoped postrenderer/v1 plugin whose subprocess command is the **absolute path** to this script. Example plugin descriptor:

```yaml
apiVersion: v1
type: postrenderer/v1
name: k6-move
version: 0.1.0
runtime: subprocess
runtimeConfig:
  platformCommand:
    - command: /absolute/repository/lab/migrations/k6-platform-tests/post-renderer.py
```

Place it at `<private-plugin-root>/k6-move/plugin.yaml`, then use `HELM_PLUGINS=<private-plugin-root>` and `--post-renderer k6-move` with the existing release in `observability-tests`. Render with the pinned chart/current values and inspect before any live upgrade; this guide does not authorize future upgrades.

Verification performed:

- Cached chart plus current values produced exactly the approved proposed manifest.
- Missing resources and unexpected RBAC subjects were rejected; independent review found no material issue for that pinned input.
- Controller ready after **11.8 seconds**; Prometheus discovered the `platform-tests` target and scraped it **UP**; Doctor **8/8**.
- Six source objects absent, six target objects present. Both CRDs and five ClusterRoles retained their UIDs/specs/rules. Only the two approved ClusterRoleBinding subjects changed namespace.
- No active TestRuns or PrivateLoadZones existed at cutover. No PVC, Infisical auth or PKI change occurred. All ten retained storage claims remained Bound.

The agreed stage rollback was `helm rollback k6-operator 1 -n observability-tests --wait --timeout 5m` if readiness failed. It was not needed. For any later change, choose its verified pre-change revision rather than blindly using revision 1. Helm rollback does not restore unrelated data or external configuration.
