# Temporal

This deploys the official Temporal Helm chart with:

- Temporal server services in the `temporal` namespace
- Temporal Web UI exposed at `https://temporal.local.dev`
- Temporal frontend exposed by LoadBalancer on port `7233`
- SQL persistence backed by the existing `storage/postgresql` service

## Deploy

Create the database secret and databases first, then install the chart:

```sh
kubectl create namespace temporal --dry-run=client -o yaml | kubectl apply -f -

export TEMPORAL_DB_PASSWORD="$(openssl rand -hex 24)"

kubectl create secret generic temporal-postgresql \
  --namespace temporal \
  --from-literal=password="$TEMPORAL_DB_PASSWORD" \
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
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'temporal') THEN
    CREATE ROLE temporal WITH LOGIN PASSWORD '${TEMPORAL_DB_PASSWORD}';
  ELSE
    ALTER ROLE temporal WITH LOGIN PASSWORD '${TEMPORAL_DB_PASSWORD}';
  END IF;
END
\$\$;

SELECT 'CREATE DATABASE temporal OWNER temporal'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'temporal')\\gexec

SELECT 'CREATE DATABASE temporal_visibility OWNER temporal'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'temporal_visibility')\\gexec

GRANT ALL PRIVILEGES ON DATABASE temporal TO temporal;
GRANT ALL PRIVILEGES ON DATABASE temporal_visibility TO temporal;
SQL

helm upgrade --install temporal temporalio/temporal \
  --version 1.6.0 \
  --namespace temporal \
  --values apps/dev/temporal/values.yaml \
  --wait \
  --timeout 10m
```

## Access

Temporal frontend:

```text
temporal-frontend.temporal.svc.cluster.local:7233
temporal.local.dev:7233
```

Temporal Web UI:

```text
https://temporal.local.dev
```

## Verify

```sh
kubectl get pods,svc,ingress -n temporal
kubectl exec -n temporal deploy/temporal-admintools -- temporal operator cluster health
```
