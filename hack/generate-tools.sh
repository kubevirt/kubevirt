#!/usr/bin/env bash
#
# Builds the generator tools and regenerates the files they produce:
# the OpenAPI spec, the metrics docs, manifests/generated/* and examples/*.
#
# Called by hack/generate.sh (full run) and hack/generate-partial.sh
# (incremental run). These outputs depend on code under pkg/, so they are
# regenerated on every run that is not skipped entirely.

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go_build)

${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json

(cd ${KUBEVIRT_DIR}/tools/resource-generator/ && go_build)
(cd ${KUBEVIRT_DIR}/tools/csv-generator/ && go_build)
(cd ${KUBEVIRT_DIR}/tools/doc-generator/ && go_build)
(
    cd ${KUBEVIRT_DIR}/docs/observability
    ${KUBEVIRT_DIR}/tools/doc-generator/doc-generator >metrics.md
)

rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/examples/*

ResourceDir=${KUBEVIRT_DIR}/manifests/generated
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=priorityclass >${ResourceDir}/kubevirt-priority-class.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${ResourceDir}/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=networkpolicies --namespace='{{.Namespace}}' >${ResourceDir}/kubevirt-network-policies.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace='{{.Namespace}}' --pullPolicy='{{.ImagePullPolicy}}' \
    --featureGates='{{.FeatureGates}}' --infraReplicas='{{.InfraReplicas}}' \
    --hypervisor='{{.Hypervisor}}' >${ResourceDir}/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=operator-rbac --namespace='{{.Namespace}}' >${ResourceDir}/rbac-operator.authorization.k8s.yaml.in

# The generation code for CSV requires a valid semver to be used.
# But we're trying to generate a template for a CSV here from code
# rather than an actual usable CSV. To work around this, we set the
# versions to something absurd and do a find/replace with our templated
# values after the file is generated.
_fake_replaces_csv_version="1111.1111.1111"
_fake_csv_version="2222.2222.2222"
csv_generator_extra_flags="${CSV_GENERATOR_EXTRA_FLAGS:-}"
${KUBEVIRT_DIR}/tools/csv-generator/csv-generator \
    --operatorImageVersion="{{.DockerTag}}" \
    --csvCreatedAtTimestamp={{.CreatedAt}} \
    --csvVersion="$_fake_csv_version" \
    --dockerPrefix={{.DockerPrefix}} \
    --kubevirtLogo={{.KubeVirtLogo}} \
    --kubeVirtVersion={{.DockerTag}} \
    --namespace={{.CSVNamespace}} \
    --pullPolicy={{.ImagePullPolicy}} \
    --replacesCsvVersion="$_fake_replaces_csv_version" \
    --verbosity={{.Verbosity}} \
    ${csv_generator_extra_flags} \
    >${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

sed -i "s/$_fake_csv_version/{{.CsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_replaces_csv_version/{{.ReplacesCsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go_build)
vms_docker_prefix=${DOCKER_PREFIX:-registry:5000/kubevirt}
vms_docker_tag=${DOCKER_TAG:-devel}
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --container-prefix=${vms_docker_prefix} --container-tag=${vms_docker_tag} --generated-vms-dir=${KUBEVIRT_DIR}/examples
