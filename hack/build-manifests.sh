#!/usr/bin/env bash
set -ex -o pipefail -o errtrace -o functrace

function catch() {
    echo "error $1 on line $2"
    exit 255
}

trap 'catch $? $LINENO' ERR TERM INT

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

function get_image_digest() {
  if [[ ! -f ${PROJECT_ROOT}/tools/digester/digester ]]; then
    (
      cd "${PROJECT_ROOT}/tools/digester"
      go build .
    )
  fi

  local image
  image=$("${PROJECT_ROOT}/tools/digester/digester" -image "$1" "$2")
  echo "${image}"
}

PROJECT_ROOT="$(readlink -e $(dirname "${BASH_SOURCE[0]}")/../)"
source "${PROJECT_ROOT}"/hack/config
source "${PROJECT_ROOT}"/deploy/images.env

DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CRD_DIR="${DEPLOY_DIR}/crds"
OLM_DIR="${DEPLOY_DIR}/olm-catalog"
CSV_VERSION=${CSV_VERSION}
CSV_TIMESTAMP=$(date +%Y%m%d%H%M -u)
PACKAGE_NAME="community-kubevirt-hyperconverged"
CSV_DIR="${OLM_DIR}/${PACKAGE_NAME}/${CSV_VERSION}"
DEFAULT_CSV_GENERATOR="/usr/bin/csv-generator"
SSP_CSV_GENERATOR="/csv-generator"

INDEX_IMAGE_DIR=${DEPLOY_DIR}/index-image
CSV_INDEX_IMAGE_DIR="${INDEX_IMAGE_DIR}/${PACKAGE_NAME}/${CSV_VERSION}"

OPERATOR_NAME="${OPERATOR_NAME:-kubevirt-hyperconverged-operator}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-kubevirt-hyperconverged}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

# Important extensions
CSV_EXT="clusterserviceversion.yaml"
CSV_CRD_EXT="csv_crds.yaml"
CRD_EXT="crd.yaml"

function gen_csv() {
  # Handle arguments
  local csvGeneratorPath="$1" && shift
  local operatorName="$1" && shift
  local imagePullUrl="$1" && shift
  local dumpCRDsArg="$1" && shift
  local operatorArgs="$@"

  # Handle important vars
  local csv="${operatorName}.${CSV_EXT}"
  local csvWithCRDs="${operatorName}.${CSV_CRD_EXT}"
  local crds="${operatorName}.crds.yaml"

  # TODO: Use oc to run if cluster is available
  local dockerArgs="docker run --rm --entrypoint=${csvGeneratorPath} ${imagePullUrl} ${operatorArgs}"

  eval $dockerArgs > $csv
  eval $dockerArgs $dumpCRDsArg > $csvWithCRDs

  # diff returns 1 when there is a diff, and there is always diff here. Added `|| :` to cancel trap here.
  diff -u $csv $csvWithCRDs | grep -E "^\+" | sed -E 's/^\+//' | tail -n+2 > $crds || :

  csplit --digits=2 --quiet --elide-empty-files \
    --prefix="${operatorName}" \
    --suffix-format="%02d.${CRD_EXT}" \
    $crds \
    "/^---$/" "{*}"
}

function create_virt_csv() {
  local apiSha
  local controllerSha
  local launcherSha
  local handlerSha

  apiSha="${KUBEVIRT_API_IMAGE/*@/}"
  controllerSha="${KUBEVIRT_CONTROLLER_IMAGE/*@/}"
  launcherSha="${KUBEVIRT_LAUNCHER_IMAGE/*@/}"
  handlerSha="${KUBEVIRT_HANDLER_IMAGE/*@/}"

  local operatorName="kubevirt"
  local dumpCRDsArg="--dumpCRDs"
  local operatorArgs
  operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csvVersion=${CSV_VERSION} \
    --operatorImageVersion=${KUBEVIRT_OPERATOR_IMAGE/*@/} \
    --dockerPrefix=${KUBEVIRT_OPERATOR_IMAGE%\/*} \
    --kubeVirtVersion=${KUBEVIRT_VERSION} \
    --apiSha=${apiSha} \
    --controllerSha=${controllerSha} \
    --handlerSha=${handlerSha} \
    --launcherSha=${launcherSha} \
  "

  gen_csv "${DEFAULT_CSV_GENERATOR}" "${operatorName}" "${KUBEVIRT_OPERATOR_IMAGE}" "${dumpCRDsArg}" "${operatorArgs}"
  echo "${operatorName}"
}

function create_cna_csv() {
  local operatorName="cluster-network-addons"
  local dumpCRDsArg="--dump-crds"
  local containerPrefix="${CNA_OPERATOR_IMAGE%/*}"
  local imageName="${CNA_OPERATOR_IMAGE#${containerPrefix}/}"
  local tag="${CNA_OPERATOR_IMAGE/*:/}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --version=${CSV_VERSION} \
    --version-replaces=${REPLACES_VERSION} \
    --image-pull-policy=IfNotPresent \
    --operator-version=${NETWORK_ADDONS_VERSION} \
    --container-tag=${CNA_OPERATOR_IMAGE/*:/} \
    --container-prefix=${containerPrefix} \
    --image-name=${imageName/:*/}
  "

  gen_csv ${DEFAULT_CSV_GENERATOR} ${operatorName} "${CNA_OPERATOR_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_ssp_csv() {
  local operatorName="scheduling-scale-performance"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image=${SSP_OPERATOR_IMAGE} \
    --operator-version=${SSP_VERSION} \
  "

  gen_csv ${SSP_CSV_GENERATOR} ${operatorName} "${SSP_OPERATOR_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_cdi_csv() {
  local operatorName="containerized-data-importer"

  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --pull-policy=IfNotPresent \
    --operator-image=${CDI_OPERATOR_IMAGE} \
    --controller-image=${CDI_CONTROLLER_IMAGE} \
    --apiserver-image=${CDI_APISERVER_IMAGE} \
    --cloner-image=${CDI_CLONER_IMAGE} \
    --importer-image=${CDI_IMPORTER_IMAGE} \
    --uploadproxy-image=${CDI_UPLOADPROXY_IMAGE} \
    --uploadserver-image=${CDI_UPLOADSERVER_IMAGE} \
    --operator-version=${CDI_VERSION} \
  "
  gen_csv ${DEFAULT_CSV_GENERATOR} ${operatorName} "${CDI_OPERATOR_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_nmo_csv() {
  local operatorName="node-maintenance"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image=${NMO_IMAGE} \
  "
  local csvGeneratorPath="/usr/local/bin/csv-generator"

  gen_csv ${csvGeneratorPath} ${operatorName} "${NMO_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_hpp_csv() {
  local operatorName="hostpath-provisioner"
  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${HPPO_IMAGE} \
    --provisioner-image-name=${HPP_IMAGE} \
    --namespace=${OPERATOR_NAMESPACE} \
    --pull-policy=IfNotPresent \
  "

  gen_csv ${DEFAULT_CSV_GENERATOR} ${operatorName} "${HPPO_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

function create_vm_import_csv() {
  local operatorName="vm-import-operator"
  local containerPrefix="${VMIMPORT_OPERATOR_IMAGE%/*}"

  local dumpCRDsArg="--dump-crds"
  local operatorArgs=" \
    --csv-version=${CSV_VERSION} \
    --operator-version=${VM_IMPORT_VERSION} \
    --operator-image=${VMIMPORT_OPERATOR_IMAGE} \
    --controller-image=${VMIMPORT_CONTROLLER_IMAGE} \
    --namespace=${OPERATOR_NAMESPACE} \
    --virtv2v-image=${VMIMPORT_VIRTV2V_IMAGE} \
    --pull-policy=IfNotPresent \
  "

  gen_csv ${DEFAULT_CSV_GENERATOR} ${operatorName} "${VMIMPORT_OPERATOR_IMAGE}" ${dumpCRDsArg} ${operatorArgs}
  echo "${operatorName}"
}

TEMPDIR=$(mktemp -d) || (echo "Failed to create temp directory" && exit 1)
pushd $TEMPDIR
virtFile=$(create_virt_csv)
virtCsv="${TEMPDIR}/${virtFile}.${CSV_EXT}"
cnaFile=$(create_cna_csv)
cnaCsv="${TEMPDIR}/${cnaFile}.${CSV_EXT}"
sspFile=$(create_ssp_csv)
sspCsv="${TEMPDIR}/${sspFile}.${CSV_EXT}"
cdiFile=$(create_cdi_csv)
cdiCsv="${TEMPDIR}/${cdiFile}.${CSV_EXT}"
nmoFile=$(create_nmo_csv)
nmoCsv="${TEMPDIR}/${nmoFile}.${CSV_EXT}"
hhpFile=$(create_hpp_csv)
hppCsv="${TEMPDIR}/${hhpFile}.${CSV_EXT}"
vmImportFile=$(create_vm_import_csv)
importCsv="${TEMPDIR}/${vmImportFile}.${CSV_EXT}"
csvOverrides="${TEMPDIR}/csv_overrides.${CSV_EXT}"
keywords="  keywords:
  - KubeVirt
  - Virtualization
  - VM"
cat > ${csvOverrides} <<- EOM
---
spec:
$keywords
EOM

# Write HCO CRDs
(cd ${PROJECT_ROOT}/tools/csv-merger/ && go build)
hco_crds=${TEMPDIR}/hco.crds.yaml
(cd ${PROJECT_ROOT} && ${PROJECT_ROOT}/tools/csv-merger/csv-merger  --api-sources=${PROJECT_ROOT}/pkg/apis/... --output-mode=CRDs > $hco_crds)
csplit --digits=2 --quiet --elide-empty-files \
  --prefix=hco \
  --suffix-format="%02d.${CRD_EXT}" \
  $hco_crds \
  "/^---$/" "{*}"

popd

rm -fr "${CSV_DIR}"
mkdir -p "${CSV_DIR}/metadata" "${CSV_DIR}/manifests"


cat << EOF > "${CSV_DIR}/metadata/annotations.yaml"
annotations:
  operators.operatorframework.io.bundle.channel.default.v1: ${CSV_VERSION}
  operators.operatorframework.io.bundle.channels.v1: ${CSV_VERSION}
  operators.operatorframework.io.bundle.manifests.v1: manifests/
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  operators.operatorframework.io.bundle.package.v1: ${PACKAGE_NAME}
EOF

SMBIOS=$(cat <<- EOM
Family: KubeVirt
Manufacturer: KubeVirt
Product: None
EOM
)

# validate CSVs. Make sure each one of them contain an image (and so, also not empty):
csvs=("${cnaCsv}" "${virtCsv}" "${sspCsv}" "${cdiCsv}" "${nmoCsv}" "${hppCsv}" "${importCsv}")
for csv in "${csvs[@]}"; do
  grep -E "^ *image: [a-zA-Z0-9/\.:@\-]+$" ${csv}
done

if [[ -n ${OPERATOR_IMAGE} ]]; then
  TEMP_IMAGE_NAME=$(get_image_digest "${OPERATOR_IMAGE}")
  DIGEST_LIST="${DIGEST_LIST/${HCO_OPERATOR_IMAGE}/${TEMP_IMAGE_NAME}}"
  HCO_OPERATOR_IMAGE=${TEMP_IMAGE_NAME}
fi

if [[ -n ${WEBHOOK_IMAGE} ]]; then
  TEMP_IMAGE_NAME=$(get_image_digest "${WEBHOOK_IMAGE}")
  if [[ -n ${HCO_WEBHOOK_IMAGE} ]]; then
    DIGEST_LIST="${DIGEST_LIST/${HCO_WEBHOOK_IMAGE}/${TEMP_IMAGE_NAME}}"
  else
    DIGEST_LIST="${DIGEST_LIST},${TEMP_IMAGE_NAME}"
  fi
  HCO_WEBHOOK_IMAGE=${TEMP_IMAGE_NAME}
fi

if [[ -z ${HCO_WEBHOOK_IMAGE} ]]; then
  HCO_WEBHOOK_IMAGE="${HCO_OPERATOR_IMAGE}"
fi

# Build and write deploy dir
(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go build)
${PROJECT_ROOT}/tools/manifest-templator/manifest-templator \
  --api-sources=${PROJECT_ROOT}/pkg/apis/... \
  --cna-csv="$(<${cnaCsv})" \
  --virt-csv="$(<${virtCsv})" \
  --ssp-csv="$(<${sspCsv})" \
  --cdi-csv="$(<${cdiCsv})" \
  --nmo-csv="$(<${nmoCsv})" \
  --hpp-csv="$(<${hppCsv})" \
  --vmimport-csv="$(<${importCsv})" \
  --ims-conversion-image-name="${CONVERSION_IMAGE}" \
  --ims-vmware-image-name="${VMWARE_IMAGE}" \
  --kv-virtiowin-image-name="${KUBEVIRT_VIRTIO_IMAGE}" \
  --operator-namespace="${OPERATOR_NAMESPACE}" \
  --smbios="${SMBIOS}" \
  --hco-kv-io-version="${CSV_VERSION}" \
  --kubevirt-version="${KUBEVIRT_VERSION}" \
  --cdi-version="${CDI_VERSION}" \
  --cnao-version="${NETWORK_ADDONS_VERSION}" \
  --ssp-version="${SSP_VERSION}" \
  --nmo-version="${NMO_VERSION}" \
  --hppo-version="${HPPO_VERSION}" \
  --vm-import-version="${VM_IMPORT_VERSION}" \
  --operator-image="${HCO_OPERATOR_IMAGE}" \
  --webhook-image="${HCO_WEBHOOK_IMAGE}"
(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go clean)

if [[ "$1" == "UNIQUE"  ]]; then
  CSV_VERSION_PARAM=${CSV_VERSION}-${CSV_TIMESTAMP}
  ENABLE_UNIQUE="true"
else
  CSV_VERSION_PARAM=${CSV_VERSION}
  ENABLE_UNIQUE="false"
fi

# Build and merge CSVs
CSV_DIR=${CSV_DIR}/manifests
${PROJECT_ROOT}/tools/csv-merger/csv-merger \
  --cna-csv="$(<${cnaCsv})" \
  --virt-csv="$(<${virtCsv})" \
  --ssp-csv="$(<${sspCsv})" \
  --cdi-csv="$(<${cdiCsv})" \
  --nmo-csv="$(<${nmoCsv})" \
  --hpp-csv="$(<${hppCsv})" \
  --vmimport-csv="$(<${importCsv})" \
  --ims-conversion-image-name="${CONVERSION_IMAGE}" \
  --ims-vmware-image-name="${VMWARE_IMAGE}" \
  --kv-virtiowin-image-name="${KUBEVIRT_VIRTIO_IMAGE}" \
  --csv-version=${CSV_VERSION_PARAM} \
  --replaces-csv-version=${REPLACES_CSV_VERSION} \
  --hco-kv-io-version="${CSV_VERSION}" \
  --spec-displayname="KubeVirt HyperConverged Cluster Operator" \
  --spec-description="$(<${PROJECT_ROOT}/docs/operator_description.md)" \
  --metadata-description="A unified operator deploying and controlling KubeVirt and its supporting operators with opinionated defaults" \
  --crd-display="HyperConverged Cluster Operator" \
  --smbios="${SMBIOS}" \
  --csv-overrides="$(<${csvOverrides})" \
  --enable-unique-version=${ENABLE_UNIQUE} \
  --kubevirt-version="${KUBEVIRT_VERSION}" \
  --cdi-version="${CDI_VERSION}" \
  --cnao-version="${NETWORK_ADDONS_VERSION}" \
  --ssp-version="${SSP_VERSION}" \
  --nmo-version="${NMO_VERSION}" \
  --hppo-version="${HPPO_VERSION}" \
  --vm-import-version="${VM_IMPORT_VERSION}" \
  --related-images-list="${DIGEST_LIST}" \
  --operator-image-name="${HCO_OPERATOR_IMAGE}" \
  --webhook-image-name="${HCO_WEBHOOK_IMAGE}" > "${CSV_DIR}/${OPERATOR_NAME}.v${CSV_VERSION}.${CSV_EXT}"

rendered_csv="$(cat "${CSV_DIR}/${OPERATOR_NAME}.v${CSV_VERSION}.${CSV_EXT}")"
rendered_keywords="$(echo "$rendered_csv" |grep 'keywords' -A 3)"
# assert that --csv-overrides work
[ "$keywords" == "$rendered_keywords" ]

# Copy all CRDs into the CRD and CSV directories
rm -f ${CRD_DIR}/*
cp -f ${TEMPDIR}/*.${CRD_EXT} ${CRD_DIR}
cp -f ${TEMPDIR}/*.${CRD_EXT} ${CSV_DIR}

# Validate the yaml files
(cd ${CRD_DIR} && docker run --rm -v "$(pwd)":/yaml quay.io/pusher/yamllint yamllint -d "{extends: relaxed, rules: {line-length: disable}}" /yaml)
(cd ${CSV_DIR} && docker run --rm -v "$(pwd)":/yaml quay.io/pusher/yamllint yamllint -d "{extends: relaxed, rules: {line-length: disable}}" /yaml)

# Check there are not API Groups overlap between different CNV operators
${PROJECT_ROOT}/tools/csv-merger/csv-merger --crds-dir=${CRD_DIR}
(cd ${PROJECT_ROOT}/tools/csv-merger/ && go clean)

if [[ "$1" == "UNIQUE"  ]]; then
  # Add the current CSV_TIMESTAMP to the currentCSV in the packages file
  sed -Ei "s/(currentCSV: ${OPERATOR_NAME}.v${CSV_VERSION}).*/\1-${CSV_TIMESTAMP}/" \
   ${PACKAGE_DIR}/kubevirt-hyperconverged.package.yaml
fi

# Intentionally removing last so failure leaves around the templates
rm -rf ${TEMPDIR}

# If the only change in the CSV file is its "created_at" field, rollback this change as it causes git conflicts for
# no good reason.
CSV_FILE="${CSV_DIR}/kubevirt-hyperconverged-operator.v${CSV_VERSION}.${CSV_EXT}"
if git difftool -y --trust-exit-code --extcmd=./hack/diff-csv.sh ${CSV_FILE}; then
  git checkout ${CSV_FILE}
fi

# Prepare files for index-image files that will be used for testing in openshift CI
rm -rf "${INDEX_IMAGE_DIR:?}"
mkdir -p "${INDEX_IMAGE_DIR:?}/${PACKAGE_NAME}"
cp -r "${CSV_DIR%/*}" "${INDEX_IMAGE_DIR:?}/${PACKAGE_NAME}/"
cp "${OLM_DIR}/bundle.Dockerfile" "${INDEX_IMAGE_DIR:?}/"

INDEX_IMAGE_CSV="${INDEX_IMAGE_DIR}/${PACKAGE_NAME}/${CSV_VERSION}/manifests/kubevirt-hyperconverged-operator.v${CSV_VERSION}.${CSV_EXT}"
sed -r -i "s|createdAt: \".*\$|createdAt: \"2020-10-23 08:58:25\"|; s|quay.io/kubevirt/hyperconverged-cluster-operator.*$|+IMAGE_TO_REPLACE+|; s|quay.io/kubevirt/hyperconverged-cluster-webhook.*$|+WEBHOOK_IMAGE_TO_REPLACE+|" ${INDEX_IMAGE_CSV}
