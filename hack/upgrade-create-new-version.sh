# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2019 Red Hat, Inc.
#
set -ex

DEPLOY_DIR="./deploy"
PACKAGE_DIR="${DEPLOY_DIR}/olm-catalog/community-kubevirt-hyperconverged"
LATEST_VERSION=$(ls -d ${PACKAGE_DIR}/*/ | sort -rV | head -1 | cut -d '/' -f 5)

OPERATOR_NAME="kubevirt-hyperconverged-operator"
LATEST_CSV_DIR="${PACKAGE_DIR}/${LATEST_VERSION}"
LATEST_CSV_NAME="${OPERATOR_NAME}.v${LATEST_VERSION}.clusterserviceversion.yaml"
UPGRADE_CSV_DIR="${PACKAGE_DIR}/${UPGRADE_VERSION}"
UPGRADE_CSV="${UPGRADE_CSV_DIR}/manifests/${OPERATOR_NAME}.v${UPGRADE_VERSION}.clusterserviceversion.yaml"

echo "LATEST_VERSION: $LATEST_VERSION"
echo "UPGRADE_VERSION: $UPGRADE_VERSION"

if [[ -z $PREV ]]; then
  cp -r "${LATEST_CSV_DIR}" "${UPGRADE_CSV_DIR}"
  REPLACES_VERSION=${LATEST_VERSION}
else
  REPLACES_VERSION=$(ls -d ${PACKAGE_DIR}/*/ | sort -rV | awk "NR==2" | cut -d '/' -f 5)
  mv "${LATEST_CSV_DIR}" "${UPGRADE_CSV_DIR}"
fi

mv "${UPGRADE_CSV_DIR}/manifests/${LATEST_CSV_NAME}" "${UPGRADE_CSV}"

sed -i "s|${OPERATOR_NAME}.v${LATEST_VERSION}|${OPERATOR_NAME}.v${UPGRADE_VERSION}|g" "${UPGRADE_CSV}"
sed -i "s|replaces:.*|replaces: ${OPERATOR_NAME}.v${REPLACES_VERSION}|" "${UPGRADE_CSV}"
sed -i "s|version:\s*${LATEST_VERSION}|version: ${UPGRADE_VERSION}|g" "${UPGRADE_CSV}"
sed -i "s|value:\s*${LATEST_VERSION}|value: ${UPGRADE_VERSION}|g" "${UPGRADE_CSV}"
if [[ -z $PREV ]]; then
  sed -i "/^channels:/a - name: \"${UPGRADE_VERSION}\"\n  currentCSV: ${OPERATOR_NAME}.v${UPGRADE_VERSION}" ${PACKAGE_DIR}/kubevirt-hyperconverged.package.yaml
else
  sed -i "s|${LATEST_VERSION}|${UPGRADE_VERSION}|g" ${PACKAGE_DIR}/kubevirt-hyperconverged.package.yaml
  sed -i "s|^defaultChannel:.*|defaultChannel: ${REPLACES_VERSION}|g" ${PACKAGE_DIR}/kubevirt-hyperconverged.package.yaml
fi

# enable KVM_EMULATION for CI, needed by kubevirt-node-labeller on AWS
find ${PACKAGE_DIR} -type f -exec sed -E -i 's|^(\s*)- name: KVM_EMULATION$|\1- name: KVM_EMULATION\n\1  value: "true"|' {} \; || :

cat ${PACKAGE_DIR}/kubevirt-hyperconverged.package.yaml
