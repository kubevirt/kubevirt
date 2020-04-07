#!/usr/bin/env bash
set -ex

# build-manifests is designed to populate the deploy directory
# with all of the manifests necessary for use in development
# and for consumption with the operator-lifecycle-manager.
# 
# First, we create a temporary directory and filling it with
# all of the component operator's ClusterServiceVersion (CSV for OLM)
# and CustomResourceDefinitions (CRDs); being sure to copy the CRDs
# into the deploy/crds directory.
# 
# The CSV manifests contain all of the information we need to 1) generate
# a combined CSV and 2) other development related manifests (like the
# operator deployment + rbac).
#
# Second, we pass all of the component CSVs off to the manifest-templator
# that handles the deployment specs, service account names, permissions, and
# clusterPermissions by converting them into their corresponding Kubernetes
# manifests (ie. permissions + serviceAccountName = role + service account
# + role binding) before writing them to disk.
#
# Lastly, we take give the component CSVs to the csv-merger that combines all
# of the manifests into a single, unified, ClusterServiceVersion.
PROJECT_ROOT="$(readlink -e $(dirname "$BASH_SOURCE[0]")/../)"
source "${PROJECT_ROOT}"/hack/config

# REPLACES_VERSION is the old CSV_VERSION
#   if REPLACES_VERSION == CSV_VERSION it will be ignored
REPLACES_CSV_VERSION="${REPLACES_VERSION:-1.0.0}"
CSV_VERSION="${CSV_VERSION:-1.1.0}"

DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CRD_DIR="${DEPLOY_DIR}/crds"
CSV_DIR="${DEPLOY_DIR}/olm-catalog/kubevirt-hyperconverged/${CSV_VERSION}"

OPERATOR_NAME="${NAME:-kubevirt-hyperconverged-operator}"
OPERATOR_NAMESPACE="${NAMESPACE:-kubevirt-hyperconverged}"
OPERATOR_IMAGE="${OPERATOR_IMAGE:-quay.io/kubevirt/hyperconverged-cluster-operator:1.1.0}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

# Component Images
KUBEVIRT_IMAGE="${KUBEVIRT_IMAGE:-docker.io/kubevirt/virt-operator:${KUBEVIRT_VERSION}}"
CNA_IMAGE="${CNA_IMAGE:-quay.io/kubevirt/cluster-network-addons-operator:${NETWORK_ADDONS_VERSION}}"
SSP_IMAGE="${SSP_IMAGE:-quay.io/fromani/kubevirt-ssp-operator-container:${SSP_VERSION}}"
CDI_IMAGE="${CDI_IMAGE:-docker.io/kubevirt/cdi-operator:${CDI_VERSION}}"
NMO_IMAGE="${NMO_IMAGE:-quay.io/kubevirt/node-maintenance-operator:${NMO_VERSION}}"
HPPO_IMAGE="${HPP_IMAGE:-quay.io/kubevirt/hostpath-provisioner-operator:${HPPO_VERSION}}"
HPP_IMAGE="${HPP_IMAGE:-quay.io/kubevirt/hostpath-provisioner:${HPP_VERSION}}"
CONVERSION_CONTAINER="${CONVERSION_CONTAINER:-quay.io/kubevirt/kubevirt-v2v-conversion:${CONVERSION_CONTAINER_VERSION}}"
VMWARE_CONTAINER="${VMWARE_CONTAINER:-quay.io/kubevirt/kubevirt-vmware:${VMWARE_CONTAINER_VERSION}}"

# Important extensions
CSV_EXT="clusterserviceversion.yaml"
CSV_CRD_EXT="csv_crds.yaml"
CRD_EXT="crd.yaml"

function gen_csv() {
  # Handle arguments
  local operatorName="$1" && shift
  local imagePullUrl="$1" && shift
  local dumpCRDsArg="$1" && shift
  local operatorArgs="$@"

  # Handle important vars
  local csv="${operatorName}.${CSV_EXT}"
  local csvWithCRDs="${operatorName}.${CSV_CRD_EXT}"
  local crds="${operatorName}.crds.yaml"

  # TODO: Use oc to run if cluster is available
  local dockerArgs="docker run --rm --entrypoint=/usr/bin/csv-generator ${imagePullUrl} ${operatorArgs}"

  eval $dockerArgs > $csv
  eval $dockerArgs $dumpCRDsArg > $csvWithCRDs

  diff -u $csv $csvWithCRDs | grep -E "^\+" | sed -E 's/^\+//' | tail -n+2 > $crds

  csplit --digits=2 --quiet --elide-empty-files \
    --prefix="${operatorName}" \
    --suffix-format="%02d.${CRD_EXT}" \
    $crds \
    "/---/" "{*}"
}

function create_virt_csv() {
  local operatorName="kubevirt"
  local imagePullUrl="${KUBEVIRT_IMAGE}"
  local dumpCRDsArg="--dumpCRDs"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csvVersion=${CSV_VERSION} \
    --operatorImageVersion=${KUBEVIRT_IMAGE/*:/} \
    --dockerPrefix=${KUBEVIRT_IMAGE%\/*} \
  "

  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_cna_csv() {
  local operatorName="cluster-network-addons"
  local imagePullUrl="${CNA_IMAGE}"
  local dumpCRDsArg="--dump-crds"
  local containerPrefix="${CNA_IMAGE%/*}"
  local tag="${CNA_IMAGE/*:/}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --version=${CSV_VERSION} \
    --version-replaces=${REPLACES_VERSION} \
    --image-pull-policy=IfNotPresent \
    --operator-version=${tag} \
    --container-tag=${tag} \
    --container-prefix=${containerPrefix} \
  "

  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_ssp_csv() {
  local operatorName="scheduling-scale-performance"
  local imagePullUrl="${SSP_IMAGE}"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image=${SSP_IMAGE} \
  "

  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_cdi_csv() {
  local operatorName="containerized-data-importer"
  local imagePullUrl="${CDI_IMAGE}"
  local containerPrefix="${CDI_IMAGE%/*}"
  local tag="${CDI_IMAGE/*:/}"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --pull-policy=IfNotPresent \
    --operator-image=${CDI_IMAGE} \
    --controller-image=${containerPrefix}/cdi-controller:${tag} \
    --apiserver-image=${containerPrefix}/cdi-apiserver:${tag} \
    --cloner-image=${containerPrefix}/cdi-cloner:${tag} \
    --importer-image=${containerPrefix}/cdi-importer:${tag} \
    --uploadproxy-image=${containerPrefix}/cdi-uploadproxy:${tag} \
    --uploadserver-image=${containerPrefix}/cdi-uploadserver:${tag} \
  "
  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_nmo_csv() {
  local operatorName="node-maintenance"
  local imagePullUrl="${NMO_IMAGE}"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image=${NMO_IMAGE} \
  "

  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_hpp_csv() {
  local operatorName="hostpath-provisioner"
  local imagePullUrl="${HPPO_IMAGE}"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${HPPO_IMAGE} \
    --provisioner-image-name=${HPP_IMAGE} \
    --namespace=${OPERATOR_NAMESPACE} \
    --pull-policy=IfNotPresent \
  "

  gen_csv ${operatorName} ${imagePullUrl} ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

TEMPDIR=$(mktemp -d) || (echo "Failed to create temp directory" && exit 1)
pushd $TEMPDIR
virtCsv="${TEMPDIR}/$(create_virt_csv).${CSV_EXT}"
cnaCsv="${TEMPDIR}/$(create_cna_csv).${CSV_EXT}"
sspCsv="${TEMPDIR}/$(create_ssp_csv).${CSV_EXT}"
cdiCsv="${TEMPDIR}/$(create_cdi_csv).${CSV_EXT}"
nmoCsv="${TEMPDIR}/$(create_nmo_csv).${CSV_EXT}"
hppCsv="${TEMPDIR}/$(create_hpp_csv).${CSV_EXT}"
csvOverrides="${TEMPDIR}/csv_overrides.${CSV_EXT}"
cat > ${csvOverrides} <<- EOM
---
spec:
  links:
  - name: KubeVirt project
    url: https://kubevirt.io
  - name: Source Code
    url: https://github.com/kubevirt/hyperconverged-cluster-operator
  maintainers:
  - email: kubevirt-dev@googlegroups.com
    name: KubeVirt project
  maturity: alpha
  provider:
    name: KubeVirt project
EOM

# Write HCO CRDs
(cd ${PROJECT_ROOT}/tools/csv-merger/ && go build)
hco_crds=${TEMPDIR}/hco.crds.yaml
${PROJECT_ROOT}/tools/csv-merger/csv-merger --output-mode=CRDs > $hco_crds
csplit --digits=2 --quiet --elide-empty-files \
  --prefix=hco \
  --suffix-format="%02d.${CRD_EXT}" \
  $hco_crds \
  "/---/" "{*}"

popd

mkdir -p "${CSV_DIR}"
rm -f ${CSV_DIR}/*

SMBIOS=$(cat <<- EOM
Family: KubeVirt
Manufacturer: KubeVirt
Product: None
EOM
)

# Build and write deploy dir
(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go build)
${PROJECT_ROOT}/tools/manifest-templator/manifest-templator \
  --cna-csv="$(<${cnaCsv})" \
  --virt-csv="$(<${virtCsv})" \
  --ssp-csv="$(<${sspCsv})" \
  --cdi-csv="$(<${cdiCsv})" \
  --nmo-csv="$(<${nmoCsv})" \
  --ims-conversion-image-name="${CONVERSION_CONTAINER}" \
  --ims-vmware-image-name="${VMWARE_CONTAINER}" \
  --operator-namespace="${OPERATOR_NAMESPACE}" \
  --smbios="${SMBIOS}" \
  --operator-image="${OPERATOR_IMAGE}"
(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go clean)

# Build and merge CSVs
${PROJECT_ROOT}/tools/csv-merger/csv-merger \
  --cna-csv="$(<${cnaCsv})" \
  --virt-csv="$(<${virtCsv})" \
  --ssp-csv="$(<${sspCsv})" \
  --cdi-csv="$(<${cdiCsv})" \
  --nmo-csv="$(<${nmoCsv})" \
  --ims-conversion-image-name="${CONVERSION_CONTAINER}" \
  --ims-vmware-image-name="${VMWARE_CONTAINER}" \
  --csv-version=${CSV_VERSION} \
  --replaces-csv-version=${REPLACES_CSV_VERSION} \
  --spec-displayname="KubeVirt HyperConverged Cluster Operator" \
  --spec-description="$(<${PROJECT_ROOT}/docs/operator_description.md)" \
  --crd-display="HyperConverged Cluster Operator" \
  --smbios="${SMBIOS}" \
  --csv-overrides="$(<${csvOverrides})" \
  --operator-image-name="${OPERATOR_IMAGE}" > "${CSV_DIR}/${OPERATOR_NAME}.v${CSV_VERSION}.${CSV_EXT}"
(cd ${PROJECT_ROOT}/tools/csv-merger/ && go clean)

# Copy all CRDs into the CRD and CSV directories
rm -f ${CRD_DIR}/*
cp -f ${TEMPDIR}/*.${CRD_EXT} ${CRD_DIR}
cp -f ${TEMPDIR}/*.${CRD_EXT} ${CSV_DIR}


# Intentionally removing last so failure leaves around the templates
rm -rf ${TEMPDIR}
