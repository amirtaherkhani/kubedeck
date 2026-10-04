# Cognee

Cognee is the home lab AI memory and knowledge-graph API for personal project
agents and experiments.

| Field | Value |
|---|---|
| Category | AI / memory and knowledge graph |
| Namespace | `ai-tools` |
| UI | No dedicated UI; API docs are served by the REST API |
| UI/API URL | `https://cognee.local.dev/docs` and `https://cognee.local.dev/api/v1` |
| Port | HTTPS `443`; Cognee service port `8000` |
| Dependencies | Infisical, local-path PVC, Traefik, OpenAI-compatible LLM key |
| Image | `cognee/cognee:1.4.1` pinned to the published manifest digest |
| Storage | `local-path` PVC, 5 Gi; SQLite, LanceDB, and KuzuDB are local to the pod |
| Credentials | `COGNEE_LLM_API_KEY` and `COGNEE_FASTAPI_USERS_JWT_SECRET` in Infisical at `/apps/ai-tools/cognee` |
| Monitoring | Alloy collects pod logs; the upstream API does not provide a Prometheus endpoint in this deployment |

Cognee is deployed as one replica because the selected file-backed database
profile is intentionally local and single-writer. Add the LLM key in Infisical
before using ingestion or search operations.
