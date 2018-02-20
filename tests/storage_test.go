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
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Storage", func() {

	nodeName := ""
	nodeIp := ""
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(nodes.Items).ToNot(BeEmpty())
		nodeName = nodes.Items[0].Name
		for _, addr := range nodes.Items[0].Status.Addresses {
			if addr.Type == k8sv1.NodeInternalIP {
				nodeIp = addr.Address
				break
			}
		}
		Expect(nodeIp).ToNot(Equal(""))
	})

	getTargetLogs := func(tailLines int64) string {
		pods, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{LabelSelector: v1.AppLabel + " in (iscsi-demo-target)"})
		Expect(err).ToNot(HaveOccurred())

		//FIXME Sometimes pods hang in terminating state, select the pod which does not have a deletion timestamp
		podName := ""
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				if pod.Status.HostIP == nodeIp {
					podName = pod.ObjectMeta.Name
					break
				}
			}
		}
		Expect(podName).ToNot(BeEmpty())

		By("Getting the ISCSI pod logs")
		logsRaw, err := virtClient.CoreV1().
			Pods(metav1.NamespaceSystem).
			GetLogs(podName,
				&k8sv1.PodLogOptions{TailLines: &tailLines}).
			DoRaw()
		Expect(err).To(BeNil())

		return string(logsRaw)
	}

	checkReadiness := func() {
		logs := getTargetLogs(75)
		By("Checking that ISCSI is ready")
		Expect(logs).To(ContainSubstring("Target 1: iqn.2017-01.io.kubevirt:sn.42"))
		Expect(logs).To(ContainSubstring("Driver: iscsi"))
		Expect(logs).To(ContainSubstring("State: ready"))
	}

	RunVMAndExpectLaunch := func(vm *v1.VirtualMachine, withAuth bool, timeout int) runtime.Object {
		By("Starting a VM")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		By("Waiting until the VM will start")
		tests.WaitForSuccessfulVMStartWithTimeout(obj, timeout)
		return obj
	}

	Context("with fresh iSCSI target", func() {
		It("should be available and ready", func() {
			checkReadiness()
		})
	})

	Describe("Starting a VM", func() {
		Context("with Alpine PVC", func() {
			It("should be successfully started", func(done Done) {
				checkReadiness()

				// Start the VM with the PVC attached
				vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				RunVMAndExpectLaunch(vm, false, 45)

				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
				defer expecter.Close()
				Expect(err).To(BeNil())

				By("Checking that the VM console has expected output")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
				}, 200*time.Second)
				Expect(err).To(BeNil())

				close(done)
			}, 240)

			It("should be successfully started and stopped multiple times", func(done Done) {
				checkReadiness()

				vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}

				num := 3
				By("Starting and stopping the VM number of times")
				for i := 1; i <= num; i++ {
					obj := RunVMAndExpectLaunch(vm, false, 60)

					// Verify console on last iteration to verify the VM is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VM console has expected output")
						expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
						defer expecter.Close()
						Expect(err).To(BeNil())
						_, err = expecter.ExpectBatch([]expect.Batcher{
							&expect.BExp{R: "Welcome to Alpine"},
						}, 200*time.Second)
						Expect(err).To(BeNil())
					}

					err = virtClient.VM(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())

					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)
				}
				close(done)
			}, 240)
		})
	})
})
