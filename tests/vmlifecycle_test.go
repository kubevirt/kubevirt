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
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vm *v1.VirtualMachine

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vm = tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
	})

	Describe("Creating a VM", func() {
		It("should success", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("should start it", func(done Done) {
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			close(done)
		}, 45)

		It("should attach virt-launcher to it", func(done Done) {
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vm) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for qemu"))
			close(done)
		}, 50)

		Context("when it already exist", func() {
			It("should be rejected", func() {
				By("Creating a VM")
				err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
				Expect(err).To(BeNil())
				By("Creating the same VM second time")
				b, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).DoRaw()
				Expect(err).ToNot(BeNil())
				By("Checking that POST return status equals to 409")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil())
				Expect(status.Code).To(Equal(int32(http.StatusConflict)))
			})
		})

		Context("with user-data", func() {
			Context("without k8s secret", func() {
				It("should retry starting the VM", func(done Done) {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					vm = tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), userData)

					for _, volume := range vm.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VM")
					obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
					Expect(err).To(BeNil())

					By("Checking that VM was restarted twice")
					retryCount := 0
					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Watch(func(event *k8sv1.Event) bool {
						if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
							retryCount++
							if retryCount >= 2 {
								// Done, two retries is enough
								return true
							}
						}
						return false
					})
					close(done)
				}, 45)

				It("should log warning and proceed once the secret is there", func(done Done) {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					userData64 := ""
					vm = tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), userData)

					for _, volume := range vm.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							userData64 = spec.UserDataBase64
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VM")
					createdVM, err := virtClient.VM(tests.NamespaceTestDefault).Create(vm)
					Expect(err).To(BeNil())

					// Wait until we see that starting the VM is failing
					By("Checking that VM start failed")
					event := tests.NewObjectEventWatcher(createdVM).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
					Expect(event.Message).To(ContainSubstring("nonexistent"))

					// Creat nonexistent secret, so that the VM can recover
					By("Creating a user-data secret")
					secret := k8sv1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "nonexistent",
							Namespace: vm.GetObjectMeta().GetNamespace(),
						},
						Type: "Opaque",
						Data: map[string][]byte{
							"userdata": []byte(userData64),
						},
					}
					_, err = virtClient.CoreV1().Secrets(vm.Namespace).Create(&secret)
					Expect(err).ToNot(HaveOccurred())

					// Wait for the VM to be started, allow warning events to occur
					By("Checking that VM start succeeded")
					tests.NewObjectEventWatcher(createdVM).SinceWatchedObjectResourceVersion().Timeout(30*time.Second).WaitFor(tests.NormalEvent, v1.Started)

					close(done)

				}, 60)
			})
		})

		Context("when virt-launcher crashes", func() {
			It("should be stopped and have Failed phase", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(10 * time.Second)

				By("Crashing the virt-launcher")
				err = pkillAllLaunchers(virtClient, nodeName)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				By("Checking that VM has 'Failed' phase")
				Expect(func() v1.VMPhase {
					vm := &v1.VirtualMachine{}
					err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 90)
		})

		Context("when virt-handler crashes", func() {
			It("should recover and continue management", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				// Start a VM
				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				// Kill virt-handler on the node the VM is active on.
				time.Sleep(5 * time.Second)
				By("Crashing the virt-handler")
				err = pkillAllHandlers(virtClient, nodeName)
				Expect(err).To(BeNil())

				// Crash the VM and verify a recovered version of virt-handler processes the crash
				time.Sleep(5 * time.Second)
				By("Killing the VM")
				err = pkillAllVms(virtClient, nodeName)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				By("Checking that VM has 'Failed' phase")
				Expect(func() v1.VMPhase {
					vm := &v1.VirtualMachine{}
					err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 120)
		})

		Context("with non default namespace", func() {
			table.DescribeTable("should log libvirt start and stop lifecycle events of the domain", func(namespace string) {

				_, exists := os.LookupEnv("JENKINS_HOME")
				if exists {
					Skip("Skip log query tests for JENKINS ci test environment")
				}
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				node := nodes.Items[0].Name

				By("Creating a VM with different namespace")
				vm = tests.NewRandomVMWithNS(namespace)
				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
				Expect(err).ToNot(HaveOccurred())

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				// Make sure we schedule the VM to master
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				// Start the VM and wait for the confirmation of the start
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(vm.GetObjectMeta().GetNamespace()).Body(vm).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(obj)

				// Check if the start event was logged
				By("Checking that virt-handler logs VM creation")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain is in state Running)", vm.GetObjectMeta().GetName()))
				// Check the VM Namespace
				Expect(vm.GetObjectMeta().GetNamespace()).To(Equal(namespace))

				// Delete the VM and wait for the confirmation of the delete
				By("Deleting the VM")
				_, err = virtClient.RestClient().Delete().Resource("virtualmachines").Namespace(vm.GetObjectMeta().GetNamespace()).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

				// Check if the stop event was logged
				By("Checking that virt-handler logs VM deletion")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain deleted)", vm.GetObjectMeta().GetName()))

			},
				table.Entry(tests.NamespaceTestDefault, tests.NamespaceTestDefault),
				table.Entry(tests.NamespaceTestAlternative, tests.NamespaceTestAlternative),
			)
		})
	})

	Describe("Get a VM", func() {
		Context("when that not exist", func() {
			It("should return 404", func() {
				b, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name("nonexistnt").DoRaw()
				Expect(err).ToNot(BeNil())
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil())
				Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
			})
		})
	})

	Describe("Delete a VM", func() {
		Context("with grace period greater than 0", func() {
			It("should run graceful shutdown", func(done Done) {
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				node := nodes.Items[0].Name

				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
				Expect(err).ToNot(HaveOccurred())

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				By("Setting a VM termination grace period to 5")
				var gracePeriod int64
				gracePeriod = int64(5)
				// Give the VM a custom grace period
				vm.Spec.TerminationGracePeriodSeconds = &gracePeriod
				// Make sure we schedule the VM to master
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				By("Creating the VM")
				obj, err := virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(obj)

				// Delete the VM and wait for the confirmation of the delete
				By("Deleting the VM")
				Expect(virtClient.VM(vm.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed())
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

				// Check if the graceful shutdown was logged
				By("Checking that virt-handler logs VM graceful shutdown")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Signaled graceful shutdown for %s", vm.GetObjectMeta().GetName())))

				// Verify VM is killed after grace period expires
				By("Checking that the VM does not exist after grace period")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("grace period expired, killing deleted VM %s", vm.GetObjectMeta().GetName())))

				close(done)
			}, 45)
		})
	})

	Describe("Killed VM", func() {
		It("should be in Failed phase", func(done Done) {
			By("Starting a VM")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())

			nodeName := tests.WaitForSuccessfulVMStart(obj)
			_, ok := obj.(*v1.VirtualMachine)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
			Expect(err).ToNot(HaveOccurred())

			By("Killing the VM")
			time.Sleep(10 * time.Second)
			err = pkillAllVms(virtClient, nodeName)
			Expect(err).To(BeNil())

			tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

			By("Checking that the VM has 'Failed' phase")
			Expect(func() v1.VMPhase {
				vm := &v1.VirtualMachine{}
				err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Phase
			}()).To(Equal(v1.Failed))

			close(done)
		}, 60)

		It("should be left alone by virt-handler", func(done Done) {
			By("Starting a VM")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())

			nodeName := tests.WaitForSuccessfulVMStart(obj)
			_, ok := obj.(*v1.VirtualMachine)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
			Expect(err).ToNot(HaveOccurred())

			By("Killing the VM")
			err = pkillAllVms(virtClient, nodeName)
			Expect(err).To(BeNil())

			// Wait for stop event of the VM
			tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

			// Wait for some time and see if a sync event happens on the stopped VM
			By("Checking that virt-handler does not try to sync stopped VM")
			event := tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(5*time.Second).
				SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
			Expect(event).To(BeNil(), "virt-handler tried to sync on a VM in final state")

			close(done)
		}, 50)
	})
})

func renderPkillAllJob(processName string) *k8sv1.Pod {
	return tests.RenderJob("vm-killer", []string{"pkill"}, []string{"-9", processName})
}

func getVirtLauncherLogs(virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine) string {
	namespace := vm.GetObjectMeta().GetNamespace()
	domain := vm.GetObjectMeta().GetName()

	labelSelector := fmt.Sprintf("kubevirt.io/domain in (%s)", domain)

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty())

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: "compute",
		}).
		DoRaw()
	Expect(err).To(BeNil())

	return string(logsRaw)
}

func pkillAllHandlers(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("virt-handler")
	job.Spec.NodeName = node
	pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
	Expect(err).ToNot(HaveOccurred())

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Get(pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(k8sv1.PodSucceeded))

	return err
}

func pkillAllLaunchers(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("virt-launcher")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}

func pkillAllVms(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("qemu")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}
