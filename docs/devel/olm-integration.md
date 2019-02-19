# KubeVirt's OLM and Operator Marketplace Integration

## Introduction

### Operator Lifecycle Manager (OLM)

https://github.com/operator-framework/operator-lifecycle-manager

OLM is the Operator Lifecycle Manager, which consists of 2 operators:

#### OLM Operator

Installs application operators based in information in ClusterServiceVersions

CRDs:

- ClusterServiceVersion (CSV):  
  contains application metadata: name, version, icon, required resources, installation, etc...
  provided by developer, together with CRD declarations and package description. The latter declares channels and their CSV version.
  installed by Catalog Operator

- OperatorGroup:  
  declares on which namespaces OLM should operate
  provided and installed by developer, or in UI

#### Catalog Operator

Prepares installations of operators by installing the application's CRDs and CSVs

CRDs:

- CatalogSource:  
  declares available packages
  provided and installed by Marketplace Operator based on CatalogSourceConfig

- Subscription:  
  declares which version of an operator to install (which channel from which source)
  provided and installed by developer, or in UI

- InstallPlan:  
  calculated list of resources to be created in order to automatically install/upgrade a CSV
  created and insalled by the Catalog Operator, needs manual or automatic approval

### Operator Marketplace

https://github.com/operator-framework/operator-marketplace

The Operator Marketplace has another operator

CRDs:

- OperatorSource:  
  declares where to find applications bundles (CSV + CRD + package)
  provided and installed by developer, and/or already installed pointing to official repositories (community operators)

- CatalogSourceConfig:  
  declares which packages to enable in the marketplace
  created and deployed by marketplace operator

## KubeVirt Manifests

Our OLM / Marketplace manifest templates live in /manifests/release/olm. As for all manifests, you need to run
`make generate && make manifests` for getting their final version in the `_out/` directory.

The bundle subdirectory contains:
  - the ClusterServiceVersion manifest
  - the CRD manifest
  - the Package manifest: this contains the available distribution channels and their corresponding CSV name
  These files are pushed to Quay (after they are processed with)

Then we have:
  - the OperatorSource manifest: this will be deployed to your cluster.
  - a Subscription manifest: only needed when not created using the OKD console.
  - a OperatorGroup manifest: can be created in the console, too??

Last but not least there is a preconditions manifest: if want to test the CSV manifest manually, without
OperatorSource and Subscription, you can deploy this manifest in order to satisfy all conditions, which are declared
in the CSV manifest, so that the OLM operator can deploy the KubeVirt operator.  

## Test a new version

Note 1: We use a k8s cluster >= v1.11 for this. You might want to use a OKD cluster with OLM and Marketplace already installed.
Note 2: You need a Quay.io account

- create manifests with your repository and version info, e.g.:
  
  TODO: actually use CSV_VERSION!!!
  
  `CSV_VERSION=<csv-version> DOCKER_PREFIX="docker.io/<docker_user>" DOCKER_TAG="<tag>" sh -c 'make generate && make manifests'`
- verify manifests:
  `make olm-verify`
- push images:
  `DOCKER_PREFIX="index.docker.io/<docker_user>" DOCKER_TAG="<tag>" make bazel-push-images`
- push the operator bundle:
  `CSV_VERSION=<csv-version> QUAY_USER=<username> QUAY_PASSWORD=<password> make olm-push`
  Note: you need to update the CSV version (and so run `make manifests`) on every push! (or maybe delete an old version before pushing again?)
  
- install OLM and Marketplace (see below)

- install KubeVirt OperatorSource:
  `cd _out/manifests/release/olm`
  `kubectl apply -f kubevirt-operatorsource.yaml`
- check that a CatalogSourceConfig and a CatalogSource were created in the marketplace namespace
- WORKAROUND: the OKD console only shows operators from CatalogSources in the `olm` namespace. In order to get it there,
  you need to edit the CatalogSourceConfig and change the targetNamespace from `marketplace` to `olm`. The new
  CatalogSource should be created automatically.
- create the kubevirt namespace:
  `kubectl create ns kubevirt`
- install the OperatorGroup for the new namespace:
  `kubectl apply -f operatorgroup.yaml`
- create a Subscription:
  `kubectl apply -f kubevirt-subsription.yaml`
- check that a InstallPlan was created
- check that the KubeVirt operator was installed
- install a KubeVirt CR

Bonus: install the OKD Console:

- we need cluster-admin permissions for the kube-system:default account:
  `kubectl create clusterrolebinding defaultadmin --clusterrole cluster-admin --serviceaccount kube-system:default`
- in the OLM repository, run `./scripts/run_console_local.sh`
- open `localhost:9000` in a browser

## Release a new version

Travis cares for this on every release.

## Installing OLM on Kubernetes

- clone github.com/operator-framework/operator-lifecycle-manager
- `cd deploy/upstream/quickstart`
- `kubectl apply -f olm.yaml`
- if you get an error, try again, CRDs might have been too slow

## Installing Marketplace on Kubernetes

- clone github.com/operator-framework/operator-marketplace
- `cd deploy/upstream/manifests`
- `kubectl apply -f upstream/`
- if you get an error about rolebinding, repeat with `--validate=false`

## Sources

CSV description: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md
Publish bundles: https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md
Install OLM: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md
Install and use Marketplace: https://github.com/operator-framework/operator-marketplace

## Important

- the Quay repo name needs to match the package name (https://github.com/operator-framework/operator-marketplace/issues/122#issuecomment-470820491)