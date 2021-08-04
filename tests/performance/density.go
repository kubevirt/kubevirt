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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/util"
	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"
	"kubevirt.io/kubevirt/tools/perfscale-audit/thresholds"
)

var _ = SIGDescribe("Control Plane Performance Density Testing", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient

		PrometheusPort       = "9090"
		PrometheusEndpoint   = fmt.Sprintf("http://127.0.0.1:%s", PrometheusPort)
		thresholdExpecations = map[audit_api.ResultType]audit_api.InputThreshold{
			"vmiCreationToRunningSecondsP99": audit_api.InputThreshold{
				Value: thresholds.VMI_CREATE_TO_RUNNING_SECONDS_99,
			},
			"vmiCreationToRunningSecondsP95": audit_api.InputThreshold{
				Value: thresholds.VMI_CREATE_TO_RUNNING_SECONDS_90,
			},
			"vmiCreationToRunningSecondsP50": audit_api.InputThreshold{
				Value: thresholds.VMI_CREATE_TO_RUNNING_SECONDS_50,
			},
		}
		auditInput = &audit_api.InputConfig{
			PrometheusURL:         PrometheusEndpoint,
			ThresholdExpectations: thresholdExpecations,
		}
	)

	BeforeEach(func() {
		if !RunPerfTests {
			Skip("Performance tests are not enabled.")
		}
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		tests.BeforeTestCleanup()
		start := time.Now()
		auditInput.StartTime = &start
	})

	AfterEach(func() {
		end := time.Now()
		auditInput.EndTime = &end
		// ensure the metrics get scraped by Prometheus till the end, since the default Prometheus scrape interval is 30s
		time.Sleep(30 * time.Second)

		By("Getting Prometheus server")
		pods, err := virtClient.CoreV1().Pods(flags.MonitoringNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=prometheus"})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())

		stopChan := make(chan struct{})
		err = tests.ForwardPorts(&pods.Items[0], []string{"9090:PrometheusPort"}, stopChan, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())

		perfScaleAudit(auditInput)
		close(stopChan)
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
