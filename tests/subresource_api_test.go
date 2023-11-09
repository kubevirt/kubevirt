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

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Subresource Api", decorators.SigCompute, func() {
	var err error
	var virtCli kubecli.KubevirtClient

	manual := v1.RunStrategyManual
	restartOnError := v1.RunStrategyRerunOnFailure

	BeforeEach(func() {
		virtCli = kubevirt.Client()
	})

	Describe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
			vm, err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with correct permissions", func() {
			It("[test_id:3170]should be allowed to access subresource endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceServiceAccountName)
				err := saClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Without permissions", func() {
			It("[test_id:3171]should not be able to access subresource endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceUnprivilegedServiceAccountName)
				err := saClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).To(HaveOccurred())
				Expect(errors.ReasonForError(err)).To(Equal(metav1.StatusReasonForbidden))
			})
		})
	})

	Describe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization For Version Command", func() {
		Context("with authenticated user", func() {
			It("[test_id:3172]should be allowed to access subresource version endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceServiceAccountName)
				_, err := saClient.ServerVersion().Get()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Without permissions", func() {
			It("[test_id:3173]should be able to access subresource version endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceUnprivilegedServiceAccountName)
				_, err := saClient.ServerVersion().Get()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization For Guestfs Command", func() {
		Context("with authenticated user", func() {
			It("should be allowed to access subresource guestfs endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceServiceAccountName)
				_, err := saClient.GuestfsVersion().Get()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Without permissions", func() {
			It("should be able to access subresource guestfs endpoint", func() {
				saClient := getClientForSA(virtCli, testsuite.SubresourceUnprivilegedServiceAccountName)
				_, err := saClient.GuestfsVersion().Get()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac Authorization For Expand-Spec Command", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-test",
					Namespace: testsuite.GetTestNamespace(nil),
				},
				Spec: v1.VirtualMachineSpec{
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: v1.VirtualMachineInstanceSpec{
							Domain: v1.DomainSpec{
								Devices: v1.Devices{},
							},
						},
					},
				},
			}
			vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
		})

		It("should be allowed to access expand-vm-spec endpoint with authenticated user", func() {
			saClient := getClientForSA(virtCli, testsuite.SubresourceServiceAccountName)
			_, err = saClient.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not be able to access expand-vm-spec endpoint without authenticated user", func() {
			saClient := getClientForSA(virtCli, testsuite.SubresourceUnprivilegedServiceAccountName)
			_, err = saClient.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(vm)
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).To(Equal(metav1.StatusReasonForbidden))
		})
	})

	Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component] VirtualMachine subresource", func() {
		Context("with a restart endpoint", func() {
			It("[test_id:1304] should restart a VM", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).NotTo(HaveOccurred())

				tests.StartVirtualMachine(vm)
				vmi, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))

				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if err != nil || vmi.UID == newVMI.UID {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			})

			It("[test_id:1305][posneg:negative] should return an error when VM is not running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).NotTo(HaveOccurred())

				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:2265][posneg:negative] should return an error when VM has not been found but VMI is running", func() {
				vmi := tests.NewRandomVMI()
				tests.RunVMIAndExpectLaunch(vmi, 60)

				err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vmi)).Restart(context.Background(), vmi.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With manual RunStrategy", func() {
			It("[test_id:3174]Should not restart when VM is not running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm.Spec.RunStrategy = &manual
				vm.Spec.Running = nil

				By("Creating VM")
				vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Trying to start VM via Restart subresource")
				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:3175]Should restart when VM is running", func() {
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
				vm.Spec.RunStrategy = &manual
				vm.Spec.Running = nil

				By("Creating VM")
				vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Starting VM via Start subresource")
				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Start(context.Background(), vm.Name, &v1.StartOptions{Paused: false})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if err != nil {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))

				vmi, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Restarting VM")
				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
				vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if err != nil {
						return v1.VmPhaseUnset
					}
					return newVMI.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))

				vmi, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Restarting VM")
				err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() v1.VirtualMachineInstancePhase {
					newVMI, err := virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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
				vmi := libvmi.NewCirros()
				vm = tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtCli.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, time.Second).Should(BeTrue())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(180),
				)
			})

			It("[test_id:7476]Freeze without guest agent", func() {
				expectedErr := "Internal error occurred"
				err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Freeze(context.Background(), vm.Name, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			})

			It("[test_id:7477]Unfreeze without guest agent", func() {
				expectedErr := "Internal error occurred"
				err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Unfreeze(context.Background(), vm.Name)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			})
		})

		Context("Freeze Unfreeze commands", func() {
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				var err error
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(vmi)
				vm = tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtCli.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, time.Second).Should(BeTrue())
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			})

			waitVMIFSFreezeStatus := func(expectedStatus string) {
				Eventually(func() bool {
					updatedVMI, err := virtCli.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == expectedStatus
				}, 30*time.Second, 2*time.Second).Should(BeTrue())
			}

			It("[test_id:7479]Freeze Unfreeze should succeed", func() {
				By("Freezing VMI")
				err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Freeze(context.Background(), vm.Name, 0)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("frozen")

				By("Unfreezing VMI")
				err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Unfreeze(context.Background(), vm.Name)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("")
			})

			It("[test_id:7480]Multi Freeze Unfreeze calls should succeed", func() {
				for i := 0; i < 5; i++ {
					By("Freezing VMI")
					err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Freeze(context.Background(), vm.Name, 0)
					Expect(err).ToNot(HaveOccurred())

					waitVMIFSFreezeStatus("frozen")
				}

				By("Unfreezing VMI")
				for i := 0; i < 5; i++ {
					err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Unfreeze(context.Background(), vm.Name)
					Expect(err).ToNot(HaveOccurred())

					waitVMIFSFreezeStatus("")
				}
			})

			It("Freeze without Unfreeze should trigger unfreeze after timeout", func() {
				By("Freezing VMI")
				unfreezeTimeout := 10 * time.Second
				err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Freeze(context.Background(), vm.Name, unfreezeTimeout)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus("frozen")

				By("Wait Unfreeze VMI to be triggered")
				waitVMIFSFreezeStatus("")
			})
		})
	})

	Describe("ExpandSpec subresource", func() {
		Context("instancetype", func() {
			var (
				instancetype               *instancetypev1beta1.VirtualMachineInstancetype
				clusterInstancetype        *instancetypev1beta1.VirtualMachineClusterInstancetype
				instancetypeMatcher        *v1.InstancetypeMatcher
				clusterInstancetypeMatcher *v1.InstancetypeMatcher
				expectedCpu                *v1.CPU

				instancetypeMatcherFn = func() *v1.InstancetypeMatcher {
					return instancetypeMatcher
				}
				clusterInstancetypeMatcherFn = func() *v1.InstancetypeMatcher {
					return clusterInstancetypeMatcher
				}
			)

			BeforeEach(func() {
				instancetype = newVirtualMachineInstancetype(nil)
				instancetype.Spec.CPU.Guest = 2
				instancetype, err = virtCli.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
					Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				instancetypeMatcher = &v1.InstancetypeMatcher{
					Name: instancetype.Name,
					Kind: instancetypeapi.SingularResourceName,
				}

				clusterInstancetype = newVirtualMachineClusterInstancetype(nil)
				clusterInstancetype.Spec.CPU.Guest = 2
				clusterInstancetype, err = virtCli.VirtualMachineClusterInstancetype().
					Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				clusterInstancetypeMatcher = &v1.InstancetypeMatcher{
					Name: clusterInstancetype.Name,
					Kind: instancetypeapi.ClusterSingularResourceName,
				}

				expectedCpu = &v1.CPU{
					Sockets: 2,
					Cores:   1,
					Threads: 1,
				}
			})

			AfterEach(func() {
				err = virtCli.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
					Delete(context.Background(), instancetype.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = virtCli.VirtualMachineClusterInstancetype().
					Delete(context.Background(), clusterInstancetype.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			Context("with existing VM", func() {
				It("[test_id:TODO] should return unchanged VirtualMachine, if instancetype is not used", func() {
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
					Expect(err).ToNot(HaveOccurred())

					expandedVm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).
						GetWithExpandedSpec(context.Background(), vm.GetName())
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec).To(Equal(vm.Spec))
				})

				DescribeTable("[test_id:TODO] should return VirtualMachine with instancetype expanded", func(matcherFn func() *v1.InstancetypeMatcher) {
					vm := tests.NewRandomVirtualMachine(libvmi.New(), false)
					vm.Spec.Instancetype = matcherFn()

					vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
					Expect(err).ToNot(HaveOccurred())

					expandedVm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).
						GetWithExpandedSpec(context.Background(), vm.GetName())
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec.Instancetype).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
					Expect(expandedVm.Spec.Template.Spec.Domain.CPU).To(Equal(expectedCpu), "VM should have instancetype expanded")
				},
					Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
					Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
				)
			})

			Context("with passed VM in request", func() {
				It("[test_id:TODO] should return unchanged VirtualMachine, if instancetype is not used", func() {
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)

					expandedVm, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec).To(Equal(vm.Spec))
				})

				DescribeTable("[test_id:TODO] should return VirtualMachine with instancetype expanded", func(matcherFn func() *v1.InstancetypeMatcher) {
					vm := tests.NewRandomVirtualMachine(libvmi.New(), false)
					vm.Spec.Instancetype = matcherFn()

					expandedVm, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec.Instancetype).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
					Expect(expandedVm.Spec.Template.Spec.Domain.CPU).To(Equal(expectedCpu), "VM should have instancetype expanded")
				},
					Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
					Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
				)

				DescribeTable("[test_id:TODO] should fail, if referenced instancetype does not exist", func(matcher *v1.InstancetypeMatcher) {
					vm := tests.NewRandomVirtualMachine(libvmi.New(), false)
					vm.Spec.Instancetype = matcher

					_, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(matcher.Kind + ".instancetype.kubevirt.io \"" + matcher.Name + "\" not found"))
				},
					Entry("with VirtualMachineInstancetype", &v1.InstancetypeMatcher{Name: "nonexisting-instancetype", Kind: instancetypeapi.PluralResourceName}),
					Entry("with VirtualMachineClusterInstancetype", &v1.InstancetypeMatcher{Name: "nonexisting-clusterinstancetype", Kind: instancetypeapi.ClusterPluralResourceName}),
				)

				DescribeTable("[test_id:TODO] should fail, if instancetype expansion hits a conflict", func(matcherFn func() *v1.InstancetypeMatcher) {
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm.Spec.Instancetype = matcherFn()

					_, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("cannot expand instancetype to VM"))
				},
					Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
					Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
				)

				DescribeTable("[test_id:TODO] should fail, if VM and endpoint namespace are different", func(matcherFn func() *v1.InstancetypeMatcher) {
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm.Spec.Instancetype = matcherFn()
					vm.Namespace = "madethisup"

					_, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(vm)
					Expect(err).To(HaveOccurred())
					errMsg := fmt.Sprintf("VM namespace must be empty or %s", testsuite.GetTestNamespace(nil))
					Expect(err).To(MatchError(errMsg))
				},
					Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
					Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
				)
			})
		})

		Context("preference", func() {
			var (
				preference        *instancetypev1beta1.VirtualMachinePreference
				clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference

				preferenceMatcher        *v1.PreferenceMatcher
				clusterPreferenceMatcher *v1.PreferenceMatcher

				preferenceMatcherFn = func() *v1.PreferenceMatcher {
					return preferenceMatcher
				}
				clusterPreferenceMatcherFn = func() *v1.PreferenceMatcher {
					return clusterPreferenceMatcher
				}
			)

			BeforeEach(func() {
				preference = newVirtualMachinePreference()
				preference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
					PreferredAutoattachGraphicsDevice: pointer.Bool(true),
				}
				preference, err = virtCli.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
					Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				preferenceMatcher = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: instancetypeapi.SingularPreferenceResourceName,
				}

				clusterPreference = newVirtualMachineClusterPreference()
				clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
					PreferredAutoattachGraphicsDevice: pointer.Bool(true),
				}
				clusterPreference, err = virtCli.VirtualMachineClusterPreference().
					Create(context.Background(), clusterPreference, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				clusterPreferenceMatcher = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
				}
			})

			AfterEach(func() {
				err = virtCli.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
					Delete(context.Background(), preference.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				err = virtCli.VirtualMachineClusterPreference().
					Delete(context.Background(), clusterPreference.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			Context("with existing VM", func() {
				It("[test_id:TODO] should return unchanged VirtualMachine, if preference is not used", func() {
					// Using NewCirros() here to have some data in spec.
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)

					vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
					Expect(err).ToNot(HaveOccurred())

					expandedVm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).
						GetWithExpandedSpec(context.Background(), vm.GetName())
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec).To(Equal(vm.Spec))
				})

				DescribeTable("[test_id:TODO] should return VirtualMachine with preference expanded", func(matcherFn func() *v1.PreferenceMatcher) {
					// Using NewCirros() here to have some data in spec.
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm.Spec.Preference = matcherFn()

					vm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
					Expect(err).ToNot(HaveOccurred())

					expandedVm, err := virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).
						GetWithExpandedSpec(context.Background(), vm.GetName())
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec.Preference).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
					Expect(*expandedVm.Spec.Template.Spec.Domain.Devices.AutoattachGraphicsDevice).To(BeTrue(), "VM should have preference expanded")
				},
					Entry("with VirtualMachinePreference", preferenceMatcherFn),
					Entry("with VirtualMachineClusterPreference", clusterPreferenceMatcherFn),
				)
			})

			Context("with passed VM in request", func() {
				It("[test_id:TODO] should return unchanged VirtualMachine, if preference is not used", func() {
					// Using NewCirros() here to have some data in spec.
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)

					expandedVm, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec).To(Equal(vm.Spec))
				})

				DescribeTable("[test_id:TODO] should return VirtualMachine with preference expanded", func(matcherFn func() *v1.PreferenceMatcher) {
					// Using NewCirros() here to have some data in spec.
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm.Spec.Preference = matcherFn()

					expandedVm, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).ToNot(HaveOccurred())
					Expect(expandedVm.Spec.Preference).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
					Expect(*expandedVm.Spec.Template.Spec.Domain.Devices.AutoattachGraphicsDevice).To(BeTrue(), "VM should have preference expanded")
				},
					Entry("with VirtualMachinePreference", preferenceMatcherFn),
					Entry("with VirtualMachineClusterPreference", clusterPreferenceMatcherFn),
				)

				DescribeTable("[test_id:TODO] should fail, if referenced preference does not exist", func(matcher *v1.PreferenceMatcher) {
					// Using NewCirros() here to have some data in spec.
					vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
					vm.Spec.Preference = matcher

					_, err := virtCli.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(matcher.Kind + ".instancetype.kubevirt.io \"" + matcher.Name + "\" not found"))
				},
					Entry("with VirtualMachinePreference", &v1.PreferenceMatcher{Name: "nonexisting-preference", Kind: instancetypeapi.PluralPreferenceResourceName}),
					Entry("with VirtualMachineClusterPreference", &v1.PreferenceMatcher{Name: "nonexisting-clusterpreference", Kind: instancetypeapi.ClusterPluralPreferenceResourceName}),
				)
			})
		})
	})
})

func getClientForSA(virtCli kubecli.KubevirtClient, saName string) kubecli.KubevirtClient {
	secret, err := virtCli.CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Get(context.Background(), saName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	token, ok := secret.Data["token"]
	Expect(ok).To(BeTrue())

	saClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
		Host: virtCli.Config().Host,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
		BearerToken: string(token),
	})
	Expect(err).ToNot(HaveOccurred())

	return saClient
}
