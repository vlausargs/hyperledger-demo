#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf

# See infra/k8s/operator/scripts/20-orderer.sh for the full rationale behind
# the two workarounds below (same cluster, same CA cert limitations):
#
# NOTE 1: the brief's --hosts=peer0-$org.localho.st flag (on `peer create`)
# provisions an Istio Gateway/VirtualService for external ingress. This kind
# cluster has no Istio CRDs installed, so --hosts would leave the FabricPeer
# resource FAILED. The operator's plain Kubernetes Service already gives the
# in-cluster endpoint peer0-$org.hlf:7051 that later tasks (channel/chaincode)
# need, so --hosts is dropped here rather than standing up Istio.
#
# NOTE 2: `kubectl hlf ca register`, run from the host, auto-discovers the
# CA's address via its NodePort + the kind node's external IP. But each CA's
# TLS serving certificate only carries SANs localhost/<ca>/<ca>.hlf/127.0.0.1,
# so a direct NodePort connection fails TLS hostname verification. Route the
# register call through a port-forward to the CA's ClusterIP service instead,
# and point --ca-url at 127.0.0.1, which IS a valid SAN. Distinct local ports
# per org so org1/org2 port-forwards don't collide.
#
# NOTE 3: the same NodePort/SAN mismatch also bites `peer create`'s
# auto-discovered CA address — the in-cluster operator pod does the enrolling
# (for the peer's TLS/signing certs) and fails the same TLS verification
# against the CA's NodePort external IP. Force the in-cluster ClusterIP DNS
# name instead via --ca-host/--ca-port, rather than relying on --ca-name's
# auto-discovery.

create_peer () {
  local org=$1 mspid=$2 ca=$3 pf_port=$4

  kubectl -n "$NS" port-forward "svc/${ca}" "${pf_port}:7054" \
    >"/tmp/${ca}-portforward.log" 2>&1 &
  local pf_pid=$!
  trap 'kill "'"$pf_pid"'" 2>/dev/null || true' EXIT
  sleep 2

  local register_log
  register_log=$(mktemp)
  if ! kubectl hlf ca register \
    --name="$ca" --namespace="$NS" \
    --user=peer --secret=peerpw \
    --type=peer --enroll-id=enroll --enroll-secret=enrollpw \
    --mspid="$mspid" \
    --ca-url="https://127.0.0.1:${pf_port}" \
    >"$register_log" 2>&1; then
    if grep -qi "already registered" "$register_log"; then
      echo "peer identity already registered at $ca — continuing (safe to ignore)"
    else
      cat "$register_log" >&2
      exit 1
    fi
  else
    cat "$register_log"
  fi

  kill "$pf_pid" 2>/dev/null || true
  trap - EXIT

  kubectl hlf peer create \
    --image=hyperledger/fabric-peer --version=2.5.15 \
    --storage-class=standard --capacity=2Gi \
    --enroll-id=peer --enroll-pw=peerpw --mspid="$mspid" \
    --name="peer0-$org" --namespace="$NS" \
    --ca-name="$ca.$NS" \
    --ca-host="$ca.$NS" --ca-port=7054 \
    --statedb=couchdb
}

create_peer org1 Org1MSP org1-ca 17055
create_peer org2 Org2MSP org2-ca 17056
