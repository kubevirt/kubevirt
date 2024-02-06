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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package network

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("Macvtap", decorators.Macvtap, Serial, func() {
	const (
		macvtapLowerDevice      = "eth0"
		macvtapNetAttachDefName = "net1"
	)

	BeforeEach(func() {
		tests.EnableFeatureGate(deprecation.MacvtapGate)
	})

	BeforeEach(func() {
		ns := testsuite.GetTestNamespace(nil)
		Expect(libnet.CreateMacvtapNetworkAttachmentDefinition(ns, macvtapNetAttachDefName, macvtapLowerDevice)).To(Succeed(),
			"A macvtap network named %s should be provisioned", macvtapNetAttachDefName)
	})

	It("should successfully create a VM with macvtap interface with custom MAC address", func() {
		macHW, err := GenerateRandomMac()
		Expect(err).ToNot(HaveOccurred())
		mac := macHW.String()

		const macvtapNetName = "test-macvtap"
		vmi := libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(*libvmi.InterfaceWithMac(v1.DefaultMacvtapNetworkInterface(macvtapNetName), mac)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetName, macvtapNetAttachDefName)),
		)
		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		Expect(vmi.Status.Interfaces).To(HaveLen(1), "should have a single interface")
		Expect(vmi.Status.Interfaces[0].MAC).To(Equal(mac), "the expected MAC address should be set in the VMI")
	})
})
