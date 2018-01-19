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

	dockerTag := os.Getenv("docker_tag")
	if dockerTag == "" {
		dockerTag = "latest"
	}

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vm *v1.VirtualMachine

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vm = tests.NewRandomVMWithDirectLun(2, true)
	})

	Context("New VM given", func() {

		It("Should be accepted on POST", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should reject posting the same VM a second time", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
			b, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusConflict)))
		})

		It("Should return 404 if VM does not exist", func() {
			b, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name("nonexistnt").DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("Should start the VM on POST", func(done Done) {
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			close(done)
		}, 30)

		It("Virt-launcher should attach to a started VM", func(done Done) {
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			logs := func() string { return getVirtLauncherLogs(virtClient, vm) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for qemu"))
			close(done)
		}, 50)

		It("if grace period greater than 0, graceful shutdown should be attempted on delete", func(done Done) {
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

			var gracePeriod int64
			gracePeriod = int64(5)
			// Give the VM a custom grace period
			vm.Spec.TerminationGracePeriodSeconds = &gracePeriod
			// Make sure we schedule the VM to master
			vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

			obj, err := virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			// Delete the VM and wait for the confirmation of the delete
			Expect(virtClient.VM(vm.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed())
			tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

			// Check if the graceful shutdown was logged
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Signaled graceful shutdown for %s", vm.GetObjectMeta().GetName())))

			// Verify VM is killed after grace period expires
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("grace period expired, killing deleted VM %s", vm.GetObjectMeta().GetName())))

			close(done)
		}, 45)

		Context("New VM which can't be started", func() {

			It("Should retry starting the VM", func(done Done) {
				vm.Spec.Volumes[0].ISCSI.SecretRef.Name = "nonexistent"
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

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
			}, 30)

			It("Should log warning if secret is not present, and proceed once the secret is there", func(done Done) {
				vm.Spec.Volumes[0].ISCSI.SecretRef.Name = "nonexistent"
				createdVM, err := virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).To(BeNil())

				// Wait until we see that starting the VM is failing
				event := tests.NewObjectEventWatcher(createdVM).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(event.Message).To(ContainSubstring("nonexistent"))

				// Creat nonexistent secret, so that the VM can recover
				secret, err := virtClient.CoreV1().Secrets(vm.Namespace).Get("iscsi-demo-secret", metav1.GetOptions{})
				secret.ObjectMeta = metav1.ObjectMeta{
					Name: "nonexistent",
				}
				_, err = virtClient.CoreV1().Secrets(vm.Namespace).Create(secret)
				Expect(err).ToNot(HaveOccurred())

				// Wait for the VM to be started, allow warning events to occur
				tests.NewObjectEventWatcher(createdVM).SinceWatchedObjectResourceVersion().Timeout(30*time.Second).WaitFor(tests.NormalEvent, v1.Started)

				close(done)

			}, 60)
		})

		Context("New VM that will be killed", func() {
			It("Should be in Failed phase", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(10 * time.Second)
				err = pkillAllVms(virtClient, nodeName, dockerTag)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				Expect(func() v1.VMPhase {
					vm := &v1.VirtualMachine{}
					err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 50)
			It("should be left alone by virt-handler", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				err = pkillAllVms(virtClient, nodeName, dockerTag)
				Expect(err).To(BeNil())

				// Wait for stop event of the VM
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				// Wait for some time and see if a sync event happens on the stopped VM
				event := tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(5*time.Second).
					SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(event).To(BeNil(), "virt-handler tried to sync on a VM in final state")

				close(done)
			}, 50)
		})

		Context("New VM when virt-launcher crashes", func() {
			It("should stop and be in Failed phase", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(10 * time.Second)
				err = pkillAllLaunchers(virtClient, nodeName, dockerTag)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				Expect(func() v1.VMPhase {
					vm := &v1.VirtualMachine{}
					err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 90)
		})
		Context("When virt-handler crashes", func() {
			It("should recover and continue management of existing VMs.", func(done Done) {
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				// Start a VM
				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VirtualMachine)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				// Kill virt-handler on the node the VM is active on.
				time.Sleep(5 * time.Second)
				err = pkillAllHandlers(virtClient, nodeName, dockerTag)
				Expect(err).To(BeNil())

				// Crash the VM and verify a recovered version of virt-handler processes the crash
				time.Sleep(5 * time.Second)
				err = pkillAllVms(virtClient, nodeName, dockerTag)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				Expect(func() v1.VMPhase {
					vm := &v1.VirtualMachine{}
					err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 120)
		})

		Context("in a non-default namespace", func() {
			table.DescribeTable("Should log libvirt start and stop lifecycle events of the domain", func(namespace string) {

				_, exists := os.LookupEnv("JENKINS_HOME")
				if exists {
					Skip("Skip log query tests for JENKINS ci test environment")
				}
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				node := nodes.Items[0].Name

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
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain is in state Running)", vm.GetObjectMeta().GetName()))
				// Check the VM Namespace
				Expect(vm.GetObjectMeta().GetNamespace()).To(Equal(namespace))

				// Delete the VM and wait for the confirmation of the delete
				_, err = virtClient.RestClient().Delete().Resource("virtualmachines").Namespace(vm.GetObjectMeta().GetNamespace()).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

				// Check if the stop event was logged
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
})

func renderPkillAllJob(dockerTag string, processName string) *k8sv1.Pod {
	job := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-killer",
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:  "vm-killer",
					Image: "kubevirt/vm-killer:" + dockerTag,
					Command: []string{
						"pkill",
						"-9",
						processName,
					},
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: newBool(true),
						RunAsUser:  new(int64),
					},
				},
			},
			HostPID: true,
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsUser: new(int64),
			},
		},
	}

	return &job
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
		GetLogs(podName,
			&k8sv1.PodLogOptions{TailLines: &tailLines}).
		DoRaw()
	Expect(err).To(BeNil())

	return string(logsRaw)
}

func pkillAllHandlers(virtCli kubecli.KubevirtClient, node, dockerTag string) error {
	job := renderPkillAllJob(dockerTag, "virt-handler")
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
func pkillAllLaunchers(virtCli kubecli.KubevirtClient, node, dockerTag string) error {
	job := renderPkillAllJob(dockerTag, "virt-launcher")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}

func pkillAllVms(virtCli kubecli.KubevirtClient, node, dockerTag string) error {
	job := renderPkillAllJob(dockerTag, "qemu")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}

func newBool(x bool) *bool {
	return &x
}
