# Hyperconverged Cluster Operator

The goal of the hyperconverged-cluster-operator (HCO) is to provide a single
entrypoint for multiple operators - kubevirt, cdi, networking, ect... - where
users can deploy and configure them in a single object. This operator is
sometimes referred to as a "meta operator" or an "operator for operators".
Most importantly, this operator doesn't replace or interfere with OLM.
It only creates operator CRs, which is the user's prerogative.

## Install OLM
**NOTE**
OLM is not a requirement to test.  Once we publish operators through
Marketplace|operatorhub.io, it will be.

https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md#installing-olm

## Using the HCO

**NOTE**
Until we publish (and consume) the HCO and component operators through
Marketplace|operatorhub.io, this is a means to demonstrate the HCO workflow
without OLM.

Create the namespace for the HCO.
```bash
kubectl create ns kubevirt-hyperconverged
```

Switch to the HCO namespace.
```bash
kubectl config set-context $(kubectl config current-context) --namespace=kubevirt-hyperconverged
```

Launch all of the CRDs.
```bash
kubectl create -f deploy/converged/crds/hco.crd.yaml
kubectl create -f deploy/converged/crds/kubevirt.crd.yaml
kubectl create -f deploy/converged/crds/cdi.crd.yaml
kubectl create -f deploy/converged/crds/cna.crd.yaml
kubectl create -f deploy/converged/crds/ssp.crd.yaml
kubectl create -f deploy/converged/crds/kwebui.crd.yaml
kubectl create -f deploy/converged/crds/nodemaintenance.crd.yaml
```

Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
```bash
kubectl create -f deploy/converged
```

Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
```bash
kubectl create -f deploy/converged/crds/hco.cr.yaml
```

## Launching the HCO through OLM

**NOTE**
Until we publish (and consume) the HCO and component operators through
Marketplace|operatorhub.io, this is a means to demonstrate the HCO workflow
without OLM. Replace `<docker_org>` with your Docker organization
as official operator-registry images for HCO will not be provided.

Build and push the converged HCO operator-registry image.

```bash
cd deploy/converged
export HCO_DOCKER_ORG=<docker_org>
docker build --no-cache -t docker.io/$HCO_DOCKER_ORG/hco-registry:example -f Dockerfile .
docker push docker.io/$HCO_DOCKER_ORG/hco-registry:example
```

Create the namespace for the HCO.
```bash
kubectl create ns kubevirt-hyperconverged
```

Create an OperatorGroup.
```bash
cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: hco-operatorgroup
  namespace: kubevirt-hyperconverged
EOF
```

Create a Catalog Source.
```bash
cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: hco-catalogsource
  namespace: openshift-operator-lifecycle-manager
spec:
  sourceType: grpc
  image: docker.io/$HCO_DOCKER_ORG/hco-registry:example
  displayName: KubeVirt HyperConverged
  publisher: Red Hat
EOF
```

Create a subscription.
```bash
cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-subscription
  namespace: kubevirt-hyperconverged
spec:
  channel: alpha
  name: kubevirt-hyperconverged
  source: hco-catalogsource
  sourceNamespace: openshift-operator-lifecycle-manager
EOF
```

Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
```bash
kubectl create -f deploy/converged/crds/hco.cr.yaml
```

## Launching the HCO on a local cluster

Lunch the HCO locally for testing, experimenting and developing.

**NOTE:** no need to install any type of kubenetes cluster as a prerequisite.

1. Navigate to the project's directory
```bash
$ cd <path>/hyperconverged-cluster-opertor
```
2. Remove an old cluster
```bash
$ make cluster-down
```
3. Create a new cluster
```bash
$ make cluster-up
```
4. Clean previous HCO deployment and re-deploy HCO \
   (When making a change, execute only this command - no need to repeat steps 1-3)
```bash
$ make cluster-sync
```
### Command-Line Tool
Use `./cluster/kubectl.sh` as the command-line tool. for example:
```bash
$ ./cluster/kubectl.sh get pods --all-namespaces
```
