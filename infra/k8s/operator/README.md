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

## Task 2: Certificate Authorities (org1, org2, orderer) (pending)

## Task 3: Orderer node (pending)

## Task 4: Peers (peer0-org1, peer0-org2) with CouchDB (pending)

## Task 5: Channel `mychannel` + peer joins (pending)

## Task 6: Chaincode `basic` as ccaas (pending)

## Task 7: Connect `api` + `web` (pending)

## Task 8: Full-run verification, retire skeleton chart, docs (pending)
