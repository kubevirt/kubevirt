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
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)

	coreCli, err := kubecli.Get()
	tests.PanicOnError(err)
	var vm *v1.VM

	BeforeEach(func() {
		vm = tests.NewRandomVM()
		tests.MustCleanup()
	})

	Context("New VM given", func() {

		It("Should be accepted on POST", func() {
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should reject posting the same VM a second time", func() {
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
			b, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusConflict)))
		})

		It("Should return 404 if VM does not exist", func() {
			b, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name("nonexistnt").DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("Should start the VM on POST", func(done Done) {
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			close(done)
		}, 30)

		It("Should log libvirt start and stop lifecycle events of the domain", func(done Done) {
			// Get the pod name of virt-handler running on the master node to inspect its logs later on
			handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=master")
			labelSelector, err := labels.Parse("daemon in (virt-handler)")
			Expect(err).NotTo(HaveOccurred())
			pods, err := coreCli.CoreV1().Pods(api.NamespaceDefault).List(kubev1.ListOptions{FieldSelector: handlerNodeSelector.String(), LabelSelector: labelSelector.String()})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))

			handlerName := pods.Items[0].GetObjectMeta().GetName()
			seconds := int64(30)
			logsQuery := coreCli.Pods(api.NamespaceDefault).GetLogs(handlerName, &kubev1.PodLogOptions{SinceSeconds: &seconds})

			// Make sure we schedule the VM to master
			vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": "master"}

			// Start the VM and wait for the confirmation of the start
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			// Check if the start event was logged
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 5, 0.1).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain is in state Running)", vm.GetObjectMeta().GetName()))

			// Delete the VM and wait for the confirmation of the delete
			_, err = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			tests.NewObjectEventWatcher(obj).WaitFor(tests.NormalEvent, v1.Deleted)

			// Check if the stop event was logged
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 5, 0.1).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain deleted)", vm.GetObjectMeta().GetName()))

			close(done)
		}, 30)

		Context("New VM which can't be started", func() {

			It("Should retry starting the VM", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				retryCount := 0
				tests.NewObjectEventWatcher(obj).Watch(func(event *kubev1.Event) bool {
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

			It("Should stop retrying invalid VM and go on to latest change request", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				// Wait until we see that starting the VM is failing
				event := tests.NewObjectEventWatcher(obj).WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(event.Message).To(ContainSubstring("nonexistent"))

				_, err = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())

				// Check that the definition is deleted from the host
				tests.NewObjectEventWatcher(obj).WaitFor(tests.NormalEvent, v1.Deleted)

				close(done)

			}, 30)
		})

		Context("New VM that will be killed", func() {
			It("Should be in Failed phase", func(done Done) {
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VM)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				restClient, err := kubecli.GetRESTClient()
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(10 * time.Second)
				err = pkillAllVms(coreCli, nodeName)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(obj).WaitFor(tests.WarningEvent, v1.Stopped)

				Expect(func() v1.VMPhase {
					vm := &v1.VM{}
					err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Into(vm)
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase
				}()).To(Equal(v1.Failed))

				close(done)
			}, 50)
			It("should be left alone by virt-handler", func(done Done) {
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VM)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				err = pkillAllVms(coreCli, nodeName)
				Expect(err).To(BeNil())

				// Wait for stop event of the VM
				tests.NewObjectEventWatcher(obj).WaitFor(tests.WarningEvent, v1.Stopped)

				// Wait for some time and see if a sync event happens on the stopped VM
				event := tests.NewObjectEventWatcher(obj).Timeout(5*time.Second).WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(event).To(BeNil(), "virt-handler tried to sync on a VM in final state")

				close(done)
			}, 50)
		})
	})
	AfterEach(func() {
		tests.MustCleanup()
	})
})

func renderPkillAllVmsJob() *kubev1.Pod {
	job := kubev1.Pod{
		ObjectMeta: kubev1.ObjectMeta{
			GenerateName: "vm-killer",
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers: []kubev1.Container{
				{
					Name:  "vm-killer",
					Image: "kubevirt/vm-killer:devel",
					Command: []string{
						"pkill",
						"-9",
						"qemu",
					},
					SecurityContext: &kubev1.SecurityContext{
						Privileged: newBool(true),
						RunAsUser:  new(int64),
					},
				},
			},
			HostPID: true,
			SecurityContext: &kubev1.PodSecurityContext{
				RunAsUser: new(int64),
			},
		},
	}

	return &job
}

func pkillAllVms(core *kubernetes.Clientset, node string) error {
	job := renderPkillAllVmsJob()
	job.Spec.NodeName = node
	_, err := core.Pods(kubev1.NamespaceDefault).Create(job)

	return err
}

func newBool(x bool) *bool {
	return &x
}
