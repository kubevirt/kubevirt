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

	"github.com/coreos/go-semver/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/compute"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/legacy"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate/storage"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
)

var (
	KubeVirtDefaultConfig v1.KubeVirtConfiguration
	originalKV            *v1.KubeVirt
	appliedE2EConfig      bool
)

func AdjustKubeVirtResource() {
	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
	originalKV = kv.DeepCopy()

	KubeVirtDefaultConfig = originalKV.Spec.Configuration

	if !flags.ApplyDefaulte2eConfiguration {
		return
	}

	appliedE2EConfig = true

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
			legacy.CPUManager,
		)
	}
	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates,
		legacy.IgnitionGate,
		legacy.SidecarGate,
		storage.IncrementalBackupGate,
		legacy.HostDiskGate,
		storage.VirtIOFSStorageVolumeGate,
		legacy.DownwardMetricsFeatureGate,
		legacy.WorkloadEncryptionSEV,
		storage.ObjectGraph,
		storage.DeclarativeHotplugVolumesGate,
		compute.DecentralizedLiveMigration,
		storage.UtilityVolumesGate,
		compute.RebootPolicy,
		storage.ContainerPathVolumesGate,
	)

	// ImageVolume is enabled by default for k8s 1.35+ (image volume feature gate in kubelet).
	// Disable it on older clusters to avoid CI failures.
	k8sVersion, err := checks.GetKubernetesVersion()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	if semver.New(k8sVersion).LessThan(*semver.New("1.35.0")) {
		kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates = append(
			kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates,
			legacy.ImageVolume,
		)
	}
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
}

func workerNodes(virtClient kubecli.KubevirtClient) []string {
	// Fall back to all nodes for single-node clusters (SNO / kubevirtci / kubeadm).
	nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "!node-role.kubernetes.io/control-plane",
	})
	Expect(err).ToNot(HaveOccurred())
	if len(nodes.Items) == 0 {
		nodes, err = virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
	Expect(nodes.Items).ToNot(BeEmpty(), "expected at least one worker node")

	names := make([]string, len(nodes.Items))
	for i := range nodes.Items {
		names[i] = nodes.Items[i].Name
	}
	return names
}

func WaitForWorkerNodesSchedulable() {
	virtClient := kubevirt.Client()
	workerNames := workerNodes(virtClient)

	// Baseline at wait start (after EnsureKubevirtReady). A later heartbeat means
	// schedulable + cpumanager were rewritten under suite config (same patch).
	var heartbeatTimestamps map[string]string
	if appliedE2EConfig {
		heartbeatTimestamps = make(map[string]string, len(workerNames))
		for _, name := range workerNames {
			node, err := virtClient.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			heartbeatTimestamps[name] = node.Annotations[v1.VirtHandlerHeartbeat]
		}
	}

	Eventually(func(g Gomega) {
		for _, name := range workerNames {
			node, err := virtClient.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(node.Labels[v1.NodeSchedulable]).To(Equal("true"),
				"node %s is not kubevirt.io/schedulable=true", name)
			if !appliedE2EConfig {
				continue
			}
			heartbeat := node.Annotations[v1.VirtHandlerHeartbeat]
			g.Expect(heartbeat).ToNot(BeEmpty(),
				"node %s is missing kubevirt.io/heartbeat", name)
			g.Expect(heartbeat).ToNot(Equal(heartbeatTimestamps[name]),
				"node %s has not heartbeated since suite config was applied", name)
		}
	}, 360*time.Second, time.Second).Should(Succeed(),
		"timed out waiting for worker nodes to become schedulable")
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
