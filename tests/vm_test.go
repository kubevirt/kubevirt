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
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

var _ = Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("A valid VirtualMachine given", func() {
		type vmiBuilder func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume)

		newVirtualMachineInstanceWithDV := func(imgUrl, sc string, volumeMode k8sv1.PersistentVolumeMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			Expect(libstorage.HasCDI()).To(BeTrue(), "Skip DataVolume tests when CDI is not present")

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(imgUrl, cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(imgUrl)),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					libdv.StorageWithVolumeMode(volumeMode),
				),
			)

			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			return libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithDataVolume("disk0", dataVolume.Name),
				libvmi.WithResourceMemory("256Mi"),
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				libvmi.WithTerminationGracePeriod(30),
			), dataVolume
		}

		newVirtualMachineInstanceWithFileDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			sc, foundSC := libstorage.GetRWOFileSystemStorageClass()
			Expect(foundSC).To(BeTrue(), "Filesystem storage is not present")
			return newVirtualMachineInstanceWithDV(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), sc, k8sv1.PersistentVolumeFilesystem)
		}

		newVirtualMachineInstanceWithBlockDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			sc, foundSC := libstorage.GetRWOBlockStorageClass()
			if !foundSC {
				Skip("Skip test when Block storage is not present")
			}
			return newVirtualMachineInstanceWithDV(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), sc, k8sv1.PersistentVolumeBlock)
		}

		validateGenerationState := func(vm *v1.VirtualMachine, expectedGeneration int, expectedDesiredGeneration int, expectedObservedGeneration int, expectedGenerationAnnotation int) {
			By("By validating the generation states")
			EventuallyWithOffset(1, func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				By("Expecting the generation to match")
				g.Expect(vm.Generation).To(Equal(int64(expectedGeneration)))

				By("Expecting the generation state in the vm status to match")
				g.Expect(vm.Status.DesiredGeneration).To(Equal(int64(expectedDesiredGeneration)))
				g.Expect(vm.Status.ObservedGeneration).To(Equal(int64(expectedObservedGeneration)))

				By("Expecting the generation annotation on the vmi to match")
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				g.Expect(vmi.Annotations).Should(HaveKeyWithValue(v1.VirtualMachineGenerationAnnotation, fmt.Sprintf("%v", expectedGenerationAnnotation)))
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		}

		DescribeTable("cpu/memory in requests/limits should allow", func(cpu, request string) {
			const oldCpu = "222"
			const oldMemory = "2222222"

			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
			vm.Namespace = testsuite.GetTestNamespace(vm)
			vm.APIVersion = "kubevirt.io/" + v1.ApiStorageVersion
			vm.Spec.Template.Spec.Domain.Resources.Limits = make(k8sv1.ResourceList)
			vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(oldCpu)
			vm.Spec.Template.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse(oldCpu)
			vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(oldMemory)
			vm.Spec.Template.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = resource.MustParse(oldMemory)

			jsonBytes, err := json.Marshal(vm)
			Expect(err).NotTo(HaveOccurred())

			match := func(str string) string {
				return fmt.Sprintf("\"%s\"", str)
			}

			jsonString := strings.Replace(string(jsonBytes), match(oldCpu), cpu, -1)
			jsonString = strings.Replace(jsonString, match(oldMemory), request, -1)

			By("Verify VM can be created")
			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(testsuite.GetTestNamespace(vm)).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do(context.Background())
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusCreated))

			By("Verify VM will run")
			libvmops.StartVirtualMachine(vm)
		},
			Entry("int type", "2", "2222222"),
			Entry("float type", "2.2", "2222222.2"),
		)

		It("[test_id:3161]should carry vm.template.spec.annotations to VMI and ignore vm ones", decorators.Conformance, func() {
			vm := libvmi.NewVirtualMachine(
				libvmifact.NewGuestless(libvmi.WithAnnotation("test.vm.template.spec.annotation", "propagated")),
			)
			vm.Annotations = map[string]string{"test.vm.annotation": "nopropagated"}
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm = libvmops.StartVirtualMachine(vm)

			By("checking for annotations propagation")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(And(
				HaveKeyWithValue("test.vm.template.spec.annotation", "propagated"),
				Not(HaveKey("test.vm.annotation")),
			))
		})

		It("should sync the generation annotation on the vmi during restarts", decorators.Conformance, func() {
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			for i := 1; i <= 3; i++ {
				// Generation increases twice for each pass, since there is a stop and a
				// start.
				expectedGeneration := i * 2

				validateGenerationState(vm, expectedGeneration, expectedGeneration, expectedGeneration, expectedGeneration)

				By("Restarting the VM")
				vm = libvmops.StartVirtualMachine(libvmops.StopVirtualMachine(vm))
			}
		})

		It("should not update the vmi generation annotation when the template changes", decorators.Conformance, func() {
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			By("Updating the VM template metadata")
			labelsPatch, err := patch.New(
				patch.WithReplace("/spec/template/metadata/labels",
					map[string]string{"testkey": "testvalue"})).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, labelsPatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			validateGenerationState(vm, 3, 3, 2, 2)

			By("Updating the VM template spec")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			labelsPatch, err = patch.New(
				patch.WithAdd("/spec/template/metadata/labels/testkey2", "testvalue2")).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, labelsPatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			validateGenerationState(vm, 4, 4, 2, 2)

			// Restart the VM to check that the state will once again sync.
			By("Restarting the VM")
			vm = libvmops.StartVirtualMachine(libvmops.StopVirtualMachine(vm))

			validateGenerationState(vm, 6, 6, 6, 6)
		})

		DescribeTable("[test_id:1521]should remove VirtualMachineInstance once the VM is marked for deletion", decorators.Conformance, func(createTemplate vmiBuilder, ensureGracefulTermination bool) {
			template, _ := createTemplate()
			vm := libvmops.StartVirtualMachine(createVM(virtClient, template))
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Delete it
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())
			// Wait until VMI is gone
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 2*time.Second).ShouldNot(Exist())
			if !ensureGracefulTermination {
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Under default settings, termination is graceful (shutdown instead of destroy)
			event := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(75*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.ShuttingDown)
			Expect(event).ToNot(BeNil(), "There should be a shutdown event")
		},
			Entry("with ContainerDisk", func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) { return libvmifact.NewCirros(), nil }, false),
			Entry("[storage-req]with Filesystem Disk", decorators.StorageReq, newVirtualMachineInstanceWithFileDisk, true),
			Entry("[storage-req]with Block Disk", decorators.StorageReq, newVirtualMachineInstanceWithBlockDisk, false),
		)

		It("[test_id:1522]should remove owner references on the VirtualMachineInstance if it is orphan deleted", decorators.Conformance, func() {
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			By("Getting owner references")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).ToNot(BeEmpty())

			By("Deleting VM")
			Expect(virtClient.VirtualMachine(vm.Namespace).
				Delete(context.Background(), vm.Name, metav1.DeleteOptions{PropagationPolicy: pointer.P(metav1.DeletePropagationOrphan)})).To(Succeed())
			// Wait until the virtual machine is deleted
			By("Waiting for VM to delete")
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).ShouldNot(Exist())

			By("Verifying orphaned VMI still exists")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).To(BeEmpty())
		})

		It("[test_id:1523]should recreate VirtualMachineInstance if it gets deleted", decorators.Conformance, func() {
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(virtClient.VirtualMachineInstance(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())

			Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeRestarted(vmi.UID))
		})

		It("[test_id:1524]should recreate VirtualMachineInstance if the VirtualMachineInstance's pod gets deleted", decorators.Conformance, func() {
			By("Start a new VM")
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))
			firstVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// get the pod backing the VirtualMachineInstance
			By("Getting the pod backing the VirtualMachineInstance")
			firstPod, err := libpod.GetPodByVirtualMachineInstance(firstVMI, firstVMI.Namespace)
			Expect(err).ToNot(HaveOccurred())

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Expect(virtClient.CoreV1().Pods(vm.Namespace).Delete(context.Background(), firstPod.Name, metav1.DeleteOptions{})).Should(Succeed())

			// Wait on the VMI controller to create a new VirtualMachineInstance
			By("Waiting for a new VirtualMachineInstance to spawn")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRestarted(firstVMI.UID))

			// sanity check that the test ran correctly by
			// verifying a different Pod backs the VMI as well.
			By("Verifying a new pod backs the VMI")
			currentVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(currentVMI, testsuite.GetTestNamespace(currentVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(vmiPod.Name).ToNot(Equal(firstPod.Name))
		})

		DescribeTable("[test_id:1525]should stop VirtualMachineInstance if running set to false", func(createTemplate vmiBuilder) {
			template, _ := createTemplate()
			libvmops.StopVirtualMachine(libvmops.StartVirtualMachine(createVM(virtClient, template)))
		},
			Entry("with ContainerDisk", func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) { return libvmifact.NewCirros(), nil }),
			Entry("[storage-req]with Filesystem Disk", decorators.StorageReq, newVirtualMachineInstanceWithFileDisk),
			Entry("[storage-req]with Block Disk", decorators.StorageReq, newVirtualMachineInstanceWithBlockDisk),
		)

		It("[test_id:1526]should start and stop VirtualMachineInstance multiple times", decorators.Conformance, func() {
			vm := createVM(virtClient, libvmifact.NewCirros())
			// Start and stop VirtualMachineInstance multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				vm = libvmops.StopVirtualMachine(libvmops.StartVirtualMachine(vm))
			}
		})

		It("[test_id:1527]should not update the VirtualMachineInstance spec if Running", decorators.Conformance, func() {
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			By("Updating the VM template spec")
			updatedVM := vm.DeepCopy()

			resourcesPatch, err := patch.New(
				patch.WithAdd("/spec/template/spec/domain/resources",
					map[string]map[string]string{
						"requests": {
							"memory": "4096Ki",
						},
					},
				)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			updatedVM, err = virtClient.VirtualMachine(updatedVM.Namespace).Patch(context.Background(), updatedVM.Name, types.JSONPatchType, resourcesPatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VirtualMachineInstance spec still running")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory := vm.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))

			By("Restarting the VM")
			vm = libvmops.StartVirtualMachine(libvmops.StopVirtualMachine(vm))

			By("Expecting updated spec running")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory = vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory = updatedVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))
		})

		It("[test_id:1528]should survive guest shutdown, multiple times", decorators.Conformance, func() {
			vm := createRunningVM(virtClient, libvmifact.NewCirros())
			Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())

			for i := 0; i < 3; i++ {
				By("Getting the running VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Obtaining the serial console")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Guest shutdown")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 240)).To(Succeed())

				By("waiting for the controller to replace the shut-down vmi with a new instance")
				Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeRestarted(vmi.UID))
			}
		})

		It("should always have updated vm revision when starting vm", decorators.Conformance, func() {
			By("Starting the VM")
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVMRevisionName := fmt.Sprintf("revision-start-vm-%s-%d", vm.UID, vm.Generation)
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(expectedVMRevisionName))
			oldVMRevisionName := expectedVMRevisionName

			cr, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vmi.Status.VirtualMachineRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Revision).To(Equal(int64(2)))
			vmRevision := &v1.VirtualMachine{}
			err = json.Unmarshal(cr.Data.Raw, vmRevision)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRevision.Spec).To(Equal(vm.Spec))

			By("Stopping the VM")
			vm = libvmops.StopVirtualMachine(vm)

			Eventually(func() error {
				resourcesPatch, err := patch.New(
					patch.WithAdd("/spec/template/spec/domain/resources",
						map[string]map[string]string{
							"requests": {
								"memory": "4096Ki",
							},
						},
					)).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, resourcesPatch, metav1.PatchOptions{})
				return err
			}, 10*time.Second, time.Second).ShouldNot(HaveOccurred())

			By("Starting the VM after update")
			vm = libvmops.StartVirtualMachine(vm)

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVMRevisionName = fmt.Sprintf("revision-start-vm-%s-%d", vm.UID, vm.Generation)
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(expectedVMRevisionName))

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), oldVMRevisionName, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

			cr, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vmi.Status.VirtualMachineRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Revision).To(Equal(int64(5)))
			vmRevision = &v1.VirtualMachine{}
			err = json.Unmarshal(cr.Data.Raw, vmRevision)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRevision.Spec).To(Equal(vm.Spec))
		})

		It("[test_id:4645]should set the Ready condition on VM", decorators.Conformance, func() {
			vm := createVM(virtClient, libvmifact.NewCirros())

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionFalse(v1.VirtualMachineReady))

			vm = libvmops.StartVirtualMachine(vm)

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionTrue(v1.VirtualMachineReady))

			vm = libvmops.StopVirtualMachine(vm)

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionFalse(v1.VirtualMachineReady))
		})

		DescribeTable("should report an error status", decorators.Conformance, func(vmi *v1.VirtualMachineInstance, expectedStatus v1.VirtualMachinePrintableStatus) {
			vm := createRunningVM(virtClient, vmi)
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HavePrintableStatus(expectedStatus))
		},
			Entry("[test_id:6867] when VM scheduling error occurs with unsatisfiable resource requirements",
				libvmi.New(
					// This may stop working sometime around 2040
					libvmi.WithResourceMemory("1Ei"),
					libvmi.WithResourceCPU("1M"),
				),
				v1.VirtualMachineStatusUnschedulable,
			),
			Entry("[test_id:6868] when VM scheduling error occurs with unsatisfiable scheduling constraints",
				libvmi.New(
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithNodeSelectorFor("that-doesnt-exist"),
				),
				v1.VirtualMachineStatusUnschedulable,
			),
			Entry(
				"[test_id:7596] when a VM with a missing PVC is started",
				libvmi.New(
					libvmi.WithPersistentVolumeClaim("disk0", "missing-pvc"),
					libvmi.WithResourceMemory("128Mi"),
				),
				v1.VirtualMachineStatusPvcNotFound,
			),
			Entry(
				"[test_id:7597] when a VM with a missing DV is started",
				libvmi.New(
					libvmi.WithDataVolume("disk0", "missing-datavolume"),
					libvmi.WithResourceMemory("128Mi"),
				),
				v1.VirtualMachineStatusPvcNotFound,
			),
		)

		It("[test_id:6869][QUARANTINE]should report an error status when image pull error occurs", decorators.Conformance, decorators.Quarantine, func() {
			vmi := libvmi.New(
				libvmi.WithContainerDisk("disk0", "no-such-image"),
				libvmi.WithResourceMemory("128Mi"),
			)

			vm := createRunningVM(virtClient, vmi)

			By("Verifying that the status toggles between ErrImagePull and ImagePullBackOff")
			const times = 2
			for i := 0; i < times; i++ {
				Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusErrImagePull))
				Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusImagePullBackOff))
			}
		})

		It("[test_id:7679]should report an error status when data volume error occurs", func() {
			By("Verifying that required StorageClass is configured")
			storageClassName := libstorage.Config.StorageRWOFileSystem

			_, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				Skip("Skipping since required StorageClass is not configured")
			}
			Expect(err).ToNot(HaveOccurred())

			By("Creating a VM with a DataVolume cloned from an invalid source")
			// Registry URL scheme validated in CDI
			vmi, _ := newVirtualMachineInstanceWithDV("docker://no.such/image", storageClassName, k8sv1.PersistentVolumeFilesystem)
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying that the VM status eventually gets set to DataVolumeError")
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusDataVolumeError))
		})

		DescribeTable("should stop a running VM", func(runStrategy, expectedRunStrategy v1.VirtualMachineRunStrategy) {
			By("Creating a VM")
			vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(runStrategy))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			if runStrategy == v1.RunStrategyManual {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("Waiting for VM to be ready")
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

			By("Stopping the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the VirtualMachineInstance is removed")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())
			By("Ensuring stateChangeRequests list gets cleared")
			Eventually(ThisVM(vm), 30*time.Second, 1*time.Second).Should(Not(HaveStateChangeRequests()))
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(expectedRunStrategy))
		},
			Entry("[test_id:3163]with RunStrategyAlways", v1.RunStrategyAlways, v1.RunStrategyHalted),
			Entry("[test_id:2186]with RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, v1.RunStrategyRerunOnFailure),
			Entry("[test_id:2189]with RunStrategyManual", v1.RunStrategyManual, v1.RunStrategyManual),
		)

		DescribeTable("should restart a running VM", func(runStrategy v1.VirtualMachineRunStrategy) {
			By("Creating a VM")
			vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(runStrategy))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			if runStrategy == v1.RunStrategyManual {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("Waiting for VM to be ready")
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

			By("Getting VMI's UUID")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Restarting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the VirtualMachineInstance is restarted")
			Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeRestarted(vmi.UID))

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

			By("Ensuring stateChangeRequests list gets cleared")
			// StateChangeRequest might still exist until the new VMI is created
			// But it must eventually be cleared
			Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(HaveStateChangeRequests()))
		},
			Entry("[test_id:3164]with RunStrategyAlways", v1.RunStrategyAlways),
			Entry("[test_id:2187]with RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure),
			Entry("[test_id:2035]with RunStrategyManual", v1.RunStrategyManual),
		)

		DescribeTable("[test_id:1529]should start a stopped VM only once", func(runStrategy, expectedRunStrategy v1.VirtualMachineRunStrategy) {
			By("Creating a VM")
			vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(runStrategy))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(expectedRunStrategy))

			By("Ensuring stateChangeRequests list is cleared")
			Expect(vm.Status.StateChangeRequests).To(BeEmpty())

			By("Ensuring a second invocation should fail")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).To(MatchError(ContainSubstring("VM is already running")))
		},
			Entry("[test_id:2036]with RunStrategyManual", v1.RunStrategyManual, v1.RunStrategyManual),
			Entry("[test_id:2037]with RunStrategyHalted", v1.RunStrategyHalted, v1.RunStrategyAlways),
		)

		DescribeTable("should not remove a succeeded VMI", func(runStrategy v1.VirtualMachineRunStrategy, verifyFn func(*v1.VirtualMachine)) {
			By("creating a VM")
			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(runStrategy))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			if runStrategy == v1.RunStrategyManual {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("Waiting for VM to be ready")
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(console.LoginToCirros(vmi)).To(Succeed())

			By("Issuing a poweroff command from inside VM")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo poweroff\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 10)).To(Succeed())

			By("Ensuring the VirtualMachineInstance enters Succeeded phase")
			Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(HaveSucceeded())

			By("Ensuring the VirtualMachine remains stopped")
			Consistently(ThisVMI(vmi), 60*time.Second, 5*time.Second).Should(HaveSucceeded())

			By("Ensuring the VirtualMachine remains Ready=false")
			Consistently(ThisVM(vm), 60*time.Second, 5*time.Second).Should(Not(BeReady()))

			verifyFn(vm)
		},
			Entry("with RunStrategyOnce", v1.RunStrategyOnce, func(vm *v1.VirtualMachine) {
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).To(MatchError(ContainSubstring("Once does not support manual start requests")))
			}),
			Entry("[test_id:2190] with RunStrategyManual", v1.RunStrategyManual, func(vm *v1.VirtualMachine) {
				// At this point, explicitly test that a start command will delete an existing
				// VMI in the Succeeded phase.
				By("Starting the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for StartRequest to be cleared")
				Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(HaveStateChangeRequests()))

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			}),
		)

		It("[test_id:6311]should start in paused state using RunStrategyManual", func() {
			By("Creating a VM with RunStrategyManual")
			vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyManual))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the VM in paused state")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{Paused: true})
			Expect(err).ToNot(HaveOccurred())

			By("Getting the status of the VM")
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeCreated())

			By("Getting running VirtualMachineInstance with paused condition")
			Eventually(func() *v1.VirtualMachineInstance {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*vmi.Spec.StartStrategy).To(Equal(v1.StartStrategyPaused))
				Eventually(ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstancePaused))
				return vmi
			}, 240*time.Second, 1*time.Second).Should(BeInPhase(v1.Running))
		})

		Context("Using RunStrategyAlways", func() {
			It("[test_id:3165]should restart a succeeded VMI", func() {
				By("Creating a VM with RunStategyRunning")
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Issuing a poweroff command from inside VM")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 10)).To(Succeed())

				By("Ensuring the VirtualMachineInstance is restarted")
				Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeRestarted(vmi.UID))
			})

			It("[test_id:4119]should migrate a running VM", func() {
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				if len(nodes.Items) < 2 {
					Skip("Migration tests require at least 2 nodes")
				}
				By("Creating a VM with RunStrategyAlways")
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				), libvmi.WithRunStrategy(v1.RunStrategyAlways))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				By("Migrating the VM")
				err = virtClient.VirtualMachine(vm.Namespace).Migrate(context.Background(), vm.Name, &v1.MigrateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring the VirtualMachineInstance is migrated")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed
				}, 240*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("[test_id:7743]should not migrate a running vm if dry-run option is passed", func() {
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				if len(nodes.Items) < 2 {
					Skip("Migration tests require at least 2 nodes")
				}
				By("Creating a VM with RunStrategyAlways")
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				), libvmi.WithRunStrategy(v1.RunStrategyAlways))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				By("Migrating the VM with dry-run option")
				err = virtClient.VirtualMachine(vm.Namespace).Migrate(context.Background(), vm.Name, &v1.MigrateOptions{DryRun: []string{metav1.DryRunAll}})
				Expect(err).ToNot(HaveOccurred())

				By("Check that no migration was actually created")
				Consistently(func() error {
					_, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					return err
				}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "migration should not be created in a dry run mode")
			})
		})

		Context("Using RunStrategyRerunOnFailure", func() {
			It("[test_id:2188] should remove a succeeded VMI", func() {
				By("Creating a VM with RunStrategyRerunOnFailure")
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyRerunOnFailure))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Issuing a poweroff command from inside VM")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 10)).To(Succeed())

				By("Waiting for the VMI to disappear")
				Eventually(func() error {
					_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					return err
				}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "migration should not be created in a dry run mode")
			})

			It("should restart a failed VMI", func() {
				By("Creating a VM with RunStrategyRerunOnFailure")
				vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyRerunOnFailure))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to exist")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				By("Waiting for VM to start")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusRunning))

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Triggering a segfault in qemu")
				domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator := filepath.Base(domSpec.Devices.Emulator)
				libpod.RunCommandOnVmiPod(vmi, []string{"killall", "-11", emulator})

				By("Ensuring the VM stops")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusStopped))

				By("Waiting for VM to start again")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusRunning))
			})
		})

		Context("Using RunStrategyOnce", func() {
			It("Should leave a failed VMI", func() {
				By("creating a VM with RunStrategyOnce")
				vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyOnce))
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be ready")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Triggering a segfault in qemu")
				domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator := filepath.Base(domSpec.Devices.Emulator)
				libpod.RunCommandOnVmiPod(vmi, []string{"killall", "-11", emulator})

				By("Ensuring the VirtualMachineInstance enters Failed phase")
				Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeInPhase(v1.Failed))

				By("Ensuring the VirtualMachine remains stopped")
				Consistently(ThisVMI(vmi), 60*time.Second, 5*time.Second).Should(BeInPhase(v1.Failed))

				By("Ensuring the VirtualMachine remains Ready=false")
				Consistently(ThisVM(vm), 60*time.Second, 5*time.Second).Should(Not(BeReady()))
			})

			DescribeTable("with a failing VMI and the kubevirt.io/keep-launcher-alive-after-failure annotation", func(keepLauncher string) {
				By("creating a Running VM")
				vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(
					libvmi.WithAnnotation(v1.KeepLauncherAfterFailureAnnotation, keepLauncher),
					libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
				), libvmi.WithRunStrategy(v1.RunStrategyOnce))

				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to fail")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 480*time.Second, 1*time.Second).Should(BeInPhase(v1.Failed))

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				getComputeContainerStateRunning := func() (*k8sv1.ContainerStateRunning, error) {
					launcherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vm.Namespace)
					if err != nil {
						return nil, err
					}
					for _, status := range launcherPod.Status.ContainerStatuses {
						if status.Name == "compute" {
							return status.State.Running, nil
						}
					}
					return nil, nil
				}

				// If the annotation v1.KeepLauncherAfterFailureAnnotation is set to true, the containerStatus of the
				// compute container of the virt-launcher pod is kept in the running state.
				// If the annotation v1.KeepLauncherAfterFailureAnnotation is set to false or not set, the virt-launcher pod will become failed.
				By("Verify that the virt-launcher pod or its container is in the expected state")
				if toKeep, _ := strconv.ParseBool(keepLauncher); toKeep {
					Consistently(func() *k8sv1.ContainerStateRunning {
						computeContainerStateRunning, err := getComputeContainerStateRunning()
						Expect(err).ToNot(HaveOccurred())
						return computeContainerStateRunning
					}).WithTimeout(10*time.Second).WithPolling(1*time.Second).ShouldNot(BeNil(), "compute container should be running")
				} else {
					Eventually(func() *k8sv1.ContainerStateRunning {
						computeContainerStateRunning, err := getComputeContainerStateRunning()
						Expect(err).ToNot(HaveOccurred())
						return computeContainerStateRunning
					}).WithTimeout(100*time.Second).WithPolling(1*time.Second).Should(BeNil(), "compute container should not be running")
					Eventually(func() k8sv1.PodPhase {
						launcherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vm.Namespace)
						Expect(err).ToNot(HaveOccurred())
						return launcherPod.Status.Phase
					}).WithTimeout(60*time.Second).WithPolling(1*time.Second).Should(Equal(k8sv1.PodFailed), "pod should fail")
				}
			},
				Entry("[test_id:7164]VMI launcher pod should fail", "false"),
				Entry("[test_id:6993]VMI launcher pod compute container should keep running", "true"),
			)
		})
	})

	DescribeTable("[release-blocker][test_id:299][test_id:264]should create and delete a VM using all supported API versions", decorators.Conformance, func(version string) {
		vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm.APIVersion = version

		By("Creating VM")
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VMI to start")
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

		By("Looking up virt-launcher pod")
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(pod).To(BeRunning())

		By("Deleting VM")
		err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(ThisVM(vm), 120*time.Second, 1*time.Second).Should(BeGone())
		Eventually(ThisPod(pod), 120*time.Second, 1*time.Second).Should(BeGone())
	},
		Entry("with v1 api", "kubevirt.io/v1"),
		Entry("with v1alpha3 api", "kubevirt.io/v1alpha3"),
	)

	Context("crash loop backoff", decorators.Conformance, func() {
		It("should backoff attempting to create a new VMI when 'runStrategy: Always' during crash loop.", func() {
			By("Creating VirtualMachine")
			vm := createRunningVM(virtClient, libvmifact.NewCirros(
				libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
			))

			By("waiting for crash loop state")
			Eventually(ThisVM(vm), 60*time.Second, 5*time.Second).Should(BeInCrashLoop())

			By("Testing that the failure count is within the expected range over a period of time")
			maxExpectedFailCount := 3
			Consistently(func() error {
				// get the VM and verify the failure count is less than 4 over a minute,
				// indicating that backoff is occuring
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				if vm.Status.StartFailure == nil {
					return fmt.Errorf("start failure count not detected")
				} else if vm.Status.StartFailure.ConsecutiveFailCount > maxExpectedFailCount {
					return fmt.Errorf("consecutive fail count is higher than %d", maxExpectedFailCount)
				}

				return nil
			}, 1*time.Minute, 5*time.Second).Should(BeNil())

			By("Updating the VMI template to correct the crash loop")
			Eventually(func() error {
				annotationRemovePatch, err := patch.New(
					patch.WithRemove(
						fmt.Sprintf("/spec/template/metadata/annotations/%s", patch.EscapeJSONPointer(v1.FuncTestLauncherFailFastAnnotation)),
					)).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, annotationRemovePatch, metav1.PatchOptions{})
				return err
			}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			By("Waiting on crash loop status to be removed.")
			Eventually(ThisVM(vm), 300*time.Second, 5*time.Second).Should(NotBeInCrashLoop())
		})

		It("should be able to stop a VM during crashloop backoff when when 'runStrategy: Always' is set", func() {
			By("Creating VirtualMachine")
			vm := createRunningVM(virtClient, libvmifact.NewCirros(
				libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
			))

			By("waiting for crash loop state")
			Eventually(ThisVM(vm), 60*time.Second, 5*time.Second).Should(BeInCrashLoop())

			By("Stopping the VM while in a crash loop")
			err := virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting on crash loop status to be removed.")
			Eventually(ThisVM(vm), 120*time.Second, 5*time.Second).Should(NotBeInCrashLoop())
		})
	})

	Context("VirtualMachineControllerFinalizer", decorators.Conformance, func() {
		const customFinalizer = "customFinalizer"

		var (
			vmi *v1.VirtualMachineInstance
			vm  *v1.VirtualMachine
		)

		BeforeEach(func() {
			vmi = libvmifact.NewGuestless()
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			Expect(vm.Finalizers).To(BeEmpty())
			vm.Finalizers = append(vm.Finalizers, customFinalizer)
		})

		AfterEach(func() {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			newVm := vm.DeepCopy()
			controller.RemoveFinalizer(newVm, customFinalizer)

			patchBytes, err := patch.New(
				patch.WithTest("/metadata/finalizers", vm.GetFinalizers()),
				patch.WithReplace("/metadata/finalizers", newVm.GetFinalizers()),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensure the vm has disappeared")
			Eventually(func() error {
				_, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err
			}, 2*time.Minute, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), fmt.Sprintf("vm %s is not deleted", vm.Name))
		})

		It("should be added when the vm is created and removed when the vm is being deleted", func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Finalizers).To(And(
					ContainElement(v1.VirtualMachineControllerFinalizer),
					ContainElement(customFinalizer),
				))
			}, 2*time.Minute, 1*time.Second).Should(Succeed())

			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Finalizers).To(And(
					Not(ContainElement(v1.VirtualMachineControllerFinalizer)),
					ContainElement(customFinalizer),
				))
			}, 2*time.Minute, 1*time.Second).Should(Succeed())
		})

		It("should be removed when the vm has child resources, such as instance type ControllerRevisions, that have been deleted before the vm - issue #9438", func() {
			By("creating a VirtualMachineClusterInstancetype")
			instancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(1),
					},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("64Mi"),
					},
				},
			}
			instancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating a VirtualMachine")
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
			}
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				},
			}
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the VirtualMachine has the VirtualMachineControllerFinalizer, customFinalizer and revisionName")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeTrue())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
				g.Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(BeEmpty())
			}, 2*time.Minute, 1*time.Second).Should(Succeed())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("deleting the ControllerRevision associated with the VirtualMachine and VirtualMachineClusterInstancetype %s", vm.Status.InstancetypeRef.ControllerRevisionRef.Name))
			err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("deleting the VirtualMachineClusterInstancetype")
			err = virtClient.VirtualMachineClusterInstancetype().Delete(context.Background(), vm.Spec.Instancetype.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("deleting the VirtualMachine")
			err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the VirtualMachineControllerFinalizer has been removed from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeFalse())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
			}, 2*time.Minute, 1*time.Second).Should(Succeed())
		})
	})

	Context(" when node becomes unhealthy", Serial, func() {
		const componentName = "virt-handler"
		var nodeName string

		AfterEach(func() {
			libpod.DeleteKubernetesAPIBlackhole(getHandlerNodePod(virtClient, nodeName), componentName)
			Eventually(func(g Gomega) {
				g.Expect(getHandlerNodePod(virtClient, nodeName).Items[0]).To(HaveConditionTrue(k8sv1.PodReady))
			}, 120*time.Second, time.Second).Should(Succeed())

			config.WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-handler", libkubevirt.GetCurrentKv(virtClient).ResourceVersion,
				config.ExpectResourceVersionToBeLessEqualThanConfigVersion, 120*time.Second)
		})

		It(" the VMs running in that node should be respawned", func() {
			By("Starting VM")
			vm := libvmops.StartVirtualMachine(createVM(virtClient, libvmifact.NewCirros()))
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			nodeName = vmi.Status.NodeName
			oldUID := vmi.UID

			By("Blocking virt-handler from reconciling the VMI")
			libpod.AddKubernetesAPIBlackhole(getHandlerNodePod(virtClient, nodeName), componentName)
			Eventually(func(g Gomega) {
				g.Expect(getHandlerNodePod(virtClient, nodeName).Items[0]).To(HaveConditionFalse(k8sv1.PodReady))
			}, 120*time.Second, time.Second).Should(Succeed())

			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			By("Simulating loss of the virt-launcher")
			err = virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
				GracePeriodSeconds: ptr.To(int64(0)),
			})
			Expect(err).ToNot(HaveOccurred())

			// These timeouts are low on purpose. The VMI should be marked as Failed and be recreated fast.
			// In case this fails, do not increase the timeouts.
			By("The VM should not be Ready")
			Eventually(ThisVM(vm), 30*time.Second, 1*time.Second).ShouldNot(BeReady())

			// This is only possible if the VMI has been in Phase Failed
			By("Check if the VMI has been recreated")
			Eventually(func(g Gomega) types.UID {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				return vmi.UID
			}, 30*time.Second, 1*time.Second).ShouldNot(BeEquivalentTo(oldUID), "The UID should not match to old VMI, new VMI should appear")
		})
	})
})

func getHandlerNodePod(virtClient kubecli.KubevirtClient, nodeName string) *k8sv1.PodList {
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{
			LabelSelector: "kubevirt.io=virt-handler",
			FieldSelector: fmt.Sprintf("spec.nodeName=" + nodeName),
		})

	Expect(err).NotTo(HaveOccurred())
	Expect(pods.Items).To(HaveLen(1))

	return pods
}

func createVM(virtClient kubecli.KubevirtClient, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
	By("Creating stopped VirtualMachine")
	vm := libvmi.NewVirtualMachine(template)
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return vm
}

func createRunningVM(virtClient kubecli.KubevirtClient, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
	By("Creating running VirtualMachine")
	vm := libvmi.NewVirtualMachine(template, libvmi.WithRunStrategy(v1.RunStrategyAlways))
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return vm
}
