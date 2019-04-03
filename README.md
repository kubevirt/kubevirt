# Hyperconverged Cluster Operator

The goal of the hyperconverged-cluster-operator (HCO) is to provide a single
entrypoint for multiple operators - kubevirt, cdi, networking, ect... - where
users can deploy and configure them in a single object. This operator is
sometimes referred to as a "meta operator" or an "operator for operators".
Most importantly, this operator doesn't replace or interfere with OLM.
It only creates operator CRs, which is the user's prerogative.

TODO:
- [x] Golang code to generate deployment manifests
- [x] Unifed CSV file that lauches all operators through OLM
- [ ] Use operator-courier to publish HCO CSV
- [ ] Manifest that launches HCO, kubevirt, CDI, network, and UI operators for
  initial non OLM deployments

## Using the HCO

Create the namespace for the HCO.
```bash
oc create ns kubevirt-hyperconverged
```

Switch to the HCO namespace.
```bash
oc project kubevirt-hyperconverged
```

Launch all of the CRDs.
```bash
oc create -f deploy/converged/crds/hco.crd.yaml
oc create -f deploy/converged/crds/kubevirt.crd.yaml
oc create -f deploy/converged/crds/cdi.crd.yaml
oc create -f deploy/converged/crds/cna.crd.yaml
```

Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
```bash
oc create -f deploy/converged
```

Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
```bash
oc create -f deploy/converged/crds/hco.cr.yaml
```

## Launching the HCO through OLM

**NOTE**
Until we determine how to publish (and consume) the HCO through
operator-courier|Marketplace|operatorhub.io, this is a means to demonstrate the
HCO workflow through OLM. Replace `<docker_org>` with your Docker organization
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
oc create ns kubevirt-hyperconverged
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
oc create -f deploy/converged/crds/hco.cr.yaml
```
