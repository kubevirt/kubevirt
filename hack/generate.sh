#!/usr/bin/env bash

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}
/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh
(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go build)

${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json

(cd ${KUBEVIRT_DIR}/tools/resource-generator/ && go build)
(cd ${KUBEVIRT_DIR}/tools/csv-generator/ && go build)
rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/examples/*
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmi >${KUBEVIRT_DIR}/manifests/generated/vmi-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmirs >${KUBEVIRT_DIR}/manifests/generated/vmirs-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmipreset >${KUBEVIRT_DIR}/manifests/generated/vmipreset-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vm >${KUBEVIRT_DIR}/manifests/generated/vm-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmim >${KUBEVIRT_DIR}/manifests/generated/vmim-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${KUBEVIRT_DIR}/manifests/generated/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace={{.Namespace}} --pullPolicy={{.ImagePullPolicy}} >${KUBEVIRT_DIR}/manifests/generated/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kubevirt-rbac --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/rbac-kubevirt.authorization.k8s.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=cluster-rbac --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/rbac-cluster.authorization.k8s.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=operator-rbac --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/rbac-operator.authorization.k8s.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=prometheus --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/prometheus.yaml.in

# used for Image fields in manifests
function getVersion() {
    echo "{{if $1}}@{{$1}}{{else}}:{{.DockerTag}}{{end}}"
}
virtapi_version=$(getVersion ".VirtApiSha")
virtcontroller_version=$(getVersion ".VirtControllerSha")
virthandler_version=$(getVersion ".VirtHandlerSha")
virtlauncher_version=$(getVersion ".VirtLauncherSha")
virtoperator_version=$(getVersion ".VirtOperatorSha")

# used as env var for operator
function getShasum() {
    echo "{{if $1}}@{{$1}}{{end}}"
}

# without the '@' symbole used in 'getShasum'
function getRawShasum() {
    echo "{{if $1}}{{$1}}{{end}}"
}
virtapi_sha=$(getShasum ".VirtApiSha")
virtcontroller_sha=$(getShasum ".VirtControllerSha")
virthandler_sha=$(getShasum ".VirtHandlerSha")
virtlauncher_sha=$(getShasum ".VirtLauncherSha")

virtapi_rawsha=$(getRawShasum ".VirtApiSha")
virtcontroller_rawsha=$(getRawShasum ".VirtControllerSha")
virthandler_rawsha=$(getRawShasum ".VirtHandlerSha")
virtlauncher_rawsha=$(getRawShasum ".VirtLauncherSha")

${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-api --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version="$virtapi_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-api.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-controller --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version="$virtcontroller_version" --launcherVersion="$virtlauncher_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-controller.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-handler --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version="$virthandler_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-handler.yaml.in

# The generation code for CSV requires a valid semver to be used.
# But we're trying to generate a template for a CSV here from code
# rather than an actual usable CSV. To work around this, we set the
# versions to something absurd and do a find/replace with our templated
# values after the file is generated.
_fake_replaces_csv_version="1111.1111.1111"
_fake_csv_version="2222.2222.2222"
${KUBEVIRT_DIR}/tools/csv-generator/csv-generator --namespace={{.CSVNamespace}} --dockerPrefix={{.DockerPrefix}} --operatorImageVersion="$virtoperator_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} --apiSha="$virtapi_rawsha" --controllerSha="$virtcontroller_rawsha" --handlerSha="$virthandler_rawsha" --launcherSha="$virtlauncher_rawsha" --kubevirtLogo={{.KubeVirtLogo}} --csvVersion="$_fake_csv_version" --replacesCsvVersion="$_fake_replaces_csv_version" --csvCreatedAtTimestamp={{.CreatedAt}} --kubeVirtVersion={{.DockerTag}} >${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_csv_version/{{.CsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_replaces_csv_version/{{.ReplacesCsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go build)
vms_docker_prefix=${DOCKER_PREFIX:-registry:5000/kubevirt}
vms_docker_tag=${DOCKER_TAG:-devel}
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --container-prefix=${vms_docker_prefix} --container-tag=${vms_docker_tag} --generated-vms-dir=${KUBEVIRT_DIR}/examples

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=kubevirt_hooks_info:pkg/hooks/info pkg/hooks/info/api.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha1:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api.proto
protoc --proto_path=pkg/hooks/v1alpha2 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha2:pkg/hooks/v1alpha2 pkg/hooks/v1alpha2/api.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/v1/notify.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/info/info.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/v1/cmd.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/info/info.proto

mockgen -source pkg/handler-launcher-com/notify/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/notify/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/cmd/info/generated_mock_info.go

# update cluster-up if needed
version_file="cluster-up/version.txt"
sha_file="cluster-up-sha.txt"
download_cluster_up=true
function getClusterUpShasum() {
    find ${KUBEVIRT_DIR}/cluster-up -type f | sort | xargs sha1sum | sha1sum | awk '{print $1}'
}
# check if we got a new cluster-up git commit hash
if [[ -f "${version_file}" ]] && [[ $(cat ${version_file}) == ${kubevirtci_git_hash} ]]; then
    # check if files are modified
    current_sha=$(getClusterUpShasum)
    if [[ -f "${sha_file}" ]] && [[ $(cat ${sha_file}) == ${current_sha} ]]; then
        echo "cluster-up is up to date and not modified"
        download_cluster_up=false
    else
        echo "cluster-up was modified"
    fi
else
    echo "cluster-up git commit hash was updated"
fi
if [[ "$download_cluster_up" == true ]]; then
    echo "downloading cluster-up"
    rm -rf cluster-up
    curl -L https://github.com/kubevirt/kubevirtci/archive/${kubevirtci_git_hash}/kubevirtci.tar.gz | tar xz kubevirtci-${kubevirtci_git_hash}/cluster-up --strip-component 1

    echo ${kubevirtci_git_hash} >${version_file}
    new_sha=$(getClusterUpShasum)
    echo ${new_sha} >${sha_file}
fi
