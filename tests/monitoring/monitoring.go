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
 * Copyright The KubeVirt Authors.
 *
 */

package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-monitoring]Monitoring", Serial, decorators.SigMonitoring, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var prometheusRule *promv1.PrometheusRule

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Kubevirt alert rules", func() {
		BeforeEach(func() {
			monv1 := virtClient.PrometheusClient().MonitoringV1()
			prometheusRule, err = monv1.PrometheusRules(flags.KubeVirtInstallNamespace).Get(context.Background(), "prometheus-kubevirt-rules", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:8821]should have all the required annotations", func() {
			for _, group := range prometheusRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						checkRequiredAnnotations(rule)
					}
				}
			}
		})

		It("[test_id:8822]should have all the required labels", func() {
			for _, group := range prometheusRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						checkRequiredLabels(rule)
					}
				}
			}
		})
	})

	Context("Migration Alerts", decorators.SigComputeMigrations, func() {
		PIt("KubeVirtVMIExcessiveMigrations should be triggered when a VMI has been migrated more than 12 times during the last 24 hours", func() {
			By("Starting the VirtualMachineInstance")
			vmi := libvmi.New(libnet.WithMasqueradeNetworking(), libvmi.WithMemoryRequest("2Mi"))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			By("Migrating the VMI 13 times")
			for i := 0; i < 13; i++ {
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			}

			By("Verifying KubeVirtVMIExcessiveMigration alert exists")
			libmonitoring.VerifyAlertExist(virtClient, "KubeVirtVMIExcessiveMigrations")

			// delete VMI
			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})
	})

	Context("System Alerts", func() {
		It("KubeVirtNoAvailableNodesToRunVMs should be triggered when there are no available nodes in the cluster to run VMs", func() {
			By("Getting all schedulable nodes")
			nodes := libnode.GetAllSchedulableNodes(virtClient)

			DeferCleanup(func() {
				By("Restoring kubevirt.io/schedulable label to all nodes")
				for _, node := range nodes.Items {
					libnode.SetNodeSchedulable(node.Name, virtClient)
				}
			})

			By("Setting all nodes to unschedulable")
			for _, node := range nodes.Items {
				libnode.SetNodeUnschedulable(node.Name, virtClient)
			}

			By("Waiting for alert to appear")
			libmonitoring.VerifyAlertExistWithCustomTime(virtClient, "KubeVirtNoAvailableNodesToRunVMs", 10*time.Minute)
		})
	})

	Context("Deprecation Alerts", decorators.SigMonitoring, func() {
		It("KubeVirtDeprecatedAPIRequested should be triggered when a deprecated API is requested", func() {
			By("Creating a VMI with deprecated API version")
			vmi := libvmifact.NewCirros()
			vmi.APIVersion = "v1alpha3"
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, "KubeVirtDeprecatedAPIRequested")

			By("Verifying the alert disappears")
			libmonitoring.WaitUntilAlertDoesNotExistWithCustomTime(virtClient, 15*time.Minute, "KubeVirtDeprecatedAPIRequested")
		})
	})

})

func checkRequiredAnnotations(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
		"%s summary is missing or empty", rule.Alert)
	ExpectWithOffset(1, rule.Annotations).To(HaveKey("runbook_url"),
		"%s runbook_url is missing", rule.Alert)
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("runbook_url", ContainSubstring(rule.Alert)),
		"%s runbook_url doesn't include alert name", rule.Alert)

	resp, err := http.Head(rule.Annotations["runbook_url"])
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("%s runbook is not available", rule.Alert))
	ExpectWithOffset(1, resp.StatusCode).Should(Equal(http.StatusOK), fmt.Sprintf("%s runbook is not available", rule.Alert))
}

func checkRequiredLabels(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("severity", BeElementOf("info", "warning", "critical")),
		"%s severity label is missing or not valid", rule.Alert)
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("operator_health_impact", BeElementOf("none", "warning", "critical")),
		"%s operator_health_impact label is missing or not valid", rule.Alert)
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
		"%s kubernetes_operator_part_of label is missing or not valid", rule.Alert)
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "kubevirt"),
		"%s kubernetes_operator_component label is missing or not valid", rule.Alert)
}
