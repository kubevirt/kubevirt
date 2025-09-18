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
 * Copyright The KubeVirt Authors.
 *
 */

package testsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
)

var (
	KubeVirtDefaultConfig v1.KubeVirtConfiguration
	originalKV            *v1.KubeVirt
)

func AdjustKubeVirtResource() {
	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
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

	lv, err := parseVerbosityEnv()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	if lv != nil {
		kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity = lv
	}

	if kv.Spec.Configuration.DeveloperConfiguration.FeatureGates == nil {
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{}
	}

	kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{
		VirtualMachineInstanceProfile: &v1.VirtualMachineInstanceProfile{
			CustomProfile: &v1.CustomProfile{
				LocalhostProfile: pointer.P("kubevirt/kubevirt.json"),
			},
		},
	}
	// Disable CPUManager Featuregate for s390x as it is not supported.
	if translateBuildArch() != "s390x" {
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
			featuregate.CPUManager,
		)
	}
	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
		featuregate.IgnitionGate,
		featuregate.SidecarGate,
		featuregate.SnapshotGate,
		featuregate.IncrementalBackupGate,
		featuregate.HostDiskGate,
		featuregate.VirtIOFSConfigVolumesGate,
		featuregate.VirtIOFSStorageVolumeGate,
		featuregate.DownwardMetricsFeatureGate,
		featuregate.ExpandDisksGate,
		featuregate.WorkloadEncryptionSEV,
		featuregate.VMExportGate,
		featuregate.KubevirtSeccompProfile,
		featuregate.ObjectGraph,
		featuregate.DeclarativeHotplugVolumesGate,
		featuregate.NodeRestrictionGate,
		featuregate.DecentralizedLiveMigration,
	)
	kv.Spec.Configuration.ChangedBlockTrackingLabelSelectors = &v1.ChangedBlockTrackingSelectors{
		VirtualMachineLabelSelector: &metav1.LabelSelector{
			MatchLabels: cbt.CBTLabel,
		},
		NamespaceLabelSelector: &metav1.LabelSelector{
			MatchLabels: cbt.CBTLabel,
		},
	}

	storageClass, exists := libstorage.GetVMStateStorageClass()
	if exists {
		kv.Spec.Configuration.VMStateStorageClass = storageClass
	}

	data, err := json.Marshal(kv.Spec)
	Expect(err).ToNot(HaveOccurred())
	patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
	adjustedKV, err := virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.JSONPatchType, []byte(patchData), metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
	KubeVirtDefaultConfig = adjustedKV.Spec.Configuration
	if checks.HasFeature(featuregate.CPUManager) {
		// CPUManager is not enabled in the control-plane node(s)
		nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: "!node-role.kubernetes.io/control-plane"})
		Expect(err).NotTo(HaveOccurred())
		waitForSchedulableNodesWithCPUManager(len(nodes.Items))
	}
}

func waitForSchedulableNodesWithCPUManager(n int) {
	virtClient := kubevirt.Client()
	Eventually(func() bool {
		nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true," + v1.CPUManager + "=true"})
		Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
		return len(nodes.Items) == n
	}, 360, 1*time.Second).Should(BeTrue())
}

func RestoreKubeVirtResource() {
	if originalKV != nil {
		virtClient := kubevirt.Client()
		data, err := json.Marshal(originalKV.Spec)
		Expect(err).ToNot(HaveOccurred())
		patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
		_, err = virtClient.KubeVirt(originalKV.Namespace).Patch(context.Background(), originalKV.Name, types.JSONPatchType, []byte(patchData), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

// UpdateKubeVirtConfigValue updates the given configuration in the kubevirt custom resource
func UpdateKubeVirtConfigValue(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {

	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
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

	kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	return kv
}

/*
translateBuildArch translates the build_arch to arch

	case1:
	  build_arch is crossbuild-s390x, which will be translated to s390x arch
	case2:
	  build_arch is s390x, which will be translated to s390x arch
*/
func translateBuildArch() string {
	buildArch := os.Getenv("BUILD_ARCH")

	if buildArch == "" {
		return ""
	}
	archElements := strings.Split(buildArch, "-")
	if len(archElements) == 2 {
		return archElements[1]
	}
	return archElements[0]
}

func parseVerbosityEnv() (*v1.LogVerbosity, error) {
	lv := &v1.LogVerbosity{}

	env := os.Getenv("KUBEVIRT_VERBOSITY")
	if env == "" {
		return nil, nil
	}

	tokens := strings.Split(env, ",")
	for _, token := range tokens {
		kv := strings.SplitN(token, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("failed to split verbosity token %s", token)
		}
		key, value := kv[0], kv[1]
		val, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %s", value)
		}
		switch key {
		case "virtAPI":
			lv.VirtAPI = uint(val)
		case "virtController":
			lv.VirtController = uint(val)
		case "virtHandler":
			lv.VirtHandler = uint(val)
		case "virtLauncher":
			lv.VirtLauncher = uint(val)
		case "virtOperator":
			lv.VirtOperator = uint(val)
		}
	}
	return lv, nil
}
