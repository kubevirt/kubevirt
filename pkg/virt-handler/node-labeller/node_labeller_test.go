//go:build amd64 || s390x

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

package nodelabeller

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

const nodeName = "testNode"

var _ = Describe("Node-labeller ", func() {
	var nlController *NodeLabeller
	var kubeClient *fake.Clientset
	var supportedFeatures []string
	var cpuCounter *libvirtxml.CapsHostCPUCounter
	var guestsCaps []libvirtxml.CapsGuest
	var hostCPUModel string
	var cpuModelVendor string
	var usableModels []string
	var cpuRequiredFeatures []string
	var sevSupported bool
	var sevSupportedES bool
	var hypervFeatures []string

	BeforeEach(func() {
		cpuCounter = &libvirtxml.CapsHostCPUCounter{
			Name:      "tsc",
			Frequency: 4008012000,
			Scaling:   "no",
		}

		guestsCaps = []libvirtxml.CapsGuest{
			{
				OSType: "test",
				Arch: libvirtxml.CapsGuestArch{
					Machines: []libvirtxml.CapsGuestMachine{
						{Name: "testmachine"},
					},
				},
			},
		}

		node := newNode(nodeName)
		kubeClient = fake.NewSimpleClientset(node)
		kubevirt := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					ObsoleteCPUModels: DefaultObsoleteCPUModels,
					MinCPUModel:       "Penryn",
				},
			},
		}
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		supportedFeatures = []string{"test", "test"}
		cpuCounter = &libvirtxml.CapsHostCPUCounter{
			Name:      "tsc",
			Frequency: 4008012000,
			Scaling:   "no",
		}
		hostCPUModel = "Skylake-Client-IBRS"
		cpuModelVendor = "test"
		usableModels = []string{"Skylake-Client-IBRS", "Penryn", "Opteron_G2"}
		cpuRequiredFeatures = []string{"test"}
		sevSupported = true
		sevSupportedES = true
		hypervFeatures = []string{"test"}

		nlController = NewNodeLabeller(
			config,
			kubeClient.CoreV1().Nodes(),
			nodeName,
			recorder,
			supportedFeatures,
			cpuCounter,
			guestsCaps,
			hostCPUModel,
			cpuModelVendor,
			usableModels,
			cpuRequiredFeatures,
			sevSupported,
			sevSupportedES,
			hypervFeatures,
		)

		mockQueue := testutils.NewMockWorkQueue(nlController.queue)
		nlController.queue = mockQueue

		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node.Name)
		mockQueue.Wait()
	})

	// TODO, there is issue with empty labels
	// The node labeller can't replace/update labels if there is no label
	// This is very unlikely in real Kubernetes cluster
	It("should run node-labelling", func() {
		res := nlController.execute()
		node := retrieveNode(kubeClient)
		Expect(node.Labels).ToNot(BeEmpty())

		Expect(res).To(BeTrue(), "labeller should end with true result")
		Expect(nlController.queue.Len()).To(BeZero(), "labeller should process all nodes from queue")
	})

	It("should re-queue node if node-labelling fail", func() {
		// node labelling will fail because of the Patch
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("failed")
		})

		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Eventually(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(1), "node should be re-queued if labeller process fails")
	})

	It("should add host cpu model label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelCPULabel)))
	})

	It("should add supported machine type labels", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SupportedMachineTypeLabel + "testmachine"))
	})

	It("should add host cpu required features", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelRequiredFeaturesLabel)))
	})

	It("should add SEV label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SEVLabel))
	})

	It("should add SEVES label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SEVESLabel))
	})

	It("should add usable cpu model labels for the host cpu model", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.HostModelCPULabel+"Skylake-Client-IBRS"),
			HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
		))
	})

	It("should add usable cpu model labels if all required features are supported", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Penryn"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Penryn"),
		))
	})

	It("should not add obsolete cpu model labels", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			Not(HaveKey(v1.CPUModelLabel+"Opteron_G2")),
			Not(HaveKey(v1.SupportedHostModelMigrationCPU+"Opteron_G2")),
		))
	})

	DescribeTable("should add cpu tsc labels if tsc counter exists, its name is tsc and according to scaling value", func(scaling, result string) {
		nlController.cpuCounter.Scaling = scaling
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKeyWithValue(v1.CPUTimerLabel+"tsc-frequency", fmt.Sprintf("%d", cpuCounter.Frequency)),
			HaveKeyWithValue(v1.CPUTimerLabel+"tsc-scalable", result),
		))
	},
		Entry("scaling is set to no", "no", "false"),
		Entry("scaling is set to yes", "yes", "true"),
	)

	It("should not add cpu tsc labels if counter name isn't tsc", func() {
		nlController.cpuCounter.Name = ""
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			Not(HaveKey(v1.CPUTimerLabel+"tsc-frequency")),
			Not(HaveKey(v1.CPUTimerLabel+"tsc-scalable")),
		))

	})

	It("should remove not found cpu model and migration model", func() {
		node := retrieveNode(kubeClient)
		node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
		node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
		node, err := kubeClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node = retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
		))
		Expect(node.Labels).ToNot(SatisfyAny(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))
	})

	It("should not remove not found cpu model and migration model when skip is requested", func() {
		node := retrieveNode(kubeClient)
		node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
		node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
		// request skip
		node.Annotations[v1.LabellerSkipNodeAnnotation] = "true"

		node, err := kubeClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node = retrieveNode(kubeClient)
		Expect(node.Labels).ToNot(SatisfyAny(
			HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
		))
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))
	})

	It("should emit event if cpu model is obsolete", func() {
		nlController.clusterConfig.GetConfig().ObsoleteCPUModels["Skylake-Client-IBRS"] = true

		res := nlController.execute()
		Expect(res).To(BeTrue())

		recorder := nlController.recorder.(*record.FakeRecorder)
		Expect(recorder.Events).To(Receive(ContainSubstring("in ObsoleteCPUModels")))
	})

	It("should keep existing label that is not owned by node labeller", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		// Added in BeforeEach
		Expect(node.Labels).To(HaveKey("INeedToBeHere"))
	})

	DescribeTable("should add machine type labels", func(machines []libvirtxml.CapsGuestMachine, arch string) {
		nlController.guestCaps = []libvirtxml.CapsGuest{{
			Arch: libvirtxml.CapsGuestArch{
				Name:     arch,
				Machines: machines,
			},
		}}
		mockQueue := testutils.NewMockWorkQueue(nlController.queue)
		nlController.queue = mockQueue

		mockQueue.ExpectAdds(1)
		nlController.queue.Add(nodeName)
		mockQueue.Wait()

		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should complete successfully")

		node := retrieveNode(kubeClient)

		for _, machine := range machines {
			expectedLabelKey := v1.SupportedMachineTypeLabel + machine.Name
			Expect(node.Labels).To(HaveKey(expectedLabelKey), "expected machine type label %s to be present", expectedLabelKey)
		}
	},
		Entry("for amd64", []libvirtxml.CapsGuestMachine{{Name: "q35"}, {Name: "q35-rhel9.6.0"}}, "amd64"),
		Entry("for arm64", []libvirtxml.CapsGuestMachine{{Name: "virt"}, {Name: "virt-rhel9.6.0"}}, "arm64"),
	)

	It("should ensure that proper labels are removed on removeLabellerLabels()", func() {
		node := &k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Labels: nodeLabels,
			},
		}

		nlController.removeLabellerLabels(node)

		badKey := ""
		for key := range node.Labels {
			for _, labellerPrefix := range nodeLabellerLabels {
				if strings.HasPrefix(key, labellerPrefix) {
					badKey = key
					break
				}
			}
		}
		Expect(badKey).To(BeEmpty())
	})
})

func newNode(name string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{"INeedToBeHere": "trustme"},
			Name:        name,
		},
		Spec: k8sv1.NodeSpec{},
	}
}

func retrieveNode(kubeClient *fake.Clientset) *k8sv1.Node {
	node, err := kubeClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return node
}

var nodeLabels = map[string]string{
	"beta.kubernetes.io/arch":                                          "amd64",
	"beta.kubernetes.io/os":                                            "linux",
	"cpu-feature.node.kubevirt.io/3dnowprefetch":                       "true",
	"cpu-feature.node.kubevirt.io/abm":                                 "true",
	"cpu-feature.node.kubevirt.io/adx":                                 "true",
	"cpu-feature.node.kubevirt.io/aes":                                 "true",
	"cpu-feature.node.kubevirt.io/amd-ssbd":                            "true",
	"cpu-feature.node.kubevirt.io/amd-stibp":                           "true",
	"cpu-feature.node.kubevirt.io/arat":                                "true",
	"cpu-feature.node.kubevirt.io/arch-capabilities":                   "true",
	"cpu-feature.node.kubevirt.io/avx":                                 "true",
	"cpu-feature.node.kubevirt.io/avx2":                                "true",
	"cpu-feature.node.kubevirt.io/bmi1":                                "true",
	"cpu-feature.node.kubevirt.io/bmi2":                                "true",
	"cpu-feature.node.kubevirt.io/clflushopt":                          "true",
	"cpu-feature.node.kubevirt.io/erms":                                "true",
	"cpu-feature.node.kubevirt.io/f16c":                                "true",
	"cpu-feature.node.kubevirt.io/fma":                                 "true",
	"cpu-feature.node.kubevirt.io/fsgsbase":                            "true",
	"cpu-feature.node.kubevirt.io/hypervisor":                          "true",
	"cpu-feature.node.kubevirt.io/ibpb":                                "true",
	"cpu-feature.node.kubevirt.io/ibrs":                                "true",
	"cpu-feature.node.kubevirt.io/ibrs-all":                            "true",
	"cpu-feature.node.kubevirt.io/invpcid":                             "true",
	"cpu-feature.node.kubevirt.io/invtsc":                              "true",
	"cpu-feature.node.kubevirt.io/md-clear":                            "true",
	"cpu-feature.node.kubevirt.io/mds-no":                              "true",
	"cpu-feature.node.kubevirt.io/movbe":                               "true",
	"cpu-feature.node.kubevirt.io/mpx":                                 "true",
	"cpu-feature.node.kubevirt.io/pcid":                                "true",
	"cpu-feature.node.kubevirt.io/pclmuldq":                            "true",
	"cpu-feature.node.kubevirt.io/pdcm":                                "true",
	"cpu-feature.node.kubevirt.io/pdpe1gb":                             "true",
	"cpu-feature.node.kubevirt.io/popcnt":                              "true",
	"cpu-feature.node.kubevirt.io/pschange-mc-no":                      "true",
	"cpu-feature.node.kubevirt.io/rdctl-no":                            "true",
	"cpu-feature.node.kubevirt.io/rdrand":                              "true",
	"cpu-feature.node.kubevirt.io/rdseed":                              "true",
	"cpu-feature.node.kubevirt.io/rdtscp":                              "true",
	"cpu-feature.node.kubevirt.io/skip-l1dfl-vmentry":                  "true",
	"cpu-feature.node.kubevirt.io/smap":                                "true",
	"cpu-feature.node.kubevirt.io/smep":                                "true",
	"cpu-feature.node.kubevirt.io/spec-ctrl":                           "true",
	"cpu-feature.node.kubevirt.io/ss":                                  "true",
	"cpu-feature.node.kubevirt.io/ssbd":                                "true",
	"cpu-feature.node.kubevirt.io/sse4.2":                              "true",
	"cpu-feature.node.kubevirt.io/stibp":                               "true",
	"cpu-feature.node.kubevirt.io/tsc-deadline":                        "true",
	"cpu-feature.node.kubevirt.io/tsc_adjust":                          "true",
	"cpu-feature.node.kubevirt.io/tsx-ctrl":                            "true",
	"cpu-feature.node.kubevirt.io/umip":                                "true",
	"cpu-feature.node.kubevirt.io/vme":                                 "true",
	"cpu-feature.node.kubevirt.io/vmx":                                 "true",
	"cpu-feature.node.kubevirt.io/x2apic":                              "true",
	"cpu-feature.node.kubevirt.io/xgetbv1":                             "true",
	"cpu-feature.node.kubevirt.io/xsave":                               "true",
	"cpu-feature.node.kubevirt.io/xsavec":                              "true",
	"cpu-feature.node.kubevirt.io/xsaveopt":                            "true",
	"cpu-feature.node.kubevirt.io/xsaves":                              "true",
	"cpu-model-migration.node.kubevirt.io/Broadwell-noTSX":             "true",
	"cpu-model-migration.node.kubevirt.io/Broadwell-noTSX-IBRS":        "true",
	"cpu-model-migration.node.kubevirt.io/Haswell-noTSX":               "true",
	"cpu-model-migration.node.kubevirt.io/Haswell-noTSX-IBRS":          "true",
	"cpu-model-migration.node.kubevirt.io/IvyBridge":                   "true",
	"cpu-model-migration.node.kubevirt.io/IvyBridge-IBRS":              "true",
	"cpu-model-migration.node.kubevirt.io/Nehalem":                     "true",
	"cpu-model-migration.node.kubevirt.io/Nehalem-IBRS":                "true",
	"cpu-model-migration.node.kubevirt.io/Opteron_G1":                  "true",
	"cpu-model-migration.node.kubevirt.io/Opteron_G2":                  "true",
	"cpu-model-migration.node.kubevirt.io/Penryn":                      "true",
	"cpu-model-migration.node.kubevirt.io/SandyBridge":                 "true",
	"cpu-model-migration.node.kubevirt.io/SandyBridge-IBRS":            "true",
	"cpu-model-migration.node.kubevirt.io/Skylake-Client-IBRS":         "true",
	"cpu-model-migration.node.kubevirt.io/Skylake-Client-noTSX-IBRS":   "true",
	"cpu-model-migration.node.kubevirt.io/Westmere":                    "true",
	"cpu-model-migration.node.kubevirt.io/Westmere-IBRS":               "true",
	"cpu-model.node.kubevirt.io/Broadwell-noTSX":                       "true",
	"cpu-model.node.kubevirt.io/Broadwell-noTSX-IBRS":                  "true",
	"cpu-model.node.kubevirt.io/Haswell-noTSX":                         "true",
	"cpu-model.node.kubevirt.io/Haswell-noTSX-IBRS":                    "true",
	"cpu-model.node.kubevirt.io/IvyBridge":                             "true",
	"cpu-model.node.kubevirt.io/IvyBridge-IBRS":                        "true",
	"cpu-model.node.kubevirt.io/Nehalem":                               "true",
	"cpu-model.node.kubevirt.io/Nehalem-IBRS":                          "true",
	"cpu-model.node.kubevirt.io/Opteron_G1":                            "true",
	"cpu-model.node.kubevirt.io/Opteron_G2":                            "true",
	"cpu-model.node.kubevirt.io/Penryn":                                "true",
	"cpu-model.node.kubevirt.io/SandyBridge":                           "true",
	"cpu-model.node.kubevirt.io/SandyBridge-IBRS":                      "true",
	"cpu-model.node.kubevirt.io/Skylake-Client-noTSX-IBRS":             "true",
	"cpu-model.node.kubevirt.io/Westmere":                              "true",
	"cpu-model.node.kubevirt.io/Westmere-IBRS":                         "true",
	"cpu-timer.node.kubevirt.io/tsc-frequency":                         "2111998000",
	"cpu-timer.node.kubevirt.io/tsc-scalable":                          "false",
	"cpu-vendor.node.kubevirt.io/Intel":                                "true",
	"cpumanager":                                                       "false",
	"host-model-cpu.node.kubevirt.io/Skylake-Client-IBRS":              "true",
	"host-model-required-features.node.kubevirt.io/amd-ssbd":           "true",
	"host-model-required-features.node.kubevirt.io/amd-stibp":          "true",
	"host-model-required-features.node.kubevirt.io/arch-capabilities":  "true",
	"host-model-required-features.node.kubevirt.io/clflushopt":         "true",
	"host-model-required-features.node.kubevirt.io/hypervisor":         "true",
	"host-model-required-features.node.kubevirt.io/ibpb":               "true",
	"host-model-required-features.node.kubevirt.io/ibrs":               "true",
	"host-model-required-features.node.kubevirt.io/ibrs-all":           "true",
	"host-model-required-features.node.kubevirt.io/invtsc":             "true",
	"host-model-required-features.node.kubevirt.io/md-clear":           "true",
	"host-model-required-features.node.kubevirt.io/mds-no":             "true",
	"host-model-required-features.node.kubevirt.io/pdcm":               "true",
	"host-model-required-features.node.kubevirt.io/pdpe1gb":            "true",
	"host-model-required-features.node.kubevirt.io/pschange-mc-no":     "true",
	"host-model-required-features.node.kubevirt.io/rdctl-no":           "true",
	"host-model-required-features.node.kubevirt.io/skip-l1dfl-vmentry": "true",
	"host-model-required-features.node.kubevirt.io/ss":                 "true",
	"host-model-required-features.node.kubevirt.io/ssbd":               "true",
	"host-model-required-features.node.kubevirt.io/stibp":              "true",
	"host-model-required-features.node.kubevirt.io/tsc_adjust":         "true",
	"host-model-required-features.node.kubevirt.io/tsx-ctrl":           "true",
	"host-model-required-features.node.kubevirt.io/umip":               "true",
	"host-model-required-features.node.kubevirt.io/vmx":                "true",
	"host-model-required-features.node.kubevirt.io/xsaves":             "true",
	"hyperv.node.kubevirt.io/base":                                     "true",
	"hyperv.node.kubevirt.io/frequencies":                              "true",
	"hyperv.node.kubevirt.io/ipi":                                      "true",
	"hyperv.node.kubevirt.io/reenlightenment":                          "true",
	"hyperv.node.kubevirt.io/reset":                                    "true",
	"hyperv.node.kubevirt.io/runtime":                                  "true",
	"hyperv.node.kubevirt.io/synic":                                    "true",
	"hyperv.node.kubevirt.io/synic2":                                   "true",
	"hyperv.node.kubevirt.io/synictimer":                               "true",
	"hyperv.node.kubevirt.io/time":                                     "true",
	"hyperv.node.kubevirt.io/tlbflush":                                 "true",
	"hyperv.node.kubevirt.io/vpindex":                                  "true",
	"kubernetes.io/arch":                                               "amd64",
	k8sv1.LabelHostname:                                                "node01",
	"kubernetes.io/os":                                                 "linux",
	"kubevirt.io/schedulable":                                          "true",
	"node-role.kubernetes.io/control-plane":                            "",
	"node-role.kubernetes.io/master":                                   "",
	"node.kubernetes.io/exclude-from-external-load-balancers":          "",
	"scheduling.node.kubevirt.io/tsc-frequency-2111998000":             "true",
}
