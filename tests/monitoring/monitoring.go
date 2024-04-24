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
	"regexp"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/clientcmd"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

type alerts struct {
	deploymentName       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
}

var (
	virtApi = alerts{
		deploymentName:       "virt-api",
		downAlert:            "VirtAPIDown",
		restErrorsBurtsAlert: "VirtApiRESTErrorsBurst",
	}
	virtController = alerts{
		deploymentName:       "virt-controller",
		downAlert:            "VirtControllerDown",
		noReadyAlert:         "NoReadyVirtController",
		restErrorsBurtsAlert: "VirtControllerRESTErrorsBurst",
	}
	virtHandler = alerts{
		deploymentName:       "virt-handler",
		restErrorsBurtsAlert: "VirtHandlerRESTErrorsBurst",
	}
	virtOperator = alerts{
		deploymentName:       "virt-operator",
		downAlert:            "VirtOperatorDown",
		noReadyAlert:         "NoReadyVirtOperator",
		restErrorsBurtsAlert: "VirtOperatorRESTErrorsBurst",
	}
)

var _ = Describe("[Serial][sig-monitoring]Prometheus Alerts", Serial, decorators.SigMonitoring, func() {

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

		It("[test_id:8821]should have all the requried annotations", func() {
			for _, group := range prometheusRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						checkRequiredAnnotations(rule)
					}
				}
			}
		})

		It("[test_id:8822]should have all the requried labels", func() {
			for _, group := range prometheusRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						checkRequiredLabels(rule)
					}
				}
			}
		})
	})

	Context("VM migration metrics", func() {
		var nodes *k8sv1.NodeList

		BeforeEach(func() {
			checks.SkipIfMigrationIsNotPossible()

			Eventually(func() []k8sv1.Node {
				nodes = libnode.GetAllSchedulableNodes(virtClient)
				return nodes.Items
			}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")
		})

		It("Should correctly update metrics on successful VMIM", func() {
			By("Creating VMIs")
			vmi := tests.NewRandomFedoraVMIWithGuestAgent()
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Migrating VMIs")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

			waitForMetricValue(virtClient, "kubevirt_migrate_vmi_pending_count", 0)
			waitForMetricValue(virtClient, "kubevirt_migrate_vmi_scheduling_count", 0)
			waitForMetricValue(virtClient, "kubevirt_migrate_vmi_running_count", 0)

			labels := map[string]string{
				"vmi": vmi.Name,
			}
			waitForMetricValueWithLabels(virtClient, "kubevirt_migrate_vmi_succeeded", 1, labels)

			By("Delete VMIs")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})

		It("Should correctly update metrics on failing VMIM", func() {
			By("Creating VMIs")
			vmi := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNodeAffinityFor(&nodes.Items[0]),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
			labels := map[string]string{
				"vmi": vmi.Name,
			}

			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "60"}
			migration = tests.RunMigration(virtClient, migration)

			waitForMetricValue(virtClient, "kubevirt_migrate_vmi_scheduling_count", 1)

			Eventually(matcher.ThisMigration(migration), 2*time.Minute, 5*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed), "migration creation should fail")

			waitForMetricValue(virtClient, "kubevirt_migrate_vmi_scheduling_count", 0)
			waitForMetricValueWithLabels(virtClient, "kubevirt_migrate_vmi_failed", 1, labels)

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
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

		It("should expose VM CPU metrics", func() {
			vm := newVirtualMachine()
			createVirtualMachine(vm)

			metrics := []string{
				"kubevirt_vmi_cpu_system_usage_seconds_total",
				"kubevirt_vmi_cpu_usage_seconds_total",
				"kubevirt_vmi_cpu_user_usage_seconds_total",
			}

			for _, metric := range metrics {
				Eventually(func() float64 {
					v, err := getMetricValueWithLabels(virtClient, metric, nil)
					if err != nil {
						return -1
					}

					i, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return -1
					}

					return i
				}, 3*time.Minute, 1*time.Second).Should(BeNumerically(">=", 0))
			}
		})
	})

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 3)
			backupScale(virtOperator.deploymentName)
			backupScale(virtController.deploymentName)
			backupScale(virtApi.deploymentName)
		})

		AfterEach(func() {
			revertScale(virtApi.deploymentName)
			revertScale(virtController.deploymentName)
			revertScale(virtOperator.deploymentName)

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist(virtOperator.downAlert)
			waitUntilAlertDoesNotExist(virtApi.downAlert)
			waitUntilAlertDoesNotExist(virtController.downAlert)
			waitUntilAlertDoesNotExist(virtHandler.downAlert)
		})

		It("VirtOperatorDown and NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			updateScale(virtOperator.deploymentName, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-operator to zero")
			verifyAlertExist(virtOperator.downAlert)
			verifyAlertExist(virtOperator.noReadyAlert)
		})

		It("VirtControllerDown and NoReadyVirtController should be triggered when virt-controller is down", func() {
			updateScale(virtOperator.deploymentName, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-controller to zero")
			updateScale(virtController.deploymentName, int32(0))
			verifyAlertExist(virtController.downAlert)
			verifyAlertExist(virtController.noReadyAlert)
		})

		It("VirtApiDown should be triggered when virt-api is down", func() {
			updateScale(virtOperator.deploymentName, int32(0))
			reduceAlertPendingTime()

			By("By scaling virt-controller to zero")
			updateScale(virtApi.deploymentName, int32(0))
			verifyAlertExist(virtApi.downAlert)
		})
	})

	Context("Errors metrics", func() {
		var crb *rbacv1.ClusterRoleBinding

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperator.deploymentName)

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
			revertScale(virtOperator.deploymentName)

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist(virtOperator.downAlert)
			waitUntilAlertDoesNotExist(virtApi.downAlert)
			waitUntilAlertDoesNotExist(virtController.downAlert)
			waitUntilAlertDoesNotExist(virtHandler.downAlert)
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
			updateScale(virtOperator.deploymentName, 0)

			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-controller", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			for i := 0; i < 60; i++ {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				time.Sleep(500 * time.Millisecond)

				err := checkAlert(virtController.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtController.restErrorsBurtsAlert)
		})

		It("VirtHandlerRESTErrorsBurst should be triggered when requests to virt-handler are failing", func() {
			updateScale(virtOperator.deploymentName, 0)

			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-handler", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			for i := 0; i < 60; i++ {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				time.Sleep(500 * time.Millisecond)

				err := checkAlert(virtHandler.restErrorsBurtsAlert)
				if err == nil {
					return
				}
			}

			verifyAlertExist(virtHandler.restErrorsBurtsAlert)
		})
	})

	Context("VM metrics that are based on the guest agent", func() {
		It("should have kubevirt_vmi_phase_count correctly configured with guest OS labels", func() {
			agentVMI := createAgentVMI()
			Expect(agentVMI.Status.GuestOSInfo.KernelRelease).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.Machine).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.Name).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.VersionID).ToNot(BeEmpty())

			labels := map[string]string{
				"guest_os_kernel_release": agentVMI.Status.GuestOSInfo.KernelRelease,
				"guest_os_machine":        agentVMI.Status.GuestOSInfo.Machine,
				"guest_os_name":           agentVMI.Status.GuestOSInfo.Name,
				"guest_os_version_id":     agentVMI.Status.GuestOSInfo.VersionID,
			}

			waitForMetricValueWithLabels(virtClient, "kubevirt_vmi_phase_count", 1, labels)
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
})

func checkRequiredAnnotations(rule promv1.Rule) {
	ExpectWithOffset(1, rule.Annotations).To(HaveKeyWithValue("summary", Not(BeEmpty())),
		fmt.Sprintf("%s summary is missing or empty", rule.Alert))
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

func createAgentVMI() *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()
	vmiAgentConnectedConditionMatcher := MatchFields(IgnoreExtras, Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})
	vmi := tests.RunVMIAndExpectLaunch(libvmi.NewFedora(libvmi.WithMasqueradeNetworking()...), 180)

	var err error
	var agentVMI *v1.VirtualMachineInstance

	By("VMI has the guest agent connected condition")
	Eventually(func() []v1.VirtualMachineInstanceCondition {
		agentVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return agentVMI.Status.Conditions
	}, 240*time.Second, 1*time.Second).Should(ContainElement(vmiAgentConnectedConditionMatcher), "Should have agent connected condition")

	return agentVMI
}
