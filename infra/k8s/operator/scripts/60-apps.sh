#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf
KIND_CLUSTER=hlf

# Resolve the repo root from this script's own location so it works
# regardless of the caller's cwd (`make kind-apps` from the repo root is the
# normal path, but this makes that assumption explicit rather than implicit).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

WORK_DIR=$(mktemp -d)
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

# This script deploys `api` (packages/api, Go/Gin, fabric-gateway SDK) and
# `web` (packages/web, SvelteKit) on the kind cluster and wires the api to
# the operator-managed network end-to-end:
#
#   Part A: build a fabric-gateway connection profile for Org1 (matching the
#           ConnectionProfile struct in packages/api/internal/fabric/connector.go
#           exactly — NOT kubectl-hlf's own networkconfig/inspect output shape,
#           which does not match) pointing at the in-cluster peer0-org1
#           gateway, and store it as Secret `api-connection`. The client
#           identity the api signs with is NOT generated here — it reuses the
#           operator's existing `org1-admin` FabricIdentity Secret directly
#           (cert.pem/key.pem), remapped onto the signcerts/keystore layout
#           connector.go's loadIdentity() expects via the Secret volume's
#           `items[].path` in the api chart's deployment template. Org1MSP's
#           admin identity is a valid Org1MSP member, sufficient for the
#           `basic` chaincode's AND(Org1MSP.member, Org2MSP.member) policy on
#           evaluate/submit.
#   Part B: build+load the api/web images and helm upgrade --install both
#           charts (NodePort 30948/30949).
#   Part C: wait for rollout and run the same health/network-backed
#           verification curls the task's acceptance criteria call for, so a
#           broken deploy fails this script instead of silently completing.
#
# Idempotent-tolerant: re-running regenerates the connection profile Secret
# (kubectl apply), rebuilds+reloads images, and `helm upgrade --install`s
# both releases — safe to run repeatedly against a live cluster.

echo "==> Part A: connection profile for Org1"

# Peer TLS: the CA's `tlsca_cert` (the TLS-issuing sub-CA embedded in each
# FabricCA, NOT `tls_cert`, which is only the CA server's own HTTPS cert) is
# what actually signs peer0-org1's TLS certificate — confirmed by verifying
# peer0-org1's serving cert against it with `openssl verify`. `tls_cert`
# looked like the obvious field to use but does not validate the peer cert.
PEER_TLS_CA_PEM=$(kubectl -n "$NS" get fabriccas org1-ca -o jsonpath='{.status.tlsca_cert}')
if [ -z "$PEER_TLS_CA_PEM" ]; then
  echo "ERROR: could not read org1-ca status.tlsca_cert" >&2
  exit 1
fi
CA_TLS_PEM=$(kubectl -n "$NS" get fabriccas org1-ca -o jsonpath='{.status.tls_cert}')

# Org2 is only listed in the profile's `organizations`/`peers` maps for
# completeness (so GET /api/v1/network/organizations — a plain echo of the
# parsed profile, see packages/api/internal/handler/network.go — reports
# both orgs) and is never dialed by connector.go's connect(), which only
# ever looks at client.organization (Org1MSP) and that org's first peer.
# The actual cross-org endorsement for the AND(Org1MSP,Org2MSP) chaincode
# policy happens peer-side via gossip/discovery once the client submits
# through peer0-org1's gateway, not via anything this profile declares.
PEER2_TLS_CA_PEM=$(kubectl -n "$NS" get fabriccas org2-ca -o jsonpath='{.status.tlsca_cert}')
CA2_TLS_PEM=$(kubectl -n "$NS" get fabriccas org2-ca -o jsonpath='{.status.tls_cert}')

printf '%s' "$PEER_TLS_CA_PEM" >"$WORK_DIR/peer1-tls-ca.pem"
printf '%s' "$CA_TLS_PEM" >"$WORK_DIR/ca1-tls.pem"
printf '%s' "$PEER2_TLS_CA_PEM" >"$WORK_DIR/peer2-tls-ca.pem"
printf '%s' "$CA2_TLS_PEM" >"$WORK_DIR/ca2-tls.pem"

python3 - "$WORK_DIR/peer1-tls-ca.pem" "$WORK_DIR/ca1-tls.pem" \
  "$WORK_DIR/peer2-tls-ca.pem" "$WORK_DIR/ca2-tls.pem" \
  "$WORK_DIR/connection-profile.yaml" <<'PYEOF'
import sys
import yaml

peer1_ca_path, ca1_path, peer2_ca_path, ca2_path, out_path = sys.argv[1:6]


def read(path):
    with open(path) as f:
        return f.read()


peer1_tls_ca_pem = read(peer1_ca_path)
ca1_tls_pem = read(ca1_path)
peer2_tls_ca_pem = read(peer2_ca_path)
ca2_tls_pem = read(ca2_path)

# Shape matches ConnectionProfile in packages/api/internal/fabric/connector.go
# exactly (field-for-field, including yaml tags): name/version/client/
# organizations/peers/certificateAuthorities. The api connects as Org1MSP
# (client.organization) via peer0-org1 only; Org2MSP/peer0-org2 are included
# for a complete/accurate profile (see comment above), not because
# connector.go reads them.
profile = {
    "name": "hlf-kind-network",
    "version": "1.0",
    "client": {
        "organization": "Org1MSP",
        "connection": {"timeout": {"peer": {"endorser": "300"}}},
    },
    "organizations": {
        "Org1MSP": {
            "mspid": "Org1MSP",
            "peers": ["peer0-org1"],
            "certificateAuthorities": ["org1-ca"],
        },
        "Org2MSP": {
            "mspid": "Org2MSP",
            "peers": ["peer0-org2"],
            "certificateAuthorities": ["org2-ca"],
        },
    },
    "peers": {
        "peer0-org1": {
            "url": "grpcs://peer0-org1.hlf:7051",
            "tlsCACerts": {"pem": peer1_tls_ca_pem},
            "grpcOptions": {"ssl-target-name-override": "peer0-org1.hlf"},
        },
        "peer0-org2": {
            "url": "grpcs://peer0-org2.hlf:7051",
            "tlsCACerts": {"pem": peer2_tls_ca_pem},
            "grpcOptions": {"ssl-target-name-override": "peer0-org2.hlf"},
        },
    },
    "certificateAuthorities": {
        "org1-ca": {
            "url": "https://org1-ca.hlf:7054",
            "caName": "ca",
            "tlsCACerts": {"pem": ca1_tls_pem},
            "httpOptions": {"verify": True},
        },
        "org2-ca": {
            "url": "https://org2-ca.hlf:7054",
            "caName": "ca",
            "tlsCACerts": {"pem": ca2_tls_pem},
            "httpOptions": {"verify": True},
        },
    },
}

with open(out_path, "w") as f:
    yaml.safe_dump(profile, f, default_flow_style=False, sort_keys=False)
PYEOF

echo "--- generated connection-profile.yaml (peer section) ---"
python3 -c "
import yaml
d = yaml.safe_load(open('$WORK_DIR/connection-profile.yaml'))
print('name:', d['name'])
print('client.organization:', d['client']['organization'])
print('peers:', list(d['peers'].keys()))
print('peers.peer0-org1.url:', d['peers']['peer0-org1']['url'])
print('has peer TLS pem:', bool(d['peers']['peer0-org1']['tlsCACerts']['pem']))
"

kubectl -n "$NS" create secret generic api-connection \
  --from-file=connection-profile.yaml="$WORK_DIR/connection-profile.yaml" \
  --dry-run=client -o yaml | kubectl apply -f -

# Sanity check: org1-admin FabricIdentity secret (reused directly as the
# api's client identity — see comment above) actually has the keys the
# deployment's volume `items` remap expects.
kubectl -n "$NS" get secret org1-admin -o jsonpath='{.data.cert\.pem}' >/dev/null
kubectl -n "$NS" get secret org1-admin -o jsonpath='{.data.key\.pem}' >/dev/null
echo "org1-admin secret has cert.pem/key.pem — OK to reuse as api identity"

echo "==> Part B: build + load images, deploy charts"
cd "$REPO_ROOT"
docker build -t hlf-api:local -f infra/docker/Dockerfiles/api.Dockerfile .
docker build -t hlf-web:local -f infra/docker/Dockerfiles/web.Dockerfile .
kind load docker-image hlf-api:local hlf-web:local --name "$KIND_CLUSTER"

# ingress/autoscaling default on in the shared chart values (meant for a
# real cluster with an ingress controller + metrics-server); this kind
# cluster has neither, so disable both here rather than changing the
# chart's general-purpose defaults.
helm upgrade --install api infra/k8s/helm/api -n "$NS" \
  --set image.repository=hlf-api \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent \
  --set ingress.enabled=false \
  --set autoscaling.enabled=false \
  --timeout=180s

helm upgrade --install web infra/k8s/helm/web -n "$NS" \
  --set image.repository=hlf-web \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent \
  --set ingress.enabled=false \
  --timeout=180s

# The image tag stays "local" across re-runs, so a `kind load` that replaces
# the underlying image content is invisible to the Deployment's pod template
# (no field actually changed) and `helm upgrade` alone will not roll the
# pods — it just re-applies the same spec onto whatever's already running.
# Force a rollout so already-running pods (potentially crash-looping on a
# stale image, e.g. before this chaincode-namespacing fix) pick up the
# freshly loaded image.
kubectl -n "$NS" rollout restart deploy/api
kubectl -n "$NS" rollout restart deploy/web

echo "==> Part C: verify end-to-end"
kubectl -n "$NS" rollout status deploy/api --timeout=180s
kubectl -n "$NS" rollout status deploy/web --timeout=180s

echo "--- GET /health ---"
curl -sf http://localhost:30948/health
echo

echo "--- login + GET /api/v1/network/organizations ---"
TOKEN=$(curl -sf -X POST http://localhost:30948/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"asdqwe123"}' | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')
curl -sf http://localhost:30948/api/v1/network/organizations -H "Authorization: Bearer $TOKEN"
echo

echo "--- GET /api/v1/products (proves namespaced chaincode calls round-trip) ---"
curl -sf http://localhost:30948/api/v1/products -H "Authorization: Bearer $TOKEN"
echo

echo "==> api + web deployed and verified against the operator-managed network"
