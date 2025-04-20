#!/usr/bin/env bash

set -ex

source "hack/cri-bin.sh"

function dump() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during HCO CR deployment: exit status: $rv"
        ${KUBECTL} logs -n olm -l app=olm-operator
        ${KUBECTL} get pod -n kubevirt-hyperconverged
        ${KUBECTL} logs -n kubevirt-hyperconverged -l name=hyperconverged-cluster-operator
        echo "*** HCO CR deployment failed ***"
    fi
    exit $rv
}

# Get golang
$CRI_BIN login --username "$(cat "${QUAY_USER}")" --password-stdin quay.io < "${QUAY_PASSWORD}"
wget -q https://dl.google.com/go/go1.23.5.linux-amd64.tar.gz
tar -C /usr/local -xf go*.tar.gz
export PATH=/usr/local/go/bin:$PATH

# add qemu-user-static
$CRI_BIN run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes

# get latest KubeVirt commit
latest_kubevirt=$(curl -sL https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/latest)
latest_kubevirt_image=$(curl -sL "https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/${latest_kubevirt}/kubevirt-operator.yaml" | grep 'OPERATOR_IMAGE' -A1 | tail -n 1 | sed 's/.*value: //g')
IFS=: read -r kv_image kv_tag <<< "${latest_kubevirt_image}"
latest_kubevirt_commit=$(curl -sL "https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/${latest_kubevirt}/commit")

# get latest CDI commit
latest_cdi=$(curl -sL https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/containerized-data-importer/latest)
latest_cdi_image=$(curl -sL "https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/containerized-data-importer/${latest_cdi}/cdi-operator.yaml" | grep "image:" | sed -E "s|^ +image: (.*)$|\1|")
IFS=: read -r cdi_image cdi_tag <<< "${latest_cdi_image}"
latest_cdi_commit=$(curl -sL "https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/containerized-data-importer/${latest_cdi}/commit")

# Update HCO dependencies
go mod tidy
go mod vendor
rm -rf kubevirt cdi

# Get latest kubevirt
git clone https://github.com/kubevirt/kubevirt.git
(cd kubevirt; git checkout "${latest_kubevirt_commit}")

# Get latest CDI
git clone https://github.com/kubevirt/containerized-data-importer.git cdi
(cd cdi; git checkout "${latest_cdi_commit}")

go mod edit -replace kubevirt.io/api=./kubevirt/staging/src/kubevirt.io/api
go mod edit -replace kubevirt.io/containerized-data-importer-api=./cdi/staging/src/kubevirt.io/containerized-data-importer-api

go mod tidy
go mod vendor

# set envs
build_date="$(date +%Y%m%d)"
export IMAGE_REGISTRY=quay.io
export IMAGE_TAG="nb_${build_date}_$(git show -s --format=%h)"
export REGISTRY_NAMESPACE=kubevirtci

# Build HCO & HCO Webhook
make build-push-multi-arch-operator-image build-push-multi-arch-webhook-image build-push-multi-arch-artifacts-server

# Update image digests
sed -i "s#quay.io/kubevirt/virt-#${kv_image/-*/-}#" deploy/images.csv
sed -i "s#^KUBEVIRT_VERSION=.*#KUBEVIRT_VERSION=\"${kv_tag}\"#" hack/config
sed -i "s#^CDI_VERSION=.*#CDI_VERSION=\"${cdi_tag}\"#" hack/config
(cd ./tools/digester && go build .)
export HCO_VERSION="${IMAGE_TAG}"
./automation/digester/update_images.sh

HCO_OPERATOR_IMAGE_DIGEST=$(tools/digester/digester --image "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/hyperconverged-cluster-operator:${IMAGE_TAG}")
HCO_WEBHOOK_IMAGE_DIGEST=$(tools/digester/digester --image "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/hyperconverged-cluster-webhook:${IMAGE_TAG}")
HCO_DOWNLOAD_IMAGE_DIGEST=$(tools/digester/digester --image "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/virt-artifacts-server:${IMAGE_TAG}")

# Build the CSV
HCO_OPERATOR_IMAGE=${HCO_OPERATOR_IMAGE_DIGEST} HCO_WEBHOOK_IMAGE=${HCO_WEBHOOK_IMAGE_DIGEST} HCO_DOWNLOADS_IMAGE=${HCO_DOWNLOAD_IMAGE_DIGEST} ./hack/build-manifests.sh

# Download OPM
OPM_VERSION=v1.47.0
wget https://github.com/operator-framework/operator-registry/releases/download/${OPM_VERSION}/linux-amd64-opm -O opm
chmod +x opm
export OPM=$(pwd)/opm

# create and push bundle image and index image
REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE} IMAGE_TAG=${IMAGE_TAG} ./hack/build-index-image.sh latest UNSTABLE

BUNDLE_REGISTRY_IMAGE_NAME=${BUNDLE_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-bundle}
INDEX_REGISTRY_IMAGE_NAME=${INDEX_REGISTRY_IMAGE_NAME:-hyperconverged-cluster-index}
BUNDLE_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${BUNDLE_REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"
INDEX_IMAGE_NAME="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${INDEX_REGISTRY_IMAGE_NAME}:${IMAGE_TAG}"

# build succeeded: publish the nightly build
hco_bucket="kubevirt-prow/devel/nightly/release/kubevirt/hyperconverged-cluster-operator"
echo "${BUNDLE_IMAGE_NAME}" > hco-bundle
echo "${INDEX_IMAGE_NAME}" > hco-index
gsutil cp ./hco-bundle "gs://${hco_bucket}/${build_date}/hco-bundle-image"
gsutil cp ./hco-index "gs://${hco_bucket}/${build_date}/hco-index-image"

# download operator-sdk
sdk_url=$(curl https://api.github.com/repos/operator-framework/operator-sdk/releases/latest | jq -rM '.assets[] | select(.name == "operator-sdk_linux_amd64") | .browser_download_url')
wget $sdk_url -O operator-sdk
chmod +x operator-sdk

OLM_VERSION=$(curl https://api.github.com/repos/operator-framework/operator-lifecycle-manager/releases/latest | jq -r .name)

# start K8s cluster
export KUBEVIRT_PROVIDER=k8s-1.32
export KUBEVIRT_MEMORY_SIZE=12G
export KUBEVIRT_NUM_NODES=4
# auto updated by hack/bump-kubevirtci.sh
export KUBEVIRTCI_TAG=${KUBEVIRTCI_TAG:-"2504171552-a558e3fe"}
make cluster-up

export KUBECONFIG=$(_kubevirtci/cluster-up/kubeconfig.sh)
export KUBECTL=$(pwd)/_kubevirtci/cluster-up/kubectl.sh

# install OLM on the cluster
./operator-sdk olm install --version "${OLM_VERSION}"

# Deploy cert-manager for webhooks
$KUBECTL apply -f deploy/cert-manager.yaml
$KUBECTL -n cert-manager wait deployment/cert-manager-webhook --for=condition=Available --timeout="300s"

trap "dump" INT TERM EXIT

# install HCO on the cluster
$KUBECTL create ns kubevirt-hyperconverged
./operator-sdk run bundle -n kubevirt-hyperconverged --timeout=10m ${BUNDLE_IMAGE_NAME}

# deploy the HyperConverged CR
$KUBECTL apply -n kubevirt-hyperconverged -f deploy/hco.cr.yaml
$KUBECTL wait -n kubevirt-hyperconverged hco kubevirt-hyperconverged --for=condition=Available --timeout=5m

# build func test
go install github.com/onsi/ginkgo/v2/ginkgo@$(grep github.com/onsi/ginkgo go.mod | cut -d " " -f2)
test_path="./tests/func-tests"
ginkgo build -o functest.test ${test_path}

REPORT_FLAG=""
if [[ -n ${ARTIFACTS} ]]; then
  REPORT_FLAG="${ARTIFACTS}/junit.xml"
fi

# run functest
./functest.test  -ginkgo.v -ginkgo.junit-report="${REPORT_FLAG}" -installed-namespace="kubevirt-hyperconverged" --ginkgo.label-filter='!OpenShift && !SINGLE_NODE_ONLY'

# functional test passed: publish latest nightly build
echo "${build_date}" > build-date
gsutil cp ./build-date gs://${hco_bucket}/latest
