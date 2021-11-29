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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	cd "kubevirt.io/kubevirt/tests/containerdisk"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = Describe("[sig-compute]Subresource Api", func() {

	var err error
	var virtCli kubecli.KubevirtClient

	manual := v1.RunStrategyManual
	restartOnError := v1.RunStrategyRerunOnFailure

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Describe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization", func() {
		var resource string
		BeforeEach(func() {
			vm := tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			resource = vm.Name
			vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with correct permissions", func() {
			It("[test_id:3170]should be allowed to access subresource endpoint", func() {
				testClientJob(virtCli, true, resource)
			}, 15)
		})
		Context("Without permissions", func() {
			It("[test_id:3171]should not be able to access subresource endpoint", func() {
				testClientJob(virtCli, false, resource)
			}, 15)
		})
	})

	Describe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization For Version Command", func() {
		resource := "version"

		Context("with authenticated user", func() {
			It("[test_id:3172]should be allowed to access subresource version endpoint", func() {
				testClientJob(virtCli, true, resource)
			})
		})
		Context("Without permissions", func() {
			It("[test_id:3173]should be able to access subresource version endpoint", func() {
				testClientJob(virtCli, false, resource)
			})
		})
	})

	Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component] VirtualMachine subresource", func() {
		Context("with a restart endpoint", func() {
			It("[test_id:1304] should restart a VM", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).NotTo(HaveOccurred())

				tests.StartVirtualMachine(vm)
				vmi, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))

				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vm.Name, &v1.RestartOptions{})
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil || vmi.UID == newVMI.UID {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			})

			It("[test_id:1305][posneg:negative] should return an error when VM is not running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).NotTo(HaveOccurred())

				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vm.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:2265][posneg:negative] should return an error when VM has not been found but VMI is running", func() {
				vmi := tests.NewRandomVMI()
				tests.RunVMIAndExpectLaunch(vmi, 60)

				err := virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vmi.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With manual RunStrategy", func() {
			It("[test_id:3174]Should not restart when VM is not running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm.Spec.RunStrategy = &manual
				vm.Spec.Running = nil

				By("Creating VM")
				vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Trying to start VM via Restart subresource")
				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vm.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:3175]Should restart when VM is running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm.Spec.RunStrategy = &manual
				vm.Spec.Running = nil

				By("Creating VM")
				vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Starting VM via Start subresource")
				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Start(vm.Name, &v1.StartOptions{Paused: false})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))

				vmi, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Restarting VM")
				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vm.Name, &v1.RestartOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil || vmi.UID == newVMI.UID {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			})
		})

		Context("With RunStrategy RerunOnFailure", func() {
			It("[test_id:3176]Should restart the VM", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm.Spec.RunStrategy = &restartOnError
				vm.Spec.Running = nil

				By("Creating VM")
				vm, err := virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))

				vmi, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Restarting VM")
				err = virtCli.VirtualMachine(util.NamespaceTestDefault).Restart(vm.Name, &v1.RestartOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil || vmi.UID == newVMI.UID {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			})
		})
	})

	Describe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] the openapi spec for the subresources", func() {
		It("[test_id:3177]should be aggregated into the apiserver openapi spec", func() {
			Eventually(func() string {
				spec, err := virtCli.RestClient().Get().AbsPath("/openapi/v2").DoRaw(context.Background())
				Expect(err).ToNot(HaveOccurred())
				return string(spec)
				// The first item in the SubresourceGroupVersions array is the preferred version
			}, 60*time.Second, 1*time.Second).Should(ContainSubstring("subresources.kubevirt.io/" + v1.SubresourceGroupVersions[0].Version))
		})
	})

	Describe("VirtualMachineInstance subresource", func() {
		Context("Freeze Unfreeze should fail", func() {
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				var err error
				vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "#!/bin/bash\necho 'hello'\n")
				vm = tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtCli.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, time.Second).Should(BeTrue())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 180)
			})

			It("[test_id:7476]Freeze without guest agent", func() {
				expectedErr := "Internal error occurred"
				err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Freeze(vm.Name, 0)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			})

			It("[test_id:7477]Unfreeze without guest agent", func() {
				expectedErr := "Internal error occurred"
				err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Unfreeze(vm.Name)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			})
		})

		Context("Freeze Unfreeze commands", func() {
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				var err error
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Namespace = util.NamespaceTestDefault
				vm = tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtCli.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, time.Second).Should(BeTrue())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 300)
				tests.WaitAgentConnected(virtCli, vmi)
			})

			waitVMIFSFreezeStatus := func(expectedStatus string) {
				Eventually(func() bool {
					updatedVMI, err := virtCli.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == expectedStatus
				}, 30*time.Second, 2*time.Second).Should(BeTrue())
			}

			It("[test_id:7479]Freeze Unfreeze should succeed", func() {
				By("Freezing VMI")
				err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Freeze(vm.Name, 0)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("frozen")

				By("Unfreezing VMI")
				err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Unfreeze(vm.Name)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("")
			})

			It("[test_id:7480]Multi Freeze Unfreeze calls should succeed", func() {
				for i := 0; i < 5; i++ {
					By("Freezing VMI")
					err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Freeze(vm.Name, 0)
					Expect(err).ToNot(HaveOccurred())

					waitVMIFSFreezeStatus("frozen")
				}

				By("Unfreezing VMI")
				for i := 0; i < 5; i++ {
					err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Unfreeze(vm.Name)
					Expect(err).ToNot(HaveOccurred())

					waitVMIFSFreezeStatus("")
				}
			})

			It("Freeze without Unfreeze should trigger unfreeze after timeout", func() {
				By("Freezing VMI")
				unfreezeTimeout := 10 * time.Second
				err = virtCli.VirtualMachineInstance(util.NamespaceTestDefault).Freeze(vm.Name, unfreezeTimeout)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("frozen")

				By("Wait Unfreeze VMI to be triggered")
				waitVMIFSFreezeStatus("")
			})
		})
	})
})

func testClientJob(virtCli kubecli.KubevirtClient, withServiceAccount bool, resource string) {
	namespace := util.NamespaceTestDefault
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
					Image:   fmt.Sprintf("%s/subresource-access-test:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag),
					Command: []string{"/subresource-access-test", "-n", namespace, resource},
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

	pod, err := virtCli.CoreV1().Pods(namespace).Create(context.Background(), job, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtCli.CoreV1().Pods(namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}

	Eventually(getStatus, 60, 0.5).Should(Equal(expectedPhase))
}
