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

package validatingadmissionpolicy

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute] permitted host devices are not allocatable by arbitrary pods", decorators.SigCompute, Serial, func() {
	const usbResource = "devices.kubevirt.io/usb"

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		enabled, err := util.IsValidatingAdmissionPolicyEnabled(virtClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(BeTrue(), "ValidatingAdmissionPolicy should be enabled")

		_, err = virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(
			context.Background(), vap.HostDevicesRestrictionPolicyName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "host device validating admission policy should be installed")
		_, err = virtClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Get(
			context.Background(), vap.HostDevicesRestrictionBindingName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "host device validating admission policy binding should be installed")
	})

	It("should admit ordinary pods and reject pods that request a KubeVirt-managed host device", func() {
		kv := libkubevirt.GetCurrentKv(virtClient)
		original := kv.Spec.Configuration.DeepCopy()
		updated := original.DeepCopy()
		updated.PermittedHostDevices = &v1.PermittedHostDevices{
			USB: []v1.USBHostDevice{{
				ResourceName: usbResource,
				Selectors: []v1.USBSelector{{
					Vendor:  "0000",
					Product: "0000",
				}},
			}},
		}
		DeferCleanup(func() {
			kvconfig.UpdateKubeVirtConfigValueAndWait(*original)
		})
		kvconfig.UpdateKubeVirtConfigValueAndWait(*updated)

		By("admitting a pod that does not request the host device")
		plain := libpod.RenderPod("no-host-device-", nil, nil)
		plain.Namespace = testsuite.GetTestNamespace(nil)
		created, err := virtClient.CoreV1().Pods(plain.Namespace).Create(context.Background(), plain, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(virtClient.CoreV1().Pods(created.Namespace).Delete(context.Background(), created.Name, metav1.DeleteOptions{})).To(Succeed())

		By("rejecting a pod that requests the permitted host device")
		// The apiserver watches the KubeVirt param, so allow a short delay after the CR update.
		// Device-plugin resources are not overcommitable: requests and limits must both be set,
		// and equal, or the pod is rejected before the ValidatingAdmissionPolicy runs.
		Eventually(func() error {
			attacker := libpod.RenderPod("host-device-", nil, nil)
			attacker.Namespace = plain.Namespace
			deviceQty := resource.MustParse("1")
			attacker.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceName(usbResource): deviceQty,
			}
			attacker.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
				corev1.ResourceName(usbResource): deviceQty,
			}
			created, err := virtClient.CoreV1().Pods(attacker.Namespace).Create(context.Background(), attacker, metav1.CreateOptions{})
			if err == nil {
				_ = virtClient.CoreV1().Pods(created.Namespace).Delete(context.Background(), created.Name, metav1.DeleteOptions{})
				return fmt.Errorf("host device pod was admitted")
			}
			if !errors.IsForbidden(err) || !strings.Contains(err.Error(), vap.HostDevicesRestrictionErr) {
				return err
			}
			return nil
		}, 30*time.Second, time.Second).Should(Succeed())
	})
})
