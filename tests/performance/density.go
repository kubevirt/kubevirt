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

package performance

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvv1 "kubevirt.io/api/core/v1"
	cd "kubevirt.io/kubevirt/tests/containerdisk"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("Control Plane Performance Density Testing", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		skipIfNoPerformanceTests()
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		tests.BeforeTestCleanup()
	})

	AfterEach(func() {
		// ensure the metrics get scraped by Prometheus till the end, since the default Prometheus scrape interval is 30s
		time.Sleep(30 * time.Second)
	})

	Describe("Density test", func() {
		vmCount := 100
		vmBatchStartupLimit := 5 * time.Minute

		Context(fmt.Sprintf("[small] create a batch of %d VMIs", vmCount), func() {
			It("should sucessfully create all VMIS", func() {
				By("Creating a batch of VMIs")
				createBatchVMIWithRateControl(virtClient, vmCount)

				By("Waiting a batch of VMIs")
				waitRunningVMI(virtClient, vmCount, vmBatchStartupLimit)
			})
		})
	})
})

// createBatchVMIWithRateControl creates a batch of vms concurrently, uses one goroutine for each creation.
// between creations there is an interval for throughput control
func createBatchVMIWithRateControl(virtClient kubecli.KubevirtClient, vmCount int) {
	for i := 1; i <= vmCount; i++ {
		vmi := createVMISpecWithResources(virtClient)
		By(fmt.Sprintf("Creating VMI %s", vmi.ObjectMeta.Name))
		_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		// interval for throughput control
		time.Sleep(100 * time.Millisecond)
	}
}

func createVMISpecWithResources(virtClient kubecli.KubevirtClient) *kvv1.VirtualMachineInstance {
	vmImage := cd.ContainerDiskFor("cirros")
	cpuLimit := "100m"
	memLimit := "90Mi"
	cloudInitUserData := "#!/bin/bash\necho 'hello'\n"
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmImage, cloudInitUserData)
	vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(memLimit),
		k8sv1.ResourceCPU:    resource.MustParse(cpuLimit),
	}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(memLimit),
		k8sv1.ResourceCPU:    resource.MustParse(cpuLimit),
	}
	return vmi
}

func waitRunningVMI(virtClient kubecli.KubevirtClient, vmiCount int, timeout time.Duration) {
	Eventually(func() int {
		vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		running := 0
		for _, vmi := range vmis.Items {
			if vmi.Status.Phase == kvv1.Running {
				running++
			}
		}
		return running
	}, timeout, 10*time.Second).Should(Equal(vmiCount))
}
