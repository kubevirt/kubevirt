/*
 * This file is part of the KubeVirt project
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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package testsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libstorage"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"
)

var (
	KubeVirtDefaultConfig v1.KubeVirtConfiguration
	originalKV            *v1.KubeVirt
)

func AdjustKubeVirtResource() {
	virtClient := kubevirt.Client()

	kv := util.GetCurrentKv(virtClient)
	originalKV = kv.DeepCopy()

	KubeVirtDefaultConfig = originalKV.Spec.Configuration

	if !flags.ApplyDefaulte2eConfiguration {
		return
	}

	// Rotate very often during the tests to ensure that things are working
	kv.Spec.CertificateRotationStrategy = v1.KubeVirtCertificateRotateStrategy{SelfSigned: &v1.KubeVirtSelfSignConfiguration{
		CA: &v1.CertConfig{
			Duration:    &metav1.Duration{Duration: 20 * time.Minute},
			RenewBefore: &metav1.Duration{Duration: 12 * time.Minute},
		},
		Server: &v1.CertConfig{
			Duration:    &metav1.Duration{Duration: 14 * time.Minute},
			RenewBefore: &metav1.Duration{Duration: 10 * time.Minute},
		},
	}}

	// match default kubevirt-config testing resource
	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
	}

	if kv.Spec.Configuration.DeveloperConfiguration.FeatureGates == nil {
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{}
	}

	kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{
		VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
			CustomProfile: &v1.CustomProfile{
				LocalhostProfile: pointer.String("kubevirt/kubevirt.json"),
			},
		},
	}
	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
		virtconfig.CPUManager,
		virtconfig.IgnitionGate,
		virtconfig.SidecarGate,
		virtconfig.SnapshotGate,
		virtconfig.HostDiskGate,
		virtconfig.VirtIOFSGate,
		virtconfig.HotplugVolumesGate,
		virtconfig.DownwardMetricsFeatureGate,
		virtconfig.NUMAFeatureGate,
		virtconfig.MacvtapGate,
		virtconfig.PasstGate,
		virtconfig.ExpandDisksGate,
		virtconfig.WorkloadEncryptionSEV,
		virtconfig.VMExportGate,
		virtconfig.KubevirtSeccompProfile,
		virtconfig.HotplugNetworkIfacesGate,
		virtconfig.VMPersistentState,
		virtconfig.VMLiveUpdateFeaturesGate,
		virtconfig.AutoResourceLimitsGate,
	)
	if flags.DisableCustomSELinuxPolicy {
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
			virtconfig.DisableCustomSELinuxPolicy,
		)
	}

	if kv.Spec.Configuration.NetworkConfiguration == nil {
		testDefaultPermitSlirpInterface := true

		kv.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{
			PermitSlirpInterface: &testDefaultPermitSlirpInterface,
		}
	}

	storageClass, exists := libstorage.GetRWXFileSystemStorageClass()
	if exists {
		kv.Spec.Configuration.VMStateStorageClass = storageClass
	}

	data, err := json.Marshal(kv.Spec)
	Expect(err).ToNot(HaveOccurred())
	patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
	adjustedKV, err := virtClient.KubeVirt(kv.Namespace).Patch(kv.Name, types.JSONPatchType, []byte(patchData), &metav1.PatchOptions{})
	util.PanicOnError(err)
	KubeVirtDefaultConfig = adjustedKV.Spec.Configuration
	nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	if checks.HasFeature(virtconfig.CPUManager) && len(nodes.Items) > 1 {
		// CPUManager is not enabled in the control-plane node
		waitForSchedulableNodeWithCPUManager()
	}
}

func waitForSchedulableNodeWithCPUManager() {

	virtClient := kubevirt.Client()
	Eventually(func() bool {
		nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true," + v1.CPUManager + "=true"})
		Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
		return len(nodes.Items) != 0
	}, 360, 1*time.Second).Should(BeTrue())
}

func RestoreKubeVirtResource() {
	if originalKV != nil {
		virtClient := kubevirt.Client()
		data, err := json.Marshal(originalKV.Spec)
		Expect(err).ToNot(HaveOccurred())
		patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
		_, err = virtClient.KubeVirt(originalKV.Namespace).Patch(originalKV.Name, types.JSONPatchType, []byte(patchData), &metav1.PatchOptions{})
		util.PanicOnError(err)
	}
}

func ShouldAllowEmulation(virtClient kubecli.KubevirtClient) bool {
	allowEmulation := false

	kv := util.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration != nil {
		allowEmulation = kv.Spec.Configuration.DeveloperConfiguration.UseEmulation
	}

	return allowEmulation
}

// UpdateKubeVirtConfigValue updates the given configuration in the kubevirt custom resource
func UpdateKubeVirtConfigValue(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {

	virtClient := kubevirt.Client()

	kv := util.GetCurrentKv(virtClient)
	old, err := json.Marshal(kv)
	Expect(err).ToNot(HaveOccurred())

	if equality.Semantic.DeepEqual(kv.Spec.Configuration, kvConfig) {
		return kv
	}

	Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "Tests which alter the global kubevirt configuration must not be executed in parallel, see https://onsi.github.io/ginkgo/#serial-specs")

	updatedKV := kv.DeepCopy()
	updatedKV.Spec.Configuration = kvConfig
	newJson, err := json.Marshal(updatedKV)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, kv)
	Expect(err).ToNot(HaveOccurred())

	kv, err = virtClient.KubeVirt(kv.Namespace).Patch(kv.GetName(), types.MergePatchType, patch, &metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	return kv
}
