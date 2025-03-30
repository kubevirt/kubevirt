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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"
	metric_client "kubevirt.io/kubevirt/tools/perfscale-audit/metric-client"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	instancetypeBuilder "kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var PrometheusScrapeInterval = time.Duration(30 * time.Second)

const (
	patchVMICountToPodCreateCountThreshold  = 2
	updateVMICountToPodCreateCountThreshold = 10
	vmiCreationToRunningSecondsP50Threshold = 45
	vmiCreationToRunningSecondsP95Threshold = 60
)

var _ = Describe(SIG("Control Plane Performance Density Testing", func() {
	var (
		virtClient kubecli.KubevirtClient
		startTime  time.Time
		primed     bool
	)
	artifactsDir, _ := os.LookupEnv("ARTIFACTS")
	BeforeEach(func() {
		skipIfNoPerformanceTests()
		virtClient = kubevirt.Client()

		if !primed {
			By("Create primer VMI")
			createBatchVMIWithRateControl(virtClient, 1)

			By("Waiting for primer VMI to be Running")
			waitRunningVMI(virtClient, 1, 1*time.Minute)

			// Leave a two scrape buffer between tests
			time.Sleep(2 * PrometheusScrapeInterval)

			primed = true
		}

		startTime = time.Now()
	})

	Describe("Density test", func() {
		vmCount := 100
		vmBatchStartupLimit := 5 * time.Minute

		Context(fmt.Sprintf("[small] create a batch of %d VMIs", vmCount), func() {
			It("should sucessfully create all VMIS", func() {
				By("Creating a batch of VMIs")
				createBatchVMIWithRateControl(virtClient, vmCount)

				By("Waiting a batch of VMIs")
				waitRunningVMI(virtClient, vmCount+1, vmBatchStartupLimit)
				collectMetrics(startTime, filepath.Join(artifactsDir, "VMI-perf-audit-results.json"))
			})
		})

		Context(fmt.Sprintf("[small] create a batch of %d running VMs", vmCount), func() {
			It("should sucessfully create all VMS", func() {
				By("Creating a batch of VMs")
				createBatchRunningVMWithResourcesWithRateControl(virtClient, vmCount)

				By("Waiting a batch of VMs")
				waitRunningVMI(virtClient, vmCount, vmBatchStartupLimit)
				collectMetrics(startTime, filepath.Join(artifactsDir, "VM-perf-audit-results.json"))
			})
		})

		Context(fmt.Sprintf("[small] create a batch of %d running VMs using a single instancetype and preference", vmCount), func() {
			It("should sucessfully create all VMS with instancetype and preference", func() {
				By("Creating an instancetype and preference for the test")
				instancetype := createInstancetype(virtClient)
				preference := createPreference(virtClient)

				By("Creating a batch of VMs")
				createBatchRunningVMWithInstancetypeWithRateControl(virtClient, vmCount, instancetype.Name, preference.Name)

				By("Waiting a batch of VMs")
				waitRunningVMI(virtClient, vmCount, vmBatchStartupLimit)
				collectMetrics(startTime, filepath.Join(artifactsDir, "VM-instance-type-preference-perf-audit-results.json"))
			})
		})
	})
}))

func collectMetrics(startTime time.Time, filepath string) {
	// ensure the metrics get scraped by Prometheus till the end, since the default Prometheus scrape interval is 30s
	time.Sleep(PrometheusScrapeInterval)
	endTime := time.Now()
	runAudit(startTime, endTime, filepath)

	// Leave two Prometheus scrapes of time between tests.
	time.Sleep(2 * PrometheusScrapeInterval)
}

func defineThresholds() map[audit_api.ResultType]audit_api.InputThreshold {
	thresholds := map[audit_api.ResultType]audit_api.InputThreshold{}
	thresholds[audit_api.ResultTypePatchVMICount] = audit_api.InputThreshold{
		Metric: audit_api.ResultTypeCreatePodsCount,
		Ratio:  patchVMICountToPodCreateCountThreshold,
	}

	thresholds[audit_api.ResultTypeUpdateVMICount] = audit_api.InputThreshold{
		Metric: audit_api.ResultTypeCreatePodsCount,
		Ratio:  updateVMICountToPodCreateCountThreshold,
	}

	thresholds[audit_api.ResultTypeVMICreationToRunningP50] = audit_api.InputThreshold{
		Value: vmiCreationToRunningSecondsP50Threshold,
	}

	thresholds[audit_api.ResultTypeVMICreationToRunningP95] = audit_api.InputThreshold{
		Value: vmiCreationToRunningSecondsP95Threshold,
	}
	return thresholds
}

func runAudit(startTime time.Time, endTime time.Time, outputFile string) {
	prometheusPort := 30007
	duration := audit_api.Duration(endTime.Sub(startTime))

	inputCfg := &audit_api.InputConfig{
		PrometheusURL:            fmt.Sprintf("http://127.0.0.1:%v", prometheusPort),
		StartTime:                &startTime,
		EndTime:                  &endTime,
		Duration:                 &duration,
		PrometheusScrapeInterval: PrometheusScrapeInterval,
		ThresholdExpectations:    defineThresholds(),
	}

	metricClient, err := metric_client.NewMetricClient(inputCfg)
	Expect(err).ToNot(HaveOccurred())

	result, err := metricClient.GenerateResults()
	Expect(err).ToNot(HaveOccurred())

	err = result.DumpToStdout()
	Expect(err).ToNot(HaveOccurred())

	err = result.DumpToFile(outputFile)
	Expect(err).ToNot(HaveOccurred())
}

// createBatchVMIWithRateControl creates a batch of vms concurrently, uses one goroutine for each creation.
// between creations there is an interval for throughput control
func createBatchVMIWithRateControl(virtClient kubecli.KubevirtClient, vmCount int) {
	for i := 1; i <= vmCount; i++ {
		vmi := createVMISpecWithResources()
		By(fmt.Sprintf("Creating VMI %s", vmi.ObjectMeta.Name))
		_, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// interval for throughput control
		time.Sleep(100 * time.Millisecond)
	}
}

func createBatchRunningVMWithInstancetypeWithRateControl(virtClient kubecli.KubevirtClient, vmCount int, instancetypeName, preferenceName string) {
	createBatchRunningVMWithRateControl(virtClient, vmCount, func() *kvv1.VirtualMachine {
		return libvmi.NewVirtualMachine(
			libvmifact.NewCirros(),
			libvmi.WithRunStrategy(kvv1.RunStrategyAlways),
			libvmi.WithInstancetype(instancetypeName),
			libvmi.WithPreference(preferenceName),
		)
	})
}

func createBatchRunningVMWithResourcesWithRateControl(virtClient kubecli.KubevirtClient, vmCount int) {
	createBatchRunningVMWithRateControl(virtClient, vmCount, func() *kvv1.VirtualMachine {
		return libvmi.NewVirtualMachine(createVMISpecWithResources(), libvmi.WithRunStrategy(kvv1.RunStrategyAlways))
	})
}

func createBatchRunningVMWithRateControl(virtClient kubecli.KubevirtClient, vmCount int, vmCreateFunc func() *kvv1.VirtualMachine) {
	for i := 1; i <= vmCount; i++ {
		vm := vmCreateFunc()
		By(fmt.Sprintf("Creating VM %s", vm.ObjectMeta.Name))
		_, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// interval for throughput control
		time.Sleep(100 * time.Millisecond)
	}
}

func createInstancetype(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachineInstancetype {
	instancetype := instancetypeBuilder.NewInstancetype(
		instancetypeBuilder.WithCPUs(1),
		instancetypeBuilder.WithMemory("90Mi"),
	)
	instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.NamespaceTestDefault).Create(context.Background(), instancetype, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return instancetype
}

func createPreference(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachinePreference {
	preference := instancetypeBuilder.NewPreference(
		instancetypeBuilder.WithPreferredDiskBus(kvv1.DiskBusVirtio),
	)
	preference, err := virtClient.VirtualMachinePreference(testsuite.NamespaceTestDefault).Create(context.Background(), preference, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return preference
}

func createVMISpecWithResources() *kvv1.VirtualMachineInstance {
	cpuLimit := "100m"
	memLimit := "90Mi"
	vmi := libvmifact.NewCirros(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(kvv1.DefaultPodNetwork()),
		libvmi.WithResourceMemory(memLimit),
		libvmi.WithLimitMemory(memLimit),
		libvmi.WithResourceCPU(cpuLimit),
		libvmi.WithLimitCPU(cpuLimit),
	)
	return vmi
}

func waitRunningVMI(virtClient kubecli.KubevirtClient, vmiCount int, timeout time.Duration) {
	Eventually(func() int {
		vmis, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
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
