#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf

# NOTE 1: the brief's --hosts=orderer0.localho.st flag (on `ordnode create`,
# below) provisions an Istio Gateway/VirtualService for external ingress
# (same as Task 2's CA --hosts flag). This kind cluster has no Istio CRDs
# installed, so --hosts would leave the FabricOrdererNode resource FAILED.
# The operator's plain Kubernetes Service already gives the in-cluster
# endpoint orderer0.hlf:7050 that later tasks (channel creation) need, so
# --hosts is dropped here rather than standing up Istio.

# NOTE 2: `kubectl hlf ca register`, run from the host, auto-discovers the
# CA's address via its NodePort + the kind node's external IP (e.g.
# 172.22.0.2:<nodePort>). But ord-ca's TLS serving certificate only carries
# SANs localhost/ord-ca/ord-ca.hlf/127.0.0.1 (Task 2 dropped --hosts, which
# only controls Istio provisioning, not the cert SAN list) — so a direct
# NodePort connection fails TLS hostname verification:
#   x509: certificate is valid for 127.0.0.1, not 172.22.0.2
# Route the register call through a port-forward to the CA's ClusterIP
# service instead, and point --ca-url at 127.0.0.1, which IS a valid SAN.
PF_LOCAL_PORT=17054
kubectl -n "$NS" port-forward svc/ord-ca "${PF_LOCAL_PORT}:7054" \
  >/tmp/ord-ca-portforward.log 2>&1 &
PF_PID=$!
trap 'kill "$PF_PID" 2>/dev/null || true' EXIT
sleep 2

REGISTER_LOG=$(mktemp)
if ! kubectl hlf ca register \
  --name=ord-ca --namespace="$NS" \
  --user=orderer --secret=ordererpw \
  --type=orderer --enroll-id=enroll --enroll-secret=enrollpw \
  --mspid=OrdererMSP \
  --ca-url="https://127.0.0.1:${PF_LOCAL_PORT}" \
  >"$REGISTER_LOG" 2>&1; then
  if grep -qi "already registered" "$REGISTER_LOG"; then
    echo "orderer identity already registered at ord-ca — continuing (safe to ignore)"
  else
    cat "$REGISTER_LOG" >&2
    exit 1
  fi
else
  cat "$REGISTER_LOG"
fi

kill "$PF_PID" 2>/dev/null || true
trap - EXIT

# NOTE 3: the same NodePort/SAN mismatch from NOTE 2 also bites
# `ordnode create`'s auto-discovered CA address — but this time it's the
# in-cluster operator pod doing the enrolling (for the orderer's TLS/signing
# certs), and it fails the same TLS verification against the CA's NodePort
# external IP. Force the in-cluster ClusterIP DNS name instead (which IS a
# valid SAN and is what the operator can actually resolve/reach) via
# --ca-host/--ca-port, rather than relying on --ca-name's auto-discovery.
kubectl hlf ordnode create \
  --image=hyperledger/fabric-orderer --version=2.5.15 \
  --storage-class=standard --capacity=1Gi \
  --enroll-id=orderer --enroll-pw=ordererpw \
  --mspid=OrdererMSP \
  --name=orderer0 --namespace="$NS" \
  --ca-name=ord-ca.$NS \
  --ca-host=ord-ca.$NS --ca-port=7054
