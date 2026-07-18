#!/usr/bin/env bash
set -euo pipefail
export PATH="$HOME/.krew/bin:$HOME/.local/bin:$PATH"

# Task 5a: install Istio (istiod + istio-ingressgateway + the
# networking.istio.io CRDs Gateway/VirtualService) and wire in-cluster
# CoreDNS so the hlf-operator's *.localho.st Istio ingress hosts resolve.
# This does NOT enable sidecar injection on the `hlf` namespace — the
# operator manages its own Gateway/VirtualService objects directly; all we
# need cluster-side is the ingress gateway Service + CRDs.

ISTIO_VERSION=1.23.2
ISTIOCTL_BIN="$HOME/.local/bin/istioctl"

# --- 1. istioctl (idempotent: skip download if already present) ---
if command -v istioctl >/dev/null 2>&1; then
  echo "istioctl already on PATH: $(command -v istioctl) ($(istioctl version --remote=false))"
else
  echo "Downloading istioctl ${ISTIO_VERSION}..."
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  curl -sL -o "$tmpdir/istio.tar.gz" \
    "https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/istio-${ISTIO_VERSION}-linux-amd64.tar.gz"
  tar xzf "$tmpdir/istio.tar.gz" -C "$tmpdir"
  mkdir -p "$(dirname "$ISTIOCTL_BIN")"
  cp "$tmpdir/istio-${ISTIO_VERSION}/bin/istioctl" "$ISTIOCTL_BIN"
  chmod +x "$ISTIOCTL_BIN"
fi
istioctl version --remote=false

# --- 2. Install Istio into the cluster (default profile: istiod + ingressgateway) ---
# Re-running `istioctl install` is idempotent (reconciles existing resources).
istioctl install --set profile=default -y

kubectl -n istio-system rollout status deploy/istiod --timeout=180s
kubectl -n istio-system rollout status deploy/istio-ingressgateway --timeout=180s

# --- 3. CoreDNS rewrite for *.localho.st -> istio-ingressgateway ---
# Approach: read the current `coredns` ConfigMap's Corefile out of the
# cluster (not hardcoded here, so we don't clobber unrelated Corefile
# changes), insert the rewrite rule as its own line inside the `.:53`
# server block (before the `kubernetes` plugin so the rewrite applies
# ahead of cluster-service resolution), and re-apply only if the line is
# missing (idempotent).
REWRITE_LINE='rewrite name regex (.*)\.localho\.st istio-ingressgateway.istio-system.svc.cluster.local'

current_corefile="$(kubectl -n kube-system get configmap coredns -o jsonpath='{.data.Corefile}')"

if grep -qF "$REWRITE_LINE" <<<"$current_corefile"; then
  echo "CoreDNS rewrite already present, skipping patch."
else
  echo "Patching CoreDNS ConfigMap with *.localho.st rewrite..."
  new_corefile="$(awk -v line="    ${REWRITE_LINE}" '
    { print }
    /^[[:space:]]*ready[[:space:]]*$/ && !done { print line; done=1 }
  ' <<<"$current_corefile")"

  patch_file="$(mktemp)"
  trap 'rm -f "$patch_file"' EXIT
  cat > "$patch_file" <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
$(sed 's/^/    /' <<<"$new_corefile")
EOF
  kubectl apply -f "$patch_file"
  kubectl -n kube-system rollout restart deploy/coredns
  kubectl -n kube-system rollout status deploy/coredns --timeout=120s
fi

echo "Done. Verify with:"
echo "  kubectl -n istio-system get pods"
echo "  kubectl -n kube-system get configmap coredns -o jsonpath='{.data.Corefile}'"
echo "  kubectl run dnstest --rm -i --restart=Never --image=busybox:1.36 -- nslookup orderer0.localho.st"
