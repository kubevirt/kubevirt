# Hyperconverged Cluster Operator

The goal of the hyperconverged-cluster-operator (HCO) is to provide a single
entrypoint for multiple operators - kubevirt, cdi, networking, ect... - where
users can deploy and configure them in a single object. This operator is
sometimes referred to as a "meta operator" or an "operator for operators".
Most importantly, this operator doesn't replace or interfere with OLM.
It only creates operator CRs, which is the user's prerogative.

## Using the HCO
TODO:
  - Golang code to generate deployment manifests
  - Manifest that launches HCO, kubevirt, CDI, network, and UI operators for
    initial non OLM deployments
  - Unifed CSV file that lauches all operators through OLM

Create component operator namespaces.
```bash
oc create ns kubevirt
oc create ns cdi
```

Switch to the kubevirt namespace.
```bash
oc project kubevirt
```

Launch the HCO.
```bash
oc create -f deploy/crds/hco_v1alpha1_hyperconverged_crd.yaml
oc create -f deploy/
```

Launch the KubeVirt operator.
```bash
oc create -f https://github.com/kubevirt/kubevirt/releases/download/v0.15.0/kubevirt-operator.yaml
```

Launch the CDI operator.
```bash
oc create -f https://github.com/kubevirt/containerized-data-importer/releases/download/v1.6.0/cdi-operator.yaml
```

Launch the Cluster Network Addons operator.
```bash
oc create -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/v0.1.0/cluster-network-addons-operator_00_namespace.yaml
oc create -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/v0.1.0/cluster-network-addons-operator_01_crd.yaml
oc create -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/v0.1.0/cluster-network-addons-operator_02_rbac.yaml
oc create -f https://github.com/kubevirt/cluster-network-addons-operator/releases/download/v0.1.0/cluster-network-addons-operator_03_deployment.yaml
```

Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
```bash
oc create -f deploy/crds/hco_v1alpha1_hyperconverged_cr.yaml
```
