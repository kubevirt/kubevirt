/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libkvconfig

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/util"
)

func RegisterKubevirtConfigChange(change func(c v1.KubeVirtConfiguration) ([]patch.PatchOperation, error)) error {
	kv := util.GetCurrentKv(kubevirt.Client())
	changePatch, err := change(kv.Spec.Configuration)
	if err != nil {
		return fmt.Errorf("failed changing the kubevirt configuration: %v", err)
	}

	if len(changePatch) == 0 {
		return nil
	}

	return patchKV(kv.Namespace, kv.Name, changePatch)
}

func patchKV(namespace, name string, patchOps []patch.PatchOperation) error {
	patchData, err := patch.GeneratePatchPayload(patchOps...)
	if err != nil {
		return err
	}
	_, err = kubevirt.Client().KubeVirt(namespace).Patch(context.Background(), name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	return err
}

func PatchWorkloadUpdateMethodAndRolloutStrategy(kvName string, virtClient kubecli.KubevirtClient, updateStrategy *v1.KubeVirtWorkloadUpdateStrategy, rolloutStrategy *v1.VMRolloutStrategy, fgs []string) {
	patch, err := patch.New(
		patch.WithAdd("/spec/workloadUpdateStrategy", updateStrategy),
		patch.WithAdd("/spec/configuration/vmRolloutStrategy", rolloutStrategy),
		patch.WithAdd("/spec/configuration/developerConfiguration/featureGates", fgs),
	).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() error {
		_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kvName, types.JSONPatchType, patch, metav1.PatchOptions{})
		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}
