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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

type alerts struct {
	deploymentName       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
	restErrorsHighAlert  string
	lowCountAlert        string
}

var (
	virtApi = alerts{
		deploymentName:       "virt-api",
		downAlert:            "VirtAPIDown",
		restErrorsBurtsAlert: "VirtApiRESTErrorsBurst",
		restErrorsHighAlert:  "VirtApiRESTErrorsHigh",
		lowCountAlert:        "LowVirtAPICount",
	}
	virtController = alerts{
		deploymentName:       "virt-controller",
		downAlert:            "VirtControllerDown",
		noReadyAlert:         "NoReadyVirtController",
		restErrorsBurtsAlert: "VirtControllerRESTErrorsBurst",
		restErrorsHighAlert:  "VirtControllerRESTErrorsHigh",
		lowCountAlert:        "LowVirtControllersCount",
	}
	virtHandler = alerts{
		deploymentName:       "virt-handler",
		restErrorsBurtsAlert: "VirtHandlerRESTErrorsBurst",
		restErrorsHighAlert:  "VirtHandlerRESTErrorsHigh",
	}
	virtOperator = alerts{
		deploymentName:       "virt-operator",
		downAlert:            "VirtOperatorDown",
		noReadyAlert:         "NoReadyVirtOperator",
		restErrorsBurtsAlert: "VirtOperatorRESTErrorsBurst",
		restErrorsHighAlert:  "VirtOperatorRESTErrorsHigh",
		lowCountAlert:        "LowVirtOperatorCount",
	}
)

var _ = Describe("[Serial][sig-monitoring]Monitoring", Serial, decorators.SigMonitoring, func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var scales map[string]*autoscalingv1.Scale
	var prometheusRule *promv1.PrometheusRule

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

	verifyAlertExistWithCustomTime := func(alertName string, timeout time.Duration) {
		Eventually(func() error {
			return checkAlert(alertName)
		}, timeout, 10*time.Second).Should(BeNil())
	}

	verifyAlertExist := func(alertName string) {
		verifyAlertExistWithCustomTime(alertName, 120*time.Second)
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

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 3)
			backupScale(virtOperator.deploymentName)
			backupScale(virtController.deploymentName)
			backupScale(virtApi.deploymentName)

			updateScale(virtOperator.deploymentName, int32(0))
			reduceAlertPendingTime()
		})

		AfterEach(func() {
			revertScale(virtApi.deploymentName)
			revertScale(virtController.deploymentName)
			revertScale(virtOperator.deploymentName)

			time.Sleep(10 * time.Second)
			alerts := []string{
				virtOperator.downAlert, virtOperator.noReadyAlert, virtOperator.lowCountAlert,
				virtController.downAlert, virtController.noReadyAlert, virtController.lowCountAlert,
				virtApi.downAlert, virtApi.noReadyAlert, virtApi.lowCountAlert,
			}
			for _, alert := range alerts {
				waitUntilAlertDoesNotExist(alert)
			}
		})

		It("VirtOperatorDown and NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			verifyAlertExist(virtOperator.downAlert)
			verifyAlertExist(virtOperator.noReadyAlert)
		})

		It("LowVirtOperatorCount should be triggered when virt-operator count is low", decorators.RequiresTwoSchedulableNodes, func() {
			verifyAlertExist(virtOperator.lowCountAlert)
		})

		It("VirtControllerDown and NoReadyVirtController should be triggered when virt-controller is down", func() {
			By("Scaling virt-controller to zero")
			updateScale(virtController.deploymentName, int32(0))

			verifyAlertExist(virtController.downAlert)
			verifyAlertExist(virtController.noReadyAlert)
		})

		It("LowVirtControllersCount should be triggered when virt-controller count is low", decorators.RequiresTwoSchedulableNodes, func() {
			By("Scaling virt-controller to zero")
			updateScale(virtController.deploymentName, int32(0))

			verifyAlertExist(virtController.lowCountAlert)
		})

		It("VirtApiDown should be triggered when virt-api is down", func() {
			By("Scaling virt-api to zero")
			updateScale(virtApi.deploymentName, int32(0))

			verifyAlertExist(virtApi.downAlert)
		})

		It("LowVirtAPICount should be triggered when virt-api count is low", decorators.RequiresTwoSchedulableNodes, func() {
			By("Scaling virt-api to zero")
			updateScale(virtApi.deploymentName, int32(0))

			verifyAlertExist(virtApi.lowCountAlert)
		})
	})

	Context("Errors metrics", func() {
		var crb *rbacv1.ClusterRoleBinding

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			crb, err = virtClient.RbacV1().ClusterRoleBindings().Get(context.Background(), "kubevirt-operator", metav1.GetOptions{})
			util.PanicOnError(err)

			increaseRateLimit()

			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperator.deploymentName)
			updateScale(virtOperator.deploymentName, 0)

			reduceAlertPendingTime()
		})

		AfterEach(func() {
			crb.Annotations = nil
			crb.ObjectMeta.ResourceVersion = ""
			crb.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
			if !errors.IsAlreadyExists(err) {
				util.PanicOnError(err)
			}
			revertScale(virtOperator.deploymentName)

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist(virtOperator.downAlert)
			waitUntilAlertDoesNotExist(virtApi.downAlert)
			waitUntilAlertDoesNotExist(virtController.downAlert)
			waitUntilAlertDoesNotExist(virtHandler.downAlert)
		})

		It("VirtApiRESTErrorsBurst and VirtApiRESTErrorsHigh should be triggered when requests to virt-api are failing", func() {
			randVmName := rand.String(6)

			Eventually(func(g Gomega) {
				cmd := clientcmd.NewVirtctlCommand("vnc", randVmName)
				err := cmd.Execute()
				Expect(err).To(HaveOccurred())

				g.Expect(checkAlert(virtApi.restErrorsBurtsAlert)).To(Not(HaveOccurred()))
				g.Expect(checkAlert(virtApi.restErrorsHighAlert)).To(Not(HaveOccurred()))
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtOperatorRESTErrorsBurst and VirtOperatorRESTErrorsHigh should be triggered when requests to virt-operator are failing", func() {
			revertScale(virtOperator.deploymentName)
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(checkAlert(virtOperator.restErrorsBurtsAlert)).To(Not(HaveOccurred()))
				g.Expect(checkAlert(virtOperator.restErrorsHighAlert)).To(Not(HaveOccurred()))
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtControllerRESTErrorsBurst and VirtControllerRESTErrorsHigh should be triggered when requests to virt-controller are failing", func() {
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-controller", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				g.Expect(checkAlert(virtController.restErrorsBurtsAlert)).To(Not(HaveOccurred()))
				g.Expect(checkAlert(virtController.restErrorsHighAlert)).To(Not(HaveOccurred()))
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtHandlerRESTErrorsBurst and VirtHandlerRESTErrorsHigh should be triggered when requests to virt-handler are failing", func() {
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-handler", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				g.Expect(checkAlert(virtHandler.restErrorsBurtsAlert)).To(Not(HaveOccurred()))
				g.Expect(checkAlert(virtHandler.restErrorsHighAlert)).To(Not(HaveOccurred()))
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
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
				migration = tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				// check VMI, confirm migration state
				tests.ConfirmVMIPostMigration(virtClient, vmi, migration)
			}

			By("Verifying KubeVirtVMIExcessiveMigration alert exists")
			verifyAlertExist("KubeVirtVMIExcessiveMigrations")

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
			verifyAlertExistWithCustomTime("KubeVirtNoAvailableNodesToRunVMs", 10*time.Minute)

			By("Restoring virt-handler")
			restoreVirtHandler(kv)
			waitUntilAlertDoesNotExist("KubeVirtNoAvailableNodesToRunVMs")
		})
	})

})

func checkRequiredAnnotations(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
		fmt.Sprintf("%s summary is missing or empty", rule.Alert))
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("runbook_url", Not(BeEmpty())),
		fmt.Sprintf("%s runbook_url is missing or empty", rule.Alert))

	resp, err := http.Head(rule.Annotations["runbook_url"])
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("%s runbook is not available", rule.Alert))
	ExpectWithOffset(1, resp.StatusCode).Should(Equal(http.StatusOK), fmt.Sprintf("%s runbook is not available", rule.Alert))
}

func checkRequiredLabels(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("severity", BeElementOf("info", "warning", "critical")),
		fmt.Sprintf("%s severity label is missing or not valid", rule.Alert))
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("operator_health_impact", BeElementOf("none", "warning", "critical")),
		fmt.Sprintf("%s operator_health_impact label is missing or not valid", rule.Alert))
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
		fmt.Sprintf("%s kubernetes_operator_part_of label is missing or not valid", rule.Alert))
	ExpectWithOffset(1, rule.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "kubevirt"),
		fmt.Sprintf("%s kubernetes_operator_component label is missing or not valid", rule.Alert))
}
