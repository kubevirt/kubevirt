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
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resourcev1 "k8s.io/api/resource/v1"
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
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	draCexDriverName         = "cex-driver.ibm.com"
	draCexDeviceClassName    = "ap-queue.virtual-machine.ibm.com"
	draCexTypeAttribute      = resourcev1.QualifiedName("cex.ibm.com/type")
	draCexResourceClaimName  = "claim0"
	draCexMDEVNameSelector   = "VFIO AP Passthrough Device"
	draCexResourceName       = "ibm.com/crypto-express"
	draCexConsoleCommandWait = 30 * time.Second
	draCexGuestAPCheckCmd    = `ls /sys/bus/ap/devices | grep -qE '^[0-9a-f]{2}\.[0-9a-f]{4}$'`
)

var _ = Describe("[sig-compute]DRA CEX", Serial, decorators.SigCompute, decorators.DRACEX, decorators.RequiresS390X, decorators.NoFlakeCheck, func() {
	var (
		cexType      string
		templateName string
	)

	BeforeEach(func() {
		cexType = cexTypeFromCluster()
		Expect(cexType).ToNot(BeEmpty(), "CEX DRA driver is not deployed; no ResourceSlice with driver %s and cex.ibm.com/type", draCexDriverName)
		templateName = fmt.Sprintf("single-%s-ap-queue-vm", cexType)

		if !cexDRAKubeVirtAlreadyConfigured() {
			originalConfig := libkubevirt.GetCurrentKv(kubevirt.Client()).Spec.Configuration.DeepCopy()
			DeferCleanup(func() {
				kvconfig.UpdateKubeVirtConfigValueAndWait(*originalConfig)
			})
			configureCexDRAKubeVirt()
		}
	})

	It("should attach a CEX AP queue to a VMI via DRA and expose it in the guest", func() {
		requestName := fmt.Sprintf("%s-ap-queue", cexType)

		By("Creating a ResourceClaimTemplate for one CEX AP queue")
		createResourceClaimTemplate(cexResourceClaimTemplate(templateName, requestName, cexType))

		By("Creating a Fedora VMI that claims one CEX AP queue")
		vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
			context.Background(), cexDRAVMI(cexType, templateName, requestName), metav1.CreateOptions{},
		)
		Expect(err).ToNot(HaveOccurred())
		waitForResourceClaimsToBeCreated(1)
		waitForBoundResourceClaimsForVMIs(1, vmi.Name)
		waitForVMIToBeRunning(vmi)

		By("Logging into the guest and verifying the AP crypto device")
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		Eventually(func() error {
			return console.RunCommand(vmi, draCexGuestAPCheckCmd, draCexConsoleCommandWait)
		}, timeout, pollingInterval).Should(Succeed(),
			"guest should have at least one AP queue device under /sys/bus/ap/devices")
	})
})

func cexDRAKubeVirtAlreadyConfigured() bool {
	if !checks.HasFeature(featuregate.HostDevicesGate) {
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
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	config := kv.Spec.Configuration.DeepCopy()
	if config.DeveloperConfiguration == nil {
		config.DeveloperConfiguration = &v1.DeveloperConfiguration{}
	}
	config.DeveloperConfiguration.DisabledFeatureGates = slices.DeleteFunc(
		config.DeveloperConfiguration.DisabledFeatureGates,
		func(fg string) bool { return fg == featuregate.HostDevicesGate },
	)
	if !slices.Contains(config.DeveloperConfiguration.FeatureGates, featuregate.HostDevicesGate) {
		config.DeveloperConfiguration.FeatureGates = append(
			config.DeveloperConfiguration.FeatureGates, featuregate.HostDevicesGate,
		)
	}
	// Empty types: virt-handler must not instantiate AP mdevs. The CEX DRA
	// driver provides vfio-ap devices; they are only keep-listed below.
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
	resourceSlices, err := kubevirt.Client().ResourceV1().ResourceSlices().List(
		context.Background(),
		metav1.ListOptions{
			FieldSelector: resourcev1.ResourceSliceSelectorDriver + "=" + draCexDriverName,
		},
	)
	Expect(err).ToNot(HaveOccurred())
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

func cexResourceClaimTemplate(templateName, requestName, cexType string) *resourcev1.ResourceClaimTemplate {
	return resourceClaimTemplate(templateName, exactDeviceRequest(
		requestName,
		draCexDeviceClassName,
		fmt.Sprintf(`device.attributes["cex.ibm.com"].type == %q`, cexType),
	))
}

func cexDRAVMI(cexType, templateName, requestName string) *v1.VirtualMachineInstance {
	return libvmifact.NewFedora(
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
}
