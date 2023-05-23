# KubeVirt's OLM and Operator Marketplace Integration

## Introduction

### Operator Lifecycle Manager (OLM)

https://github.com/operator-framework/operator-lifecycle-manager

OLM is the Operator Lifecycle Manager, which consists of 2 operators:

#### OLM Operator

Installs application operators based on the information in ClusterServiceVersions.

CRDs:

- ClusterServiceVersion (CSV):  
  contains application metadata: name, version, icon, required resources, installation, etc...
  provided by developer, together with CRD declarations and package description. The latter declares channels and their CSV version.
  installed by Catalog Operator

- OperatorGroup:  
  declares which namespaces OLM should operate on
  provided and installed by developer, or in UI

#### Catalog Operator

Prepares installations of operators by installing the application's CRDs and CSVs.

CRDs:

- CatalogSource:  
  declares available packages
  provided and installed by Marketplace Operator based on CatalogSourceConfig

- Subscription:  
  declares which version of the operator to install (which channel from which source)
  provided and installed by developer, or in UI

- InstallPlan:  
  calculated list of resources to be created in order to automatically install/upgrade a CSV
  created and installed by the Catalog Operator, needs manual or automatic approval

### Operator Marketplace

https://github.com/operator-framework/operator-marketplace

The Operator Marketplace has another operator, the Marketplace Operator.

CRDs:

- OperatorSource:  
  declares where to find applications bundles (CSV + CRD + package)
  provided and installed by developer, and/or already installed pointing to official repositories (community operators)

- CatalogSourceConfig:  
  declares which packages to enable in the marketplace
  created and deployed by marketplace operator

## KubeVirt Manifests

KubeVirt's OLM / Marketplace manifest templates live in `/manifests/release/olm`. As for all manifests, you need to run
`make generate && make manifests` for getting their final version in the `_out/` directory.

The bundle subdirectory contains 3 files, which are uploaded as a bundle to Quay.io:
  - the ClusterServiceVersion manifest
  - the CRD manifest
  - the Package manifest: this contains the available distribution channels and their corresponding CSV name

Then we have:
  - the OperatorSource manifest: this will be deployed to your cluster.
  - a Subscription manifest: only needed when not created using the OKD console.
  - a OperatorGroup manifest: only needed when not created using the OKD console.

Last but not least there is a preconditions manifest: if there is a need to test the CSV manifest manually, without
OperatorSource and Subscription, you can deploy this manifest in order to satisfy all conditions, which are declared
in the CSV manifest, so that the OLM operator can deploy the KubeVirt operator.  

## Test a new version

>**Note:** This example uses a k8s cluster >= v1.11, with OLM and Marketplace manually being installed.
You might want to use a preconfigured OKD cluster. Namespaces might vary in that case.

>**Note:** You need a Quay.io account

- create manifests with your repository and version info, e.g.:  
  `CSV_VERSION=<csv-version> DOCKER_PREFIX="docker.io/<docker_user>" DOCKER_TAG="<tag>" sh -c 'make generate && make manifests'`
- verify manifests:  
  `make olm-verify`
- push images:  
  `DOCKER_PREFIX="index.docker.io/<docker_user>" DOCKER_TAG="<tag>" make bazel-push-images`
- push the operator bundle:  
  `CSV_VERSION=<csv-version> QUAY_USER=<username> QUAY_PASSWORD=<password> QUAY_REPOSITORY=<repository> make olm-push`
>**Note:** we need Quay user and repository, because for robot accounts it's not the same  
>**Note:** you need to update the CSV version (and also run `make manifests`) on every push!
  
- install OLM and Marketplace (see below)

- install KubeVirt OperatorSource:  
  `cd _out/manifests/release/olm`
  `kubectl apply -f kubevirt-operatorsource.yaml`
- check that a CatalogSourceConfig and a CatalogSource were created in the `marketplace` namespace

- now a workaround is needed, at least when using a k8s cluster with OLM and Marketplace manually installed, in order
  to see the KubeVirt operator in the OperatorHub later on and make the following steps work:  
  The UI only shows CatalogSources from the `openshift-marketplace` or the `olm` namespace. Also, the catalog operator only works
  when Subscriptions point to CatalogSources in the `olm` namespace. But now the kubevirt CatalogSource is in the `marketplace` namespace.  
  For moving the it into the `olm` namespace, edit its CatalogSourceConfig and change the targetNamespace from
  `marketplace` to `olm`. A few seconds later the CatalogSource should be recreated in the `olm` namespace.  
  Heads up, this is a temporary workaround only, the marketplace operator will overwrite this change on the next sync!

- create the `kubevirt` namespace:  
  `kubectl create ns kubevirt`
- install the OperatorGroup for the `kubevirt` namespace:  
  `kubectl apply -f operatorgroup.yaml`
- create a Subscription:  
  `kubectl apply -f kubevirt-subsription.yaml`
- check that a InstallPlan was created
- check that the KubeVirt operator was installed
- install a KubeVirt CR:  
  `kubectl apply -f ../kubevirt-cr.yaml`
- check that KubeVirt components are running

Bonus: install the OKD Console:

- we need cluster-admin permissions for the kube-system:default account:
  `kubectl create clusterrolebinding defaultadmin --clusterrole cluster-admin --serviceaccount kube-system:default`
- in the OLM repository, run `./scripts/run_console_local.sh`
- open `localhost:9000` in a browser

## Release a new version

Travis cares for this on every release.

## Installing OLM on Kubernetes

- clone the [OLM repository](https://github.com/operator-framework/operator-lifecycle-manager)
- `cd deploy/upstream/quickstart`
- `kubectl apply -f olm.yaml`
>**Note:** if you get an error, try again, CRD creation might have been too slow
- check that the olm and catalog operators are running in the `olm` namespace

## Installing Marketplace on Kubernetes

- clone the [Marketplace repository](https://github.com/operator-framework/operator-marketplace)
- `cd deploy`
- `kubectl apply -f upstream/ --validate=false`
- check that the marketplace operator is running in the `marketplace` namespace
## Sources

[CSV description](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md)  
[CSV required fields](https://github.com/operator-framework/community-operators/blob/master/docs/packaging-required-fields.md)  
[Publish bundles](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md)  
[Install OLM](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md)  
[Install and use Marketplace](https://github.com/operator-framework/operator-marketplace)  

## Important

- the Quay repo name needs to match the package name (https://github.com/operator-framework/operator-marketplace/issues/122#issuecomment-470820491)
