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
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/clientcmd"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

type alerts struct {
	deploymentDame       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
}

var (
	virtApi = alerts{
		deploymentDame:       "virt-api",
		downAlert:            "VirtAPIDown",
		restErrorsBurtsAlert: "VirtApiRESTErrorsBurst",
	}
	virtController = alerts{
		deploymentDame:       "virt-controller",
		downAlert:            "VirtControllerDown",
		noReadyAlert:         "NoReadyVirtController",
		restErrorsBurtsAlert: "VirtControllerRESTErrorsBurst",
	}
	virtHandler = alerts{
		deploymentDame:       "virt-handler",
		restErrorsBurtsAlert: "VirtHandlerRESTErrorsBurst",
	}
	virtOperator = alerts{
		deploymentDame:       "virt-operator",
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

	checkAlert := func(alertName string) error {
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
	}

	verifyAlertExist := func(alertName string) {
		Eventually(func() error {
			return checkAlert(alertName)
		}, 120*time.Second, 1*time.Second).Should(BeNil())
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

	increaseRateLimit := func() {
		rateLimitConfig := &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{
				RateLimiter: &v1.RateLimiter{
					TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
						Burst: 300,
						QPS:   300,
					},
				},
			},
		}
		originalKubeVirt := util.GetCurrentKv(virtClient)
		originalKubeVirt.Spec.Configuration.ControllerConfiguration = rateLimitConfig
		originalKubeVirt.Spec.Configuration.HandlerConfiguration = rateLimitConfig
		tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
	}

	waitForMetricValue := func(client kubecli.KubevirtClient, metric string, expectedValue int64) {
		Eventually(func() int {
			v, err := getMetricValue(client, metric)
			if err != nil {
				return -1
			}
			i, err := strconv.Atoi(v)
			Expect(err).ToNot(HaveOccurred())
			return i
		}, 3*time.Minute, 1*time.Second).Should(BeNumerically("==", expectedValue))
	}

	updatePromRules := func(newRules *promv1.PrometheusRule) {
		err = virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Delete(context.Background(), "prometheus-kubevirt-rules", metav1.DeleteOptions{})

		Expect(err).To(BeNil())

		_, err = virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Create(context.Background(), newRules, metav1.CreateOptions{})

		Expect(err).To(BeNil())
	}

	reduceAlertPendingTime := func() {
		By("Reducing alert pending time")
		promRules, err := virtClient.
			PrometheusClient().MonitoringV1().
			PrometheusRules(flags.KubeVirtInstallNamespace).
			Get(context.Background(), "prometheus-kubevirt-rules", metav1.GetOptions{})
		Expect(err).To(BeNil())

		var newRules promv1.PrometheusRule
		promRules.DeepCopyInto(&newRules)

		newRules.Annotations = nil
		newRules.ObjectMeta.ResourceVersion = ""
		newRules.ObjectMeta.UID = ""

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

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtClient).ToNot(BeNil())

		checks.SkipIfPrometheusRuleIsNotEnabled(virtClient)
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

		It("[test_id:8639]Number of disks restored and total restored bytes metric values should be correct", func() {
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

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 3)
			backupScale(virtOperator.deploymentDame)
			backupScale(virtController.deploymentDame)
			backupScale(virtApi.deploymentDame)
		})

		AfterEach(func() {
			revertScale(virtApi.deploymentDame)
			revertScale(virtController.deploymentDame)
			revertScale(virtOperator.deploymentDame)

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist("VirtOperatorDown")
			waitUntilAlertDoesNotExist("NoReadyVirtOperator")
		})

		It("VirtOperatorDown and NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			updateScale(virtOperator.deploymentDame, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-operator to zero")
			verifyAlertExist(virtOperator.downAlert)
			verifyAlertExist(virtOperator.noReadyAlert)
		})

		It("VirtControllerDown and NoReadyVirtController should be triggered when virt-controller is down", func() {
			updateScale(virtOperator.deploymentDame, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-controller to zero")
			updateScale(virtController.deploymentDame, int32(0))
			verifyAlertExist(virtController.downAlert)
			verifyAlertExist(virtController.noReadyAlert)
		})

		It("VirtApiDown should be triggered when virt-api is down", func() {
			updateScale(virtOperator.deploymentDame, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-controller to zero")
			updateScale(virtApi.deploymentDame, int32(0))
			verifyAlertExist(virtApi.downAlert)
		})
	})

	Context("Errors metrics", func() {
		var crb *rbacv1.ClusterRoleBinding

		BeforeEach(func() {
			virtClient, err = kubecli.GetKubevirtClient()
			util.PanicOnError(err)

			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperator.deploymentDame)

			crb, err = virtClient.RbacV1().ClusterRoleBindings().Get(context.Background(), "kubevirt-operator", metav1.GetOptions{})
			util.PanicOnError(err)

			reduceAlertPendingTime()
			increaseRateLimit()
		})

		AfterEach(func() {
			crb.Annotations = nil
			crb.ObjectMeta.ResourceVersion = ""
			crb.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
			if !errors.IsAlreadyExists(err) {
				util.PanicOnError(err)
			}
			revertScale(virtOperator.deploymentDame)

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist("VirtOperatorDown")
			waitUntilAlertDoesNotExist("NoReadyVirtOperator")
		})

		It("VirtApiRESTErrorsBurst should be triggered when requests to virt-api are failing", func() {
			for i := 0; i < 120; i++ {
				cmd := clientcmd.NewVirtctlCommand("vnc", "test"+rand.String(6))
				err := cmd.Execute()
				Expect(err).To(HaveOccurred())

				time.Sleep(500 * time.Millisecond)

				err = checkAlert(virtApi.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtApi.restErrorsBurtsAlert)
		})

		It("VirtOperatorRESTErrorsBurst should be triggered when requests to virt-operator are failing", func() {
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 60; i++ {
				time.Sleep(500 * time.Millisecond)

				err := checkAlert(virtOperator.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtOperator.restErrorsBurtsAlert)
		})

		It("VirtControllerRESTErrorsBurst should be triggered when requests to virt-controller are failing", func() {
			updateScale(virtOperator.deploymentDame, 0)

			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-controller", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			for i := 0; i < 60; i++ {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})

				time.Sleep(500 * time.Millisecond)

				err := checkAlert(virtController.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtController.restErrorsBurtsAlert)
		})

		It("VirtHandlerRESTErrorsBurst should be triggered when requests to virt-handler are failing", func() {
			updateScale(virtOperator.deploymentDame, 0)

			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-handler", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			for i := 0; i < 60; i++ {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})

				time.Sleep(500 * time.Millisecond)

				err := checkAlert(virtHandler.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtHandler.restErrorsBurtsAlert)
		})
	})
})
