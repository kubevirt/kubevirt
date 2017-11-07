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

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Health Monitoring", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	virtConfig, err := kubecli.GetKubevirtClientConfig()
	tests.PanicOnError(err)

	launchVM := func(vm *v1.VirtualMachine) {
		obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())

		tests.WaitForSuccessfulVMStart(obj)
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("Watchdog device", func() {
		It("should cause VM to shutdown when watchdog expires", func(done Done) {
			vm := tests.NewRandomVMWithWatchdog()
			Expect(err).ToNot(HaveOccurred())
			launchVM(vm)

			expecter, _, err := tests.NewConsoleExpecter(virtConfig, vm, "serial0", 10*time.Second)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BExp{R: "Welcome to Alpine"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "login"},
				&expect.BSnd{S: "root\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "watchdog -t 2000ms -T 4000ms /dev/watchdog && sleep 5 && killall -9 watchdog\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 150*time.Second)
			Expect(err).ToNot(HaveOccurred())

			namespace := vm.ObjectMeta.Namespace
			name := vm.ObjectMeta.Name

			Eventually(func() v1.VMPhase {
				startedVM, err := virtClient.VM(namespace).Get(name, metav1.GetOptions{})

				Expect(err).ToNot(HaveOccurred())
				return startedVM.Status.Phase
			}, 40*time.Second).Should(Equal(v1.Failed))

			close(done)
		}, 200)
	})
})
