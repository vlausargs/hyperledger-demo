#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/manifests"
mkdir -p "$MANIFEST_DIR"

# This script creates channel `mychannel` (FabricMainChannel, peer orgs
# Org1MSP+Org2MSP, orderer org OrdererMSP) and joins both peers
# (FabricFollowerChannel per org). It discovers the real v1.14 CRD schema
# empirically rather than trusting the brief's guessed field names — see the
# NOTEs below for what was actually verified against the running operator.
#
# NOTE 0: `kubectl hlf channelcrd --help` exposes `main`/`follower`
# subcommands (not a single `generate`/`channel generate` command as the
# brief guessed). `kubectl hlf channelcrd main create -o ...` and
# `... follower create -o ...` PRINT a FabricMainChannel/FabricFollowerChannel
# manifest to stdout without applying it (confirmed: after several -o runs,
# `kubectl -n hlf get fabricmainchannels` still showed nothing) — they query
# the live FabricCA/FabricOrdererNode/FabricPeer CRs to embed real certs, so
# generating from THIS cluster's live state is far more reliable than
# hand-authoring the CRD YAML from scratch.

# ─── Part A: register admin identities at each CA (host-side, SAN workaround) ─
# Same NodePort/SAN mismatch as Tasks 3/4 (see 20-orderer.sh NOTE 2): `kubectl
# hlf ca register` run from the host auto-discovers the CA's NodePort/kind-node
# external IP, which isn't a SAN on the CA's TLS serving cert. Route through a
# port-forward to the CA's ClusterIP service and point --ca-url at 127.0.0.1.
register_admin() {
  local ca=$1 mspid=$2 port=$3
  kubectl -n "$NS" port-forward "svc/${ca}" "${port}:7054" \
    >"/tmp/${ca}-admin-portforward.log" 2>&1 &
  local pid=$!
  trap 'kill "'"$pid"'" 2>/dev/null || true' EXIT
  sleep 2

  local log
  log=$(mktemp)
  if ! kubectl hlf ca register \
    --name="$ca" --namespace="$NS" \
    --user=admin --secret=adminpw \
    --type=admin --enroll-id=enroll --enroll-secret=enrollpw \
    --mspid="$mspid" \
    --ca-url="https://127.0.0.1:${port}" >"$log" 2>&1; then
    if grep -qi "already registered" "$log"; then
      echo "$ca: admin identity already registered — continuing (safe to ignore)"
    else
      cat "$log" >&2
      kill "$pid" 2>/dev/null || true
      exit 1
    fi
  else
    echo "$ca: admin identity registered"
  fi

  kill "$pid" 2>/dev/null || true
  trap - EXIT
}

register_admin org1-ca Org1MSP 17057
register_admin org2-ca Org2MSP 17058
register_admin ord-ca OrdererMSP 17059

# ─── Part B: enroll the admins via FabricIdentity CRs (operator-side) ────────
# FabricIdentity's `register` block is optional (nullable) — since the admins
# are already registered above, we omit it and only enroll (fetch cert+key).
#
# NOTE 1 (the real hard part): FabricIdentity.spec.catls.cacert must be the
# CA server's own TLS *serving* certificate (FabricCA status.tls_cert), NOT
# the tlsca sub-CA's root (status.tlsca_cert) and NOT the enrollment-signing
# CA's root (status.ca_cert). Verified empirically:
#   openssl s_client -connect 127.0.0.1:<port-forwarded-ca> -showcerts
# shows the CA's actual listener cert is SELF-SIGNED (subject == issuer,
# "O=Hyperledger, OU=Fabric") — its sha256 fingerprint matches
# status.tls_cert exactly, and it is NOT chained to status.tlsca_cert
# ("O=Kung Fu Software, CN=tlsca"; `openssl verify` against it fails with
# "self-signed certificate" / "unable to get local issuer certificate").
# Using tlsca_cert (or ca_cert) here makes the operator's in-cluster enroll
# call fail with "x509: certificate signed by unknown authority". This is
# true for BOTH `caname: ca` (MSP identity) and `caname: tlsca` (TLS
# identity, see NOTE 1b) enrollments below — confirmed empirically both use
# status.tls_cert as catls.cacert.
create_identity() {
  local name=$1 ca=$2 mspid=$3 caname=$4
  local tlscert
  tlscert=$(kubectl -n "$NS" get fabriccas.hlf.kungfusoftware.es "$ca" \
    -o jsonpath='{.status.tls_cert}' | base64 -w0)
  cat <<EOF | kubectl apply -f -
apiVersion: hlf.kungfusoftware.es/v1alpha1
kind: FabricIdentity
metadata:
  name: ${name}
  namespace: $NS
spec:
  cahost: ${ca}.${NS}
  caname: ${caname}
  caport: 7054
  catls:
    cacert: $tlscert
  enrollid: admin
  enrollsecret: adminpw
  mspid: $mspid
EOF
}

create_identity org1-admin org1-ca Org1MSP ca
create_identity org2-admin org2-ca Org2MSP ca
create_identity ord-admin ord-ca OrdererMSP ca

# NOTE 1b (delta discovered during live debugging — see
# .superpowers/sdd/progress.md Task 5): the operator's osnadmin
# (channel-participation) call, used both for the orderer's internal join
# (NOTE 5 below) and for the follower peers' join, needs a TLS *client*
# identity keyed "<MSP>-tls" in the channel CR's `identities` map, or the
# join fails with "tls: certificate required". Enroll a second identity per
# org against the CA's `tlsca` profile (same admin/adminpw credentials
# registered in Part A — registration is not caname-specific on this
# fabric-ca-server, so no extra `ca register` call is needed). Reuses the
# same create_identity helper, just enrolled against `caname: tlsca` and
# named with a `-tls` suffix so it doesn't collide with the MSP identity.
create_identity org1-admin-tls org1-ca Org1MSP tlsca
create_identity org2-admin-tls org2-ca Org2MSP tlsca
create_identity ord-admin-tls ord-ca OrdererMSP tlsca

echo "Waiting for admin FabricIdentity CRs to reach RUNNING..."
for name in org1-admin org2-admin ord-admin org1-admin-tls org2-admin-tls ord-admin-tls; do
  state=""
  for _ in $(seq 1 30); do
    state=$(kubectl -n "$NS" get fabricidentities.hlf.kungfusoftware.es "$name" \
      -o jsonpath='{.status.status}' 2>/dev/null || true)
    [[ "$state" == "RUNNING" ]] && break
    sleep 3
  done
  echo "  $name: ${state:-<none>}"
  if [[ "$state" != "RUNNING" ]]; then
    kubectl -n "$NS" get fabricidentities.hlf.kungfusoftware.es "$name" -o yaml >&2
    exit 1
  fi
done
# Each FabricIdentity's operator-managed Secret stores the enrolled cert+key
# combined under a single key "user.yaml" (confirmed via
# `kubectl -n hlf get secret org1-admin -o jsonpath='{.data}'` ->
# cert.pem/key.pem/root.pem/user.yaml) — that's the secretKey referenced by
# the channel CRs' identity/hlfIdentity fields below.

# ─── Part C: generate FabricMainChannel + FabricFollowerChannel manifests ────
ORDERER_TLSCERT_FILE=$(mktemp)
kubectl -n "$NS" get fabricorderernodes.hlf.kungfusoftware.es orderer0 \
  -o jsonpath='{.status.tlsCert}' >"$ORDERER_TLSCERT_FILE"

# NOTE 2: `channelcrd main create`'s --identities flag takes
# "<mspid>;<secretKey>" pairs (NOT "<mspid>=<secretName>" as one might guess),
# and a single --secret-name/--secret-ns pair is applied to ALL identities —
# there's no way to point different MSPs at different secrets from the CLI.
# Since our three admins live in three separate per-org secrets
# (org1-admin/org2-admin/ord-admin), generate with a placeholder shared
# secret name and patch spec.identities afterwards to the real per-org
# secretName values.
#
# NOTE 3: the CLI auto-populates ordererOrganizations[].ordererEndpoints by
# discovering the orderer's NodePort/kind-node external address (observed:
# "172.22.0.2:0" — wrong port too), not the in-cluster orderer0.hlf:7050
# endpoint the channel config actually needs. Patched below.
kubectl hlf channelcrd main create -o \
  --channel-name=mychannel \
  --peer-orgs=Org1MSP --peer-orgs=Org2MSP \
  --orderer-orgs=OrdererMSP \
  --admin-peer-orgs=Org1MSP --admin-peer-orgs=Org2MSP \
  --admin-orderer-orgs=OrdererMSP \
  --consenters="orderer0.$NS:7050" \
  --consenter-certificates="$ORDERER_TLSCERT_FILE" \
  --identities="Org1MSP;user.yaml" \
  --identities="Org2MSP;user.yaml" \
  --identities="OrdererMSP;user.yaml" \
  --secret-name=org1-admin \
  --secret-ns="$NS" \
  --name=mychannel >"$MANIFEST_DIR/mainchannel.yaml.raw"

python3 - "$MANIFEST_DIR/mainchannel.yaml.raw" "$MANIFEST_DIR/mainchannel.yaml" "$NS" "$ORDERER_TLSCERT_FILE" <<'PYEOF'
import sys
import yaml

src, dst, ns, tlscert_file = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
with open(src) as f:
    doc = yaml.safe_load(f)
with open(tlscert_file) as f:
    orderer_tlscert = f.read()

# NOTE 1b: the MSP identities alone aren't enough — the operator's
# osnadmin/channel-participation call additionally needs a TLS client
# identity per org, keyed "<MSP>-tls", pointing at the tlsca-enrolled
# FabricIdentity secrets created in Part B. Without these the orderer join
# (and the followers' peer join, which shares this identities map's TLS
# entries via mspId lookup) fails with "tls: certificate required".
doc["spec"]["identities"] = {
    "Org1MSP": {"secretKey": "user.yaml", "secretName": "org1-admin", "secretNamespace": ns},
    "Org1MSP-tls": {"secretKey": "user.yaml", "secretName": "org1-admin-tls", "secretNamespace": ns},
    "Org2MSP": {"secretKey": "user.yaml", "secretName": "org2-admin", "secretNamespace": ns},
    "Org2MSP-tls": {"secretKey": "user.yaml", "secretName": "org2-admin-tls", "secretNamespace": ns},
    "OrdererMSP": {"secretKey": "user.yaml", "secretName": "ord-admin", "secretNamespace": ns},
    "OrdererMSP-tls": {"secretKey": "user.yaml", "secretName": "ord-admin-tls", "secretNamespace": ns},
}
for org in doc["spec"]["ordererOrganizations"]:
    if org["mspID"] == "OrdererMSP":
        org["ordererEndpoints"] = [f"orderer0.{ns}:7050"]
        org["caName"] = "ord-ca"
        org["caNamespace"] = ns
        # NOTE 5 (superseded — corrected after live debugging, see
        # .superpowers/sdd/progress.md Task 5): an earlier attempt at
        # `orderersToJoin` (in-cluster name/namespace ref) failed with "dial
        # tcp 172.22.0.2:0: connect: connection refused", which looked like
        # a FabricOrdererNode.status.port/adminPort (both 0, no Istio)
        # problem, so that attempt switched to `externalOrderersToJoin`
        # instead. That workaround turned out to be masking the real cause:
        # the channel identities map was missing the "OrdererMSP-tls" TLS
        # client identity (NOTE 1b) osnadmin needs to authenticate the join
        # call at all — the connection-refused symptom was a side effect of
        # the reconcile failing before it got far enough to matter which
        # join path was used. With the TLS identity present, the in-cluster
        # `orderersToJoin` (name/namespace ref, resolved by the operator
        # directly against the FabricOrdererNode's Kubernetes Service —
        # doesn't actually require status.port/Istio) works and is what's
        # live on the cluster. Keep externalOrderersToJoin empty.
        org["orderersToJoin"] = [{"name": "orderer0", "namespace": ns}]
        org["externalOrderersToJoin"] = []

# Safety refresh (NOT needed on a clean run — the orderer is created once,
# before the channel is generated, so orderers[0].tlsCert below already
# matches; this only matters if orderer0 was ever recreated after this
# manifest was first generated, which would rotate its TLS cert and make a
# stale committed manifest mismatch the live orderer). Re-stamp from the
# orderer's current status.tlsCert (fetched fresh above) on every run.
for orderer in doc["spec"]["orderers"]:
    if orderer.get("host") == f"orderer0.{ns}":
        orderer["tlsCert"] = orderer_tlscert

doc["metadata"].pop("creationTimestamp", None)

with open(dst, "w") as f:
    yaml.safe_dump(doc, f, default_flow_style=False, sort_keys=False)
PYEOF
rm -f "$MANIFEST_DIR/mainchannel.yaml.raw"

# NOTE 4: `channelcrd follower create --peers` takes "<name>.<namespace>"
# (dot-separated) — every other separator tried (";", ":", ",", "/", "@")
# was rejected with "invalid peer format".
PATCH_FOLLOWER_PY=$(mktemp)
cat >"$PATCH_FOLLOWER_PY" <<'PYEOF'
import sys
import yaml

src, org, ns, tlsca_cert, orderer_tlscert = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5]
with open(src) as f:
    doc = yaml.safe_load(f)
doc["metadata"].pop("creationTimestamp", None)

# NOTE 6: same root cause as NOTE 5 originally suspected, but for peers —
# `peersToJoin` (in-cluster name/namespace ref) resolves the join URL from
# FabricPeer.status.port, which is also 0 here (same missing-Istio reason;
# confirmed via `kubectl -n hlf get fabricpeers peer0-org1 -o
# jsonpath='{.status.port}'` -> 0). Use `externalPeersToJoin` instead: a
# direct grpcs:// URL at the peer's real in-cluster port 7051 (confirmed via
# `kubectl -n hlf get svc peer0-org1`: ports 443/7051/7052/7053/9443, main
# "peer" listener — which also serves the Admin/join service in Fabric
# 2.x — on 7051), with tlsCACert set to the org CA's tlsca root (verified
# this equals FabricPeer.status.tlsCaCert byte-for-byte via sha256
# fingerprint match). Unlike the orderer join (NOTE 5), this path is
# confirmed to still need externalPeersToJoin — it was NOT switched to the
# in-cluster peersToJoin ref, since it already works and the live cluster
# uses this exact form.
doc["spec"]["peersToJoin"] = []
doc["spec"]["externalPeersToJoin"] = [{
    "url": f"grpcs://peer0-{org}.{ns}:7051",
    "tlsCACert": tlsca_cert,
}]

# Safety refresh, same rationale as the mainchannel patch above: re-stamp
# the follower's copy of the orderer's TLS cert from live status.tlsCert on
# every run (no-op on a clean run; only matters if orderer0 was recreated).
for orderer in doc["spec"]["orderers"]:
    orderer["certificate"] = orderer_tlscert

yaml.safe_dump(doc, sys.stdout, default_flow_style=False, sort_keys=False)
PYEOF

gen_follower() {
  local org=$1 mspid=$2 secret=$3 ca=$4 out=$5
  local tlsca_cert raw
  tlsca_cert=$(kubectl -n "$NS" get fabriccas.hlf.kungfusoftware.es "$ca" \
    -o jsonpath='{.status.tlsca_cert}')
  raw=$(mktemp)
  kubectl hlf channelcrd follower create -o \
    --channel-name=mychannel \
    --mspid="$mspid" \
    --peers="peer0-$org.$NS" \
    --anchor-peers="peer0-$org.$NS:7051" \
    --orderer-urls="grpcs://orderer0.$NS:7050" \
    --orderer-certificates="$ORDERER_TLSCERT_FILE" \
    --secret-key=user.yaml \
    --secret-name="$secret" \
    --secret-ns="$NS" \
    --name="follower-mychannel-$org" >"$raw"
  python3 "$PATCH_FOLLOWER_PY" "$raw" "$org" "$NS" "$tlsca_cert" "$(cat "$ORDERER_TLSCERT_FILE")" >"$out"
  rm -f "$raw"
}

gen_follower org1 Org1MSP org1-admin org1-ca "$MANIFEST_DIR/follower-org1.yaml"
gen_follower org2 Org2MSP org2-admin org2-ca "$MANIFEST_DIR/follower-org2.yaml"
rm -f "$PATCH_FOLLOWER_PY"

rm -f "$ORDERER_TLSCERT_FILE"

# ─── Part D: apply + wait for readiness ──────────────────────────────────────
kubectl apply -f "$MANIFEST_DIR/mainchannel.yaml"

echo "Waiting for FabricMainChannel/mychannel to reach RUNNING..."
state=""
for _ in $(seq 1 40); do
  state=$(kubectl -n "$NS" get fabricmainchannels.hlf.kungfusoftware.es mychannel \
    -o jsonpath='{.status.status}' 2>/dev/null || true)
  [[ "$state" == "RUNNING" ]] && break
  sleep 5
done
echo "  mychannel: ${state:-<none>}"
if [[ "$state" != "RUNNING" ]]; then
  kubectl -n "$NS" describe fabricmainchannels.hlf.kungfusoftware.es mychannel >&2
  exit 1
fi

kubectl apply -f "$MANIFEST_DIR/follower-org1.yaml" -f "$MANIFEST_DIR/follower-org2.yaml"

echo "Waiting for FabricFollowerChannels to reach RUNNING..."
for name in follower-mychannel-org1 follower-mychannel-org2; do
  state=""
  for _ in $(seq 1 40); do
    state=$(kubectl -n "$NS" get fabricfollowerchannels.hlf.kungfusoftware.es "$name" \
      -o jsonpath='{.status.status}' 2>/dev/null || true)
    [[ "$state" == "RUNNING" ]] && break
    sleep 5
  done
  echo "  $name: ${state:-<none>}"
  if [[ "$state" != "RUNNING" ]]; then
    kubectl -n "$NS" describe fabricfollowerchannels.hlf.kungfusoftware.es "$name" >&2
    exit 1
  fi
done

echo "Channel mychannel created; peer0-org1 and peer0-org2 joined."
