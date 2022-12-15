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
	"regexp"
	"strconv"
	"strings"
	"time"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

type alerts struct {
	deploymentName       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
}

var (
	virtOperator = alerts{
		deploymentName:       "virt-operator",
		downAlert:            "VirtOperatorDown",
		noReadyAlert:         "NoReadyVirtOperator",
		restErrorsBurtsAlert: "VirtOperatorRESTErrorsBurst",
	}
)

var _ = Describe("[Serial][sig-monitoring]Prometheus Alerts", func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var scales map[string]*autoscalingv1.Scale

	backupScale := func(operatorName string) {
		Eventually(func() error {
			virtOperatorCurrentScale, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).GetScale(context.TODO(), operatorName, metav1.GetOptions{})
			if err == nil {
				scales[operatorName] = virtOperatorCurrentScale
			}
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	revertScale := func(operatorName string) {
		revert := scales[operatorName].DeepCopy()
		revert.ResourceVersion = ""
		Eventually(func() error {
			_, err = virtClient.
				AppsV1().
				Deployments(flags.KubeVirtInstallNamespace).
				UpdateScale(context.TODO(), operatorName, revert, metav1.UpdateOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	updateScale := func(operatorName string, replicas int32) {
		scale := scales[operatorName].DeepCopy()
		scale.Spec.Replicas = replicas
		Eventually(func() error {
			_, err = virtClient.
				AppsV1().
				Deployments(flags.KubeVirtInstallNamespace).
				UpdateScale(context.TODO(), operatorName, scale, metav1.UpdateOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	verifyAlertExist := func(alertName string) {
		Eventually(func() error {
			alerts, err := getAlerts(virtClient)
			if err != nil {
				return err
			}
			for _, alert := range alerts {
				if string(alert.Labels["alertname"]) == alertName {
					return nil
				}
			}
			return fmt.Errorf("alert doesn't exist: %v", alertName)
		}, 120*time.Second, 1*time.Second).Should(BeNil())

	}

	waitUntilThereIsNoAlert := func() {
		Eventually(func() error {
			alerts, err := getAlerts(virtClient)
			if err != nil {
				return err
			}
			if len(alerts) == 0 {
				return nil
			}
			return fmt.Errorf("some alerts exist: %v", alerts)
		}, 120*time.Second, 1*time.Second).Should(BeNil())

	}

	waitForMetricValueWithLabels := func(client kubecli.KubevirtClient, metric string, expectedValue int64, labels map[string]string) {
		Eventually(func() int {
			v, err := getMetricValueWithLabels(client, metric, labels)
			if err != nil {
				return -1
			}
			i, err := strconv.Atoi(v)
			Expect(err).ToNot(HaveOccurred())
			return i
		}, 3*time.Minute, 1*time.Second).Should(BeNumerically("==", expectedValue))
	}

	waitForMetricValue := func(client kubecli.KubevirtClient, metric string, expectedValue int64) {
		waitForMetricValueWithLabels(client, metric, expectedValue, nil)
	}

	updatePromRules := func(newRules *promv1.PrometheusRule) {
		err = virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Delete(context.Background(), "prometheus-kubevirt-rules", metav1.DeleteOptions{})

		Expect(err).ToNot(HaveOccurred())

		_, err = virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Create(context.Background(), newRules, metav1.CreateOptions{})

		Expect(err).ToNot(HaveOccurred())
	}

	getPrometheusAlerts := func() promv1.PrometheusRule {
		promRules, err := virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Get(context.Background(), "prometheus-kubevirt-rules", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		var newRules promv1.PrometheusRule
		promRules.DeepCopyInto(&newRules)

		newRules.Annotations = nil
		newRules.ObjectMeta.ResourceVersion = ""
		newRules.ObjectMeta.UID = ""

		return newRules
	}

	reduceAlertPendingTime := func() {
		By("Reducing alert pending time")
		newRules := getPrometheusAlerts()
		var re = regexp.MustCompile("\\[\\d+m\\]")

		var gs []promv1.RuleGroup
		for _, group := range newRules.Spec.Groups {
			var rs []promv1.Rule
			for _, rule := range group.Rules {
				var r promv1.Rule
				rule.DeepCopyInto(&r)
				if r.Alert != "" {
					r.For = "0m"
					r.Expr = intstr.FromString(re.ReplaceAllString(r.Expr.String(), `[1m]`))
					r.Expr = intstr.FromString(strings.ReplaceAll(r.Expr.String(), ">= 300", ">= 0"))
				}
				rs = append(rs, r)
			}

			gs = append(gs, promv1.RuleGroup{
				Name:  group.Name,
				Rules: rs,
			})
		}
		newRules.Spec.Groups = gs

		updatePromRules(&newRules)
	}

	waitUntilAlertDoesNotExist := func(alertName string) {
		Eventually(func() error {
			alerts, err := getAlerts(virtClient)
			if err != nil {
				return err
			}
			for _, alert := range alerts {
				if alertName == string(alert.Labels["alertname"]) {
					return fmt.Errorf("alert exist: %v", alertName)
				}
			}
			return nil
		}, 5*time.Minute, 1*time.Second).Should(BeNil())
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtClient).ToNot(BeNil())

		checks.SkipIfPrometheusRuleIsNotEnabled(virtClient)
		tests.BeforeTestCleanup()
	})

	Context("VM status metrics", func() {
		newVirtualMachine := func() *v1.VirtualMachine {
			vmi := tests.NewRandomVMI()
			return tests.NewRandomVirtualMachine(vmi, true)
		}

		createVirtualMachine := func(vm *v1.VirtualMachine) {
			By("Creating VirtualMachine")
			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		}

		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperator.deploymentName)
			updateScale(virtOperator.deploymentName, 0)

			reduceAlertPendingTime()
		})

		AfterEach(func() {
			revertScale(virtOperator.deploymentName)

			waitUntilAlertDoesNotExist("KubeVirtVMStuckInStartingState")
			waitUntilAlertDoesNotExist("KubeVirtVMStuckInErrorState")
		})

		It("KubeVirtVMStuckInStartingState should be triggered if VM is taking more than 5 minutes to start", func() {
			vm := newVirtualMachine()
			vm.Spec.Template.Spec.PriorityClassName = "non-preemtible"
			createVirtualMachine(vm)

			verifyAlertExist("KubeVirtVMStuckInStartingState")
		})

		It("KubeVirtVMStuckInErrorState should be triggered if VM is taking more than 5 minutes in Error state", func() {
			vm := newVirtualMachine()
			vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5000000Gi"),
				corev1.ResourceCPU:    resource.MustParse("5000000Gi"),
			}
			createVirtualMachine(vm)

			verifyAlertExist("KubeVirtVMStuckInErrorState")
		})
	})

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperator.deploymentName)
		})

		AfterEach(func() {
			revertScale(virtOperator.deploymentName)
			waitUntilThereIsNoAlert()
		})

		It("VirtOperatorDown should be triggered when virt-operator is down", func() {
			By("By scaling virt-operator to zero")
			updateScale(virtOperator.deploymentName, int32(0))
			verifyAlertExist("VirtOperatorDown")
		})
	})

	Context("Alerts runbooks", func() {
		It("Should have available URLs", func() {
			alerts, err := getAlerts(virtClient)
			Expect(err).ToNot(HaveOccurred())
			for _, alert := range alerts {
				Expect(alert.Annotations).ToNot(BeNil())
				url, ok := alert.Annotations["runbook_url"]
				Expect(ok).To(BeTrue())
				resp, err := http.Head(string(url))
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).Should(Equal(http.StatusOK))
			}
		})
	})

	Context("VM snapshot metrics", func() {
		quantity, _ := resource.ParseQuantity("500Mi")

		createSimplePVCWithRestoreLabels := func(name string) {
			_, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"restore.kubevirt.io/source-vm-name":      "simple-vm",
						"restore.kubevirt.io/source-vm-namespace": util.NamespaceTestDefault,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": quantity,
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("Number of disks restored and total restored bytes metric values should be correct", func() {
			totalMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_total{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			bytesMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_bytes{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			numPVCs := 2

			for i := 1; i < numPVCs+1; i++ {
				// Create dummy PVC that is labelled as "restored" from VM snapshot
				createSimplePVCWithRestoreLabels(fmt.Sprintf("vmsnapshot-restored-pvc-%d", i))
				// Metric values increases per restored disk
				waitForMetricValue(virtClient, totalMetric, int64(i))
				waitForMetricValue(virtClient, bytesMetric, quantity.Value()*int64(i))
			}
		})
	})
})
