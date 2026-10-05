# n8n

This deploys n8n as the repository-managed `n8n` Helm release.

## Bootstrap

Create the namespace, generated secret values, Postgres role, and Postgres
database before applying the chart:

```sh
kubectl create namespace n8n --dry-run=client -o yaml | kubectl apply -f -

export N8N_DB_PASSWORD="$(openssl rand -hex 24)"
export N8N_ENCRYPTION_KEY="$(openssl rand -hex 32)"

kubectl create secret generic n8n \
  --namespace n8n \
  --from-literal=DB_POSTGRESDB_PASSWORD="$N8N_DB_PASSWORD" \
  --from-literal=N8N_ENCRYPTION_KEY="$N8N_ENCRYPTION_KEY" \
  --dry-run=client -o yaml | kubectl apply -f -

POSTGRES_ADMIN_PASSWORD="$(
  kubectl get secret -n platform-storage postgresql \
    -o jsonpath='{.data.POSTGRES_POSTGRES_PASSWORD}' | base64 -d
)"

kubectl exec -i -n platform-storage postgresql-0 -- \
  env PGPASSWORD="$POSTGRES_ADMIN_PASSWORD" \
  psql -U postgres -d postgres <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'n8n') THEN
    CREATE ROLE n8n WITH LOGIN PASSWORD '${N8N_DB_PASSWORD}';
  ELSE
    ALTER ROLE n8n WITH LOGIN PASSWORD '${N8N_DB_PASSWORD}';
  END IF;
END
\$\$;

SELECT 'CREATE DATABASE n8n OWNER n8n'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'n8n')\\gexec

GRANT ALL PRIVILEGES ON DATABASE n8n TO n8n;
SQL
```

## Deploy

```sh
make helm-validate RELEASE=n8n
make helm-apply RELEASE=n8n
```

## Access

```text
https://n8n.local.dev
```

## Verify

```sh
kubectl get pods,svc,ingress,servicemonitor -n n8n
curl -fsS https://n8n.local.dev/healthz
curl -fsS https://n8n.local.dev/healthz/readiness
curl -fsS https://n8n.local.dev/metrics | head
```
