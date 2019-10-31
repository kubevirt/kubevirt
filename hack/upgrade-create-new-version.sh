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

LATEST_VERSION=$(ls -d ./deploy/olm-catalog/kubevirt-hyperconverged/*/ | sort -r | head -1 | cut -d '/' -f 5);

echo "LATEST_VERSION: $LATEST_VERSION"
echo "UPGRADE_VERSION: $UPGRADE_VERSION"

export REPLACES_VERSION=$LATEST_VERSION
export CSV_VERSION=$UPGRADE_VERSION
./hack/build-manifests.sh 

sed -i "s|currentCSV: kubevirt-hyperconverged-operator.v$LATEST_VERSION|currentCSV: kubevirt-hyperconverged-operator.v$UPGRADE_VERSION|g" ./deploy/olm-catalog/kubevirt-hyperconverged/kubevirt-hyperconverged.package.yaml