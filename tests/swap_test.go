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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

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
	"kubevirt.io/kubevirt/tests/libwait"

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

var _ = Describe("[Serial][sig-compute]SwapTest", Serial, decorators.SigCompute, func() {
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
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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

	getAffinityForTargetNode := func(targetNode *v1.Node) (nodeAffinity *v1.Affinity, err error) {
		nodeAffinityRuleForVmiToFill, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(targetNode, targetNode)
		return &v1.Affinity{
			NodeAffinity: nodeAffinityRuleForVmiToFill,
		}, err
	}

	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()

		virtClient = kubevirt.Client()

		nodes := libnode.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items)).To(BeNumerically(">", 1),
			"should have at least two schedulable nodes in the cluster")

		skipIfSwapOff(fmt.Sprintf("swap should be enabled through env var: KUBEVIRT_SWAP_ON=true "+
			"and contain at least %dMi in the nodes when running these tests", maxSwapSizeToUseKib/bytesInKib))

	})

	Context("Migration to/from memory overcommitted nodes", decorators.SigComputeMigrations, func() {
		It("Postcopy Migration of vmi that is dirtying(stress-ng) more memory than the source node's memory", func() {
			sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
			Expect(err).ToNot(HaveOccurred(), "should be able to get valid source and target nodes for migartion")
			totalMemKib := getTotalMemSizeInKib(*sourceNode)
			availableMemSizeKib := getAvailableMemSizeInKib(*sourceNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(*sourceNode)
			swapSizeKib := getSwapSizeInKib(*sourceNode)
			swapSizeToUseKib := min(maxSwapSizeToUseKib, int(swapPartToUse*float64(availableSwapSizeKib)))

			Expect(availableSwapSizeKib).Should(BeNumerically(">", maxSwapSizeToUseKib), "not enough available swap space")
			//use more memory than what node can handle without swap memory
			memToUseInTheVmKib := availableMemSizeKib + swapSizeToUseKib

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

			vmi := tests.NewRandomFedoraVMI()
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmi.Spec.Affinity = &v1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
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
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(*sourceNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(*sourceNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib))

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, libmigration.MigrationWaitTime*2)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			confirmMigrationMode(vmi, virtv1.MigrationPostCopy)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			kv = util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.MigrationConfiguration = oldMigrationConfiguration
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

		})

		It("Migration of vmi to memory overcommited node", func() {
			sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
			Expect(err).ToNot(HaveOccurred(), "should be able to get valid source and target nodes for migartion")
			vmMemoryRequestkib := 512000
			availableMemSizeKib := getAvailableMemSizeInKib(*targetNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(*targetNode)
			swapSizeToUsekib := min(maxSwapSizeToUseKib, int(swapPartToUse*float64(availableSwapSizeKib)))

			//make sure that the vm migrate data to swap space (leave enough space for the vm that we will migrate)
			memToUseInTargetNodeVmKib := availableMemSizeKib + swapSizeToUsekib - vmMemoryRequestkib

			//The vmi should have more memory than memToUseInTheVm
			vmiMemSize := resource.MustParse(fmt.Sprintf("%dMi", int((float64(memToUseInTargetNodeVmKib)+float64(gigbytesInkib*2))/bytesInKib)))
			vmiMemReq := resource.MustParse(fmt.Sprintf("%dMi", vmMemoryRequestkib/bytesInKib))
			vmiToFillTargetNodeMem := tests.NewRandomFedoraVMI()
			//we want vmiToFillTargetNodeMem to land on the target node to achieve memory-overcommitment in target
			affinityRuleForVmiToFill, err := getAffinityForTargetNode(targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToFillTargetNodeMem.Spec.Affinity = affinityRuleForVmiToFill
			vmiToFillTargetNodeMem.Spec.Domain.Resources.OvercommitGuestOverhead = true
			vmiToFillTargetNodeMem.Spec.Domain.Memory = &virtv1.Memory{Guest: &vmiMemSize}
			vmiToFillTargetNodeMem.Spec.Domain.Resources.Requests["memory"] = vmiMemReq

			By("Starting the VirtualMachineInstance")
			vmiToFillTargetNodeMem = runVMIAndExpectLaunch(vmiToFillTargetNodeMem, 240)
			Expect(console.LoginToFedora(vmiToFillTargetNodeMem)).To(Succeed())

			By("reaching memory overcommitment in the target node")
			err = fillMemWithStressFedoraVMI(vmiToFillTargetNodeMem, memToUseInTargetNodeVmKib)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			vmiToMigrate := tests.NewRandomFedoraVMI()
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &v1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			vmiToMigrate.Spec.Domain.Resources.Requests["memory"] = vmiMemReq
			//add label the source node to make sure that the vm we want to migrate will be scheduled to the source node

			By("Starting the VirtualMachineInstance that we should migrate to the target node")
			vmiToMigrate = runVMIAndExpectLaunch(vmiToMigrate, 240)
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("The workloads in the node should consume more memory than the memory size eventually.")
			swapSizeKib := getSwapSizeInKib(*targetNode)
			totalMemKib := getTotalMemSizeInKib(*targetNode)
			Eventually(func() int {
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(*targetNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(*targetNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib),
				"at this point the node should has use more memory than totalMemKib "+
					"we check this to insure we migrated into memory overcommited node")
			By("Deleting the VMIs")
			Expect(virtClient.VirtualMachineInstance(vmiToFillTargetNodeMem.Namespace).Delete(context.Background(), vmiToFillTargetNodeMem.Name, &metav1.DeleteOptions{})).To(Succeed())
			Expect(virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Delete(context.Background(), vmiToMigrate.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMIs to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmiToFillTargetNodeMem, 240)
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmiToMigrate, 240)
		})

	})
})
