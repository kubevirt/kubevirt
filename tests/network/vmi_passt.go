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
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("[Serial] Passt", decorators.PasstGate, Serial, func() {
	BeforeEach(func() {
		tests.EnableFeatureGate(deprecation.PasstGate)
	})

	It("can be used by a VMI as its primary network", func() {
		const (
			macAddress = "02:00:00:00:00:02"
		)

		vmi := libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(v1.Interface{
				Name:                   v1.DefaultPodNetwork().Name,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}},
				Ports:                  []v1.Port{{Port: 1234, Protocol: "TCP"}},
				MacAddress:             macAddress,
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		var err error
		namespace := testsuite.GetTestNamespace(nil)
		vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(
			vmi,
			console.LoginToAlpine,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].MAC).To(Equal(macAddress))
	})
})
