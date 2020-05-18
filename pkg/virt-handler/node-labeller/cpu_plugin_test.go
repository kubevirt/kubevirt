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

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Node-labeller config", func() {
	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	clusterConfig, configMapInformer, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

	BeforeSuite(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.ObsoleteCPUsKey: "486, pentium, pentium2, pentium3, pentiumpro, coreduo, n270, core2duo, Conroe, athlon, phenom",
				virtconfig.MinCPUKey:       "Penryn",
			},
		})

		nlController = &NodeLabeller{
			namespace:         k8sv1.NamespaceDefault,
			clientset:         virtClient,
			configMapInformer: configMapInformer,
			clusterConfig:     clusterConfig,
		}
		os.MkdirAll(nodeLabellerVolumePath+"/cpu_map", 0777)
	})

	AfterSuite(func() {
		os.Remove(nodeLabellerVolumePath + "/cpu_map")
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
	err := writeMockDataFile(domCapabilitiesFilePath, domainCapabilities)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFileDomCapabilitiesNothingUsable() {
	err := writeMockDataFile(domCapabilitiesFilePath, domainCapabilitiesNothingUsable)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFilesFeatures() {
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
	deleteMockFile(getPathCPUFefatures("Penryn"))
	deleteMockFile(getPathCPUFefatures("Haswell"))
	deleteMockFile(getPathCPUFefatures("IvyBridge"))
	deleteMockFile("/tmp/virsh-domcapabilities.xml")
}
