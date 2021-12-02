#!/usr/bin/env bash

set -ex

IMAGES_NS=${IMAGES_NS:-kubevirt-os-images}

if [[ $(${KUBECTL_BINARY} get ssp -n ${INSTALLED_NAMESPACE}) ]]; then
  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": true }]'"
  sleep 10

  ${KUBECTL_BINARY} get hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -o jsonpath='{.spec.featureGates.enableCommonBootImageImport}'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos8-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="fedora-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos8-image-cron-is")'

  ./hack/retry.sh 12 5 "[[ $(${KUBECTL_BINARY} get DataImportCron -A --no-headers | wc -l) -eq 3 ]]"

  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} centos8-image-cron
  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} fedora-image-cron
  ${KUBECTL_BINARY} get DataImportCron -o yaml -n ${IMAGES_NS} centos8-image-cron-is

  [[ $(${KUBECTL_BINARY} get DataImportCron -o json -n ${IMAGES_NS} centos8-image-cron-is | jq -cM '.spec.template.spec.source.registry') == '{"imageStream":"centos8","pullMethod":"node"}' ]]

  [[ $(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} --no-headers | wc -l) -eq 1 ]]
  [[ "$(${KUBECTL_BINARY} get imageStream centos8  -n ${IMAGES_NS} -o json | jq -cM '.spec.tags[0].from')" == '{"kind":"DockerImage","name":"quay.io/kubevirt/centos8-container-disk-images"}' ]]

  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": false }]'"
  sleep 10

  ./hack/retry.sh 10 3 "[[ $(${KUBECTL_BINARY} get DataImportCron -A --no-headers | wc -l) -eq 0 ]]"
fi
