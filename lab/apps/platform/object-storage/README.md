# Docker Desktop object storage

This independent Helm release runs SeaweedFS `weed mini` as a single-node S3 service. It is intended for the Docker Desktop development cluster; the release owns only its own StatefulSet, Service, Ingress and PVC.

## Dependencies

- The `platform-storage` release and Infisical Operator must have populated the `platform-storage/minio` Kubernetes Secret with `MINIO_ACCESS_KEY` and `MINIO_SECRET_KEY`. This is a V1 credential bridge; a later module contract can give object storage its own Secret.
- Traefik must watch `platform-storage`; `local-dev-tls` must exist in that namespace; Technitium must resolve `s3.local.dev` to the Mac LAN address.
- The Docker Desktop `standard` StorageClass must be available.

From the repository root, run `lab/scripts/docker-desktop-object-storage.sh validate`, then `lab/scripts/docker-desktop-object-storage.sh apply`. The apply step enables versioning on the `default` bucket and verifies its status. The API is `https://s3.local.dev` from a configured client and `http://object-storage.platform-storage.svc.cluster.local:8333` inside the cluster. Use path-style S3 requests and the `default` bucket.

The older `data-minio-0` PVC is retained separately. This chart does not migrate its objects. The SeaweedFS `data-object-storage-0` PVC contains object bytes, metadata and the SSE-S3 key; back it up and restore it as one unit before relying on this service for irreplaceable data. No SeaweedFS admin endpoint is exposed through the Service or Ingress.
