# Infisical Secrets Operator

Synchronizes Infisical secret and configuration values into Kubernetes Secrets.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | No |
| Dependency | Infisical backend |
| Observability | Controller health and metrics when supported |

Application charts reference stable Secret names; the operator owns their
contents.

The scoped watch and RBAC list also includes `vero-vault-finance` for Finance
application configuration reconciliation.

The scoped watch and RBAC list also includes
`vero-vault-finance-load-test`. This namespace is dedicated to Finance k6
runner configuration; the operator may reconcile only the InfisicalSecret and
native Secret resources required by that namespace. Finance application
runtime credentials remain in the separate application namespace and are not
copied into load-test runners.
