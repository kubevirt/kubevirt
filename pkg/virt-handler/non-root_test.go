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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virthandler

import (
	"fmt"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("FindTapDeviceIfindexPath", func() {
	var mockIsolationResult *isolation.MockIsolationResult
	var testsRootDir string

	BeforeEach(func() {
		var err error
		testsRootDir, err = os.MkdirTemp("", "ifindex-tests-")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(testsRootDir)).To(Succeed()) })

		mockIsolationResult = isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
		mockIsolationResult.EXPECT().Pid().Return(1).AnyTimes()

		testsRootDirPath, err := safepath.JoinAndResolveWithRelativeRoot(testsRootDir)
		Expect(err).ToNot(HaveOccurred())
		mockIsolationResult.EXPECT().MountRoot().Return(testsRootDirPath, nil).AnyTimes()
	})

	createFileInTestsRootDir := func(path string) *unsafepath.Path {
		testIfaceIndexAbsolutePath := fmt.Sprintf("%s/%s", testsRootDir, path)
		Expect(os.MkdirAll(testIfaceIndexAbsolutePath, 0777)).To(Succeed())

		return unsafepath.New(testsRootDir, path)
	}

	DescribeTable("should return pod interface ifindex path given",
		func(networkName, podIfaceName string) {
			testsNetworks := []v1.Network{
				podNetwork(),
				multusNetwork(networkName),
			}
			podIfaceIndexPath := fmt.Sprintf("/sys/class/net/%s/ifindex", podIfaceName)
			expectedPodIfaceIndexPath := createFileInTestsRootDir(podIfaceIndexPath)

			path, err := FindInterfaceIndexPath(mockIsolationResult, podIfaceName, networkName, testsNetworks)
			Expect(err).ToNot(HaveOccurred())
			Expect(path.Raw()).To(Equal(expectedPodIfaceIndexPath))
		},
		Entry("hashed pod interface", "red", "b1f51a511f1"),
		Entry("ordinal pod interface", "red", "net1"),
	)

	It("when path not found using hashed pod interface, fall back using ordinal pod interface name", func() {
		testsNetworks := []v1.Network{
			podNetwork(),
			multusNetwork("red"),
		}
		const existingOrdinalNamePodIfaceIndexPath = "/sys/class/net/net1/ifindex"
		expectedOrdinalNamePodIfaceIndexPath := createFileInTestsRootDir(existingOrdinalNamePodIfaceIndexPath)

		path, err := FindInterfaceIndexPath(mockIsolationResult, "b1f51a511f1", "red", testsNetworks)
		Expect(err).ToNot(HaveOccurred())
		Expect(path.Raw()).To(Equal(expectedOrdinalNamePodIfaceIndexPath))
	})

	It("should fail when path not found using hashed pod interface, and ordinal pod interface name not found", func() {
		const existingOrdinalNamePodIfaceIndexPath = "/sys/class/net/net1/ifindex"
		createFileInTestsRootDir(existingOrdinalNamePodIfaceIndexPath)

		testsNetworks := []v1.Network{
			podNetwork(),
			multusNetwork("red"),
		}

		const hashedPodIfaceName = "b1f51a511f1"
		_, err := FindInterfaceIndexPath(mockIsolationResult, hashedPodIfaceName, "wrong-network-name", testsNetworks)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to find network \"wrong-network-name\" pod interface name"))
	})

	It("should fail when ifindex not found using both hashed and ordinal pod interface names", func() {
		createFileInTestsRootDir("/sys/class/net/b1f51a511f1")

		testsNetworks := []v1.Network{
			podNetwork(),
			multusNetwork("red"),
		}

		_, err := FindInterfaceIndexPath(mockIsolationResult, "b1f51a511f1", "red", testsNetworks)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("/sys/class/net/b1f51a511f1/ifindex: no such file or directory"))
		Expect(err.Error()).To(ContainSubstring("/sys/class/net/net1: no such file or directory"))
	})
})

func multusNetwork(name string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{NetworkName: name + "vnet"},
		},
	}
}

func podNetwork() v1.Network {
	return v1.Network{
		Name: "default",
		NetworkSource: v1.NetworkSource{
			Pod: &v1.PodNetwork{},
		},
	}
}
