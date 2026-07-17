# hlf-operator on kind — runbook

Rehearses moving the Fabric supply-chain network onto Kubernetes via
[hlf-operator](https://github.com/kfsoftware/hlf-operator), on a local
[kind](https://kind.sigs.k8s.io/) cluster before targeting a real cluster.

Plan: `docs/superpowers/plans/2026-07-17-k8s-operator-deployment.md`

Cluster name: `hlf` (kubectl context `kind-hlf`). Namespace: `hlf`.

Task order and commands, appended as each task lands:

## Task 1: Host prereqs, kind cluster, operator install (DONE)

Tools (host, one-time): `kubectl`, `kind`, `helm`, and the `kubectl-hlf` krew
plugin. Confirm with `kubectl hlf version`.

Create the cluster and namespace:

```bash
kind create cluster --config infra/k8s/operator/kind-config.yaml
kubectl cluster-info --context kind-hlf
kubectl apply -f infra/k8s/operator/00-namespace.yaml
```

Install the operator (chart version pinned to match the installed
`kubectl-hlf` plugin's minor line — see Makefile `kind-operator` target for
the exact `--version`):

```bash
helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts
helm repo update
helm install hlf-operator --version=1.14.0 kfs/hlf-operator -n hlf
```

Verify:

```bash
kubectl -n hlf rollout status deploy/hlf-operator-controller-manager --timeout=180s
kubectl get crds | grep hlf.kungfusoftware.es
```

Expected CRDs include `fabriccas`, `fabricpeers`, `fabricorderernodes`,
`fabricmainchannels`, `fabricfollowerchannels`, `fabricchaincodes`.

Makefile targets: `make kind-up`, `make kind-operator`, `make kind-down`.

## Task 2: Certificate Authorities (org1, org2, orderer) (DONE)

Creates `org1-ca`, `org2-ca`, `ord-ca` via the operator, enroll id/pw
`enroll`/`enrollpw`, Fabric CA version 1.5.5, storage class `standard`, 1Gi:

```bash
bash infra/k8s/operator/scripts/10-cas.sh
```

Verify (operator provisions asynchronously — wait/re-run until `Running`):

```bash
kubectl -n hlf get fabriccas.hlf.kungfusoftware.es
kubectl -n hlf get pods -l app=hlf-ca
kubectl -n hlf wait --for=condition=ready pod -l app=hlf-ca --timeout=300s
```

Expected: `org1-ca`, `org2-ca`, `ord-ca` each `RUNNING`/`Running` and `1/1`.
Each CA is reachable in-cluster via its Kubernetes Service, e.g.
`org1-ca.hlf.svc.cluster.local:7054` (short form `org1-ca.hlf`).

Note: the plugin's `--hosts` flag (Istio ingress `Gateway`/`VirtualService`)
is intentionally omitted from the script — this kind cluster has no Istio
CRDs installed, and using `--hosts` makes the CA's chart install fail
(`FabricCA` stuck in `FAILED` state) since the operator can't create those
Istio resource kinds. In-cluster reachability, which is all later
enroll/register tasks need, comes from the plain Kubernetes `Service` the
operator creates regardless of `--hosts`.

Also note: this plugin version has no `kubectl hlf ca list` subcommand; use
`kubectl -n hlf get fabriccas.hlf.kungfusoftware.es` instead.

Makefile target: `make kind-cas`.

## Task 3: Orderer node (pending)

## Task 4: Peers (peer0-org1, peer0-org2) with CouchDB (pending)

## Task 5: Channel `mychannel` + peer joins (pending)

## Task 6: Chaincode `basic` as ccaas (pending)

## Task 7: Connect `api` + `web` (pending)

## Task 8: Full-run verification, retire skeleton chart, docs (pending)
