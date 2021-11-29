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
	"encoding/xml"
	"fmt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Sound", func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with default sound support", func() {
		BeforeEach(func() {
			vmi, err = createSoundVMI(virtClient, "test-model-empty")
			Expect(err).To(BeNil())
			vmi = tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
		})

		It("should create an ich9 sound device on empty model", func() {
			checkAudioDevice(vmi, "ich9")
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with ich9 sound support", func() {
		BeforeEach(func() {
			vmi, err = createSoundVMI(virtClient, "ich9")
			Expect(err).To(BeNil())
			vmi = tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
		})

		It("should create ich9 sound device on ich9 model ", func() {
			checkXMLSoundCard(virtClient, vmi, "ich9")
			checkAudioDevice(vmi, "ich9")
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with ac97 sound support", func() {
		BeforeEach(func() {
			vmi, err = createSoundVMI(virtClient, "ac97")
			Expect(err).To(BeNil())
			vmi = tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
		})

		It("should create ac97 sound device on ac97 model", func() {
			checkXMLSoundCard(virtClient, vmi, "ac97")
			checkAudioDevice(vmi, "ac97")
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with unsupported sound support", func() {
		It("should fail to create VMI with unsupported sound device", func() {
			vmi, err = createSoundVMI(virtClient, "ich7")
			Expect(err).To(HaveOccurred())
		})
	})
})

func createSoundVMI(virtClient kubecli.KubevirtClient, soundDevice string) (*v1.VirtualMachineInstance, error) {
	randomVmi := libvmi.NewCirros()
	if soundDevice != "" {
		model := soundDevice
		if soundDevice == "test-model-empty" {
			model = ""
		}
		randomVmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
			Name:  "test-audio-device",
			Model: model,
		}
	}
	return virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(randomVmi)
}

func checkXMLSoundCard(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, model string) {
	domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
	Expect(err).ToNot(HaveOccurred())
	domSpec := &api.DomainSpec{}
	Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
	Expect(len(domSpec.Devices.SoundCards)).To(Equal(1))
	Expect(domSpec.Devices.SoundCards).To(ContainElement(api.SoundCard{
		Alias: api.NewUserDefinedAlias("test-audio-device"),
		Model: model,
	}))
}

func checkAudioDevice(vmi *v1.VirtualMachineInstance, name string) {
	// Audio device: Intel Corporation 82801I (ICH9 Family) HD Audio Controller
	deviceId := "8086:293e"
	if name == "ac97" {
		// Multimedia audio controller: Intel Corporation 82801AA AC'97 Audio Controller
		deviceId = "8086:2415"
	}
	cmdCheck := fmt.Sprintf("lspci | grep %s\n", deviceId)

	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: ".*8086.*"},
	}, 15)
	Expect(err).ToNot(HaveOccurred(), "Console command timeout")
}
