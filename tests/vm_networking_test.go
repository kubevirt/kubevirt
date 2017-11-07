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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/goexpect"

	"fmt"

	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/ginkgo/extensions/table"

	"sync"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Networking", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	virtConfig, err := kubecli.GetKubevirtClientConfig()
	tests.PanicOnError(err)

	var inboundVM *v1.VirtualMachine
	var outboundVM *v1.VirtualMachine

	// TODO this is not optimal, since the one test which will initiate this, will look slow
	BeforeAll(func() {
		tests.BeforeTestCleanup()

		var wg sync.WaitGroup

		createAndLogin := func() *v1.VirtualMachine {
			vm, err := tests.NewRandomVMWithEphemeralDiskAndUserdata("kubevirt/cirros-registry-disk-demo:devel", "noCloud", "#!/bin/bash\necho 'hello'\n")
			Expect(err).ToNot(HaveOccurred())

			// add node network
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{{Type: "nodeNetwork"}}

			// Start VM
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Fetch the new VM with updated status
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Get(vm.ObjectMeta.Name, v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			expecter, _, err := tests.NewConsoleExpecter(virtConfig, vm, "", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "cirros login: "},
				&expect.BSnd{S: "cirros\n"},
				&expect.BExp{R: "Password: "},
				&expect.BSnd{S: "cubswin:)\n"},
				&expect.BExp{R: "\\$ "},
			}, 90*time.Second)
			Expect(err).ToNot(HaveOccurred())
			return vm
		}
		wg.Add(2)

		// Create inbound VM which listens on port 1500 for incoming connections and repeatedly returns "Hello World!"
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			inboundVM = createAndLogin()
			expecter, _, err := tests.NewConsoleExpecter(virtConfig, inboundVM, "", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "nc -klp 1500 -e echo -e \"Hello World!\"\n"},
			}, 60*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}()

		// Create a VM and log in, to allow executing arbitrary commands from the terminal
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			outboundVM = createAndLogin()
		}()

		wg.Wait()
	})

	Context("VirtualMachine with nodeNetwork definition given", func() {

		It("should be able to reach the internet", func() {
			// Wait until the VM is booted, ping google and check if we can reach the internet
			expecter, _, err := tests.NewConsoleExpecter(virtConfig, outboundVM, "", 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ping www.google.com -c 1 -w 5\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 90*time.Second)
			Expect(err).ToNot(HaveOccurred())
		}, 120)

		table.DescribeTable("should be reachable via the propagated IP from a Pod", func(op v12.NodeSelectorOperator, hostNetwork bool) {

			ip := inboundVM.Status.Interfaces[0].IP

			// Run netcat and give it one second to ghet "Hello World!" back from the VM
			check := []string{fmt.Sprintf("while read x; do test \"$x\" = \"Hello World!\"; exit $?; done < <(nc %s 1500 -i 1 -w 1)", ip)}
			job := tests.RenderJob("netcat", tests.GetDockerTag(), []string{"/bin/bash", "-c"}, check)
			job.Spec.Affinity = &v12.Affinity{
				NodeAffinity: &v12.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v12.NodeSelector{
						NodeSelectorTerms: []v12.NodeSelectorTerm{
							{
								MatchExpressions: []v12.NodeSelectorRequirement{
									{Key: "kubernetes.io/hostname", Operator: op, Values: []string{inboundVM.Status.NodeName}},
								},
							},
						},
					},
				},
			}
			job.Spec.HostNetwork = hostNetwork

			job, err = virtClient.CoreV1().Pods(inboundVM.ObjectMeta.Namespace).Create(job)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v12.PodPhase {
				j, err := virtClient.Core().Pods(inboundVM.ObjectMeta.Namespace).Get(job.ObjectMeta.Name, v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(j.Status.Phase).ToNot(Equal(v12.PodFailed))
				return j.Status.Phase
			}, 30*time.Second, 1*time.Second).Should(Equal(v12.PodSucceeded))

		},
			table.Entry("on the same node from Pod", v12.NodeSelectorOpIn, false),
			table.Entry("on a different node from Pod", v12.NodeSelectorOpNotIn, false),
			table.Entry("on the same node from Node", v12.NodeSelectorOpIn, true),
			table.Entry("on a different node from Node", v12.NodeSelectorOpNotIn, true),
		)
	})

})

func BeforeAll(fn func()) {
	first := true
	BeforeEach(func() {
		if first {
			fn()
			first = false
		}
	})
}
