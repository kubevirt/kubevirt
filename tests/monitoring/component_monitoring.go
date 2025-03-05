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

type alerts struct {
	deploymentName       string
	downAlert            string
	noReadyAlert         string
	restErrorsBurtsAlert string
	lowCountAlert        string
	lowReadyAlert        string
}

var (
	virtApi = alerts{
		deploymentName:       "virt-api",
		downAlert:            "VirtAPIDown",
		restErrorsBurtsAlert: "VirtApiRESTErrorsBurst",
		lowCountAlert:        "LowVirtAPICount",
	}
	virtController = alerts{
		deploymentName:       "virt-controller",
		downAlert:            "VirtControllerDown",
		noReadyAlert:         "NoReadyVirtController",
		restErrorsBurtsAlert: "VirtControllerRESTErrorsBurst",
		lowCountAlert:        "LowVirtControllersCount",
		lowReadyAlert:        "LowReadyVirtControllersCount",
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
		lowCountAlert:        "LowVirtOperatorCount",
	}
)

var _ = Describe("[sig-monitoring]Component Monitoring", Serial, Ordered, decorators.SigMonitoring, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var scales *libmonitoring.Scaling

	BeforeAll(func() {
		virtClient = kubevirt.Client()
		scales = libmonitoring.NewScaling(virtClient, []string{virtOperator.deploymentName})
	})

	BeforeEach(func() {
		By("Increasing the rate limit")
		increaseRateLimit(virtClient)

		By("Backing up the operator, controller, api and handler scales")
		scales.UpdateScale(virtOperator.deploymentName, int32(0))

		By("Reducing the alert pending time")
		libmonitoring.ReduceAlertPendingTime(virtClient)
	})

	AfterEach(func() {
		By("Restoring the operator")
		restoreOperator(virtClient, scales)
	})

	Context("Up metrics", func() {
		It("VirtOperatorDown should be triggered when virt-operator is down", func() {
			By("Waiting for the operator to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtOperator.downAlert)
		})

		It("NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			By("Waiting for the operator to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_ready", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtOperator.noReadyAlert)
		})

		It("LowVirtOperatorCount should be triggered when virt-operator count is low", decorators.RequiresTwoSchedulableNodes, func() {
			By("Waiting for the operator to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtOperator.lowCountAlert)
		})

		It("VirtControllerDown should be triggered when virt-controller is down", func() {
			By("Scaling down the controller")
			scales.UpdateScale(virtController.deploymentName, int32(0))

			By("Waiting for the controller to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_controller_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtController.downAlert)
		})

		It("NoReadyVirtController should be triggered when virt-controller is down", func() {
			By("Scaling down the controller")
			scales.UpdateScale(virtController.deploymentName, int32(0))

			By("Waiting for the controller to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_controller_ready", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtController.noReadyAlert)
		})

		It("LowVirtControllersCount should be triggered when virt-controller count is low", decorators.RequiresTwoSchedulableNodes, func() {
			By("Scaling down the controller")
			scales.UpdateScale(virtController.deploymentName, int32(0))

			By("Waiting for the controller to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_controller_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtController.lowCountAlert)
		})

		It("VirtApiDown should be triggered when virt-api is down", func() {
			By("Scaling down the api")
			scales.UpdateScale(virtApi.deploymentName, int32(0))

			By("Waiting for the api to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_api_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtApi.downAlert)
		})

		It("LowVirtAPICount should be triggered when virt-api count is low", decorators.RequiresTwoSchedulableNodes, func() {
			By("Scaling down the api")
			scales.UpdateScale(virtApi.deploymentName, int32(0))

			By("Waiting for the api to be down")
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_api_up", 0)

			By("Verifying the alert exists")
			libmonitoring.VerifyAlertExist(virtClient, virtApi.lowCountAlert)
		})
	})

	Context("Low ready alerts", decorators.RequiresTwoSchedulableNodes, func() {
		It("LowReadyVirtControllersCount should be triggered when virt-controller pods exist but are not ready", func() {
			virtControllerDeployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), virtController.deploymentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			container := &virtControllerDeployment.Spec.Template.Spec.Containers[0]
			container.Image = libregistry.GetUtilityImageFromRegistry("vm-killer") // any random image
			container.Command = []string{"tail", "-f", "/dev/null"}
			container.Args = []string{}
			container.ReadinessProbe = nil
			container.LivenessProbe = nil

			patch, err := json.Marshal(virtControllerDeployment)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Patch(context.Background(), virtControllerDeployment.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			libmonitoring.VerifyAlertExist(virtClient, virtController.lowReadyAlert)
		})
	})

	Context("Errors metrics", func() {
		var crb *rbacv1.ClusterRoleBinding
		const operatorRoleBindingName = "kubevirt-operator-rolebinding"
		var operatorRoleBinding *rbacv1.RoleBinding

		BeforeEach(func() {
			By("Backing up the operator cluster role binding")
			crb, err = virtClient.RbacV1().ClusterRoleBindings().Get(context.Background(), "kubevirt-operator", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Backing up the operator role binding")
			operatorRoleBinding, err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Get(context.Background(), operatorRoleBindingName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Restoring the operator cluster role binding")
			crb.Annotations = nil
			crb.ObjectMeta.ResourceVersion = ""
			crb.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
			Expect(err).To(Or(Not(HaveOccurred()), MatchError(errors.IsAlreadyExists, "IsAlreadyExists")))

			By("Restoring the operator role binding")
			operatorRoleBinding.Annotations = nil
			operatorRoleBinding.ObjectMeta.ResourceVersion = ""
			operatorRoleBinding.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Create(context.Background(), operatorRoleBinding, metav1.CreateOptions{})
			Expect(err).To(Or(Not(HaveOccurred()), MatchError(errors.IsAlreadyExists, "IsAlreadyExists")))
		})

		It("VirtApiRESTErrorsBurst should be triggered when requests to virt-api are failing", func() {
			By("Creating VNC connections to the virt-api")
			Eventually(func(g Gomega) {
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).VNC(rand.String(6), false)
				g.Expect(err).To(MatchError(ContainSubstring("not found")))
				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtApi.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtOperatorRESTErrorsBurst should be triggered when requests to virt-operator are failing", func() {
			By("Restoring the operator")
			scales.RestoreScale(virtOperator.deploymentName)

			By("Deleting the operator cluster role binding")
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Deleting the operator role binding")
			err = virtClient.RbacV1().RoleBindings(flags.KubeVirtInstallNamespace).Delete(context.Background(), operatorRoleBindingName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the alert to exist")
			Eventually(func(g Gomega) {
				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtOperator.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		PIt("VirtControllerRESTErrorsBurst should be triggered when requests to virt-controller are failing", func() {
			By("Deleting the controller cluster role binding")
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-controller", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmifact.NewGuestless()

			By("Trying to create a guestless vmi until the alert exists")
			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
				_ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})

				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtController.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtHandlerRESTErrorsBurst should be triggered when requests to virt-handler are failing", func() {
			By("Deleting the handler cluster role binding")
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-handler", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmifact.NewGuestless()

			By("Trying to create a guestless vmi until the alert exists")
			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
				_ = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})

				g.Expect(libmonitoring.CheckAlertExists(virtClient, virtHandler.restErrorsBurtsAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})
	})
})

func restoreOperator(virtClient kubecli.KubevirtClient, scales *libmonitoring.Scaling) {
	oldVirtOperatorReplicas := scales.GetScale(virtOperator.deploymentName)

	for i := range oldVirtOperatorReplicas {
		replica := i + 1

		By(fmt.Sprintf("Updating the operator scale to %d", replica))
		scales.UpdateScale(virtOperator.deploymentName, int32(replica))

		By("Waiting for the operator to be up")
		libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_up", float64(replica))

		By("Waiting for the operator to be ready")
		libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_ready", float64(replica))
	}

	By("Waiting for an operator to be leading")
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_virt_operator_leading", 1.0)
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
