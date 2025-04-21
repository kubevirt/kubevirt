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

package tests_test

import (
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/libvmops"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("[rfe_id:609][sig-compute]VMIheadless", decorators.SigCompute, func() {

	Describe("[rfe_id:609]Creating a VirtualMachineInstance", func() {

		Context("with headless", func() {

			It("[test_id:709][posneg:positive]should connect to console", func() {
				vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(libvmi.WithAutoattachGraphicsDevice(false)), 30)

				By("checking that console works")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
			})

		})
	})

})
