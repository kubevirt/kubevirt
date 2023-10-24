//go:build amd64

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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"path"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/testutils"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var features = []string{"apic", "clflush", "cmov"}

const (
	x86PenrynXml = "x86_Penryn.xml"
)

var _ = Describe("Node-labeller config", func() {
	var nlController *NodeLabeller

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)

		kv := &kubevirtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: kubevirtv1.KubeVirtSpec{
				Configuration: kubevirtv1.KubeVirtConfiguration{
					ObsoleteCPUModels: util.DefaultObsoleteCPUModels,
					MinCPUModel:       util.DefaultMinCPUModel,
				},
			},
		}

		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

		nlController = &NodeLabeller{
			namespace:               k8sv1.NamespaceDefault,
			clientset:               virtClient,
			clusterConfig:           clusterConfig,
			logger:                  log.DefaultLogger(),
			volumePath:              "testdata",
			domCapabilitiesFileName: "virsh_domcapabilities.xml",
			hostCPUModel:            hostCPUModel{requiredFeatures: make(map[string]bool, 0)},
		}
	})

	It("should return correct cpu file path", func() {
		p := getPathCPUFeatures(nlController.volumePath, x86PenrynXml)
		correctPath := path.Join(nlController.volumePath, "cpu_map", x86PenrynXml)
		Expect(p).To(Equal(correctPath), "cpu file path is not the same")
	})

	It("should load cpu features", func() {
		fileName := x86PenrynXml
		f, err := nlController.loadFeatures(fileName)
		Expect(err).ToNot(HaveOccurred())
		for _, val := range features {
			if _, ok := f[val]; !ok {
				Expect(ok).To(BeFalse(), "expect feature")
			}
		}

	})

	It("should return correct cpu models, features and tsc freqnency", func() {
		err := nlController.loadDomCapabilities()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadHostSupportedFeatures()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadCPUInfo()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadHostCapabilities()
		Expect(err).ToNot(HaveOccurred())

		cpuModels := nlController.getSupportedCpuModels(nlController.clusterConfig.GetObsoleteCPUModels())
		cpuFeatures := nlController.getSupportedCpuFeatures()

		Expect(cpuModels).To(HaveLen(5), "number of models must match")

		Expect(cpuFeatures).To(HaveLen(4), "number of features must match")
		counter, err := nlController.capabilities.GetTSCCounter()
		Expect(err).ToNot(HaveOccurred())
		Expect(counter).ToNot(BeNil())
		Expect(counter.Frequency).To(BeNumerically("==", 4008012000))

	})

	It("No cpu model is usable", func() {
		nlController.domCapabilitiesFileName = "virsh_domcapabilities_nothing_usable.xml"
		err := nlController.loadDomCapabilities()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadCPUInfo()
		Expect(err).ToNot(HaveOccurred())

		Expect(nlController.loadHostSupportedFeatures()).To(Succeed())

		cpuModels := nlController.getSupportedCpuModels(nlController.clusterConfig.GetObsoleteCPUModels())
		cpuFeatures := nlController.getSupportedCpuFeatures()

		Expect(cpuModels).To(BeEmpty(), "no CPU models are expected to be supported")

		Expect(cpuFeatures).To(HaveLen(4), "number of features doesn't match")
	})

	Context("should return correct host cpu", func() {
		var hostCpuModel hostCPUModel

		BeforeEach(func() {
			nlController.domCapabilitiesFileName = "virsh_domcapabilities.xml"
			err := nlController.loadDomCapabilities()
			Expect(err).ToNot(HaveOccurred())

			err = nlController.loadHostSupportedFeatures()
			Expect(err).ToNot(HaveOccurred())

			hostCpuModel = nlController.GetHostCpuModel()
		})

		It("model", func() {
			Expect(hostCpuModel.Name).To(Equal("Skylake-Client-IBRS"))
			Expect(hostCpuModel.fallback).To(Equal("allow"))
		})

		It("required features", func() {
			features := hostCpuModel.requiredFeatures
			Expect(features).To(HaveLen(3))
			Expect(features).Should(And(
				HaveKey("ds"),
				HaveKey("acpi"),
				HaveKey("ss"),
			))
		})
	})

	Context("return correct SEV capabilities", func() {
		DescribeTable("for SEV and SEV-ES",
			func(isSupported bool, withES bool) {
				if isSupported && withES {
					nlController.domCapabilitiesFileName = "domcapabilities_sev.xml"
				} else if isSupported {
					nlController.domCapabilitiesFileName = "domcapabilities_noseves.xml"
				} else {
					nlController.domCapabilitiesFileName = "domcapabilities_nosev.xml"
				}
				err := nlController.loadDomCapabilities()
				Expect(err).ToNot(HaveOccurred())

				if isSupported {
					Expect(nlController.SEV.Supported).To(Equal("yes"))
					Expect(nlController.SEV.CBitPos).To(Equal(uint(47)))
					Expect(nlController.SEV.ReducedPhysBits).To(Equal(uint(1)))
					Expect(nlController.SEV.MaxGuests).To(Equal(uint(15)))

					if withES {
						Expect(nlController.SEV.SupportedES).To(Equal("yes"))
						Expect(nlController.SEV.MaxESGuests).To(Equal(uint(15)))
					} else {
						Expect(nlController.SEV.SupportedES).To(Equal("no"))
						Expect(nlController.SEV.MaxESGuests).To(BeZero())
					}
				} else {
					Expect(nlController.SEV.Supported).To(Equal("no"))
					Expect(nlController.SEV.CBitPos).To(BeZero())
					Expect(nlController.SEV.ReducedPhysBits).To(BeZero())
					Expect(nlController.SEV.MaxGuests).To(BeZero())
					Expect(nlController.SEV.SupportedES).To(Equal("no"))
					Expect(nlController.SEV.MaxESGuests).To(BeZero())
				}
			},
			Entry("when only SEV is supported", true, false),
			Entry("when both SEV and SEV-ES are supported", true, true),
			Entry("when neither SEV nor SEV-ES are supported", false, false),
		)
	})

	It("Make sure proper labels are removed on removeLabellerLabels()", func() {
		node := &k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Labels: nodeLabels,
			},
		}

		nlController.removeLabellerLabels(node)

		badKey := ""
		for key, _ := range node.Labels {
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
	"kubernetes.io/hostname":                                           "node01",
	"kubernetes.io/os":                                                 "linux",
	"kubevirt.io/schedulable":                                          "true",
	"node-role.kubernetes.io/control-plane":                            "",
	"node-role.kubernetes.io/master":                                   "",
	"node.kubernetes.io/exclude-from-external-load-balancers":          "",
	"scheduling.node.kubevirt.io/tsc-frequency-2111998000":             "true",
}
