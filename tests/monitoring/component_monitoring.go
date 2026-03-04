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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	randNodeSelectorSuffixLength   = 8
	lowReadySingletonWorkloadCount = 1
	lowReadyDeploymentRunningCount = 2
)

type alerts struct {
	deploymentName       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
	lowReadyAlert        string
}

var (
	virtAPI = alerts{
		deploymentName:       "virt-api",
		downAlert:            "VirtAPIDown",
		noReadyAlert:         "NoReadyVirtAPI",
		restErrorsBurtsAlert: "VirtApiRESTErrorsBurst",
		lowReadyAlert:        "LowReadyVirtAPICount",
	}
	virtController = alerts{
		deploymentName:       "virt-controller",
		downAlert:            "VirtControllerDown",
		noReadyAlert:         "NoReadyVirtController",
		restErrorsBurtsAlert: "VirtControllerRESTErrorsBurst",
		lowReadyAlert:        "LowReadyVirtControllersCount",
	}
	virtHandler = alerts{
		deploymentName:       "virt-handler",
		downAlert:            "VirtHandlerDown",
		noReadyAlert:         "NoReadyVirtHandler",
		restErrorsBurtsAlert: "VirtHandlerRESTErrorsBurst",
		lowReadyAlert:        "LowReadyVirtHandlerCount",
	}
	virtOperator = alerts{
		deploymentName:       "virt-operator",
		downAlert:            "VirtOperatorDown",
		noReadyAlert:         "NoReadyVirtOperator",
		restErrorsBurtsAlert: "VirtOperatorRESTErrorsBurst",
		lowReadyAlert:        "LowReadyVirtOperatorsCount",
	}
)

var _ = Describe("[sig-monitoring]Component Monitoring", Serial, Ordered, decorators.SigMonitoring, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var scales *libmonitoring.Scaling

	BeforeAll(func() {
		virtClient = kubevirt.Client()
		scales = libmonitoring.NewScaling(virtClient, []string{
			virtOperator.deploymentName,
			virtController.deploymentName,
			virtAPI.deploymentName,
		})
	})

	BeforeEach(func() {
		By("Increasing the rate limit")
		increaseRateLimit(virtClient)

		By("Scaling down the operator to prevent reconciliation")
		scales.UpdateScale(virtOperator.deploymentName, int32(0))

		By("Reducing the alert pending time")
		libmonitoring.ReduceAlertPendingTime(virtClient)
	})

	AfterEach(func() {
		By("Restoring the controller and api scales")
		scales.RestoreScale(virtController.deploymentName)
		scales.RestoreScale(virtAPI.deploymentName)

		By("Restoring the operator")
		restoreOperator(virtClient, scales)
	})

	Context("Up metrics", func() {
		It("VirtOperatorDown should be triggered when virt-operator is down", func() {
			By("Waiting for no running virt-operator pods")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_operator_pods_running:count", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtOperator.downAlert)
		})

		It("NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			By("Waiting for the operator to be down")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_operator_ready:sum", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtOperator.noReadyAlert)
		})

		It("VirtControllerDown should be triggered when virt-controller is down", func() {
			By("Scaling down the controller")
			scales.UpdateScale(virtController.deploymentName, int32(0))

			By("Waiting for no running virt-controller pods")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_controller_pods_running:count", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtController.downAlert)
		})

		It("NoReadyVirtController should be triggered when virt-controller is down", func() {
			By("Scaling down the controller")
			scales.UpdateScale(virtController.deploymentName, int32(0))

			By("Waiting for the controller to be down")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_controller_ready:sum", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtController.noReadyAlert)
		})

		It("VirtAPIDown should be triggered when virt-api is down", func() {
			By("Scaling down the api")
			scales.UpdateScale(virtAPI.deploymentName, int32(0))

			By("Waiting for no running virt-api pods")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_api_pods_running:count", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtAPI.downAlert)
		})

		It("NoReadyVirtAPI should be triggered when virt-api is down", func() {
			By("Scaling down the api")
			scales.UpdateScale(virtAPI.deploymentName, int32(0))

			By("Waiting for the api ready metric to be zero")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_api_ready:sum", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtAPI.noReadyAlert)
		})

		It("VirtHandlerDown should be triggered when virt-handler is down", func() {
			By("Patching virt-handler DaemonSet with a nodeSelector that matches no node (no running virt-handler pods)")
			daemonSet, getErr := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(
				context.Background(), virtHandler.deploymentName, metav1.GetOptions{},
			)
			Expect(getErr).ToNot(HaveOccurred())

			if daemonSet.Spec.Template.Spec.NodeSelector == nil {
				daemonSet.Spec.Template.Spec.NodeSelector = map[string]string{}
			}
			// VirtHandlerDown uses running pod count; a non-scrapable-but-Running pod would not satisfy the alert.
			daemonSet.Spec.Template.Spec.NodeSelector["kubevirt.io/e2e-virt-handler-down"] = rand.String(randNodeSelectorSuffixLength)

			patchBytes, patchErr := json.Marshal(daemonSet)
			Expect(patchErr).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Patch(
				context.Background(), virtHandler.deploymentName, types.MergePatchType, patchBytes, metav1.PatchOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for no running virt-handler pods")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_handler_pods_running:count", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtHandler.downAlert)
		})

		It("NoReadyVirtHandler should be triggered when virt-handler is not ready", func() {
			daemonSet, getErr := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(
				context.Background(), virtHandler.deploymentName, metav1.GetOptions{},
			)
			Expect(getErr).ToNot(HaveOccurred())

			originalDaemonSet := daemonSet.DeepCopy()
			defer func() {
				restoreDaemonSetImage(
					virtClient, virtHandler.deploymentName,
					originalDaemonSet, virtHandler.noReadyAlert,
				)
			}()

			badContainer := daemonSet.Spec.Template.Spec.Containers[0]
			badContainer.Image = libregistry.GetUtilityImageFromRegistry("vm-killer")
			badContainer.Command = []string{"tail", "-f", "/dev/null"}
			badContainer.Args = []string{}
			badContainer.ReadinessProbe = nil
			badContainer.LivenessProbe = nil

			err = patchDaemonSetFirstContainer(virtClient, daemonSet.Name, badContainer)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-handler ready metric to be zero")
			libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_handler_ready:sum", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtHandler.noReadyAlert)
		})
	})

	Context("Low ready alerts", func() {
		It("LowReadyVirtControllersCount should be triggered when virt controller pods ready are less than running and ready > 0", func() {
			runLowReadyDecoyDeploymentTest(
				virtClient, scales,
				virtController.deploymentName, virtController.lowReadyAlert,
				"cluster:kubevirt_virt_controller_ready:sum",
				"cluster:kubevirt_virt_controller_pods_running:count",
			)
		})

		It("LowReadyVirtAPICount should be triggered when virt api pods ready are less than running and ready > 0", func() {
			runLowReadyDecoyDeploymentTest(
				virtClient, scales,
				virtAPI.deploymentName, virtAPI.lowReadyAlert,
				"cluster:kubevirt_virt_api_ready:sum",
				"cluster:kubevirt_virt_api_pods_running:count",
			)
		})

		It("LowReadyVirtHandlerCount should be triggered when virt-handler pods ready are less than running and ready > 0", func() {
			runLowReadyDecoyDaemonSetTest(virtClient, virtHandler.lowReadyAlert)
		})

		It("LowReadyVirtOperatorsCount should be triggered when virt-operator pods ready are less than running and ready > 0", func() {
			runLowReadyDecoyDeploymentTest(
				virtClient, scales,
				virtOperator.deploymentName, virtOperator.lowReadyAlert,
				"cluster:kubevirt_virt_operator_ready:sum",
				"cluster:kubevirt_virt_operator_pods_running:count",
			)
		})
	})

	Context("Errors metrics", func() {
		const (
			operatorRoleBindingName = "kubevirt-operator-rolebinding"
			randStrLen              = 6
		)

		var crb *rbacv1.ClusterRoleBinding
		var operatorRoleBinding *rbacv1.RoleBinding

		BeforeEach(func() {
			By("Backing up the operator cluster role binding")
			crb, err = virtClient.RbacV1().ClusterRoleBindings().Get(
				context.Background(), "kubevirt-operator", metav1.GetOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Backing up the operator role binding")
			operatorRoleBinding, err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Get(
				context.Background(), operatorRoleBindingName, metav1.GetOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Restoring the operator cluster role binding")
			crb.Annotations = nil
			crb.ObjectMeta.ResourceVersion = ""
			crb.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().ClusterRoleBindings().Create(
				context.Background(), crb, metav1.CreateOptions{},
			)
			Expect(err).To(Or(Not(HaveOccurred()), MatchError(errors.IsAlreadyExists, "IsAlreadyExists")))

			By("Restoring the operator role binding")
			operatorRoleBinding.Annotations = nil
			operatorRoleBinding.ObjectMeta.ResourceVersion = ""
			operatorRoleBinding.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Create(
				context.Background(), operatorRoleBinding, metav1.CreateOptions{},
			)
			Expect(err).To(Or(Not(HaveOccurred()), MatchError(errors.IsAlreadyExists, "IsAlreadyExists")))
		})

		It("VirtApiRESTErrorsBurst should be triggered when requests to virt-api are failing", func() {
			By("Creating VNC connections to the virt-api")
			Eventually(func(g Gomega) {
				_, vncErr := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).VNC(
					rand.String(randStrLen), false,
				)
				g.Expect(vncErr).To(MatchError(ContainSubstring("not found")))
				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtAPI.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtOperatorRESTErrorsBurst should be triggered when requests to virt-operator are failing", func() {
			By("Restoring the operator")
			scales.RestoreScale(virtOperator.deploymentName)

			By("Deleting the operator cluster role binding")
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(
				context.Background(), crb.Name, metav1.DeleteOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting the operator role binding")
			err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Delete(
				context.Background(), operatorRoleBindingName, metav1.DeleteOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the alert to exist")
			Eventually(func(g Gomega) {
				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtOperator.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		PIt("VirtControllerRESTErrorsBurst should be triggered for virt-controller", func() {
			checkRESTErrorsBurst(virtClient, "kubevirt-controller", virtController.restErrorsBurtsAlert)
		})

		It("VirtHandlerRESTErrorsBurst should be triggered when requests to virt-handler are failing", func() {
			checkRESTErrorsBurst(virtClient, "kubevirt-handler", virtHandler.restErrorsBurtsAlert)
		})
	})
})

func restoreOperator(virtClient kubecli.KubevirtClient, scales *libmonitoring.Scaling) {
	oldVirtOperatorReplicas := scales.GetScale(virtOperator.deploymentName)

	for i := range oldVirtOperatorReplicas {
		replica := i + 1

		By(fmt.Sprintf("Updating the operator scale to %d", replica))
		scales.UpdateScale(virtOperator.deploymentName, replica)

		By("Waiting for running virt-operator pods")
		libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_operator_pods_running:count", float64(replica))

		By("Waiting for the operator to be ready")
		libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_operator_ready:sum", float64(replica))
	}

	By("Waiting for an operator to be leading")
	libmonitoring.WaitForMetricValue(virtClient, "cluster:kubevirt_virt_operator_leading:sum", 1.0)
}

func checkRESTErrorsBurst(virtClient kubecli.KubevirtClient, roleBindingName, alertName string) {
	By("Deleting the cluster role binding")
	err := virtClient.RbacV1().ClusterRoleBindings().Delete(
		context.Background(), roleBindingName, metav1.DeleteOptions{},
	)
	Expect(err).ToNot(HaveOccurred())

	vmi := libvmifact.NewGuestless()

	By("Trying to create a guestless vmi until the alert exists")
	Eventually(func(g Gomega) {
		_, _ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(
			context.Background(), vmi, metav1.CreateOptions{},
		)
		_ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Delete(
			context.Background(), vmi.Name, metav1.DeleteOptions{},
		)

		g.Expect(libmonitoring.CheckAlertExists(virtClient, alertName)).To(BeTrue())
	}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
}

func increaseRateLimit(virtClient kubecli.KubevirtClient) {
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
	originalKubeVirt := libkubevirt.GetCurrentKv(virtClient)
	originalKubeVirt.Spec.Configuration.ControllerConfiguration = rateLimitConfig
	originalKubeVirt.Spec.Configuration.HandlerConfiguration = rateLimitConfig
	config.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
}

func runLowReadyDecoyDeploymentTest(
	virtClient kubecli.KubevirtClient,
	scales *libmonitoring.Scaling,
	deploymentName, lowReadyAlert, readyMetric, runningMetric string,
) {
	ns := flags.KubeVirtInstallNamespace
	ctx := context.Background()

	By("Scaling the deployment to one real pod (decoy will be the second running pod)")
	scales.UpdateScale(deploymentName, int32(lowReadySingletonWorkloadCount))

	libmonitoring.WaitForMetricValue(virtClient, runningMetric, float64(lowReadySingletonWorkloadCount))
	libmonitoring.WaitForMetricValue(virtClient, readyMetric, float64(lowReadySingletonWorkloadCount))

	deployment, getErr := virtClient.AppsV1().Deployments(ns).Get(ctx, deploymentName, metav1.GetOptions{})
	Expect(getErr).ToNot(HaveOccurred())

	decoyName := fmt.Sprintf("%s-e2e-lowready-decoy-%s", deploymentName, rand.String(randNodeSelectorSuffixLength))
	decoyPod := newLowReadyDecoyPod(decoyName, deployment.Spec.Template.Spec)

	defer func() {
		By("Deleting the low-ready decoy pod")
		_ = virtClient.CoreV1().Pods(ns).Delete(ctx, decoyName, metav1.DeleteOptions{})
		By("Waiting for the low ready alert to not be firing anymore")
		libmonitoring.WaitUntilAlertDoesNotExist(virtClient, lowReadyAlert)
		By("Restoring the deployment replica count")
		scales.RestoreScale(deploymentName)
	}()

	By("Creating a decoy pod that counts as running but does not contribute to the KubeVirt ready sum")
	_, createErr := virtClient.CoreV1().Pods(ns).Create(ctx, decoyPod, metav1.CreateOptions{})
	Expect(createErr).ToNot(HaveOccurred())

	By("Waiting for two running pods and one ready pod in metrics (decoy running, single real pod ready)")
	libmonitoring.WaitForMetricValue(virtClient, runningMetric, lowReadyDeploymentRunningCount)
	libmonitoring.WaitForMetricValue(virtClient, readyMetric, float64(lowReadySingletonWorkloadCount))

	libmonitoring.VerifyAlertExist(virtClient, lowReadyAlert)
}

func runLowReadyDecoyDaemonSetTest(virtClient kubecli.KubevirtClient, lowReadyAlert string) {
	ns := flags.KubeVirtInstallNamespace
	ctx := context.Background()

	daemonSet, getErr := virtClient.AppsV1().DaemonSets(ns).Get(ctx, virtHandler.deploymentName, metav1.GetOptions{})
	Expect(getErr).ToNot(HaveOccurred())

	decoyName := fmt.Sprintf("%s-e2e-lowready-decoy-%s", virtHandler.deploymentName, rand.String(randNodeSelectorSuffixLength))
	decoyPod := newLowReadyDecoyPod(decoyName, daemonSet.Spec.Template.Spec)

	defer func() {
		By("Deleting the virt-handler low-ready decoy pod")
		_ = virtClient.CoreV1().Pods(ns).Delete(ctx, decoyName, metav1.DeleteOptions{})
		By("Waiting for the low ready alert to not be firing anymore")
		libmonitoring.WaitUntilAlertDoesNotExist(virtClient, lowReadyAlert)
	}()

	By("Creating a virt-handler decoy pod so running count exceeds the KubeVirt ready sum")
	_, createErr := virtClient.CoreV1().Pods(ns).Create(ctx, decoyPod, metav1.CreateOptions{})
	Expect(createErr).ToNot(HaveOccurred())

	const (
		handlerReadyMetric   = "cluster:kubevirt_virt_handler_ready:sum"
		handlerRunningMetric = "cluster:kubevirt_virt_handler_pods_running:count"
	)
	By("Waiting for running count to exceed ready sum with ready still positive")
	Eventually(func(g Gomega) {
		ready, errR := libmonitoring.GetMetricValueWithLabels(virtClient, handlerReadyMetric, nil)
		running, errRun := libmonitoring.GetMetricValueWithLabels(virtClient, handlerRunningMetric, nil)
		g.Expect(errR).ToNot(HaveOccurred())
		g.Expect(errRun).ToNot(HaveOccurred())
		g.Expect(ready).To(BeNumerically(">", 0))
		g.Expect(ready).To(BeNumerically("<", running))
	}, 5*time.Minute, 2*time.Second).Should(Succeed())

	libmonitoring.VerifyAlertExist(virtClient, lowReadyAlert)
}

func lowReadyDecoyPodSpecFromTemplate(spec corev1.PodSpec) corev1.PodSpec {
	out := *spec.DeepCopy()
	Expect(out.Containers).NotTo(BeEmpty())
	c := &out.Containers[0]
	c.Image = libregistry.GetUtilityImageFromRegistry("vm-killer")
	c.Command = []string{"tail", "-f", "/dev/null"}
	c.Args = nil
	c.ReadinessProbe = nil
	c.LivenessProbe = nil
	c.StartupProbe = nil
	return out
}

func newLowReadyDecoyPod(decoyPodName string, templateSpec corev1.PodSpec) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      decoyPodName,
			Namespace: flags.KubeVirtInstallNamespace,
			Labels: map[string]string{
				"kubevirt.io/e2e-lowready-decoy": decoyPodName,
			},
		},
		Spec: lowReadyDecoyPodSpecFromTemplate(templateSpec),
	}
}

func restoreImageAndWaitForAlert(
	virtClient kubecli.KubevirtClient,
	alertName string,
	restoreMsg string,
	doRestore func() error,
) {
	By(restoreMsg)
	Expect(doRestore()).ToNot(HaveOccurred())
	By("Waiting for the low ready alert to not be firing anymore")
	libmonitoring.WaitUntilAlertDoesNotExist(virtClient, alertName)
}

func restoreDaemonSetImage(
	virtClient kubecli.KubevirtClient,
	daemonSetName string,
	originalDaemonSet *appsv1.DaemonSet,
	alertName string,
) {
	restoreImageAndWaitForAlert(virtClient, alertName, "Restoring the virt-handler DaemonSet to the correct image", func() error {
		return patchDaemonSetFirstContainer(
			virtClient,
			daemonSetName,
			originalDaemonSet.Spec.Template.Spec.Containers[0],
		)
	})
}

func patchDaemonSetFirstContainer(virtClient kubecli.KubevirtClient, daemonSetName string, container corev1.Container) error {
	ns := flags.KubeVirtInstallNamespace
	ctx := context.Background()

	ds, err := virtClient.AppsV1().DaemonSets(ns).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ds.Spec.Template.Spec.Containers[0] = container

	return mergePatchDaemonSet(ctx, virtClient, ns, ds)
}

func mergePatchDaemonSet(
	ctx context.Context,
	virtClient kubecli.KubevirtClient,
	ns string,
	daemonSet *appsv1.DaemonSet,
) error {
	patchBytes, err := json.Marshal(daemonSet)
	if err != nil {
		return err
	}

	_, err = virtClient.AppsV1().DaemonSets(ns).Patch(
		ctx, daemonSet.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{},
	)

	return err
}
