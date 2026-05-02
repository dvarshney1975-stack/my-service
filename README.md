# my-service

Minimal Go HTTP service deployed via GitOps to k3s on AWS EC2.

## Endpoints
- `GET /` — hello message
- `GET /healthz` — health check
- `GET /info` — JSON service info

## Local dev
```bash
go run .
curl localhost:8080/healthz
```

## Deploy
Push to `main` → GitHub Actions builds image to GHCR → updates `my-gitops` repo → Flux syncs to cluster.
