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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

const (
	pubKey       = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	rootPassword = "password"
	rootCmd      = "%23%21/bin/sh%0Aecho%20password%20%7C%20passwd%20--stdin%20root%0A"
	motd         = "Welcome%20to%20Kubevirt%0A"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component]IgnitionData", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	LaunchVMI := func(vmi *v1.VirtualMachineInstance) {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
		Expect(err).To(BeNil())

		By("Waiting the VirtualMachineInstance start")
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
	}

	VerifyIgnitionDataVMI := func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
		Expect(err).ToNot(HaveOccurred())
		defer expecter.Close()

		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		resp, err := expecter.ExpectBatch(commands, timeout)
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with cloudInitNoCloud userDataBase64 source", func() {
			Context("with injected ssh-key", func() {
				It("[test_id:1616]should have ssh-key under authorized keys", func() {
					vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
					ignitionData := fmt.Sprintf("{ \"ignition\": { \"config\": {}, \"version\": \"2.2.0\" }, \"networkd\": {}, \"passwd\": { \"users\": [ { \"name\": \"core\", \"sshAuthorizedKeys\": [ \"%s\" ] } ] }, \"storage\": { \"files\": [ { \"contents\": { \"source\": \"data:%s\", \"verification\": {} }, \"filesystem\": \"root\", \"mode\": 420, \"path\": \"/etc/motd\" }, { \"contents\": { \"source\": \"data:%s\", \"verification\": {} }, \"filesystem\": \"root\", \"mode\": 448, \"path\": \"/root/first.sh\" } ] }, \"systemd\": { \"units\": [ { \"contents\": \"[Service]\nType=oneshot\nExecStart=/root/first.sh\n[Install]\nWantedBy=multi-user.target\n\", \"enabled\": true, \"name\": \"first-boot.service\" } ] } }", pubKey, motd, rootCmd)
					vmi.Annotations = map[string]string{"kubevirt.io/ignitiondata": ignitionData}

					LaunchVMI(vmi)

					VerifyIgnitionDataVMI(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "root\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: rootPassword + "\n"},
						&expect.BExp{R: "$"},
						&expect.BSnd{S: "cat /home/core/.ssh/authorized_keys\n"},
						&expect.BExp{R: "test-ssh-key"},
					}, time.Second*300)
				})
			})
		})

	})
})
