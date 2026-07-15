/*
 * This file is part of the kubevirt project
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

package network

import (
	"context"
	"fmt"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/tests/infrastructure"

	"kubevirt.io/kubevirt/tests/libpod"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	passtNetAttDefName   = "netbindingpasst"
	draClaimTemplateName = "passt-net-claim-template"
	draClaimName         = "passt-net-claim"
	draRequestName       = "passt-net-device"
	draNetName           = "passtnet"
)

var _ = Describe(SIG("binding plugin with DRA", Serial, decorators.NetCustomBindingPlugins, func() {
	BeforeEach(func() {
		objects, err := infrastructure.DeployDRATestDriver()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			for i := len(objects) - 1; i >= 0; i-- {
				if objects[i].GetNamespace() != "" {
					continue
				}
				Expect(testsuite.DeleteRawManifest(objects[i])).To(Succeed())
			}
		})
	})

	BeforeEach(func() {
		infrastructure.DeployDRASidecarResourceClaimsPolicy()
	})

	BeforeEach(func() {
		config.EnableFeatureGate("NetworkDevicesWithDRA")
	})

	BeforeEach(func() {
		const passtBindingName = "passt"
		passtSidecarImage := libregistry.GetUtilityImageFromRegistry("network-passt-binding")

		err := config.RegisterKubevirtConfigChange(
			config.WithNetBindingPluginIfNotPresent(passtBindingName, v1.InterfaceBindingPlugin{
				SidecarImage:                passtSidecarImage,
				NetworkAttachmentDefinition: passtNetAttDefName,
			}),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		netAttachDef := libnet.NewPasstNetAttachDef(passtNetAttDefName)
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		resourceClaimTemplate := newResourceClaimTemplate(draClaimTemplateName, draRequestName)
		_, err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(
			testsuite.GetTestNamespace(nil)).Create(
			context.Background(), resourceClaimTemplate, metav1.CreateOptions{},
		)
		Expect(err).ToNot(HaveOccurred())
	})

	FIt("migrate VM with DRA based NB", func() {
		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(draNetName, v1.PluginBinding{Name: "passt"})),
			libvmi.WithNetwork(libvmi.DRANetwork(draNetName, draClaimName, draRequestName)),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
				Name:                      draClaimName,
				ResourceClaimTemplateName: new(draClaimTemplateName),
			}),
		)
		testNamespace := testsuite.GetTestNamespace(nil)
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(vm)).WithTimeout(5 * time.Minute).WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())

		By("verifying passt wrote its log to the host dir mounted from the resource claim")
		Expect(validatePasstLogFileExistsOnHost(vmi)).Should(Succeed())

		By("migrating the VM")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

		By("verifying VM is functional after migration")
		Eventually(matcher.ThisVM(vm)).WithTimeout(5 * time.Minute).WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())

		By("verifying passt wrote its log to the host dir mounted from the resource claim")
		Expect(validatePasstLogFileExistsOnHost(vmi)).Should(Succeed())
	})
}),
)

func validatePasstLogFileExistsOnHost(vmi *v1.VirtualMachineInstance) error {
	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	if err != nil {
		return err
	}

	if vmiPod.Status.ResourceClaimStatuses == nil ||
		len(vmiPod.Status.ResourceClaimStatuses) < 1 ||
		vmiPod.Status.ResourceClaimStatuses[0].Name != draClaimName {
		return fmt.Errorf("ResourceClaimStatuses not found in virt-launcher pod")
	}

	resourceClaimName := ptr.Deref(vmiPod.Status.ResourceClaimStatuses[0].ResourceClaimName, "")
	if resourceClaimName == "" {
		return fmt.Errorf("ResourceClaimName is empty in virt-launcher pod status")
	}

	logPathOnHost := path.Join("/var/run/kubevirt/dra", resourceClaimName, "passt-dra.log")
	args := []string{"/bin/test", "-s", logPathOnHost}
	args = append([]string{"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--"}, args...)
	_, err = libnode.ExecuteCommandInVirtHandlerPod(vmi.Status.NodeName, args)
	return err
}

func newResourceClaimTemplate(templateName, requestName string) *resourcev1.ResourceClaimTemplate {
	namespace := testsuite.GetTestNamespace(nil)
	return &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templateName,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{
						{
							Name: requestName,
							Exactly: &resourcev1.ExactDeviceRequest{
								DeviceClassName: "test.kubevirt.io",
								Count:           1,
							},
						},
					},
				},
			},
		},
	}
}
