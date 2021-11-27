#!/usr/bin/env bash

set -ex

if [[ $(${KUBECTL_BINARY} get ssp -n ${INSTALLED_NAMESPACE}) ]]; then
  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": true }]'"
  sleep 10

  ${KUBECTL_BINARY} get hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -o jsonpath='{.spec.featureGates.enableCommonBootImageImport}'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="centos8-image-cron")'
  ${KUBECTL_BINARY} get ssp -n "${INSTALLED_NAMESPACE}" ssp-kubevirt-hyperconverged -o jsonpath='{.spec.commonTemplates.dataImportCronTemplates}' | jq -e '.[] |select(.metadata.name=="fedora-image-cron")'

  ${KUBECTL_BINARY} get DataImportCron -o yaml -A

  ./hack/retry.sh 10 3 "${KUBECTL_BINARY} patch hco -n \"${INSTALLED_NAMESPACE}\" --type=json kubevirt-hyperconverged -p '[{ \"op\": \"replace\", \"path\": \"/spec/featureGates/enableCommonBootImageImport\", \"value\": false }]'"
  sleep 10

  ./hack/retry.sh 10 3 "[[ $(${KUBECTL_BINARY} get DataImportCron -A --no-headers | wc -l) -eq 0 ]]"

fi