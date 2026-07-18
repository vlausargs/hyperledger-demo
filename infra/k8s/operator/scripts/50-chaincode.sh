#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf
CC=basic
CC_LABEL=basic_1.0
# version/sequence 1.1/2, not 1.0/1: an earlier interactive dry-run (before
# this script existed / before tar output was made reproducible, see the
# TAR_REPRO note below) already committed a 1.0/sequence-1 definition on
# this live channel under a package-id that doesn't match this script's
# deterministic build. Fabric lifecycle sequences are append-only/immutable
# once committed, so the script targets the next sequence instead of
# fighting the stale one.
CC_VERSION=1.1
CC_SEQUENCE=2
POLICY="AND('Org1MSP.member','Org2MSP.member')"
IMAGE=hlf-basic-cc:1.0
# Query used purely to prove the committed chaincode answers on-chain: a
# real, zero-side-effect read from the `basic` chaincode's InventoryContract
# (contractapi namespaces functions by Go struct name, so it's
# "<Contract>:<Function>", not the bare function name).
VERIFY_FCN="InventoryContract:GetInventory"
VERIFY_ARG="Org1MSP"

WORK_DIR=$(mktemp -d)
cleanup() {
  # PF_ISTIO_PRIMARY_PID is `sudo`'s own pid, which after exec runs as
  # root — a plain `kill` from this (non-root) script cannot signal it, so
  # the primary tunnel needs `sudo kill` explicitly, not just a bare kill.
  kill "${PF_ISTIO_SECONDARY_PID:-0}" "${PF_ORDERER_PID:-0}" 2>/dev/null || true
  sudo -n kill "${PF_ISTIO_PRIMARY_PID:-0}" 2>/dev/null || true
  sudo -n pkill -f "port-forward svc/istio-ingressgateway 443:443" 2>/dev/null || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT
cd "$WORK_DIR"

# This script deploys `basic` (packages/chaincode) as chaincode-as-a-service
# (ccaas): a Deployment/Service running the pre-built hlf-basic-cc:1.0 image
# (built + `kind load`ed by `make kind-chaincode`, not by this script), wired
# into the peer's external-builder lifecycle via `kubectl hlf externalchaincode
# sync` + the standard connection.json/metadata.json ccaas package, then
# installed on both peers, approved by both orgs, and committed on
# `mychannel` with endorsement AND(Org1MSP,Org2MSP). Idempotent-tolerant: a
# second run recomputes the same package-id, redeploys the ccaas pod, skips
# straight to verification if already committed at CC_VERSION/CC_SEQUENCE
# (re-approving/re-committing an already-committed sequence fails with
# ENDORSEMENT_POLICY_FAILURE / "must be sequence N+1" — confirmed empirically
# on this cluster — so re-running install, which IS naturally idempotent, is
# safe, but approve/commit are gated on querycommitted first).

# ─── Part A: build the ccaas package (connection.json + metadata.json) ──────
# Fabric's standard external-builder package layout for "type": "ccaas":
# code.tar.gz (containing only connection.json, the runtime address the peer
# dials) wrapped with metadata.json into the outer chaincode.tgz that
# `chaincode install` uploads. The peer image ships the ccaas_builder
# external builder (confirmed via `strings` on the kubectl-hlf binary and
# corroborated by kfsoftware/meetup-k8s-hlf-2024) that unpacks this at
# install time instead of compiling Go source.
#
# Port 7052: the Service `kubectl hlf externalchaincode sync` creates for
# the ccaas pod is ALWAYS on port 7052 (confirmed empirically: its `--port`
# flag only rewrites the Deployment's CHAINCODE_SERVER_ADDRESS env value and
# liveness/readiness probe port, not the Service) — so connection.json must
# point at "$CC.$NS:7052" regardless of what the container itself listens
# on, and packages/chaincode/Dockerfile's default (0.0.0.0:7052) is chosen
# to match so the two stay aligned without relying on `sync`'s env override.
cat >metadata.json <<EOF
{
    "type": "ccaas",
    "label": "${CC_LABEL}"
}
EOF
cat >connection.json <<EOF
{
  "address": "${CC}.${NS}:7052",
  "dial_timeout": "10s",
  "tls_required": false
}
EOF
# --sort/--mtime/--owner/--group/--numeric-owner: calculatepackageid hashes
# the raw tgz bytes, and plain `tar cfz` embeds each file's mtime/uid/gid —
# so a naive rebuild with byte-identical connection.json/metadata.json still
# produces a DIFFERENT package-id every run (confirmed empirically: a second
# run computed a different id from the first despite identical content),
# which then desyncs the freshly-deployed ccaas pod's CORE_CHAINCODE_ID_NAME
# from the package-id already approved+committed on the channel — the
# `query` in Part G hangs (peer can't match the running ccaas server's
# advertised ID to the committed definition's) rather than erroring
# cleanly. Force reproducible tar output so the package-id is stable
# run-to-run for the same package contents.
TAR_REPRO=(--sort=name --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner)
tar "${TAR_REPRO[@]}" -czf code.tar.gz connection.json
tar "${TAR_REPRO[@]}" -czf chaincode.tgz metadata.json code.tar.gz

PACKAGE_ID=$(kubectl hlf chaincode calculatepackageid \
  --path=chaincode.tgz --language=golang --label="$CC_LABEL")
echo "Package ID: $PACKAGE_ID"

# ─── Part B: deploy the ccaas pod ────────────────────────────────────────────
# NOTE: `externalchaincode sync`'s *update* path (i.e. calling it a second
# time against an existing FabricChaincode) silently drops --env — confirmed
# empirically: `spec.env` came back `[]` after a second sync despite passing
# --env, even though the identical flags on a *fresh* object populate it
# correctly. Delete-then-recreate instead of relying on sync's update/--force
# path, to guarantee CORE_CHAINCODE_ID_NAME (which contract-api-go v2's
# contractapi.(*ContractChaincode).Start() requires to branch into ccaas
# server mode — see packages/chaincode/Dockerfile's comment) is always set.
kubectl -n "$NS" delete fabricchaincode "$CC" --ignore-not-found >/dev/null 2>&1 || true
for _ in $(seq 1 20); do
  kubectl -n "$NS" get fabricchaincode "$CC" >/dev/null 2>&1 || break
  sleep 2
done

kubectl hlf externalchaincode sync \
  --image="$IMAGE" --name="$CC" --namespace="$NS" \
  --package-id="$PACKAGE_ID" --tls-required=false --replicas=1 \
  --env "CORE_CHAINCODE_ID_NAME=$PACKAGE_ID"

echo "Waiting for ccaas deployment/$CC to be Available..."
kubectl -n "$NS" rollout status deployment/"$CC" --timeout=120s

# ─── Part C: build an SDK network-config for the host-side kubectl-hlf CLI ──
# `kubectl hlf chaincode install/approveformyorg/commit/query` run from the
# HOST (not from a cluster pod), so they need externally-reachable
# endpoints, not in-cluster DNS (`*.hlf` names only resolve inside the
# cluster). `kubectl hlf inspect` (without --internal) emits the Istio
# ingress hostnames set up in Task 5a (`peer0-orgN.localho.st`, which
# publicly resolves to 127.0.0.1/::1 — the whole point of the localho.st
# service) already wired up for Fabric's SNI-passthrough routing; only the
# YAML's `orderers` fields need patching (see NOTE below), plus the admin
# identities added and a live orderer TLS-reachable endpoint filled in.
kubectl -n "$NS" get secret org1-admin -o jsonpath='{.data.user\.yaml}' | base64 -d >org1-admin-user.yaml
kubectl -n "$NS" get secret org2-admin -o jsonpath='{.data.user\.yaml}' | base64 -d >org2-admin-user.yaml

kubectl hlf inspect -o Org1MSP -o Org2MSP -c mychannel -n "$NS" --output config.yaml

# NOTE: fabric-sdk-go's endpoint-config parser wants the top-level
# `orderers:` key to be a MAP but `channels.<name>.orderers` to be a LIST —
# `kubectl hlf inspect` emits `orderers: []` (a list) in both places when no
# orderer node was queried, which fails top-level parsing ("expected a map,
# got slice"). Patch both to their required shapes and fill in a real,
# TLS-reachable orderer endpoint (direct port-forward to the orderer's
# ClusterIP Service below — its cert's SAN includes 127.0.0.1, confirmed via
# openssl, so no SNI/hostname override is needed).
ORDERER_TLSCERT=$(kubectl -n "$NS" get fabricorderernodes.hlf.kungfusoftware.es orderer0 \
  -o jsonpath='{.status.tlsCert}')

python3 - "$ORDERER_TLSCERT" <<'PYEOF'
import sys
import yaml

orderer_tlscert = sys.argv[1]
with open("config.yaml") as f:
    doc = yaml.safe_load(f)

doc["client"]["organization"] = "Org1MSP"
doc["orderers"] = {
    "orderer0.hlf": {
        "url": "grpcs://127.0.0.1:17050",
        "grpcOptions": {"allow-insecure": False},
        "tlsCACerts": {"pem": orderer_tlscert},
    }
}
for org in doc["organizations"].values():
    org["orderers"] = []
doc["channels"]["mychannel"]["orderers"] = ["orderer0.hlf"]

with open("config.yaml", "w") as f:
    yaml.safe_dump(doc, f, default_flow_style=False, sort_keys=False)
PYEOF

kubectl hlf utils adduser --userPath=org1-admin-user.yaml --config=config.yaml --username=admin --mspid=Org1MSP
kubectl hlf utils adduser --userPath=org2-admin-user.yaml --config=config.yaml --username=admin --mspid=Org2MSP

# A second copy whose peer0-org2 entry is routed through the secondary Istio
# tunnel (below) rather than the primary one — used only for `commit`, which
# needs simultaneous connections to both peers' Istio-fronted endpoints;
# empirically, two concurrent SNI connections through the SAME `kubectl
# port-forward` local listener race and one times out ("context deadline
# exceeded"), so `commit` gets org1 on the primary tunnel and org2 on a
# second, independent tunnel to avoid the contention.
python3 <<'PYEOF'
import yaml
with open("config.yaml") as f:
    doc = yaml.safe_load(f)
doc["peers"]["peer0-org2.hlf"]["url"] = "grpcs://peer0-org2.localho.st:28443"
with open("config-commit.yaml", "w") as f:
    yaml.safe_dump(doc, f, default_flow_style=False, sort_keys=False)
PYEOF

# ─── Part D: open the tunnels the host-side CLI calls need ──────────────────
# 1) A literal :443 port-forward to the Istio ingress gateway. Required
#    because `kubectl hlf chaincode approveformyorg` (and `commit`'s
#    submitting-org leg) registers for the commit's TxStatus event on a
#    peer0-orgN.localho.st:443 endpoint it derives itself — confirmed
#    empirically this ignores --config's peer URL/port entirely, so this
#    exact port must be reachable. 443 is a privileged port -> sudo.
# 2) A second, unprivileged port-forward to the same Service, used only for
#    the non-submitting org's leg of `commit` (see config-commit.yaml above)
#    to dodge the single-tunnel concurrency issue.
# 3) A direct port-forward to the orderer0 Service (bypasses Istio; its cert
#    already carries a 127.0.0.1 SAN) for the broadcast/orderer leg of
#    approve+commit.
pkill -f "port-forward svc/istio-ingressgateway" 2>/dev/null || true
pkill -f "port-forward svc/orderer0" 2>/dev/null || true
sleep 1

sudo -n -E --preserve-env=KUBECONFIG,HOME env "PATH=$PATH" "HOME=$HOME" \
  kubectl -n istio-system port-forward svc/istio-ingressgateway 443:443 \
  >"$WORK_DIR/pf-istio-primary.log" 2>&1 &
PF_ISTIO_PRIMARY_PID=$!

kubectl -n istio-system port-forward svc/istio-ingressgateway 28443:443 \
  >"$WORK_DIR/pf-istio-secondary.log" 2>&1 &
PF_ISTIO_SECONDARY_PID=$!

kubectl -n "$NS" port-forward svc/orderer0 17050:7050 \
  >"$WORK_DIR/pf-orderer.log" 2>&1 &
PF_ORDERER_PID=$!

sleep 3
for name in PF_ISTIO_PRIMARY PF_ISTIO_SECONDARY PF_ORDERER; do
  pid_var="${name}_PID"
  if ! kill -0 "${!pid_var}" 2>/dev/null; then
    echo "Tunnel $name failed to start:" >&2
    cat "$WORK_DIR/pf-$(echo "$name" | tr 'A-Z_' 'a-z-' | sed 's/pf-//').log" >&2 2>/dev/null || true
    exit 1
  fi
done

# ─── Part E: install on both peers ───────────────────────────────────────────
# `chaincode install` is naturally idempotent (installing the same
# package-id twice just re-reports success) — confirmed empirically — so no
# extra guard is needed here.
kubectl hlf chaincode install --path=./chaincode.tgz --config=config.yaml \
  --language=golang --label="$CC_LABEL" --user=admin --peer=peer0-org1.hlf
kubectl hlf chaincode install --path=./chaincode.tgz --config=config.yaml \
  --language=golang --label="$CC_LABEL" --user=admin --peer=peer0-org2.hlf

# ─── Part F: approve (both orgs) + commit — skipped if already committed ────
# Re-approving/re-committing an already-committed version+sequence fails
# (ENDORSEMENT_POLICY_FAILURE, or "requested sequence is N, but new
# definition must be sequence N+1" — both confirmed empirically), so check
# querycommitted first and only run approve/commit if this exact
# version+sequence isn't live yet.
ALREADY_COMMITTED=0
if kubectl hlf chaincode querycommitted --config=config.yaml --user=admin \
  --peer=peer0-org1.hlf --channel=mychannel 2>/dev/null \
  | grep -q "^${CC}[[:space:]]*${CC_VERSION}[[:space:]]*${CC_SEQUENCE}[[:space:]]"; then
  ALREADY_COMMITTED=1
fi

if [[ "$ALREADY_COMMITTED" == "1" ]]; then
  echo "basic ${CC_VERSION} (sequence ${CC_SEQUENCE}) already committed on mychannel — skipping approve/commit."
else
  kubectl hlf chaincode approveformyorg --config=config.yaml --user=admin --peer=peer0-org1.hlf \
    --package-id="$PACKAGE_ID" --version="$CC_VERSION" --sequence="$CC_SEQUENCE" \
    --name="$CC" --channel=mychannel --policy="$POLICY"
  kubectl hlf chaincode approveformyorg --config=config.yaml --user=admin --peer=peer0-org2.hlf \
    --package-id="$PACKAGE_ID" --version="$CC_VERSION" --sequence="$CC_SEQUENCE" \
    --name="$CC" --channel=mychannel --policy="$POLICY"

  kubectl hlf chaincode checkcommitreadiness --config=config.yaml --user=admin --peer=peer0-org1.hlf \
    --version="$CC_VERSION" --sequence="$CC_SEQUENCE" --chaincode="$CC" --channel=mychannel --policy="$POLICY"

  kubectl hlf chaincode commit --config=config-commit.yaml --user=admin --mspid=Org1MSP \
    --version="$CC_VERSION" --sequence="$CC_SEQUENCE" --name="$CC" --channel=mychannel --policy="$POLICY"
fi

# ─── Part G: verify with a query ─────────────────────────────────────────────
echo "Chaincode pod:"
kubectl -n "$NS" get pod -l chaincode="$CC"

echo "Committed definition:"
kubectl hlf chaincode querycommitted --config=config.yaml --user=admin --peer=peer0-org1.hlf --channel=mychannel

echo "Verification query ($VERIFY_FCN):"
kubectl hlf chaincode query --config=config.yaml --user=admin --peer=peer0-org1.hlf \
  --chaincode="$CC" --channel=mychannel --fcn="$VERIFY_FCN" -a "$VERIFY_ARG"

echo "basic chaincode committed on mychannel and reachable via query."
