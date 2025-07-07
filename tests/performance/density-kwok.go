/*
Copyright 2024 The KubeVirt Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	vmBatchStartupLimit = 5 * time.Minute
	defaultVMCount      = 1000
)

var _ = Describe(KWOK("Control Plane Performance Density Testing using kwok", func() {
	var (
		virtClient  kubecli.KubevirtClient
		startTime   time.Time
		primed      bool
		vmCount     = getVMCount()
		metricsPath = filepath.Join(os.Getenv("ARTIFACTS"), "VMI-kwok-perf-audit-results.json")
	)

	BeforeEach(func() {
		if !flags.DeployFakeKWOKNodesFlag {
			Skip("Skipping test as KWOK flag is not enabled")
		}

		virtClient = kubevirt.Client()

		if !primed {
			By("Create primer VMI")
			createFakeVMIBatchWithKWOK(virtClient, 1)

			By("Waiting for primer VMI to be Running")
			waitRunningVMI(virtClient, 1, 1*time.Minute)

			// Leave a two scrape buffer between tests
			time.Sleep(2 * PrometheusScrapeInterval)

			primed = true
		}

		startTime = time.Now()
	})

	AfterEach(func() {
		// Leave a two scrape buffer between tests
		time.Sleep(2 * PrometheusScrapeInterval)
	})

	Describe("kwok density tests", func() {
		Context("create a batch of fake VMIs", func() {
			It("should successfully create all fake VMIs", func() {
				By(fmt.Sprintf("creating a batch of %d fake VMIs", vmCount))
				createFakeVMIBatchWithKWOK(virtClient, vmCount)

				By("Waiting for a batch of VMIs")
				waitRunningVMI(virtClient, vmCount+1, vmBatchStartupLimit)

				By("Deleting fake VMIs")
				deleteAndVerifyFakeVMIBatch(virtClient, vmBatchStartupLimit)

				By("Collecting metrics")
				collectMetrics(startTime, metricsPath)
			})
		})

		Context("create a batch of fake VMs", func() {
			It("should successfully create all fake VMs", func() {
				By(fmt.Sprintf("creating a batch of %d fake VMs", vmCount))
				createFakeBatchRunningVMWithKWOK(virtClient, vmCount)

				By("Waiting for a batch of VMs")
				waitRunningVMI(virtClient, vmCount, vmBatchStartupLimit)

				By("Deleting fake VMs")
				deleteAndVerifyFakeVMBatch(virtClient, vmBatchStartupLimit)

				By("Collecting metrics")
				collectMetrics(startTime, metricsPath)
			})
		})
	})
}))

func createFakeVMIBatchWithKWOK(virtClient kubecli.KubevirtClient, vmCount int) {
	for i := 1; i <= vmCount; i++ {
		vmi := newFakeVMISpecWithResources()

		_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

func deleteAndVerifyFakeVMIBatch(virtClient kubecli.KubevirtClient, timeout time.Duration) {
	err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() []v1.VirtualMachineInstance {
		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		return vmis.Items
	}, timeout, 10*time.Second).Should(BeEmpty())
}

func deleteAndVerifyFakeVMBatch(virtClient kubecli.KubevirtClient, timeout time.Duration) {
	err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() []v1.VirtualMachineInstance {
		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		return vmis.Items
	}, timeout, 10*time.Second).Should(BeEmpty())
}

func createFakeBatchRunningVMWithKWOK(virtClient kubecli.KubevirtClient, vmCount int) {
	for i := 1; i <= vmCount; i++ {
		vmi := newFakeVMISpecWithResources()
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

		_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

func newFakeVMISpecWithResources() *v1.VirtualMachineInstance {
	return libvmifact.NewAlpine(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithMemoryRequest("90Mi"),
		libvmi.WithMemoryLimit("90Mi"),
		libvmi.WithCPURequest("100m"),
		libvmi.WithCPULimit("100m"),
		libvmi.WithNodeSelector("type", "kwok"),
		libvmi.WithToleration(k8sv1.Toleration{
			Key:      "CriticalAddonsOnly",
			Operator: k8sv1.TolerationOpExists,
		}),
		libvmi.WithToleration(k8sv1.Toleration{
			Key:      "kwok.x-k8s.io/node",
			Effect:   k8sv1.TaintEffectNoSchedule,
			Operator: k8sv1.TolerationOpExists,
		}),
	)
}

func getVMCount() int {
	vmCountString := os.Getenv("VM_COUNT")
	if vmCountString == "" {
		return defaultVMCount
	}

	vmCount, err := strconv.Atoi(vmCountString)
	if err != nil {
		return defaultVMCount
	}

	return vmCount
}
