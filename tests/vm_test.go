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
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/pborman/uuid"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	virtctl "kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var _ = Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	runStrategyManual := v1.RunStrategyManual

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("An invalid VirtualMachine given", func() {
		It("[test_id:1518]should be rejected on POST", func() {
			vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
			// because we're marshaling this ourselves, we have to make sure
			// we're using the same version the virtClient is using.
			vm.APIVersion = "kubevirt.io/" + v1.ApiStorageVersion

			jsonBytes, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(testsuite.GetTestNamespace(vm)).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do(context.Background())
			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		})
		It("[test_id:1519]should reject POST if validation webhoook deems the spec is invalid", func() {
			template := libvmi.NewCirros()
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			template.Spec.Domain.Devices.Disks = append(template.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vm := tests.NewRandomVirtualMachine(template, false)

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(testsuite.GetTestNamespace(vm)).Body(vm).Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &k8smetav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).ToNot(HaveOccurred())

			Expect(reviewResponse.Details.Causes).To(HaveLen(1))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].name"))
		})
	})

	Context("[Serial]A mutated VirtualMachine given", Serial, func() {
		const testingMachineType = "pc-q35-2.7"

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)
			kubevirtConfiguration := kv.Spec.Configuration
			kubevirtConfiguration.MachineType = ""
			kubevirtConfiguration.ArchitectureConfiguration = &v1.ArchConfiguration{Amd64: &v1.ArchSpecificConfiguration{}, Arm64: &v1.ArchSpecificConfiguration{}, Ppc64le: &v1.ArchSpecificConfiguration{}}

			kubevirtConfiguration.ArchitectureConfiguration.Amd64.MachineType = testingMachineType
			kubevirtConfiguration.ArchitectureConfiguration.Arm64.MachineType = testingMachineType
			kubevirtConfiguration.ArchitectureConfiguration.Ppc64le.MachineType = testingMachineType

			tests.UpdateKubeVirtConfigValueAndWait(kubevirtConfiguration)
		})

		It("[test_id:3312]should set the default MachineType when created without explicit value", func() {
			By("Creating VirtualMachine")
			template := libvmi.NewCirros()
			template.Spec.Domain.Machine = nil
			vm := createVM(virtClient, template)

			Expect(vm.Spec.Template.Spec.Domain.Machine.Type).To(Equal(testingMachineType))
		})

		It("[test_id:3311]should keep the supplied MachineType when created", func() {
			By("Creating VirtualMachine")
			const explicitMachineType = "pc-q35-3.0"
			template := libvmi.NewCirros()
			template.Spec.Domain.Machine = &v1.Machine{Type: explicitMachineType}
			vm := createVM(virtClient, template)

			Expect(vm.Spec.Template.Spec.Domain.Machine.Type).To(Equal(explicitMachineType))
		})
	})

	Context("A valid VirtualMachine given", func() {
		type vmiBuilder func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume)

		newVirtualMachineInstanceWithFileDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			return tests.NewRandomVirtualMachineInstanceWithFileDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), corev1.ReadWriteOnce)
		}

		newVirtualMachineInstanceWithBlockDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			return tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), corev1.ReadWriteOnce)
		}

		newVirtualMachineWithRunStrategy := func(runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
			vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			), false)
			vm.Spec.Running = nil
			vm.Spec.RunStrategy = &runStrategy

			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			return vm
		}

		validateGenerationState := func(vm *v1.VirtualMachine, expectedGeneration int, expectedDesiredGeneration int, expectedObservedGeneration int, expectedGenerationAnnotation int) {
			By("By validating the generation states")
			EventuallyWithOffset(1, func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				By("Expecting the generation to match")
				g.Expect(vm.Generation).To(Equal(int64(expectedGeneration)))

				By("Expecting the generation state in the vm status to match")
				g.Expect(vm.Status.DesiredGeneration).To(Equal(int64(expectedDesiredGeneration)))
				g.Expect(vm.Status.ObservedGeneration).To(Equal(int64(expectedObservedGeneration)))

				By("Expecting the generation annotation on the vmi to match")
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				g.Expect(vmi.Annotations).Should(HaveKeyWithValue(v1.VirtualMachineGenerationAnnotation, fmt.Sprintf("%v", expectedGenerationAnnotation)))
			}, 10*time.Second, 1*time.Second).Should(Succeed())
		}

		DescribeTable("cpu/memory in requests/limits should allow", func(cpu, request string) {
			const oldCpu = "222"
			const oldMemory = "2222222"

			vm := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
			vm.Namespace = testsuite.GetTestNamespace(vm)
			vm.APIVersion = "kubevirt.io/" + v1.ApiStorageVersion
			vm.Spec.Template.Spec.Domain.Resources.Limits = make(k8sv1.ResourceList)
			vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(oldCpu)
			vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(oldCpu)
			vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(oldMemory)
			vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(oldMemory)

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
			vm = startVM(virtClient, vm)
		},
			Entry("int type", "2", "2222222"),
			Entry("float type", "2.2", "2222222.2"),
		)

		It("[test_id:3161]should carry annotations to VMI", func() {
			annotations := map[string]string{
				"testannotation": "test",
			}

			vm := createVM(virtClient, libvmi.NewCirros())

			err = tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta k8smetav1.ObjectMeta) error {
				vm, err = virtClient.VirtualMachine(meta.Namespace).Get(context.Background(), meta.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Template.ObjectMeta.Annotations = annotations
				vm, err = virtClient.VirtualMachine(meta.Namespace).Update(context.Background(), vm)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			vm = startVM(virtClient, vm)

			By("checking for annotations to be present")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(HaveKeyWithValue("testannotation", "test"))
		})

		It("[test_id:3162]should ignore kubernetes and kubevirt annotations to VMI", func() {
			annotations := map[string]string{
				"kubevirt.io/test":   "test",
				"kubernetes.io/test": "test",
			}

			vm := createVM(virtClient, libvmi.NewCirros())

			err = tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta k8smetav1.ObjectMeta) error {
				vm, err = virtClient.VirtualMachine(meta.Namespace).Get(context.Background(), meta.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Annotations = annotations
				vm, err = virtClient.VirtualMachine(meta.Namespace).Update(context.Background(), vm)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			vm = startVM(virtClient, vm)

			By("checking for annotations to not be present")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).ShouldNot(HaveKey("kubevirt.io/test"), "kubevirt internal annotations should be ignored")
			Expect(vmi.Annotations).ShouldNot(HaveKey("kubernetes.io/test"), "kubernetes internal annotations should be ignored")
		})

		It("should sync the generation annotation on the vmi during restarts", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			for i := 1; i <= 3; i++ {
				// Generation increases twice for each pass, since there is a stop and a
				// start.
				expectedGeneration := (i * 2)

				validateGenerationState(vm, expectedGeneration, expectedGeneration, expectedGeneration, expectedGeneration)

				By("Restarting the VM")
				vm = startVM(virtClient, stopVM(virtClient, vm))
			}
		})

		It("should not update the vmi generation annotation when the template changes", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			By("Updating the VM template spec")
			vm.Spec.Template.ObjectMeta.Labels["testkey"] = "testvalue"
			_, err := virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			validateGenerationState(vm, 3, 3, 2, 2)

			By("Updating the VM template spec")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.ObjectMeta.Labels["testkey2"] = "testvalue2"
			_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			validateGenerationState(vm, 4, 4, 2, 2)

			// Restart the VM to check that the state will once again sync.
			By("Restarting the VM")
			vm = startVM(virtClient, stopVM(virtClient, vm))

			validateGenerationState(vm, 6, 6, 6, 6)
		})

		DescribeTable("[test_id:1520]should update VirtualMachine once VMIs are up", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer libstorage.DeleteDataVolume(&dv)
			startVM(virtClient, createVM(virtClient, template))
		},
			Entry("with ContainerDisk", func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) { return libvmi.NewCirros(), nil }),
			Entry("[Serial][storage-req]with Filesystem Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithFileDisk),
			Entry("[Serial][storage-req]with Block Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithBlockDisk),
		)

		DescribeTable("[test_id:1521]should remove VirtualMachineInstance once the VM is marked for deletion", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer libstorage.DeleteDataVolume(&dv)
			vm := startVM(virtClient, createVM(virtClient, template))
			// Delete it
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &k8smetav1.DeleteOptions{})).To(Succeed())
			// Wait until VMI is gone
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 2*time.Second).ShouldNot(Exist())
		},
			Entry("with ContainerDisk", func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) { return libvmi.NewCirros(), nil }),
			Entry("[Serial][storage-req]with Filesystem Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithFileDisk),
			Entry("[Serial][storage-req]with Block Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithBlockDisk),
		)

		It("[test_id:1522]should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			By("Getting owner references")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).ToNot(BeEmpty())

			// Delete it
			orphanPolicy := k8smetav1.DeletePropagationOrphan
			By("Deleting VM")
			Expect(virtClient.VirtualMachine(vm.Namespace).
				Delete(context.Background(), vm.Name, &k8smetav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the virtual machine is deleted
			By("Waiting for VM to delete")
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).ShouldNot(Exist())

			By("Verifying orphaned VMI still exists")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).To(BeEmpty())
		})

		It("[test_id:1523]should recreate VirtualMachineInstance if it gets deleted", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(virtClient.VirtualMachineInstance(vm.Namespace).Delete(context.Background(), vm.Name, &k8smetav1.DeleteOptions{})).To(Succeed())

			Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))
		})

		It("[test_id:1524]should recreate VirtualMachineInstance if the VirtualMachineInstance's pod gets deleted", func() {
			By("Start a new VM")
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))
			firstVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// get the pod backing the VirtualMachineInstance
			By("Getting the pod backing the VirtualMachineInstance")
			pods, err := virtClient.CoreV1().Pods(vm.Namespace).List(context.Background(), tests.UnfinishedVMIPodSelector(firstVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))
			firstPod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(vm.Namespace).Delete(context.Background(), firstPod.Name, k8smetav1.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// Wait on the VMI controller to create a new VirtualMachineInstance
			By("Waiting for a new VirtualMachineInstance to spawn")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(beRestarted(firstVMI.UID))

			// sanity check that the test ran correctly by
			// verifying a different Pod backs the VMI as well.
			By("Verifying a new pod backs the VMI")
			currentVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			pods, err = virtClient.CoreV1().Pods(vm.Namespace).List(context.Background(), tests.UnfinishedVMIPodSelector(currentVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))
			pod := pods.Items[0]
			Expect(pod.Name).ToNot(Equal(firstPod.Name))
		})

		DescribeTable("[test_id:1525]should stop VirtualMachineInstance if running set to false", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer libstorage.DeleteDataVolume(&dv)
			stopVM(virtClient, startVM(virtClient, createVM(virtClient, template)))
		},
			Entry("with ContainerDisk", func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) { return libvmi.NewCirros(), nil }),
			Entry("[Serial][storage-req]with Filesystem Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithFileDisk),
			Entry("[Serial][storage-req]with Block Disk", Serial, decorators.StorageReq, newVirtualMachineInstanceWithBlockDisk),
		)

		It("[test_id:1526]should start and stop VirtualMachineInstance multiple times", func() {
			vm := createVM(virtClient, libvmi.NewCirros())
			// Start and stop VirtualMachineInstance multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				vm = stopVM(virtClient, startVM(virtClient, vm))
			}
		})

		It("[test_id:1527]should not update the VirtualMachineInstance spec if Running", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			By("Updating the VM template spec")
			updatedVM := vm.DeepCopy()
			updatedVM.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedVM, err := virtClient.VirtualMachine(updatedVM.Namespace).Update(context.Background(), updatedVM)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VirtualMachineInstance spec still running")
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory := vm.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))

			By("Restarting the VM")
			vm = startVM(virtClient, stopVM(virtClient, vm))

			By("Expecting updated spec running")
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory = vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory = updatedVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))
		})

		It("[test_id:1528]should survive guest shutdown, multiple times", func() {
			By("Creating new VM, not running")
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			for i := 0; i < 3; i++ {
				By("Getting the running VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Obtaining the serial console")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Guest shutdown")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 240)).To(Succeed())

				By("waiting for the controller to replace the shut-down vmi with a new instance")
				Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))

				By("VMI should run the VirtualMachineInstance again")
			}
		})

		It("should create vm revision when starting vm", func() {
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVMRevisionName := fmt.Sprintf("revision-start-vm-%s-%d", vm.UID, vm.Generation)
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(expectedVMRevisionName))

			cr, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vmi.Status.VirtualMachineRevisionName, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Revision).To(Equal(int64(2)))
			vmRevision := &v1.VirtualMachine{}
			err = json.Unmarshal(cr.Data.Raw, vmRevision)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRevision.Spec).To(Equal(vm.Spec))
		})

		It("should delete old vm revision and create new one when restarting vm", func() {
			By("Starting the VM")
			vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVMRevisionName := fmt.Sprintf("revision-start-vm-%s-%d", vm.UID, vm.Generation)
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(expectedVMRevisionName))
			oldVMRevisionName := expectedVMRevisionName

			By("Stopping the VM")
			vm = stopVM(virtClient, vm)

			updatedVM := vm.DeepCopy()
			Eventually(func() error {
				updatedVM, err = virtClient.VirtualMachine(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				updatedVM.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("4096Ki"),
				}
				updatedVM, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(context.Background(), updatedVM)
				return err
			}, 10*time.Second, time.Second).ShouldNot(HaveOccurred())

			By("Starting the VM after update")
			vm = startVM(virtClient, updatedVM)

			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedVMRevisionName = fmt.Sprintf("revision-start-vm-%s-%d", vm.UID, vm.Generation)
			Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(expectedVMRevisionName))

			cr, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), oldVMRevisionName, k8smetav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())

			cr, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vmi.Status.VirtualMachineRevisionName, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Revision).To(Equal(int64(5)))
			vmRevision := &v1.VirtualMachine{}
			err = json.Unmarshal(cr.Data.Raw, vmRevision)
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRevision.Spec).To(Equal(vm.Spec))
		})

		It("[test_id:4645]should set the Ready condition on VM", func() {
			vm := createVM(virtClient, libvmi.NewCirros())

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionFalse(v1.VirtualMachineReady))

			vm = startVM(virtClient, vm)

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionTrue(v1.VirtualMachineReady))

			vm = stopVM(virtClient, vm)

			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(HaveConditionFalse(v1.VirtualMachineReady))
		})

		DescribeTable("should report an error status when VM scheduling error occurs", func(unschedulableFunc func(vmi *v1.VirtualMachineInstance)) {
			vmi := libvmi.New(
				libvmi.WithContainerImage("no-such-image"),
				libvmi.WithResourceMemory("128Mi"),
			)
			unschedulableFunc(vmi)

			vm := createRunningVM(virtClient, vmi)

			By("Verifying that the VM status eventually gets set to FailedUnschedulable")
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(havePrintableStatus(v1.VirtualMachineStatusUnschedulable))
		},
			Entry("[test_id:6867]with unsatisfiable resource requirements", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.Domain.Resources.Requests = corev1.ResourceList{
					// This may stop working sometime around 2040
					corev1.ResourceMemory: resource.MustParse("1Ei"),
					corev1.ResourceCPU:    resource.MustParse("1M"),
				}
			}),
			Entry("[test_id:6868]with unsatisfiable scheduling constraints", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.NodeSelector = map[string]string{
					"node-label": "that-doesnt-exist",
				}
			}),
		)

		It("[test_id:6869]should report an error status when image pull error occurs", func() {
			vmi := libvmi.New(
				libvmi.WithContainerImage("no-such-image"),
				libvmi.WithResourceMemory("128Mi"),
			)

			vm := createRunningVM(virtClient, vmi)

			By("Verifying that the status toggles between ErrImagePull and ImagePullBackOff")
			const times = 2
			for i := 0; i < times; i++ {
				Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(havePrintableStatus(v1.VirtualMachineStatusErrImagePull))
				Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(havePrintableStatus(v1.VirtualMachineStatusImagePullBackOff))
			}
		})

		DescribeTable("should report an error status when a VM with a missing PVC/DV is started", func(vmiFunc func() *v1.VirtualMachineInstance, status v1.VirtualMachinePrintableStatus) {
			vm := createRunningVM(virtClient, vmiFunc())
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(havePrintableStatus(status))
		},
			Entry(
				"[test_id:7596]missing PVC",
				func() *v1.VirtualMachineInstance {
					return libvmi.New(
						libvmi.WithPersistentVolumeClaim("disk0", "missing-pvc"),
						libvmi.WithResourceMemory("128Mi"),
					)
				},
				v1.VirtualMachineStatusPvcNotFound,
			),
			Entry(
				"[test_id:7597]missing DataVolume",
				func() *v1.VirtualMachineInstance {
					return libvmi.New(
						libvmi.WithDataVolume("disk0", "missing-datavolume"),
						libvmi.WithResourceMemory("128Mi"),
					)
				},
				v1.VirtualMachineStatusPvcNotFound,
			),
		)

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
			vm := tests.NewRandomVMWithDataVolumeWithRegistryImport("docker://no.such/image",
				testsuite.GetTestNamespace(nil), storageClassName, k8sv1.ReadWriteOnce)
			vm.Spec.Running = pointer.BoolPtr(true)
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying that the VM status eventually gets set to DataVolumeError")
			Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(havePrintableStatus(v1.VirtualMachineStatusDataVolumeError))
		})

		Context("Using virtctl interface", func() {
			It("[test_id:1529]should start a VirtualMachineInstance once", func() {
				By("getting a VM")
				vm := createVM(virtClient, libvmi.NewCirros())

				By("Invoking virtctl start")
				startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
				Expect(startCommand()).To(Succeed())

				By("Getting the status of the VM")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

				By("Getting the running VirtualMachineInstance")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).Should(BeRunning())

				By("Ensuring a second invocation should fail")
				Expect(startCommand()).To(MatchError(fmt.Sprintf(`Error starting VirtualMachine Operation cannot be fulfilled on virtualmachine.kubevirt.io "%s": VM is already running`, vm.Name)))
			})

			It("[test_id:1530]should stop a VirtualMachineInstance once", func() {
				By("getting a VM")
				vm := startVM(virtClient, createVM(virtClient, libvmi.NewCirros()))

				By("Invoking virtctl stop")
				stopCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name)
				Expect(stopCommand()).To(Succeed())

				By("Ensuring VM is not running")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(And(Not(beCreated()), Not(beReady())))

				By("Ensuring the VirtualMachineInstance is removed")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())

				By("Ensuring a second invocation should fail")
				Expect(stopCommand()).To(MatchError(fmt.Sprintf(`Error stopping VirtualMachine Operation cannot be fulfilled on virtualmachine.kubevirt.io "%s": VM is not running`, vm.Name)))
			})

			It("[test_id:6310]should start a VirtualMachineInstance in paused state", func() {
				By("getting a VM")
				vm := createVM(virtClient, libvmi.NewCirros())

				By("Invoking virtctl start")
				startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name, "--paused")
				Expect(startCommand()).To(Succeed())

				By("Getting the status of the VM")
				Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beCreated())

				By("Getting running VirtualMachineInstance with paused condition")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*vmi.Spec.StartStrategy).To(Equal(v1.StartStrategyPaused))
					Eventually(ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstancePaused))
					return vmi.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("[test_id:3007]Should force restart a VM with terminationGracePeriodSeconds>0", func() {
				By("getting a VM with high TerminationGracePeriod")
				vm := startVM(virtClient, createVM(virtClient, libvmi.NewFedora(
					libvmi.WithTerminationGracePeriod(600),
				)))

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Invoking virtctl --force restart")
				forceRestart := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_RESTART, "--namespace", vm.Namespace, "--force", vm.Name, "--grace-period=0")
				Expect(forceRestart()).To(Succeed())

				zeroGracePeriod := int64(0)
				// Checks if the old VMI Pod still exists after force-restart command
				Eventually(func() string {
					pod, err := tests.GetRunningPodByLabel(string(vmi.UID), v1.CreatedByLabel, vm.Namespace, "")
					if err != nil {
						return err.Error()
					}
					if pod.GetDeletionGracePeriodSeconds() == &zeroGracePeriod && pod.GetDeletionTimestamp() != nil {
						return "old VMI Pod still not deleted"
					}
					return ""
				}, 120*time.Second, 1*time.Second).Should(ContainSubstring("failed to find pod"))

				Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))

				By("Comparing the new UID and CreationTimeStamp with the old one")
				newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(newVMI.CreationTimestamp).ToNot(Equal(vmi.CreationTimestamp))
				Expect(newVMI.UID).ToNot(Equal(vmi.UID))
			})

			It("Should force stop a VMI", func() {
				By("getting a VM with high TerminationGracePeriod")
				vm := startVM(virtClient, createVM(virtClient, libvmi.New(
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithTerminationGracePeriod(1600),
				)))

				By("setting up a watch for vmi")
				lw, err := virtClient.VirtualMachineInstance(vm.Namespace).Watch(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				terminationGracePeriodUpdated := func(stopCn <-chan bool, eventsCn <-chan watch.Event, updated chan<- bool) {
					for {
						select {
						case <-stopCn:
							return
						case e := <-eventsCn:
							vmi, ok := e.Object.(*v1.VirtualMachineInstance)
							Expect(ok).To(BeTrue())
							if vmi.Name != vm.Name {
								continue
							}

							if *vmi.Spec.TerminationGracePeriodSeconds == 0 {
								updated <- true
							}
						}
					}
				}
				stopCn := make(chan bool, 1)
				updated := make(chan bool, 1)
				go terminationGracePeriodUpdated(stopCn, lw.ResultChan(), updated)

				By("Invoking virtctl --force stop")
				forceStop := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, vm.Name, "--namespace", vm.Namespace, "--force", "--grace-period=0")
				Expect(forceStop()).To(Succeed())

				By("Ensuring the VirtualMachineInstance is removed")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())

				Expect(updated).To(Receive(), "vmi should be updated")
				stopCn <- true
			})

			Context("Using RunStrategyAlways", func() {
				It("[test_id:3163]should stop a running VM", func() {
					By("creating a VM with RunStrategyAlways")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Invoking virtctl stop")
					stopCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name)
					Expect(stopCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())

					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyHalted))
					Expect(vm.Status.StateChangeRequests).To(BeEmpty())
				})

				It("[test_id:3164]should restart a running VM", func() {
					By("creating a VM with RunStrategyAlways")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Getting VMI's UUID")
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Invoking virtctl restart")
					restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_RESTART, "--namespace", vm.Namespace, vm.Name)
					Expect(restartCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))

					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))

					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))
				})

				It("[test_id:3165]should restart a succeeded VMI", func() {
					By("creating a VM with RunStategyRunning")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(console.LoginToCirros(vmi)).To(Succeed())

					By("Issuing a poweroff command from inside VM")
					Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
						&expect.BExp{R: console.PromptExpression},
					}, 10)).To(Succeed())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))
				})

				It("[test_id:4119]should migrate a running VM", func() {
					nodes := libnode.GetAllSchedulableNodes(virtClient)
					if len(nodes.Items) < 2 {
						Skip("Migration tests require at least 2 nodes")
					}
					By("creating a VM with RunStrategyAlways")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Invoking virtctl migrate")
					migrateCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_MIGRATE, "--namespace", vm.Namespace, vm.Name)
					Expect(migrateCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is migrated")
					Eventually(func() bool {
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed
					}, 240*time.Second, 1*time.Second).Should(BeTrue())
				})

				It("[test_id:7743]should not migrate a running vm if dry-run option is passed", func() {
					nodes := libnode.GetAllSchedulableNodes(virtClient)
					if len(nodes.Items) < 2 {
						Skip("Migration tests require at least 2 nodes")
					}
					By("creating a VM with RunStrategyAlways")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Invoking virtctl migrate with dry-run option")
					migrateCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_MIGRATE, "--dry-run", "--namespace", vm.Namespace, vm.Name)
					Expect(migrateCommand()).To(Succeed())

					By("Check that no migration was actually created")
					Consistently(func() bool {
						_, err = virtClient.VirtualMachineInstanceMigration(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 60*time.Second, 5*time.Second).Should(BeTrue(), "migration should not be created in a dry run mode")
				})
			})

			Context("Using RunStrategyRerunOnFailure", func() {
				It("[test_id:2186] should stop a running VM", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Invoking virtctl stop")
					stopCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name)
					Expect(stopCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())

					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyHalted))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(vm.Status.StateChangeRequests).To(BeEmpty())
				})

				It("[test_id:2187] should restart a running VM", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Getting VMI's UUID")
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Invoking virtctl restart")
					restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_RESTART, "--namespace", vm.Namespace, vm.Name)
					Expect(restartCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))

					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyRerunOnFailure))

					By("Ensuring stateChangeRequests list gets cleared")
					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))
				})

				It("[test_id:2188] should not remove a succeeded VMI", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					By("Waiting for VMI to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(console.LoginToCirros(vmi)).To(Succeed())

					By("Issuing a poweroff command from inside VM")
					Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
						&expect.BExp{R: console.PromptExpression},
					}, 10)).To(Succeed())

					By("Ensuring the VirtualMachineInstance enters Succeeded phase")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(HaveSucceeded())

					// At this point, explicitly test that a start command will delete an existing
					// VMI in the Succeeded phase.
					By("Invoking virtctl start")
					restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(restartCommand()).To(Succeed())

					By("Waiting for StartRequest to be cleared")
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())
				})
			})

			Context("Using RunStrategyHalted", func() {
				It("[test_id:2037] should start a stopped VM", func() {
					By("creating a VM with RunStrategyHalted")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)

					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(vm.Status.StateChangeRequests).To(BeEmpty())
				})
			})

			Context("Using RunStrategyOnce", func() {
				It("[Serial] Should leave a failed VMI", Serial, func() {
					By("creating a VM with RunStrategyOnce")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyOnce)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("killing qemu process")
					Expect(pkillAllVMIs(virtClient, vmi.Status.NodeName)).To(Succeed(), "Should kill VMI successfully")

					By("Ensuring the VirtualMachineInstance enters Failed phase")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(BeInPhase(v1.Failed))

					By("Ensuring the VirtualMachine remains stopped")
					Consistently(ThisVMI(vmi), 60*time.Second, 5*time.Second).Should(BeInPhase(v1.Failed))

					By("Ensuring the VirtualMachine remains Ready=false")
					Consistently(ThisVM(vm), 60*time.Second, 5*time.Second).Should(Not(beReady()))
				})

				It("Should leave a succeeded VMI", func() {
					By("creating a VM with RunStrategyOnce")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyOnce)

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
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
					Consistently(ThisVM(vm), 60*time.Second, 5*time.Second).Should(Not(beReady()))
				})
			})

			Context("Using RunStrategyManual", func() {
				It("[test_id:2036] should start", func() {
					By("creating a VM with RunStrategyManual")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyManual))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(vm.Status.StateChangeRequests).To(BeEmpty())
				})

				It("[test_id:2189] should stop", func() {
					By("creating a VM with RunStrategyManual")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					stopCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name)
					Expect(stopCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(Exist())

					By("Ensuring stateChangeRequests list is cleared")
					Eventually(ThisVM(vm), 30*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))
				})

				It("[test_id:6311]should start in paused state", func() {
					By("creating a VM with RunStrategyManual")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					By("Invoking virtctl start")
					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name, "--paused")
					Expect(startCommand()).To(Succeed())

					By("Getting the status of the VM")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beCreated())

					By("Getting running VirtualMachineInstance with paused condition")
					Eventually(func() bool {
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(*vmi.Spec.StartStrategy).To(Equal(v1.StartStrategyPaused))
						Eventually(ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstancePaused))
						return vmi.Status.Phase == v1.Running
					}, 240*time.Second, 1*time.Second).Should(BeTrue())
				})

				It("[test_id:2035] should restart", func() {
					By("creating a VM with RunStrategyManual")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					stopCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name)
					restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_RESTART, "--namespace", vm.Namespace, vm.Name)

					By("Invoking virtctl restart should fail")
					Expect(restartCommand()).ToNot(Succeed())

					By("Invoking virtctl start")
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(beReady())

					By("Invoking virtctl stop")
					Expect(stopCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is stopped")
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(beCreated()))

					By("Waiting state change request to clear for stopped VM")
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))

					By("Invoking virtctl start")
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					By("Getting VMI's UUID")
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Invoking virtctl restart")
					Expect(restartCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(beRestarted(vmi.UID))

					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyManual))

					By("Ensuring stateChangeRequests list gets cleared")
					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))
				})

				It("[test_id:2190] should not remove a succeeded VMI", func() {
					By("creating a VM with RunStrategyManual")
					vm := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(console.LoginToCirros(vmi)).To(Succeed())

					By("Issuing a poweroff command from inside VM")
					Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
						&expect.BExp{R: console.PromptExpression},
					}, 10)).To(Succeed())

					By("Ensuring the VirtualMachineInstance enters Succeeded phase")
					Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(HaveSucceeded())

					// At this point, explicitly test that a start command will delete an existing
					// VMI in the Succeeded phase.
					By("Invoking virtctl start")
					restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(restartCommand()).To(Succeed())

					By("Waiting for StartRequest to be cleared")
					Eventually(ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(haveStateChangeRequests()))

					By("Waiting for VM to be ready")
					Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())
				})
				DescribeTable("with a failing VMI and the kubevirt.io/keep-launcher-alive-after-failure annotation", func(keepLauncher string) {
					// The estimated execution time of one test is 400 seconds.
					By("Creating a Kernel Boot VMI with a mismatched disk")
					vmi := utils.GetVMIKernelBootWithRandName()
					vmi.Spec.Domain.Firmware.KernelBoot.Container.Image = cd.ContainerDiskFor(cd.ContainerDiskCirros)

					By("Creating a VM with RunStrategyManual")
					vm := tests.NewRandomVirtualMachine(vmi, false)
					vm.Spec.Running = nil
					vm.Spec.RunStrategy = &runStrategyManual

					By("Annotate the VM with regard for leaving launcher pod after qemu exit")
					vm.Spec.Template.ObjectMeta.Annotations = map[string]string{
						v1.KeepLauncherAfterFailureAnnotation: keepLauncher,
					}
					vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
					Expect(err).ToNot(HaveOccurred())

					By("Starting the VMI with virtctl")
					startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
					Expect(startCommand()).To(Succeed())

					By("Waiting for VM to be in Starting status")
					Eventually(ThisVM(vm), 160*time.Second, 1*time.Second).Should(havePrintableStatus(v1.VirtualMachineStatusStarting))

					By("Waiting for VMI to fail")
					Eventually(ThisVMIWith(vm.Namespace, vm.Name), 480*time.Second, 1*time.Second).Should(BeInPhase(v1.Failed))

					// If the annotation v1.KeepLauncherAfterFailureAnnotation is set to true, the containerStatus of the
					// compute container of the virt-launcher pod is kept in the running state.
					// If the annotation v1.KeepLauncherAfterFailureAnnotation is set to false or not set, the virt-launcher pod will become failed.
					By("Verify that the virt-launcher pod or its container is in the expected state")
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					launcherPod, err := libvmi.GetPodByVirtualMachineInstance(vmi, vm.Namespace)
					Expect(err).ToNot(HaveOccurred())

					if toKeep, _ := strconv.ParseBool(keepLauncher); toKeep {
						Consistently(func() bool {
							for _, status := range launcherPod.Status.ContainerStatuses {
								if status.Name == "compute" && status.State.Running != nil {
									return true
								}
							}
							return false
						}, 10*time.Second, 1*time.Second).Should(BeTrue())
					} else {
						Eventually(launcherPod.Status.Phase).
							Within(160 * time.Second).
							WithPolling(time.Second).
							Should(Equal(k8sv1.PodFailed))
					}
				},
					Entry("[test_id:7164]VMI launcher pod should fail", "false"),
					Entry("[test_id:6993]VMI launcher pod compute container should keep running", "true"),
				)
			})

			Context("Using expand command", func() {
				var vm *v1.VirtualMachine

				BeforeEach(func() {
					vm = createRunningVM(virtClient, libvmi.NewCirros())
				})

				It("should fail without arguments", func() {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND)
					Expect(expandCommand()).To(MatchError("error invalid arguments - VirtualMachine name or file must be provided"))
				})

				It("should expand vm", func() {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--namespace", vm.Namespace, "--vm", vm.Name)
					Expect(expandCommand()).To(Succeed())
				})

				It("should fail with non existing vm", func() {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--namespace", "default", "--vm", "non-existing-vm")
					Expect(expandCommand()).To(MatchError("error expanding VirtualMachine - non-existing-vm in namespace - default: virtualmachine.kubevirt.io \"non-existing-vm\" not found"))
				})

				DescribeTable("should expand vm with", func(formatName string) {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--namespace", vm.Namespace, "--output", formatName, "--vm", vm.Name)
					Expect(expandCommand()).To(Succeed())
				},
					Entry("supported format output json", virtctl.JSON),
					Entry("supported format output yaml", virtctl.YAML),
				)

				It("should fail with unsupported output format", func() {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--namespace", vm.Namespace, "--output", "fakeJson", "--vm", vm.Name)
					Expect(expandCommand()).To(MatchError("error not supported output format defined: fakeJson"))
				})
			})

			Context("Using expand command with file input", func() {
				const (
					invalidVmSpec = `apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  annotations: {}
  labels: {}
  name: testvm
spec: {}
`
					vmSpec = `apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: testvm
spec:
  runStrategy: Always
  template:
    spec:
      domain:
        devices: {}
        machine:
          type: q35
        resources: {}
        volumes:
status:
`
				)

				var file *os.File

				BeforeEach(func() {
					file, err = os.CreateTemp("", "file-*")
					Expect(err).ToNot(HaveOccurred())
				})

				It("should expand vm defined in file", func() {
					Expect(os.WriteFile(file.Name(), []byte(vmSpec), 0777)).To(Succeed())
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", file.Name())
					Expect(expandCommand()).To(Succeed())
				})

				It("should fail expanding invalid vm defined in file", func() {
					Expect(os.WriteFile(file.Name(), []byte(invalidVmSpec), 0777)).To(Succeed())
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--namespace", "default", "--file", file.Name())
					Expect(expandCommand()).To(MatchError("error expanding VirtualMachine - testvm in namespace - default: Object is not a valid VirtualMachine"))
				})

				It("should fail expanding vm when input file does not exist", func() {
					expandCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_EXPAND, "--file", "invalid/path")
					Expect(expandCommand()).To(MatchError("error reading file open invalid/path: no such file or directory"))
				})
			})
		})
	})

	Context("[rfe_id:273]with oc/kubectl", func() {
		var k8sClient string
		var workDir string
		var vmRunningRe *regexp.Regexp

		BeforeEach(func() {
			k8sClient = clientcmd.GetK8sCmdClient()
			clientcmd.SkipIfNoCmd(k8sClient)
			workDir = GinkgoT().TempDir()

			// By default "." does not match newline: "Phase" and "Running" only match if on same line.
			vmRunningRe = regexp.MustCompile("Phase.*Running")
		})

		createVMAndGenerateJson := func(running bool) (*v1.VirtualMachine, string) {
			vm := tests.NewRandomVirtualMachine(libvmi.NewAlpine(), running)
			vm.Namespace = testsuite.GetTestNamespace(vm)

			vmJson, err := tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

			return vm, vmJson
		}

		It("[test_id:243][posneg:negative]should create VM only once", func() {
			vm, vmJson := createVMAndGenerateJson(true)

			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying VM is created")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "New VM was not created")

			By("Creating the VM again")
			_, stdErr, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).To(HaveOccurred())

			Expect(strings.HasPrefix(stdErr, "Error from server (AlreadyExists): error when creating")).To(BeTrue(), "command should error when creating VM second time")
		})

		DescribeTable("[release-blocker][test_id:299]should create VM via command line using all supported API versions", func(version string) {
			vmi := libvmi.NewAlpine()
			vm := tests.NewRandomVirtualMachine(vmi, true)
			vm.Namespace = testsuite.GetTestNamespace(vm)
			vm.APIVersion = version

			vmJson, err := tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

			By("Creating VM using k8s client binary")
			_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

			By("Listing running pods")
			stdout, _, err := clientcmd.RunCommand(k8sClient, "get", "pods")
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring pod is running")
			expectedPodName := getExpectedPodName(vm)
			podRunningRe, err := regexp.Compile(fmt.Sprintf("%s.*Running", expectedPodName))
			Expect(err).ToNot(HaveOccurred())

			Expect(podRunningRe.FindString(stdout)).ToNot(Equal(""), "Pod is not Running")

			By("Checking that VM is running")
			stdout, _, err = clientcmd.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")
		},
			Entry("with v1 api", "kubevirt.io/v1"),
			Entry("with v1alpha3 api", "kubevirt.io/v1alpha3"),
		)

		It("[test_id:264]should create and delete via command line", func() {
			vm, vmJson := createVMAndGenerateJson(false)

			By("Creating VM using k8s client binary")
			_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Invoking virtctl start")
			startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, vm.Name)
			Expect(startCommand()).To(Succeed())

			By("Waiting for VMI to start")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

			By("Checking that VM is running")
			stdout, _, err := clientcmd.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")

			By("Deleting VM using k8s client binary")
			_, _, err = clientcmd.RunCommand(k8sClient, "delete", "vm", vm.GetName())
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the VM gets deleted")
			waitForResourceDeletion(k8sClient, "vms", vm.GetName())

			By("Verifying pod gets deleted")
			expectedPodName := getExpectedPodName(vm)
			waitForResourceDeletion(k8sClient, "pods", expectedPodName)
		})

		Context("Deleting a running VM with high TerminationGracePeriod via command line", func() {
			DescribeTable("should force delete the VM", func(deleteFlags []string) {
				By("getting a VM with a high TerminationGracePeriod")
				vmi := libvmi.New(
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithTerminationGracePeriod(1600),
				)
				vm := tests.NewRandomVirtualMachine(vmi, true)
				vm.Namespace = testsuite.GetTestNamespace(vm)

				vmJson, err := tests.GenerateVMJson(vm, workDir)
				Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

				By("Creating VM using k8s client binary")
				_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).Should(BeRunning())

				By("Checking that VM is running")
				stdout, _, err := clientcmd.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
				Expect(err).ToNot(HaveOccurred())

				Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")

				By("Sending a force delete VM request using k8s client binary")
				deleteCmd := append([]string{"delete", "vm", vm.GetName()}, deleteFlags...)
				_, _, err = clientcmd.RunCommand(k8sClient, deleteCmd...)
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the VM gets deleted")
				waitForResourceDeletion(k8sClient, "vms", vm.GetName())

				By("Verifying pod gets deleted")
				expectedPodName := getExpectedPodName(vm)
				waitForResourceDeletion(k8sClient, "pods", expectedPodName)
			},
				Entry("when --force and --grace-period=0 are provided", []string{"--force", "--grace-period=0"}),
				Entry("when --now is provided", []string{"--now"}),
			)
		})

		Context("should not change anything if dry-run option is passed", func() {
			It("[test_id:7530]in start command", func() {
				vm, vmJson := createVMAndGenerateJson(false)

				By("Creating VM using k8s client binary")
				_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
				Expect(err).ToNot(HaveOccurred())

				By("Invoking virtctl start with dry-run option")
				startCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_START, "--namespace", vm.Namespace, "--dry-run", vm.Name)
				Expect(startCommand()).To(Succeed())

				_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			DescribeTable("in stop command", func(flags ...string) {
				vm, vmJson := createVMAndGenerateJson(true)

				By("Creating VM using k8s client binary")
				_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

				By("Getting current vmi instance")
				originalVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Getting current vm instance")
				originalVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				var args = []string{virtctl.COMMAND_STOP, "--namespace", vm.Namespace, vm.Name, "--dry-run"}
				if flags != nil {
					args = append(args, flags...)
				}
				By("Invoking virtctl stop with dry-run option")
				stopCommand := clientcmd.NewRepeatableVirtctlCommand(args...)
				Expect(stopCommand()).To(Succeed())

				By("Checking that VM is still running")
				stdout, _, err := clientcmd.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")

				By("Checking VM Running spec does not change")
				actualVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(actualVM.Spec.Running).To(BeEquivalentTo(originalVM.Spec.Running))
				actualRunStrategy, err := actualVM.RunStrategy()
				Expect(err).ToNot(HaveOccurred())
				originalRunStrategy, err := originalVM.RunStrategy()
				Expect(err).ToNot(HaveOccurred())
				Expect(actualRunStrategy).To(BeEquivalentTo(originalRunStrategy))

				By("Checking VMI TerminationGracePeriodSeconds does not change")
				actualVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(actualVMI.Spec.TerminationGracePeriodSeconds).To(BeEquivalentTo(originalVMI.Spec.TerminationGracePeriodSeconds))
				Expect(actualVMI.Status.Phase).To(BeEquivalentTo(originalVMI.Status.Phase))
			},

				Entry("[test_id:7529]with no other flags"),
				Entry("[test_id:7604]with grace period", "--grace-period=10", "--force"),
			)

			It("[test_id:7528]in restart command", func() {
				vm, vmJson := createVMAndGenerateJson(true)

				By("Creating VM using k8s client binary")
				_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to start")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

				By("Getting current vmi instance")
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Invoking virtctl restart with dry-run option")
				restartCommand := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_RESTART, "--namespace", vm.Namespace, "--dry-run", vm.Name)
				Expect(restartCommand()).To(Succeed())

				By("Comparing the CreationTimeStamp and UUID and check no Deletion Timestamp was set")
				newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.ObjectMeta.CreationTimestamp).To(Equal(newVMI.ObjectMeta.CreationTimestamp))
				Expect(vmi.ObjectMeta.UID).To(Equal(newVMI.ObjectMeta.UID))
				Expect(newVMI.ObjectMeta.DeletionTimestamp).To(BeNil())

				By("Checking that VM is running")
				stdout, _, err := clientcmd.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")
			})
		})

		It("[test_id:232]should create same manifest twice via command line", func() {
			vm, vmJson := createVMAndGenerateJson(true)

			By("Creating VM using k8s client binary")
			_, _, err := clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())

			By("Deleting VM using k8s client binary")
			_, _, err = clientcmd.RunCommand(k8sClient, "delete", "vm", vm.GetName())
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the VM gets deleted")
			waitForResourceDeletion(k8sClient, "vms", vm.GetName())

			By("Creating same VM using k8s client binary and same manifest")
			_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(BeRunning())
		})

		It("[test_id:233][posneg:negative]should fail when deleting nonexistent VM", func() {
			vmi := tests.NewRandomVirtualMachine(libvmi.NewAlpine(), false)

			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, stdErr, err := clientcmd.RunCommand(k8sClient, "delete", "vm", vmi.Name)
			Expect(err).To(HaveOccurred())
			Expect(strings.HasPrefix(stdErr, "Error from server (NotFound): virtualmachines.kubevirt.io")).To(BeTrue(), "should fail when deleting non existent VM")
		})

		Context("as ordinary OCP user trough test service account", func() {
			var testUser string

			BeforeEach(func() {
				testUser = "testuser-" + uuid.NewRandom().String()
			})

			Context("should succeed with right rights", func() {
				BeforeEach(func() {
					// kubectl doesn't have "adm" subcommand -- only oc does
					clientcmd.SkipIfNoCmd("oc")
					By("Ensuring the cluster has new test serviceaccount")
					stdOut, stdErr, err := clientcmd.RunCommand(k8sClient, "create", "user", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

					By("Ensuring user has the admin rights for the test namespace project")
					// This simulates the ordinary user as an admin in this project
					stdOut, stdErr, err = clientcmd.RunCommand(k8sClient, "adm", "policy", "add-role-to-user", "admin", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				AfterEach(func() {
					stdOut, stdErr, err := clientcmd.RunCommand(k8sClient, "adm", "policy", "remove-role-from-user", "admin", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

					stdOut, stdErr, err = clientcmd.RunCommand(k8sClient, "delete", "user", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				It("[test_id:2839]should create VM via command line", func() {
					By("Checking VM creation permission using k8s client binary")
					stdOut, _, err := clientcmd.RunCommand(k8sClient, "auth", "can-i", "create", "vms", "--as", testUser)
					Expect(err).ToNot(HaveOccurred())
					Expect(strings.TrimSpace(stdOut)).To(Equal("yes"))
				})
			})

			Context("should fail without right rights", func() {
				BeforeEach(func() {
					By("Ensuring the cluster has new test serviceaccount")
					stdOut, stdErr, err := clientcmd.RunCommandWithNS(testsuite.GetTestNamespace(nil), k8sClient, "create", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				AfterEach(func() {
					stdOut, stdErr, err := clientcmd.RunCommandWithNS(testsuite.GetTestNamespace(nil), k8sClient, "delete", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				It("[test_id:2914]should create VM via command line", func() {
					By("Checking VM creation permission using k8s client binary")
					stdOut, _, err := clientcmd.RunCommand(k8sClient, "auth", "can-i", "create", "vms", "--as", testUser)
					// non-zero exit code
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("exit status 1"))
					Expect(strings.TrimSpace(stdOut)).To(Equal("no"))
				})
			})
		})

	})

	Context("crash loop backoff", func() {
		It("should backoff attempting to create a new VMI when 'runStrategy: Always' during crash loop.", func() {
			By("Creating VirtualMachine")
			vm := createRunningVM(virtClient, libvmi.NewCirros(
				libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
			))

			By("waiting for crash loop state")
			Eventually(ThisVM(vm), 60*time.Second, 5*time.Second).Should(beInCrashLoop())

			By("Testing that the failure count is within the expected range over a period of time")
			maxExpectedFailCount := 3
			Consistently(func() error {
				// get the VM and verify the failure count is less than 4 over a minute,
				// indicating that backoff is occuring
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if vm.Status.StartFailure == nil {
					return fmt.Errorf("start failure count not detected")
				} else if vm.Status.StartFailure.ConsecutiveFailCount > maxExpectedFailCount {
					return fmt.Errorf("consecutive fail count is higher than %d", maxExpectedFailCount)
				}

				return nil
			}, 1*time.Minute, 5*time.Second).Should(BeNil())

			By("Updating the VMI template to correct the crash loop")
			Eventually(func() error {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				delete(vm.Spec.Template.ObjectMeta.Annotations, v1.FuncTestLauncherFailFastAnnotation)
				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
				return err
			}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			By("Waiting on crash loop status to be removed.")
			Eventually(ThisVM(vm), 300*time.Second, 5*time.Second).Should(notBeInCrashLoop())
		})

		It("should be able to stop a VM during crashloop backoff when when 'runStrategy: Always' is set", func() {
			By("Creating VirtualMachine")
			vm := createRunningVM(virtClient, libvmi.NewCirros(
				libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
			))

			By("waiting for crash loop state")
			Eventually(ThisVM(vm), 60*time.Second, 5*time.Second).Should(beInCrashLoop())

			By("Invoking virtctl stop while in a crash loop")
			stopCmd := clientcmd.NewRepeatableVirtctlCommand(virtctl.COMMAND_STOP, vm.Name, "--namespace", vm.Namespace)
			Expect(stopCmd()).To(Succeed())

			By("Waiting on crash loop status to be removed.")
			Eventually(ThisVM(vm), 120*time.Second, 5*time.Second).Should(notBeInCrashLoop())
		})
	})

	Context("VirtualMachineControllerFinalizer", func() {
		const customFinalizer = "customFinalizer"

		var (
			vmi *v1.VirtualMachineInstance
			vm  *v1.VirtualMachine
		)

		BeforeEach(func() {
			vmi = tests.NewRandomVMI()
			vm = tests.NewRandomVirtualMachine(vmi, true)
			Expect(vm.Finalizers).To(BeEmpty())
			vm.Finalizers = append(vm.Finalizers, customFinalizer)
		})

		AfterEach(func() {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			oldFinalizers, err := json.Marshal(vm.GetFinalizers())
			Expect(err).ToNot(HaveOccurred())

			newVm := vm.DeepCopy()
			controller.RemoveFinalizer(newVm, customFinalizer)
			newFinalizers, err := json.Marshal(newVm.GetFinalizers())
			Expect(err).ToNot(HaveOccurred())

			var ops []string
			ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/finalizers", "value": %s }`, string(oldFinalizers)))
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/finalizers", "value": %s }`, string(newFinalizers)))

			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensure the vm has disappeared")
			Eventually(func() bool {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 2*time.Minute, 1*time.Second).Should(BeTrue(), fmt.Sprintf("vm %s is not deleted", vm.Name))
		})

		It("should be added when the vm is created and removed when the vm is being deleted", func() {
			By("Creating VirtualMachine")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeTrue())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
			}, 2*time.Minute, 1*time.Second).Should(Succeed())

			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeFalse())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
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
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the VirtualMachine has the VirtualMachineControllerFinalizer, customFinalizer and revisionName")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeTrue())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
				g.Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
			}, 2*time.Minute, 1*time.Second).Should(Succeed())

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("deleting the ControllerRevision associated with the VirtualMachine and VirtualMachineClusterInstancetype %s", vm.Spec.Instancetype.RevisionName))
			err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("deleting the VirtualMachineClusterInstancetype")
			err = virtClient.VirtualMachineClusterInstancetype().Delete(context.Background(), vm.Spec.Instancetype.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("deleting the VirtualMachine")
			err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the VirtualMachineControllerFinalizer has been removed from the VirtualMachine")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(controller.HasFinalizer(vm, v1.VirtualMachineControllerFinalizer)).To(BeFalse())
				g.Expect(controller.HasFinalizer(vm, customFinalizer)).To(BeTrue())
			}, 2*time.Minute, 1*time.Second).Should(Succeed())
		})
	})
})

func getExpectedPodName(vm *v1.VirtualMachine) string {
	maxNameLength := 63
	podNamePrefix := "virt-launcher-"
	podGeneratedSuffixLen := 5
	charCountFromName := maxNameLength - len(podNamePrefix) - podGeneratedSuffixLen
	expectedPodName := fmt.Sprintf(fmt.Sprintf("virt-launcher-%%.%ds", charCountFromName), vm.GetName())
	return expectedPodName
}

func waitForResourceDeletion(k8sClient string, resourceType string, resourceName string) {
	Eventually(func() bool {
		stdout, _, err := clientcmd.RunCommand(k8sClient, "get", resourceType)
		Expect(err).ToNot(HaveOccurred())
		return strings.Contains(stdout, resourceName)
	}, 120*time.Second, 1*time.Second).Should(BeFalse(), "VM was not deleted")
}

func createVM(virtClient kubecli.KubevirtClient, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
	By("Creating stopped VirtualMachine")
	vm := tests.NewRandomVirtualMachine(template, false)
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
	Expect(err).ToNot(HaveOccurred())
	return vm
}

func createRunningVM(virtClient kubecli.KubevirtClient, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
	By("Creating running VirtualMachine")
	vm := tests.NewRandomVirtualMachine(template, true)
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
	Expect(err).ToNot(HaveOccurred())
	return vm
}

func startVM(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Starting the VirtualMachine")
	err := tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta k8smetav1.ObjectMeta) error {
		vm, err := virtClient.VirtualMachine(meta.Namespace).Get(context.Background(), meta.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm.Spec.Running = nil
		runStrategyAlways := v1.RunStrategyAlways
		vm.Spec.RunStrategy = &runStrategyAlways
		_, err = virtClient.VirtualMachine(meta.Namespace).Update(context.Background(), vm)
		return err
	})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for VMI to be running")
	Eventually(ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 1*time.Second).Should(BeRunning())

	By("Waiting for VM to be ready")
	Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())

	vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	return vm
}

func stopVM(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Stopping the VirtualMachine")
	err := tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta k8smetav1.ObjectMeta) error {
		vm, err := virtClient.VirtualMachine(meta.Namespace).Get(context.Background(), meta.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm.Spec.Running = nil
		runStrategyHalted := v1.RunStrategyHalted
		vm.Spec.RunStrategy = &runStrategyHalted
		_, err = virtClient.VirtualMachine(meta.Namespace).Update(context.Background(), vm)
		return err
	})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for VMI to not exist")
	Eventually(ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 1*time.Second).ShouldNot(Exist())

	By("Waiting for VM to not be ready")
	Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(Not(beReady()))

	vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &k8smetav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	return vm
}

func beCreated() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Created": BeTrue(),
		}),
	}))
}

func beReady() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Ready": BeTrue(),
		}),
	}))
}

func beRestarted(oldUID types.UID) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"UID": Not(Equal(oldUID)),
		}),
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Phase": Equal(v1.Running),
		}),
	}))
}

func beInCrashLoop() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"PrintableStatus": Equal(v1.VirtualMachineStatusCrashLoopBackOff),
			"StartFailure": gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ConsecutiveFailCount": BeNumerically(">", 0),
			})),
		}),
	}))
}

func notBeInCrashLoop() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"StartFailure": BeNil(),
		}),
	}))
}

func havePrintableStatus(status v1.VirtualMachinePrintableStatus) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"PrintableStatus": Equal(status),
		}),
	}))
}

func haveStateChangeRequests() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"StateChangeRequests": Not(BeEmpty()),
		}),
	}))
}
