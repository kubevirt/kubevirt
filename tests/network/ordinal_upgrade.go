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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Ordinal naming scheme upgrade", Serial, func() {
	const nadName = "my-net"

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.LibvirtHooksServerAndClient)
		config.EnableFeatureGate(featuregate.PodSecondaryInterfaceNamingUpgrade)
	})

	BeforeEach(func() {
		netAttachDef := libnet.NewBridgeNetAttachDef(nadName, "br10")
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should upgrade the ordinal naming scheme", func() {
		const secondaryNetName = "sec"

		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithLabel("use-ordinal", "true"),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
		)

		var err error
		namespace := testsuite.GetTestNamespace(nil)
		vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(
			vmi,
			console.LoginToAlpine,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].PodInterfaceName).To(Equal("net1"))

		vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		patchset := patch.New(
			patch.WithRemove("/metadata/labels/use-ordinal"),
		)

		patchBytes, err := patchset.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		_, err = kubevirt.Client().VirtualMachineInstance(namespace).Patch(context.Background(), vmi.Name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Perform migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		vmi = libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].PodInterfaceName).To(Equal("podadd93534eeb"))
	})
}))
