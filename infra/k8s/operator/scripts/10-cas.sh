#!/usr/bin/env bash
set -euo pipefail
NS=hlf
# NOTE: the brief's --hosts flag provisions Istio Gateway/VirtualService
# objects for external ingress. This kind cluster has no Istio CRDs
# installed, so --hosts makes the chart install fail (CA stuck FAILED).
# In-cluster reachability (org1-ca.hlf etc., which is all later enrollment
# tasks need) comes from the operator's plain Kubernetes Service regardless
# of --hosts, so the flag is dropped here rather than standing up Istio.
for org in org1 org2 ord; do
  kubectl hlf ca create \
    --storage-class=standard \
    --capacity=1Gi \
    --name="${org}-ca" \
    --namespace="$NS" \
    --enroll-id=enroll \
    --enroll-pw=enrollpw \
    --version=1.5.5
done
