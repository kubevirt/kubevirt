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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Node-labeller config", func() {
	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	var cm *k8sv1.ConfigMap

	BeforeSuite(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		cm = &k8sv1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt-cpu-plugin-configmap",
				Namespace: k8sv1.NamespaceDefault,
			},
			Data: map[string]string{"cpu-plugin-configmap.yaml": cpuConfig},
		}

		cmInformer, _ := testutils.NewFakeInformerFor(&k8sv1.ConfigMap{})

		nlController = &NodeLabeller{
			namespace:         k8sv1.NamespaceDefault,
			clientset:         virtClient,
			configMapInformer: cmInformer,
		}
		nlController.configMapInformer.GetStore().Add(cm)
		os.Mkdir("/tmp/cpu_map", 0777)
	})

	AfterSuite(func() {
		os.RemoveAll("/tmp/cpu_map")
	})

	BeforeEach(func() {
		domCapabilitiesFilePath = "/tmp/virsh-domcapabilities.xml"

		configPath = "/tmp/cpu-plugin-configmap.yaml"
	})

	AfterEach(func() {
		deleteFiles()
	})

	It("should return correct cpu file path", func() {
		cpuName := "Penryn"
		path := getPathCPUFefatures(cpuName)
		Expect(path).To(Equal("/var/lib/kubevirt-node-labeller/cpu_map/x86_Penryn.xml"), "cpu file path is not the same")
	})

	It("should load cpu features", func() {
		nodeLabellerVolumePath = "/tmp"
		cpuName := "Penryn"

		path := getPathCPUFefatures(cpuName)

		err := writeMockDataFile(path, cpuModelPenrynFeatures)
		Expect(err).ToNot(HaveOccurred())
		features, err := loadFeatures(cpuName)
		Expect(err).ToNot(HaveOccurred())

		for key := range features {
			if _, ok := features[key]; !ok {
				Expect(ok).To(Equal(true), "expect feature")
			}
		}

		deleteMockFile(path)
	})

	It("should return correct cpu models and features", func() {
		prepareFileDomCapabilities()
		prepareFilesFeatures()

		cpuModels, cpuFeatures, err := nlController.getCPUInfo()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(cpuModels)).To(Equal(3), "number of models must match")

		Expect(len(cpuFeatures)).To(Equal(2), "number of features must match")

		for _, feature := range newFeatures {
			if _, ok := cpuFeatures[feature]; !ok {
				Expect(ok).To(Equal(true), "feature is missing")
			}
		}
	})

	It("Domcapabilities file is not ready", func() {
		_, _, err := nlController.getCPUInfo()
		Expect(err).To(HaveOccurred(), "It doesn't throw error")
	})

	It("No cpu model is usable", func() {
		prepareFileDomCapabilitiesNothingUsable()
		prepareFilesFeatures()
		cpuModels, cpuFeatures, err := nlController.getCPUInfo()

		Expect(err).ToNot(HaveOccurred())

		Expect(len(cpuModels)).To(Equal(0), "number of models doesn't match")

		Expect(len(cpuFeatures)).To(Equal(0), "number of features doesn't match")
	})

})

func prepareFileDomCapabilities() {
	domCapabilitiesFilePath := "/tmp/virsh-domcapabilities.xml"
	err := writeMockDataFile(domCapabilitiesFilePath, domainCapabilities)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFileDomCapabilitiesNothingUsable() {
	domCapabilitiesFilePath := "/tmp/virsh-domcapabilities.xml"
	err := writeMockDataFile(domCapabilitiesFilePath, domainCapabilitiesNothingUsable)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFilesFeatures() {
	nodeLabellerVolumePath = "/tmp"
	penrynPath := getPathCPUFefatures("Penryn")
	err := writeMockDataFile(penrynPath, cpuModelPenrynFeatures)
	Expect(err).ToNot(HaveOccurred())

	ivyBridgePath := getPathCPUFefatures("IvyBridge")
	err = writeMockDataFile(ivyBridgePath, cpuModelIvyBridgeFeatures)
	Expect(err).ToNot(HaveOccurred())

	haswellPath := getPathCPUFefatures("Haswell")
	err = writeMockDataFile(haswellPath, cpuModelHaswellFeatures)
	Expect(err).ToNot(HaveOccurred())
}

func deleteFiles() {
	nodeLabellerVolumePath = "/tmp/"
	deleteMockFile(getPathCPUFefatures("Penryn"))
	deleteMockFile(getPathCPUFefatures("Haswell"))
	deleteMockFile(getPathCPUFefatures("IvyBridge"))
	deleteMockFile("/tmp/virsh-domcapabilities.xml")
}
