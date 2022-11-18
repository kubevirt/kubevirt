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

package tests_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/checks"

	expect "github.com/google/goexpect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/pointer"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"

	"k8s.io/apimachinery/pkg/api/resource"

	virtv1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/util"
)

const (
	// Define relevant k8s versions
	k8sSwapVer = "1.22"
	//20% of the default size which is 2G in kubevirt-ci
	maxSwapSizeToUseKib = 415948
	swapPartToUse       = 0.2
	gigbytesInkib       = 1048576
	bytesInKib          = 1024
)

var _ = Describe("[Serial][sig-compute]SwapTest", Serial, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	getMemInfoByString := func(node v1.Node, field string) (size int) {
		stdout, stderr, err := tests.ExecuteCommandOnNodeThroughVirtHandler(virtClient, node.Name, []string{"grep", field, "/proc/meminfo"})
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("stderr: %v \n", stderr))
		fields := strings.Fields(stdout)
		size, err = strconv.Atoi(fields[1])
		Expect(err).ToNot(HaveOccurred())
		return size
	}
	getAvailableMemSizeInKib := func(node v1.Node) (size int) {
		return getMemInfoByString(node, "MemAvailable")
	}

	getTotalMemSizeInKib := func(node v1.Node) (size int) {
		return getMemInfoByString(node, "MemTotal")
	}
	getSwapFreeSizeInKib := func(node v1.Node) (size int) {
		return getMemInfoByString(node, "SwapFree")
	}

	getSwapSizeInKib := func(node v1.Node) (size int) {
		return getMemInfoByString(node, "SwapTotal")
	}
	runVMIAndExpectLaunch := func(vmi *virtv1.VirtualMachineInstance, timeout int) *virtv1.VirtualMachineInstance {
		return tests.RunVMIAndExpectLaunch(vmi, timeout)
	}

	fillMemWithStressFedoraVMI := func(vmi *virtv1.VirtualMachineInstance, memToUseInTheVmKib int) error {
		_, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("stress-ng --vm-bytes %db --vm-keep -m 1 &\n", memToUseInTheVmKib*bytesInKib)},
			&expect.BExp{R: console.PromptExpression},
		}, 15)
		return err
	}

	confirmMigrationMode := func(vmi *virtv1.VirtualMachineInstance, expectedMode virtv1.MigrationMode) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("couldn't find vmi err: %v \n", err))

		By("Verifying the VMI's migration mode")
		Expect(vmi.Status.MigrationState.Mode).To(Equal(expectedMode), fmt.Sprintf("expected migration state: %v got :%v \n", vmi.Status.MigrationState.Mode, expectedMode))
	}

	skipIfSwapOff := func(message string) {
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		for _, node := range nodes.Items {
			swapSizeKib := getSwapSizeInKib(node)
			if swapSizeKib < maxSwapSizeToUseKib {
				Skip(message)
			}
		}
	}

	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		nodes := libnode.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items)).To(BeNumerically(">", 1),
			"should have at least two schedulable nodes in the cluster")

		checks.SkipIfVersionBelow("swap requires v1.22 and above", k8sSwapVer)
		skipIfSwapOff(fmt.Sprintf("swap should be enabled through env var: KUBEVIRT_SWAP_ON=true "+
			"and contain at least %dMi in the nodes when running these tests", maxSwapSizeToUseKib/bytesInKib))

	})

	Context("Migration to/from memory overcommitted nodes", func() {
		It("Postcopy Migration of vmi that is dirtying(stress-ng) more memory than the source node's memory", func() {
			nodes := libnode.GetAllSchedulableNodes(virtClient).Items
			sourceNode := nodes[0]
			targetNode := nodes[1]
			totalMemKib := getTotalMemSizeInKib(sourceNode)

			availableMemSizeKib := getAvailableMemSizeInKib(sourceNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(sourceNode)
			swapSizeKib := getSwapSizeInKib(sourceNode)
			swapSizeToUseKib := min(maxSwapSizeToUseKib, int(swapPartToUse*float64(availableSwapSizeKib)))

			Expect(availableSwapSizeKib).Should(BeNumerically(">", maxSwapSizeToUseKib), "not enough available swap space")
			//use more memory than what node can handle without swap memory
			memToUseInTheVmKib := availableMemSizeKib + swapSizeToUseKib

			//add label the node so we could schedule the vmi to it through node selector
			libnode.AddLabelToNode(sourceNode.Name, "swaptest", "swaptest")
			defer libnode.RemoveLabelFromNode(sourceNode.Name, "swaptest")

			By("Allowing post-copy")
			kv := util.GetCurrentKv(virtClient)
			oldMigrationConfiguration := kv.Spec.Configuration.MigrationConfiguration
			kv.Spec.Configuration.MigrationConfiguration = &virtv1.MigrationConfiguration{
				AllowPostCopy:           pointer.BoolPtr(true),
				CompletionTimeoutPerGiB: pointer.Int64Ptr(1),
			}
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			//The vmi should have more memory than memToUseInTheVmKib
			vmiMemSizeMi := resource.MustParse(fmt.Sprintf("%dMi", int((float64(memToUseInTheVmKib)+float64(gigbytesInkib*2))/bytesInKib)))

			vmi := tests.NewRandomFedoraVMIWithGuestAgent()
			vmi.Spec.NodeSelector = map[string]string{"swaptest": "swaptest"}
			vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true
			vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &vmiMemSizeMi}

			By("Starting the VirtualMachineInstance")
			vmi = runVMIAndExpectLaunch(vmi, 240)
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			By("Consume more memory than the node's memory")
			err = fillMemWithStressFedoraVMI(vmi, memToUseInTheVmKib)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("The workloads in the node should consume more memory than the memory size eventually.")
			Eventually(func() int {
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(sourceNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(sourceNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib))

			//add the test label to the target node
			libnode.AddLabelToNode(targetNode.Name, "swaptest", "swaptest")
			defer libnode.RemoveLabelFromNode(targetNode.Name, "swaptest")

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime*2)

			// check VMI, confirm migration state
			tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
			confirmMigrationMode(vmi, virtv1.MigrationPostCopy)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			kv = util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.MigrationConfiguration = oldMigrationConfiguration
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

		})

		It("Migration of vmi to memory overcommited node", func() {
			nodes := libnode.GetAllSchedulableNodes(virtClient).Items
			targetNode := nodes[0]
			sourceNode := nodes[1]

			vmMemoryRequestkib := 512000
			availableMemSizeKib := getAvailableMemSizeInKib(targetNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(targetNode)
			swapSizeToUsekib := min(maxSwapSizeToUseKib, int(swapPartToUse*float64(availableSwapSizeKib)))

			//make sure that the vm migrate data to swap space (leave enough space for the vm that we will migrate)
			memToUseInTargetNodeVmKib := availableMemSizeKib + swapSizeToUsekib - vmMemoryRequestkib

			//add label the node so we could schedule the vmi to it through the targert node selector
			libnode.AddLabelToNode(targetNode.Name, "swaptest", "swaptest")
			defer libnode.RemoveLabelFromNode(targetNode.Name, "swaptest")

			//The vmi should have more memory than memToUseInTheVm
			vmiMemSize := resource.MustParse(fmt.Sprintf("%dMi", int((float64(memToUseInTargetNodeVmKib)+float64(gigbytesInkib*2))/bytesInKib)))
			vmiMemReq := resource.MustParse(fmt.Sprintf("%dMi", vmMemoryRequestkib/bytesInKib))
			vmiToFillTargetNodeMem := tests.NewRandomFedoraVMIWithGuestAgent()
			vmiToFillTargetNodeMem.Spec.NodeSelector = map[string]string{"swaptest": "swaptest"}
			vmiToFillTargetNodeMem.Spec.Domain.Resources.OvercommitGuestOverhead = true
			vmiToFillTargetNodeMem.Spec.Domain.Memory = &virtv1.Memory{Guest: &vmiMemSize}
			vmiToFillTargetNodeMem.Spec.Domain.Resources.Requests["memory"] = vmiMemReq

			By("Starting the VirtualMachineInstance")
			vmiToFillTargetNodeMem = runVMIAndExpectLaunch(vmiToFillTargetNodeMem, 240)
			Expect(console.LoginToFedora(vmiToFillTargetNodeMem)).To(Succeed())

			By("reaching memory overcommitment in the target node")
			err = fillMemWithStressFedoraVMI(vmiToFillTargetNodeMem, memToUseInTargetNodeVmKib)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			vmiToMigrate := tests.NewRandomFedoraVMIWithGuestAgent()
			vmiToMigrate.Spec.NodeSelector = map[string]string{"swaptestmigrate": "swaptestmigrate"}
			vmiToMigrate.Spec.Domain.Resources.Requests["memory"] = vmiMemReq
			//add label the source node to make sure that the vm we want to migrate will be scheduled to the source node
			libnode.AddLabelToNode(sourceNode.Name, "swaptestmigrate", "swaptestmigrate")
			defer libnode.RemoveLabelFromNode(sourceNode.Name, "swaptestmigrate")

			By("Starting the VirtualMachineInstance that we should migrate to the target node")
			vmiToMigrate = runVMIAndExpectLaunch(vmiToMigrate, 240)
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())
			//add label the target node so the vm could be scheduled to it
			libnode.AddLabelToNode(targetNode.Name, "swaptestmigrate", "swaptestmigrate")
			defer libnode.RemoveLabelFromNode(targetNode.Name, "swaptestmigrate")

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmiToMigrate.Name, vmiToMigrate.Namespace)
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			By("The workloads in the node should consume more memory than the memory size eventually.")
			swapSizeKib := getSwapSizeInKib(targetNode)
			totalMemKib := getTotalMemSizeInKib(targetNode)
			Eventually(func() int {
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(targetNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(targetNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib),
				"at this point the node should has use more memory than totalMemKib "+
					"we check this to insure we migrated into memory overcommited node")
			By("Deleting the VMIs")
			Expect(virtClient.VirtualMachineInstance(vmiToFillTargetNodeMem.Namespace).Delete(vmiToFillTargetNodeMem.Name, &metav1.DeleteOptions{})).To(Succeed())
			Expect(virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Delete(vmiToMigrate.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMIs to disappear")
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmiToFillTargetNodeMem, 240)
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmiToMigrate, 240)
		})

	})
})
