# Infisical Secrets Operator

Synchronizes Infisical secret and configuration values into Kubernetes Secrets.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | No |
| Dependency | Infisical backend |
| Observability | Controller health and metrics when supported |

The operator watches `platform-secrets`, `observability`, and
`development-tools` for this stack. Grafana's `InfisicalSecret` creates the
`grafana-admin` Secret; the operator owns its contents. The
`development-tools` scope supports a separately enabled, value-free Helm
integration probe. The operator reads its existing Universal Auth Secret in
`platform-secrets`; the credentials are not copied into an application chart.

## TokenRequest permission policy

The approved [targeted narrowing](../../../../docs/infisical-tokenrequest-narrowing.md)
is deployed at Helm revision 4. TokenRequest is limited to
`development-tools/kubedesk-infisical-reader`; the other two watched namespaces
have no manager-Role TokenRequest grant. See the
[observability audit](../../../../docs/infisical-observability-audit-2026-10-10.md).
No current configured consumer uses Kubernetes auth; preserve Universal Auth until
the separately approved activation has its required human-session handoff.

For offline preparation, download only the official pinned archive, then run
from `lab/` (use a new output directory; the builder refuses to overwrite one):

```sh
mkdir -p .generated
python3 scripts/operator-token-scope/prepare_chart.py \
  --archive /path/to/secrets-operator-0.11.11.tgz \
  --output .generated/operator-token-scope
helm template infisical-operator .generated/operator-token-scope/secrets-operator \
  -n platform-secrets -f apps/platform/infisical-operator/values.yaml \
  -f .generated/operator-token-scope/token-request-values.json
```

The archive URL and checksum are in the plan. The offline builder changes only
TokenRequest handling in the existing manager Roles. Do not replace this with a
bare upstream chart upgrade: it would restore broad access. The local chart and
explicit values overlay must be used for subsequent approved upgrades.

**After the separate security-setting approval and current-state diff checks**,
the intended command is:

```sh
helm upgrade infisical-operator .generated/operator-token-scope/secrets-operator \
  -n platform-secrets --reuse-values \
  -f .generated/operator-token-scope/token-request-values.json --wait
python3 scripts/operator-token-scope/check_permissions.py
```

The checker only submits authorization reviews; it never creates tokens. It
passed all four allow/deny checks after the approved revision-4 rollout. Tests run via
`go test ./...`; include the pinned chart integration cases with:

```sh
KUCHDESK_OPERATOR_CHART=/path/to/secrets-operator-0.11.11.tgz \
  python3 -m unittest discover -s scripts/operator-token-scope -v
```
