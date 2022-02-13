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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"
	metric_client "kubevirt.io/kubevirt/tools/perfscale-audit/metric-client"

	kvv1 "kubevirt.io/api/core/v1"
	cd "kubevirt.io/kubevirt/tests/containerdisk"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

var PrometheusScrapeInterval = time.Duration(30 * time.Second)

const (
	patchVMICountToPodCreateCountThreshold  = 2
	updateVMICountToPodCreateCountThreshold = 10
	vmiCreationToRunningSecondsP50Threshold = 45
	vmiCreationToRunningSecondsP95Threshold = 60
)

var _ = SIGDescribe("Control Plane Performance Density Testing", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
		startTime  time.Time
		endTime    time.Time
		primed     bool
	)
	BeforeEach(func() {
		skipIfNoPerformanceTests()
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

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
		tests.BeforeTestCleanup()
	})

	AfterEach(func() {
		// ensure the metrics get scraped by Prometheus till the end, since the default Prometheus scrape interval is 30s
		time.Sleep(PrometheusScrapeInterval)
		endTime = time.Now()
		runAudit(startTime, endTime)

		// Leave two Prometheus scrapes of time between tests.
		time.Sleep(2 * PrometheusScrapeInterval)
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

func runAudit(startTime time.Time, endTime time.Time) {
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
}

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
