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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Slirp", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	Context("VirtualMachineInstance with slirp interface", func() {
		tests.BeforeAll(func() {
			ports := []v1.Port{{Port: 80}}
			vmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
			generateHelloWorldServer(vmi, virtClient, 80, "tcp")
		})

		It("should start the virtual machine with slirp interface", func() {
			vmiPod := tests.GetRunningPodByLabel(vmi.Name, v1.DomainLabel, tests.NamespaceTestDefault)
			output, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[1].Name,
				[]string{"netstat", "-tnlp"},
			)
			log.Log.Infof("%v", output)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Contains(output, "0.0.0.0:80")).To(BeTrue())
		})
		It("should return \"Hello World!\" when connecting to localhost on port 80", func() {
			vmiPod := tests.GetRunningPodByLabel(vmi.Name, v1.DomainLabel, tests.NamespaceTestDefault)
			output, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[1].Name,
				[]string{"curl", "-s", "--retry", "30", "--retry-delay", "30", "127.0.0.1"},
			)
			fmt.Println(err)
			log.Log.Infof("%v", output)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Contains(output, "Hello World!")).To(BeTrue())
		})
		It("should reject connecting to localhost and port different than 80", func() {
			vmiPod := tests.GetRunningPodByLabel(vmi.Name, v1.DomainLabel, tests.NamespaceTestDefault)
			output, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[1].Name,
				[]string{"curl", "127.0.0.1:9080"},
			)
			log.Log.Infof("%v", output)
			Expect(err).To(HaveOccurred())
		})
		It("should be able to communicate with the outside world", func() {
			expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "curl -o /dev/null -s -w \"%{http_code}\\n\" -k https://google.com\n"},
				&expect.BExp{R: "301"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
