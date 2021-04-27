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
	"os"
	"path"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
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

	clusterConfig, _, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

	BeforeSuite(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		nlController = &NodeLabeller{
			namespace:     k8sv1.NamespaceDefault,
			clientset:     virtClient,
			clusterConfig: clusterConfig,
			logger:        log.DefaultLogger(),
		}

		os.MkdirAll(path.Join(nodeLabellerVolumePath, "cpu_map"), 0777)
	})

	AfterSuite(func() {
		os.Remove(path.Join(nodeLabellerVolumePath, "cpu_map"))
	})

	BeforeEach(func() {
		prepareFilesFeatures()
	})

	AfterEach(func() {
		os.Remove(path.Join(nodeLabellerVolumePath, "virsh_domcapabilities.xml"))
	})
	It("should return correct cpu file path", func() {
		fileName := "x86_Penryn.xml"
		p := getPathCPUFeatures(fileName)
		correctPath := path.Join(nodeLabellerVolumePath, "cpu_map", "x86_Penryn.xml")
		Expect(p).To(Equal(correctPath), "cpu file path is not the same")
	})

	It("should load cpu features", func() {
		fileName := "x86_Penryn.xml"
		f, err := nlController.loadFeatures(fileName)
		Expect(err).ToNot(HaveOccurred())
		for _, val := range features {
			if _, ok := f[val]; !ok {
				Expect(ok).To(Equal(false), "expect feature")
			}
		}

	})

	It("should return correct cpu models and features", func() {
		prepareFileDomCapabilities()

		err := nlController.loadHostCapabilities()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadHostSupportedFeatures()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadCPUInfo()
		Expect(err).ToNot(HaveOccurred())

		cpuModels, cpuFeatures := nlController.getCPUInfo()

		Expect(len(cpuModels)).To(Equal(3), "number of models must match")

		Expect(len(cpuFeatures)).To(Equal(2), "number of features must match")

	})

	It("No cpu model is usable", func() {
		prepareFileDomCapabilitiesNothingUsable()

		err := nlController.loadHostCapabilities()
		Expect(err).ToNot(HaveOccurred())

		err = nlController.loadCPUInfo()
		Expect(err).ToNot(HaveOccurred())

		cpuModels, cpuFeatures := nlController.getCPUInfo()

		Expect(len(cpuModels)).To(Equal(0), "number of models doesn't match")

		Expect(len(cpuFeatures)).To(Equal(2), "number of features doesn't match")
	})

})

func prepareFileDomCapabilities() {
	err := writeMockDataFile(path.Join(nodeLabellerVolumePath, "virsh_domcapabilities.xml"), domainCapabilities)
	Expect(err).ToNot(HaveOccurred())
	err = writeMockDataFile(path.Join(nodeLabellerVolumePath+"supported_features.xml"), hostSupportedFeatures)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFileDomCapabilitiesNothingUsable() {
	err := writeMockDataFile(path.Join(nodeLabellerVolumePath, "virsh_domcapabilities.xml"), domainCapabilitiesNothingUsable)
	Expect(err).ToNot(HaveOccurred())
}

func prepareFilesFeatures() {
	penrynPath := getPathCPUFeatures("x86_Penryn.xml")
	err := writeMockDataFile(penrynPath, cpuModelPenrynFeatures)
	Expect(err).ToNot(HaveOccurred())
}
