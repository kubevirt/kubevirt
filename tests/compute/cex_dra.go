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

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resourcev1 "k8s.io/api/resource/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	draCexDeviceClassName    = "ap-queue.virtual-machine.ibm.com"
	draCexTypeAttribute      = resourcev1.QualifiedName("cex.ibm.com/type")
	draCexResourceClaimName  = "claim0"
	draCexMDEVNameSelector   = "VFIO AP Passthrough Device"
	draCexResourceName       = "ibm.com/crypto-express"
	draCexGuestCheckTimeout  = 6 * time.Minute
	draCexVMIDeleteTimeout   = 180 * time.Second
	draCexConsoleCommandWait = 30 * time.Second
	// lszcrypt prints a two-line header even with no devices; require at least one data row.
	draCexGuestAPCheckCmd = `lszcrypt | awk 'NF && $1 != "CARD.DOM" && $1 !~ /^-+$/ { found=1 } END { exit !found }'`
)

// DRA-CEX requires a real s390x node with the CEX DRA driver and a crypto
// queue, so it cannot run on the flake-check lane.
var _ = Describe("[sig-compute]DRA CEX", Serial, decorators.SigCompute, decorators.DRACEX, decorators.WgS390x, decorators.RequiresS390X, decorators.NoFlakeCheck, func() {
	var (
		originalConfig *v1.KubeVirtConfiguration
		vmi            *v1.VirtualMachineInstance
		cexType        string
		templateName   string
	)

	BeforeEach(func() {
		cexType = cexTypeFromCluster()
		if cexType == "" {
			Skip("CEX DRA driver is not deployed; no ResourceSlice with cex.ibm.com/type")
		}
		templateName = fmt.Sprintf("single-%s-ap-queue-vm", cexType)

		if !cexDRAKubeVirtAlreadyConfigured() {
			originalConfig = libkubevirt.GetCurrentKv(kubevirt.Client()).Spec.Configuration.DeepCopy()
			configureCexDRAKubeVirt()
		}
	})

	AfterEach(func() {
		if vmi != nil {
			By("Deleting the VMI so the CEX queue is released")
			err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, draCexVMIDeleteTimeout)).To(Succeed())
			vmi = nil
		}

		if templateName != "" {
			By("Waiting for ResourceClaims to be released")
			waitForResourceClaimsToBeGone()
			err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(testsuite.GetTestNamespace(nil)).Delete(
				context.Background(), templateName, metav1.DeleteOptions{},
			)
			if err != nil && !k8serrors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			templateName = ""
		}

		if originalConfig != nil {
			kvconfig.UpdateKubeVirtConfigValueAndWait(*originalConfig)
			originalConfig = nil
		}
	})

	It("should attach a CEX AP queue to a VMI via DRA and expose it in the guest", func() {
		namespace := testsuite.GetTestNamespace(nil)
		requestName := fmt.Sprintf("%s-ap-queue", cexType)

		By("Creating a ResourceClaimTemplate for one CEX AP queue")
		createCexResourceClaimTemplate(namespace, templateName, requestName, cexType)

		By("Creating a Fedora VMI that claims one CEX AP queue")
		vmi = createCexDRAVMI(cexType, templateName, requestName)
		waitForResourceClaimsToBeCreated(1)
		waitForBoundResourceClaimsForVMIs(1, vmi.Name)
		waitForVMIToBeRunning(vmi)

		By("Logging into the guest and verifying the AP crypto device")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		Eventually(func() error {
			return console.RunCommand(vmi, draCexGuestAPCheckCmd, draCexConsoleCommandWait)
		}, draCexGuestCheckTimeout, pollingInterval).Should(Succeed(),
			"guest lszcrypt should list at least one AP device")
	})
})

func cexDRAKubeVirtAlreadyConfigured() bool {
	if !checks.HasFeature(featuregate.HostDevicesGate) || !checks.HasFeature(featuregate.HostDevicesWithDRAGate) {
		return false
	}
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	return hasCexVfioAPKeepList(&kv.Spec.Configuration)
}

func hasCexVfioAPKeepList(config *v1.KubeVirtConfiguration) bool {
	if config.PermittedHostDevices == nil {
		return false
	}
	for _, device := range config.PermittedHostDevices.MediatedDevices {
		if device.MDEVNameSelector == draCexMDEVNameSelector &&
			device.ResourceName == draCexResourceName &&
			device.ExternalResourceProvider {
			return true
		}
	}
	return false
}

func configureCexDRAKubeVirt() {
	kvconfig.EnableFeatureGate(featuregate.HostDevicesGate)
	kvconfig.EnableFeatureGate(featuregate.HostDevicesWithDRAGate)

	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	config := kv.Spec.Configuration.DeepCopy()
	config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
		MediatedDeviceTypes: []string{},
	}
	config.PermittedHostDevices = &v1.PermittedHostDevices{
		MediatedDevices: []v1.MediatedHostDevice{{
			MDEVNameSelector:         draCexMDEVNameSelector,
			ResourceName:             draCexResourceName,
			ExternalResourceProvider: true,
		}},
	}
	kvconfig.UpdateKubeVirtConfigValueAndWait(*config)
}

func cexTypeFromCluster() string {
	resourceSlices, err := kubevirt.Client().ResourceV1().ResourceSlices().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return ""
	}
	for _, resourceSlice := range resourceSlices.Items {
		for _, device := range resourceSlice.Spec.Devices {
			attr, ok := device.Attributes[draCexTypeAttribute]
			if !ok || attr.StringValue == nil || *attr.StringValue == "" {
				continue
			}
			return *attr.StringValue
		}
	}
	return ""
}

func createCexResourceClaimTemplate(namespace, templateName, requestName, cexType string) {
	resourceClaimTemplate := &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templateName,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{{
						Name: requestName,
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: draCexDeviceClassName,
							AllocationMode:  resourcev1.DeviceAllocationModeExactCount,
							Count:           1,
							Selectors: []resourcev1.DeviceSelector{{
								CEL: &resourcev1.CELDeviceSelector{
									Expression: fmt.Sprintf(`device.attributes["cex.ibm.com"].type == %q`, cexType),
								},
							}},
						},
					}},
				},
			},
		},
	}
	_, err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(namespace).Create(
		context.Background(), resourceClaimTemplate, metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())
}

func createCexDRAVMI(cexType, templateName, requestName string) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewFedora(
		libvmi.WithMemoryRequest("1Gi"),
		libvmi.WithMemoryLimit("1Gi"),
		libvmi.WithTerminationGracePeriod(0),
		libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
			Name:                      draCexResourceClaimName,
			ResourceClaimTemplateName: &templateName,
		}),
		libvmi.WithHostDevice(v1.HostDevice{
			Name: fmt.Sprintf("cex-%s", cexType),
			ClaimRequest: &v1.ClaimRequest{
				ClaimName:   draCexResourceClaimName,
				RequestName: requestName,
			},
		}),
	)

	createdVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
		context.Background(), vmi, metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())
	return createdVMI
}

func waitForResourceClaimsToBeGone() {
	Eventually(func(g Gomega) {
		claims, err := kubevirt.Client().ResourceV1().ResourceClaims(testsuite.GetTestNamespace(nil)).List(
			context.Background(), metav1.ListOptions{},
		)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(claims.Items).To(BeEmpty())
	}, timeout, pollingInterval).Should(Succeed())
}
