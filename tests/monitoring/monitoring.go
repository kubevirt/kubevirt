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

package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
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
		checks.SkipIfPrometheusRuleIsNotEnabled(virtClient)
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
			vmi := libvmi.New(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory("2Mi"))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

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
		var originalKv *v1.KubeVirt

		BeforeEach(func() {
			originalKv = libkubevirt.GetCurrentKv(virtClient)
		})

		It("KubeVirtNoAvailableNodesToRunVMs should be triggered when there are no available nodes in the cluster to run VMs", func() {
			By("Scaling down virt-handler")
			err := disableVirtHandler(virtClient, originalKv)
			Expect(err).ToNot(HaveOccurred(), "Failed to disable virt-handler")

			By("Ensuring virt-handler is unschedulable on all nodes")
			waitForVirtHandlerNodeSelector(1, virtClient, originalKv.Namespace, "does-not-exist", 90*time.Second, 5*time.Second)

			By("Verifying virt-handler has no available pods")
			waitForVirtHandlerPodCount(1, virtClient, originalKv.Namespace, 0, 90*time.Second, 5*time.Second)

			By("Verifying KubeVirtNoAvailableNodesToRunVMs alert exists if emulation is disabled")
			libmonitoring.VerifyAlertExistWithCustomTime(virtClient, "KubeVirtNoAvailableNodesToRunVMs", 10*time.Minute)

			By("Restoring virt-handler")
			err = restoreVirtHandler(virtClient, originalKv)
			Expect(err).ToNot(HaveOccurred(), "Failed to restore virt-handler")
			libmonitoring.WaitUntilAlertDoesNotExist(virtClient, "KubeVirtNoAvailableNodesToRunVMs")
		})
	})

	Context("Deprecation Alerts", decorators.SigComputeMigrations, func() {
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

func disableVirtHandler(virtClient kubecli.KubevirtClient, originalKv *v1.KubeVirt) error {
	customizedComponents := v1.CustomizeComponents{
		Patches: []v1.CustomizeComponentsPatch{
			{
				ResourceName: virtHandler.deploymentName,
				ResourceType: "DaemonSet",
				Patch:        `{"spec":{"template":{"spec":{"nodeSelector":{"kubernetes.io/hostname":"does-not-exist"}}}}}`,
				Type:         v1.StrategicMergePatchType,
			},
		},
	}

	patchBytes, err := patch.New(patch.WithAdd("/spec/customizeComponents", customizedComponents)).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = virtClient.KubeVirt(originalKv.Namespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func restoreVirtHandler(virtClient kubecli.KubevirtClient, originalKv *v1.KubeVirt) error {
	patchBytes, err := patch.New(patch.WithRemove("/spec/customizeComponents")).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = virtClient.KubeVirt(originalKv.Namespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func waitForVirtHandlerNodeSelector(offset int, virtClient kubecli.KubevirtClient, namespace, expectedValue string, timeout, polling time.Duration) {
	EventuallyWithOffset(offset, func() (string, error) {
		vh, err := virtClient.AppsV1().DaemonSets(namespace).Get(context.Background(), virtHandler.deploymentName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return vh.Spec.Template.Spec.NodeSelector[k8sv1.LabelHostname], nil
	}).WithTimeout(timeout).WithPolling(polling).Should(Equal(expectedValue))
}

func waitForVirtHandlerPodCount(offset int, virtClient kubecli.KubevirtClient, namespace string, expectedCount int, timeout, polling time.Duration) {
	EventuallyWithOffset(offset, func() (int, error) {
		vh, err := virtClient.AppsV1().DaemonSets(namespace).Get(context.Background(), virtHandler.deploymentName, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}
		return int(vh.Status.NumberAvailable + vh.Status.NumberUnavailable), nil
	}).WithTimeout(timeout).WithPolling(polling).Should(Equal(expectedCount))
}

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
