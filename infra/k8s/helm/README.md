# HLF Helm Charts

Kubernetes Helm charts for the production-ready Hyperledger Fabric supply-chain stack.

## Charts

| Chart        | Purpose                                                      |
|--------------|--------------------------------------------------------------|
| `fabric/`    | Orderer + peers + CouchDB + CAs + per-CA PostgreSQL backends |
| `api/`       | Go REST API with HPA (CPU 70 percent, min 2, max 10)         |
| `web/`       | SvelteKit frontend                                           |
| `monitoring/`| Prometheus + Loki + Grafana stack                            |

All images expected to live in `ghcr.io/myindo/*`. Override `image.tag` per release.

## Install

```bash
# Lint everything first
helm lint infra/k8s/helm/*/

# Fabric network (run once)
helm install fabric infra/k8s/helm/fabric \
  --namespace hlf --create-namespace \
  --set global.storageClass=standard

# API (one release per org)
helm install api-org1 infra/k8s/helm/api \
  --namespace hlf \
  --set env.MSP_ID=Org1MSP \
  --set ingress.hosts[0].host=api-org1.example.com

# Web
helm install web infra/k8s/helm/web \
  --namespace hlf \
  --set ingress.hosts[0].host=app.example.com

# Monitoring
helm install monitoring infra/k8s/helm/monitoring \
  --namespace observability --create-namespace \
  --set ingress.hosts.grafana=grafana.example.com
```

## Notes

- HPA on `api/` targets 70 percent CPU per spec section 8 (50K+ txns/day capacity).
- Per-CA PostgreSQL instances for blast-radius isolation; switch to a single shared instance by overriding `postgres.replicas`.
- `ingressClassName` defaults to `nginx`; override for Traefik/Caddy ingress controllers.
- Fabric crypto material (MSP, TLS) is expected to be mounted via separate Secrets bootstrapped by the deployment scripts in `fabric-network/scripts/deployment/` (out of scope for these charts).
