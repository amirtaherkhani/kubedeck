# KubeDeck

Personal home-lab management dashboard.

| Field | Value |
|---|---|
| Namespace | `development-tools` |
| UI | Yes |
| URL | `https://kubedeck.local.dev` |
| Service | HTTP `80` -> container `3000` |
| Storage | `local-path` PVC, 1 Gi |
| Dependencies | KubeDeck Agent, Infisical, Traefik |
| Observability | Health probes; agent-backed cluster data |

The chart references the external KubeDeck project image and does not clone
the project source into this repository.
