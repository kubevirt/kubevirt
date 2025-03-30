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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Sound", decorators.SigCompute, func() {
	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with default sound support", func() {
		It("should create an ich9 sound device on empty model", func() {
			vmi := libvmifact.NewCirros()
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name: "test-audio-device",
			}
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
			err = checkICH9AudioDeviceInGuest(vmi)
			Expect(err).ToNot(HaveOccurred(), "ICH9 sound device was no found")
		})
	})
})

func checkICH9AudioDeviceInGuest(vmi *v1.VirtualMachineInstance) error {
	// Audio device: Intel Corporation 82801I (ICH9 Family) HD Audio Controller
	const deviceId = "8086:293e"

	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: fmt.Sprintf("lspci | grep %s\n", deviceId)},
		&expect.BExp{R: ".*8086.*"},
	}, 15)
}
