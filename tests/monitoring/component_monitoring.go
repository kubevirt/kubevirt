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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package monitoring

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
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

var _ = Describe("[Serial][sig-monitoring]Component Monitoring", Serial, decorators.SigMonitoring, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var scales *Scaling

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

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

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = NewScaling(virtClient, []string{virtOperator.deploymentName, virtController.deploymentName, virtApi.deploymentName})
			scales.UpdateScale(virtOperator.deploymentName, int32(0))

			reduceAlertPendingTime(virtClient)
		})

		AfterEach(func() {
			scales.RestoreAllScales()

			time.Sleep(10 * time.Second)
			alerts := []string{
				virtOperator.downAlert, virtOperator.noReadyAlert, virtOperator.lowCountAlert,
				virtController.downAlert, virtController.noReadyAlert, virtController.lowCountAlert,
				virtApi.downAlert, virtApi.noReadyAlert, virtApi.lowCountAlert,
			}
			waitUntilAlertDoesNotExist(virtClient, alerts...)
		})

		It("VirtOperatorDown and NoReadyVirtOperator should be triggered when virt-operator is down", func() {
			verifyAlertExist(virtClient, virtOperator.downAlert)
			verifyAlertExist(virtClient, virtOperator.noReadyAlert)
		})

		It("LowVirtOperatorCount should be triggered when virt-operator count is low", decorators.RequiresTwoSchedulableNodes, func() {
			verifyAlertExist(virtClient, virtOperator.lowCountAlert)
		})

		It("VirtControllerDown and NoReadyVirtController should be triggered when virt-controller is down", func() {
			scales.UpdateScale(virtController.deploymentName, int32(0))
			verifyAlertExist(virtClient, virtController.downAlert)
			verifyAlertExist(virtClient, virtController.noReadyAlert)
		})

		It("LowVirtControllersCount should be triggered when virt-controller count is low", decorators.RequiresTwoSchedulableNodes, func() {
			scales.UpdateScale(virtController.deploymentName, int32(0))
			verifyAlertExist(virtClient, virtController.lowCountAlert)
		})

		It("VirtApiDown should be triggered when virt-api is down", func() {
			scales.UpdateScale(virtApi.deploymentName, int32(0))
			verifyAlertExist(virtClient, virtApi.downAlert)
		})

		It("LowVirtAPICount should be triggered when virt-api count is low", decorators.RequiresTwoSchedulableNodes, func() {
			scales.UpdateScale(virtApi.deploymentName, int32(0))
			verifyAlertExist(virtClient, virtApi.lowCountAlert)
		})
	})

	Context("Errors metrics", func() {
		var crb *rbacv1.ClusterRoleBinding

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			crb, err = virtClient.RbacV1().ClusterRoleBindings().Get(context.Background(), "kubevirt-operator", metav1.GetOptions{})
			util.PanicOnError(err)

			increaseRateLimit()

			scales = NewScaling(virtClient, []string{virtOperator.deploymentName})
			scales.UpdateScale(virtOperator.deploymentName, int32(0))

			reduceAlertPendingTime(virtClient)
		})

		AfterEach(func() {
			crb.Annotations = nil
			crb.ObjectMeta.ResourceVersion = ""
			crb.ObjectMeta.UID = ""
			_, err = virtClient.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
			if !errors.IsAlreadyExists(err) {
				util.PanicOnError(err)
			}
			scales.RestoreAllScales()

			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist(virtClient, virtOperator.downAlert, virtApi.downAlert, virtController.downAlert, virtHandler.downAlert)
		})

		It("VirtApiRESTErrorsBurst and VirtApiRESTErrorsHigh should be triggered when requests to virt-api are failing", func() {
			randVmName := rand.String(6)

			Eventually(func(g Gomega) {
				cmd := clientcmd.NewVirtctlCommand("vnc", randVmName)
				err := cmd.Execute()
				Expect(err).To(HaveOccurred())

				g.Expect(checkAlertExists(virtClient, virtApi.restErrorsBurtsAlert)).To(BeTrue())
				g.Expect(checkAlertExists(virtClient, virtApi.restErrorsHighAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtOperatorRESTErrorsBurst and VirtOperatorRESTErrorsHigh should be triggered when requests to virt-operator are failing", func() {
			scales.RestoreScale(virtOperator.deploymentName)
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), crb.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(checkAlertExists(virtClient, virtOperator.restErrorsBurtsAlert)).To(BeTrue())
				g.Expect(checkAlertExists(virtClient, virtOperator.restErrorsHighAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtControllerRESTErrorsBurst and VirtControllerRESTErrorsHigh should be triggered when requests to virt-controller are failing", func() {
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-controller", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				g.Expect(checkAlertExists(virtClient, virtController.restErrorsBurtsAlert)).To(BeTrue())
				g.Expect(checkAlertExists(virtClient, virtController.restErrorsHighAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})

		It("VirtHandlerRESTErrorsBurst and VirtHandlerRESTErrorsHigh should be triggered when requests to virt-handler are failing", func() {
			err = virtClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "kubevirt-handler", metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()

			Eventually(func(g Gomega) {
				_, _ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				_ = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})

				g.Expect(checkAlertExists(virtClient, virtHandler.restErrorsBurtsAlert)).To(BeTrue())
				g.Expect(checkAlertExists(virtClient, virtHandler.restErrorsHighAlert)).To(BeTrue())
			}, 5*time.Minute, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("Resource metrics", func() {
		var resourceAlerts = []string{
			"KubeVirtComponentExceedsRequestedCPU",
			"KubeVirtComponentExceedsRequestedMemory",
		}

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			scales = NewScaling(virtClient, []string{virtOperator.deploymentName})
			scales.UpdateScale(virtOperator.deploymentName, int32(0))
			reduceAlertPendingTime(virtClient)
		})

		AfterEach(func() {
			scales.RestoreAllScales()
			time.Sleep(10 * time.Second)
			waitUntilAlertDoesNotExist(virtClient, resourceAlerts...)
		})

		It("KubeVirtComponentExceedsRequestedCPU should be triggered when virt-api exceeds requested CPU", func() {
			By("updating virt-api deployment CPU and Memory requests")
			updateDeploymentResourcesRequest(virtClient, virtApi.deploymentName, resource.MustParse("0m"), resource.MustParse("0Mi"))

			By("waiting for KubeVirtComponentExceedsRequestedCPU and KubeVirtComponentExceedsRequestedMemory alerts")
			verifyAlertExist(virtClient, "KubeVirtComponentExceedsRequestedCPU")
			verifyAlertExist(virtClient, "KubeVirtComponentExceedsRequestedMemory")
		})
	})
})

func updateDeploymentResourcesRequest(virtClient kubecli.KubevirtClient, deploymentName string, cpu, memory resource.Quantity) {
	deployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceCPU:    cpu,
		k8sv1.ResourceMemory: memory,
	}

	patchDeployment(virtClient, deployment)
}

func patchDeployment(virtClient kubecli.KubevirtClient, deployment *appsv1.Deployment) {
	patchOp := patch.PatchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: deployment.Spec,
	}

	payload, err := patch.GeneratePatchPayload(patchOp)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Patch(context.Background(), deployment.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}
