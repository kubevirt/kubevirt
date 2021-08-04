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
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/util"
	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"
	metric_client "kubevirt.io/kubevirt/tools/perfscale-audit/metric-client"
)

var RunPerfTests = false

func init() {
	flag.BoolVar(&RunPerfTests, "performance-test", false, "run performance tests. If false, all performance tests will be skiped.")
	if ptest, _ := strconv.ParseBool(os.Getenv("KUBEVIRT_E2E_PERF_TEST")); ptest {
		RunPerfTests = true
	}
}

func SIGDescribe(text string, body func()) bool {
	return Describe("[sig-performance][Serial] "+text, body)
}

func FSIGDescribe(text string, body func()) bool {
	return FDescribe("[sig-performance][Serial] "+text, body)
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

func perfScaleAudit(inputConfig *audit_api.InputConfig) {
	metricClient, err := metric_client.NewMetricClient(inputConfig)
	Expect(err).ToNot(HaveOccurred())

	result, err := metricClient.GenerateResults()
	Expect(err).ToNot(HaveOccurred())

	err = result.DumpToStdout()
	Expect(err).ToNot(HaveOccurred())
}
