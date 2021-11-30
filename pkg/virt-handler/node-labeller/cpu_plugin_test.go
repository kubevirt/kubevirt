// +build amd64

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

	"kubevirt.io/kubevirt/tests"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
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
	var virtClient *kubecli.MockKubevirtClient

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

	BeforeSuite(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

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
				Expect(ok).To(Equal(false), "expect feature")
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

		cpuModels := nlController.getSupportedCpuModels()
		cpuFeatures := nlController.getSupportedCpuFeatures()

		Expect(len(cpuModels)).To(Equal(3), "number of models must match")

		Expect(len(cpuFeatures)).To(Equal(2), "number of features must match")
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

		cpuModels := nlController.getSupportedCpuModels()
		cpuFeatures := nlController.getSupportedCpuFeatures()

		Expect(len(cpuModels)).To(Equal(0), "number of models doesn't match")

		Expect(len(cpuFeatures)).To(Equal(2), "number of features doesn't match")
	})

	Context("should return correct host cpu", func() {
		var hostCpuModel hostCPUModel

		tests.BeforeAll(func() {
			err := nlController.loadHostSupportedFeatures()
			Expect(err).ToNot(HaveOccurred())

			hostCpuModel = nlController.getHostCpuModel()
		})

		It("model", func() {
			Expect(hostCpuModel.name).To(Equal("Skylake-Client-IBRS"))
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

})
