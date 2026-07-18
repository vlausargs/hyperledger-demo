#!/usr/bin/env bash
set -euo pipefail
# Ensure the kubectl-hlf krew plugin + host-installed tools resolve even when
# invoked from a non-interactive shell (e.g. `make kind-cas`), which does not
# source ~/.bashrc where ~/.krew/bin is added.
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"
NS=hlf
# NOTE: the brief's --hosts flag provisions Istio Gateway/VirtualService
# objects for external ingress. This kind cluster has no Istio CRDs
# installed, so --hosts makes the chart install fail (CA stuck FAILED).
# In-cluster reachability (org1-ca.hlf etc., which is all later enrollment
# tasks need) comes from the operator's plain Kubernetes Service regardless
# of --hosts, so the flag is dropped here rather than standing up Istio.
for org in org1 org2 ord; do
  # tolerate re-runs: skip create if the CA already exists
  if kubectl -n "$NS" get fabriccas.hlf.kungfusoftware.es "${org}-ca" >/dev/null 2>&1; then
    echo "${org}-ca already exists — skipping create"
    continue
  fi
  kubectl hlf ca create \
    --storage-class=standard \
    --capacity=1Gi \
    --name="${org}-ca" \
    --namespace="$NS" \
    --enroll-id=enroll \
    --enroll-pw=enrollpw \
    --version=1.5.5
done

# Block until every CA reaches RUNNING. The operator provisions CAs
# asynchronously; without this wait, a fast `make kind-all` proceeds to the
# orderer/peer steps while a CA is still PENDING, which fails their
# register/enroll ("ca <name> is in PENDING status").
echo "==> waiting for CAs to reach RUNNING"
for org in org1 org2 ord; do
  st=""
  for _ in $(seq 1 60); do
    st=$(kubectl -n "$NS" get fabriccas.hlf.kungfusoftware.es "${org}-ca" \
      -o jsonpath='{.status.status}' 2>/dev/null || true)
    [ "$st" = "RUNNING" ] && break
    sleep 5
  done
  [ "$st" = "RUNNING" ] || { echo "ERROR: ${org}-ca not RUNNING (last status: ${st:-<none>})" >&2; exit 1; }
  echo "  ${org}-ca RUNNING"
done
