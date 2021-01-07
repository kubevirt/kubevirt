#!/bin/bash -e
#
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
# Usage:
# export KUBEVIRT_PROVIDER=okd-4.1
# make cluster-up
# make upgrade-test
#
# Start deploying the HCO cluster using the latest images shipped
# in quay.io with latest tag:
# - quay.io/kubevirt/hyperconverged-cluster-operator:latest
# - quay.io/kubevirt/hco-container-registry:latest
#
# A new bundle, named 100.0.0, is then created with the content of
# the open PR (this can include new dependent images, new CRDs...).
# A new hco-operator image is created based off of the code in the
# current checkout.
#
# Both the hco-operator image and new registry image is pushed
# to the local registry.
#
# The subscription is checked to verify that it progresses
# to the new version.
#
# The hyperconverged-cluster deployment's image is also checked
# to verify that it is updated to the new operator image from
# the local registry.

# This script checks that old SSP operator resources are removed from the cluster

OLD_SSP_CRDS=(
	"kubevirttemplatevalidators.ssp.kubevirt.io"
	"kubevirtcommontemplatesbundles.ssp.kubevirt.io"
	"kubevirtnodelabellerbundles.ssp.kubevirt.io"
	"kubevirtmetricsaggregations.ssp.kubevirt.io"
)

OLD_SSP_KINDS=(
	"KubevirtTemplateValidator"
	"KubevirtCommonTemplatesBundle"
	"KubevirtNodeLabellerBundle"
	"KubevirtMetricsAggregation"
)

echo "Check that all old SSP CRDs were removed"

for CRD in "${OLD_SSP_CRDS[@]}"; do
  echo "Check that $CRD CRD was removed"
	CRD_RESPONSE=$( $CMD get customresourcedefinition $CRD 2>&1 || true)
	if ! grep -q "NotFound" <(echo "$CRD_RESPONSE"); then
		echo "expected $CRD CRD to be removed from the cluster"
		exit 1
	fi
done

echo "Check that all old SSP CRs were removed from HCO's related objects"

RELATED_OBJECTS=$( $CMD get hco $HCO_RESOURCE_NAME -n $HCO_NAMESPACE -o jsonpath="{.status.relatedObjecs}")

for CR in "${OLD_SSP_KINDS[@]}"; do
  echo "Check that $CR CR was removed from HCO's related objects"
  if grep -q $CR <(echo "$RELATED_OBJECTS"); then
    echo "expected $CR CR to be removed from HCO related objects, but found the following entry:"
    echo $RELATED_OBJECTS | jq -r ".[] | select(.kind==\"$CR\")"
    exit 1
  fi
done

