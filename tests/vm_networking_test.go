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
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Networking", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var inboundVM *v1.VirtualMachine
	var outboundVM *v1.VirtualMachine

	// TODO this is not optimal, since the one test which will initiate this, will look slow
	BeforeAll(func() {
		tests.BeforeTestCleanup()

		var wg sync.WaitGroup

		createAndLogin := func() (vm *v1.VirtualMachine) {
			vm = tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")

			// Start VM
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStartIgnoreWarnings(vm)

			// Fetch the new VM with updated status
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Get(vm.ObjectMeta.Name, v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Lets make sure that the OS is up by waiting until we can login
			expecter, err := tests.LoggedInCirrosExpecter(vm)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())
			return vm
		}
		wg.Add(2)

		// Create inbound VM which listens on port 1500 for incoming connections and repeatedly returns "Hello World!"
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			inboundVM = createAndLogin()
			expecter, _, err := tests.NewConsoleExpecter(virtClient, inboundVM, 10*time.Second)
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

		table.DescribeTable("should be able to reach", func(destination string) {
			var cmdCheck, addr string

			// Wait until the VM is booted, ping google and check if we can reach the internet
			expecter, _, err := tests.NewConsoleExpecter(virtClient, outboundVM, 10*time.Second)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			switch destination {
			case "Internet":
				addr = "www.google.com"
			case "InboundVM":
				addr = inboundVM.Status.Interfaces[0].IP
			}
			cmdCheck = fmt.Sprintf("ping %s -c 1 -w 5\n", addr)
			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		},
			table.Entry("the Inbound VM", "InboundVM"),
			table.Entry("the internet", "Internet"),
		)

		table.DescribeTable("should be reachable via the propagated IP from a Pod", func(op v12.NodeSelectorOperator, hostNetwork bool) {

			ip := inboundVM.Status.Interfaces[0].IP

			//TODO if node count 1, skip whe nv12.NodeSelectorOpOut
			nodes, err := virtClient.CoreV1().Nodes().List(v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes.Items).ToNot(BeEmpty())
			if len(nodes.Items) == 1 && op == v12.NodeSelectorOpNotIn {
				Skip("Skip network test that requires multiple nodes when only one node is present.")
			}

			// Run netcat and give it one second to ghet "Hello World!" back from the VM
			check := []string{fmt.Sprintf("while read x; do test \"$x\" = \"Hello World!\"; exit $?; done < <(nc %s 1500 -i 1 -w 1)", ip)}
			job := tests.RenderJob("netcat", []string{"/bin/bash", "-c"}, check)
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
