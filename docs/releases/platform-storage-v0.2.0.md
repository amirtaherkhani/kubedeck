# platform-storage v0.2.0

## Highlights

The managed local platform now contains only the shared PostgreSQL, Redis, RabbitMQ, and NATS services. Infisical, DNS, ingress, TLS, and Kubernetes support services remain separate.

## Removed

Unused admin UIs, Kafka and Schema Registry, MongoDB, MinIO, Jaeger, Mailpit, grpcui, Radar, k6 Operator, and inactive observability and autoscaling modules are removed from the managed service catalog. Previously provisioned PVCs are retained.

## Breaking Changes

The removed chart templates and release entries can no longer be deployed through this repository. Consumers of those services must use another deployment source before upgrading.

## Configuration

The Helm workflow selects the Docker Desktop context and applies the storage chart's Docker Desktop overrides. Redis remains reachable only through its in-cluster ClusterIP Service.
