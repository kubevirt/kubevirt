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

	"kubevirt.io/kubevirt/tests/libmigration"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/decorators"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[Serial][sig-monitoring]Monitoring", Serial, decorators.SigMonitoring, func() {
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
		It("KubeVirtVMIExcessiveMigrations should be triggered when a VMI has been migrated more than 12 times during the last 24 hours", func() {
			By("Starting the VirtualMachineInstance")
			opts := append(libvmi.WithMasqueradeNetworking(), libvmi.WithResourceMemory("2Mi"))
			vmi := libvmi.New(opts...)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Migrating the VMI 13 times")
			for i := 0; i < 13; i++ {
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			}

			By("Verifying KubeVirtVMIExcessiveMigration alert exists")
			verifyAlertExist(virtClient, "KubeVirtVMIExcessiveMigrations")

			// delete VMI
			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})
	})

	Context("System Alerts", func() {
		disableVirtHandler := func() *v1.KubeVirt {
			originalKv := util.GetCurrentKv(virtClient)
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.CustomizeComponents = v1.CustomizeComponents{
				Patches: []v1.CustomizeComponentsPatch{
					{
						ResourceName: virtHandler.deploymentName,
						ResourceType: "DaemonSet",
						Patch:        `{"spec":{"template":{"spec":{"nodeSelector":{"kubernetes.io/hostname":"does-not-exist"}}}}}`,
						Type:         v1.StrategicMergePatchType,
					},
				},
			}

			Eventually(func() error {
				kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
				return err
			}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			Eventually(func() string {
				vh, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), virtHandler.deploymentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vh.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]
			}, 90*time.Second, 5*time.Second).Should(Equal("does-not-exist"))

			Eventually(func() int {
				vh, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), virtHandler.deploymentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return int(vh.Status.NumberAvailable + vh.Status.NumberUnavailable)
			}, 90*time.Second, 5*time.Second).Should(Equal(0))

			return kv
		}

		restoreVirtHandler := func(kv *v1.KubeVirt) {
			originalKv := util.GetCurrentKv(virtClient)
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			kv.Spec.CustomizeComponents = v1.CustomizeComponents{}

			Eventually(func() error {
				kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
				return err
			}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		It("KubeVirtNoAvailableNodesToRunVMs should be triggered when there are no available nodes in the cluster to run VMs", func() {
			By("Scaling down virt-handler")
			kv := disableVirtHandler()

			By("Verifying KubeVirtNoAvailableNodesToRunVMs alert exists if emulation is disabled")
			verifyAlertExistWithCustomTime(virtClient, "KubeVirtNoAvailableNodesToRunVMs", 10*time.Minute)

			By("Restoring virt-handler")
			restoreVirtHandler(kv)
			waitUntilAlertDoesNotExist(virtClient, "KubeVirtNoAvailableNodesToRunVMs")
		})
	})

	Context("Deprecation Alerts", decorators.SigComputeMigrations, func() {
		It("KubeVirtDeprecatedAPIRequested should be triggered when a deprecated API is requested", func() {
			By("Creating a VMI with deprecated API version")
			vmi := libvmi.NewCirros()
			vmi.APIVersion = "v1alpha3"
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the alert exists")
			verifyAlertExist(virtClient, "KubeVirtDeprecatedAPIRequested")

			By("Verifying the alert disappears")
			waitUntilAlertDoesNotExistWithCustomTime(virtClient, 15*time.Minute, "KubeVirtDeprecatedAPIRequested")
		})
	})

})

func checkRequiredAnnotations(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
		"%s summary is missing or empty", rule.Alert)
	ExpectWithOffset(1, rule.Annotations).To(HaveKey("runbook_url"),
		"%s runbook_url is missing", rule.Alert)
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("runbook_url", HaveSuffix(rule.Alert)),
		"%s runbook_url is not equal to alert name", rule.Alert)

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
