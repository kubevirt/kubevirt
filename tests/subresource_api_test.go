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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Subresource Api", func() {

	flag.Parse()

	virtCli, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("Rbac Authorization", func() {
		resource := "virtualmachineinstances"

		Context("with correct permissions", func() {
			It("should be allowed to access subresource endpoint", func() {
				testClientJob(virtCli, true, resource)
			}, 15)
		})
		Context("Without permissions", func() {
			It("should not be able to access subresource endpoint", func() {
				testClientJob(virtCli, false, resource)
			}, 15)
		})
	})

	Describe("Rbac Authorization For Version Command", func() {
		resource := "version"

		Context("with authenticated user", func() {
			It("should be allowed to access subresource version endpoint", func() {
				testClientJob(virtCli, true, resource)
			}, 15)
		})
		Context("Without permissions", func() {
			It("should be able to access subresource version endpoint", func() {
				testClientJob(virtCli, false, resource)
			}, 15)
		})
	})

	Describe("VirtualMachine subresource", func() {
		Context("with a restart endpoint", func() {
			It("should restart a VM", func() {
				vm := NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).NotTo(HaveOccurred())

				tests.StartVirtualMachine(vm)
				vmi, err := virtCli.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))

				err = virtCli.VirtualMachine(tests.NamespaceTestDefault).Restart(vm.Name)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil || vmi.UID == newVMI.UID {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			})

			It("should return an error when VM is not running", func() {
				vm := NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).NotTo(HaveOccurred())

				err = virtCli.VirtualMachine(tests.NamespaceTestDefault).Restart(vm.Name)
				Expect(err).To(HaveOccurred())
			})

			It("should return an error when VM has not been found but VMI is running", func() {
				vmi := tests.NewRandomVMI()
				tests.RunVMIAndExpectLaunch(vmi, false, 60)

				err := virtCli.VirtualMachine(tests.NamespaceTestDefault).Restart(vmi.Name)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("the openapi spec for the subresources", func() {
		It("should be aggregated into the the apiserver openapi spec", func() {
			Eventually(func() string {
				spec, err := virtCli.RestClient().Get().AbsPath("/openapi/v2").DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(spec)
			}, 60*time.Second, 1*time.Second).Should(ContainSubstring("subresources.kubevirt.io"))
		})
	})
})

func testClientJob(virtCli kubecli.KubevirtClient, withServiceAccount bool, resource string) {
	namespace := tests.NamespaceTestDefault
	expectedPhase := k8sv1.PodFailed
	name := "subresource-access-tester"
	job := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				v1.AppLabel: tests.SubresourceTestLabel,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   fmt.Sprintf("%s/subresource-access-test:%s", tests.KubeVirtRepoPrefix, tests.KubeVirtVersionTag),
					Command: []string{"/subresource-access-test", resource},
				},
			},
		},
	}

	if withServiceAccount {
		job.Spec.ServiceAccountName = tests.SubresourceServiceAccountName
		expectedPhase = k8sv1.PodSucceeded
	} else if resource == "version" {
		expectedPhase = k8sv1.PodSucceeded
	}

	pod, err := virtCli.CoreV1().Pods(namespace).Create(job)
	Expect(err).ToNot(HaveOccurred())

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtCli.CoreV1().Pods(namespace).Get(pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(expectedPhase))
}
