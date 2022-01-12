#!/usr/bin/env bash

set -ex

IMAGES_NS=${IMAGES_NS:-kubevirt-os-images}

function count_data_import_crons() {
  echo $(${KUBECTL_BINARY} get DataImportCron -A --no-headers | wc -l);
}

export -f count_data_import_crons

if [[ $(${KUBECTL_BINARY} get ssp -n ${INSTALLED_NAMESPACE}) ]]; then

  # test image streams
  [[ $(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} --no-headers | wc -l) -eq 1 ]]
  [[ "$(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} -o json | jq -cM '.spec.tags[0].from')" == '{"kind":"DockerImage","name":"quay.io/kubevirt/centos8-container-disk-images"}' ]]

  # check that HCO reconciles the image stream
  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch imageStream -n ${IMAGES_NS} centos8 --type=json -p '[{\"op\": \"add\", \"path\": \"/metadata/labels/test-label\", \"value\": \"test\"}]'"
  sleep 10
  # HCO expect to remove the test-label label from the image stream
  ./hack/retry.sh 10 3 "[[ -z '$(${KUBECTL_BINARY} get imageStream -n ${IMAGES_NS} centos8 -o jsonpath='{.metadata.labels.test-label}')' ]]" "${KUBECTL_BINARY} get imageStream -n ${IMAGES_NS} centos8 -o yaml"

  ${KUBECTL_BINARY} get hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -o jsonpath='{.spec.featureGates.enableCommonBootImageImport}'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos-stream8-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos-stream9-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="fedora-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos8-image-cron-is")'

  ./hack/retry.sh 10 30 "[[ \$(count_data_import_crons) -eq 4 ]]" "${KUBECTL_BINARY} get DataImportCron -A"

  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} centos-stream8-image-cron
  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} centos-stream9-image-cron
  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} fedora-image-cron
  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} centos8-image-cron-is

  [[ $(${KUBECTL_BINARY} get DataImportCron -o json -n ${IMAGES_NS} centos8-image-cron-is | jq -cM '.spec.template.spec.source.registry') == '{"imageStream":"centos8","pullMethod":"node"}' ]]

  # disable the feature
  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": false }]'"
  sleep 10

  # check that the image streams and the DataImportCron were removed
  ./hack/retry.sh 10 3 "[[ $(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} --no-headers | wc -l) -eq 0 ]]"
  ./hack/retry.sh 10 3 "[[ $(${KUBECTL_BINARY} get DataImportCron -A --no-headers | wc -l) -eq 0 ]]"

  # enable it back
  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": true }]'"
  sleep 10

  # test image streams
  [[ $(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} --no-headers | wc -l) -eq 1 ]]
  [[ "$(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} -o json | jq -cM '.spec.tags[0].from')" == '{"kind":"DockerImage","name":"quay.io/kubevirt/centos8-container-disk-images"}' ]]

fi
