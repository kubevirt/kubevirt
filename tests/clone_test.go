package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

	virtsnapshot "kubevirt.io/api/snapshot"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clone "kubevirt.io/api/clone/v1beta1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

const (
	vmAPIGroup = "kubevirt.io"
)

var _ = Describe("VirtualMachineClone Tests", Serial, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		config.EnableFeatureGate(featuregate.SnapshotGate)

		format.MaxLength = 0
	})

	createClone := func(vmClone *clone.VirtualMachineClone) *clone.VirtualMachineClone {
		By(fmt.Sprintf("Creating clone object %s", vmClone.Name))
		vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Create(context.Background(), vmClone, v1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		return vmClone
	}
	waitCloneSucceeded := func(vmClone *clone.VirtualMachineClone) {
		By(fmt.Sprintf("Waiting for the clone %s to finish", vmClone.Name))
		Eventually(func() clone.VirtualMachineClonePhase {
			vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, v1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			return vmClone.Status.Phase
		}, 3*time.Minute, 3*time.Second).Should(Equal(clone.Succeeded), "clone should finish successfully")
	}

	createCloneAndWaitForCompletion := func(vmClone *clone.VirtualMachineClone) {
		vmClone = createClone(vmClone)
		waitCloneSucceeded(vmClone)
	}

	expectVMRunnable := func(vm *virtv1.VirtualMachine, login console.LoginToFunction) *virtv1.VirtualMachine {
		By(fmt.Sprintf("Starting VM %s", vm.Name))
		Expect(virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &virtv1.StartOptions{})).To(Succeed())
		Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())
		targetVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, v1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		err = login(targetVMI)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &virtv1.StopOptions{})).To(Succeed())
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 1*time.Second).ShouldNot(Exist())
		Eventually(ThisVM(vm), 300*time.Second, 1*time.Second).Should(Not(BeReady()))

		return vm
	}

	Context("VM clone", func() {

		const (
			targetVMName                     = "vm-clone-target"
			cloneShouldEqualSourceMsgPattern = "cloned VM's %s should be equal to source"

			key1   = "key1"
			key2   = "key2"
			value1 = "value1"
			value2 = "value2"
		)

		var (
			sourceVM, targetVM *virtv1.VirtualMachine
			vmClone            *clone.VirtualMachineClone
			defaultVMIOptions  = []libvmi.Option{
				libvmi.WithLabel(key1, value1),
				libvmi.WithLabel(key2, value2),
				libvmi.WithAnnotation(key1, value1),
				libvmi.WithAnnotation(key2, value2),
			}
		)

		expectEqualStrMap := func(actual, expected map[string]string, expectationMsg string, keysToExclude ...string) {
			expected = filterOutIrrelevantKeys(expected)
			actual = filterOutIrrelevantKeys(actual)

			for _, key := range keysToExclude {
				delete(expected, key)
			}

			Expect(actual).To(Equal(expected), expectationMsg)
		}

		expectEqualLabels := func(targetVM, sourceVM *virtv1.VirtualMachine, keysToExclude ...string) {
			expectEqualStrMap(targetVM.Labels, sourceVM.Labels, fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "labels"), keysToExclude...)
		}
		expectEqualTemplateLabels := func(targetVM, sourceVM *virtv1.VirtualMachine, keysToExclude ...string) {
			expectEqualStrMap(targetVM.Spec.Template.ObjectMeta.Labels, sourceVM.Spec.Template.ObjectMeta.Labels, fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "template.labels"), keysToExclude...)
		}

		expectEqualAnnotations := func(targetVM, sourceVM *virtv1.VirtualMachine, keysToExclude ...string) {
			expectEqualStrMap(targetVM.Annotations, sourceVM.Annotations, fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "annotations"), keysToExclude...)
		}
		expectEqualTemplateAnnotations := func(targetVM, sourceVM *virtv1.VirtualMachine, keysToExclude ...string) {
			expectEqualStrMap(targetVM.Spec.Template.ObjectMeta.Annotations, sourceVM.Spec.Template.ObjectMeta.Annotations, fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "template.annotations"), keysToExclude...)
		}

		expectVMsToEqualExcludingMACAndFirmwareIDs := func(vm1, vm2 *virtv1.VirtualMachine) {
			vm1Spec := vm1.Spec.DeepCopy()
			vm2Spec := vm2.Spec.DeepCopy()

			for _, spec := range []*virtv1.VirtualMachineSpec{vm1Spec, vm2Spec} {
				for i := range spec.Template.Spec.Domain.Devices.Interfaces {
					spec.Template.Spec.Domain.Devices.Interfaces[i].MacAddress = ""
				}
				if spec.Template.Spec.Domain.Firmware != nil {
					spec.Template.Spec.Domain.Firmware.UUID = ""
					spec.Template.Spec.Domain.Firmware.Serial = ""
				}
			}

			Expect(vm1Spec).To(Equal(vm2Spec), fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "spec not including mac addresses and firmware UUID"))
		}

		expectSpecsToEqualExceptForFirmwareUUIDAndSerial := func(vm1, vm2 *virtv1.VirtualMachine) {
			vm1Spec := vm1.Spec.DeepCopy()
			vm2Spec := vm2.Spec.DeepCopy()

			for _, spec := range []*virtv1.VirtualMachineSpec{vm1Spec, vm2Spec} {
				if spec.Template.Spec.Domain.Firmware != nil {
					spec.Template.Spec.Domain.Firmware.UUID = ""
					spec.Template.Spec.Domain.Firmware.Serial = ""
				}
			}

			Expect(vm1Spec).To(Equal(vm2Spec), fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "spec not including firmware UUID"))
		}

		generateCloneFromVM := func() *clone.VirtualMachineClone {
			return generateCloneFromVMWithParams(sourceVM.Name, sourceVM.Namespace, targetVMName)
		}

		Context("[sig-compute]simple VM and cloning operations", decorators.SigCompute, func() {

			expectVMRunnable := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
				return expectVMRunnable(vm, console.LoginToCirros)
			}

			It("simple default clone", func() {
				sourceVM, err = createSourceVM(defaultVMIOptions...)
				Expect(err).ShouldNot(HaveOccurred())
				vmClone = generateCloneFromVM()

				createCloneAndWaitForCompletion(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				expectSpecsToEqualExceptForFirmwareUUIDAndSerial(sourceVM, targetVM)
				expectEqualLabels(targetVM, sourceVM)
				expectEqualAnnotations(targetVM, sourceVM)
				expectEqualTemplateLabels(targetVM, sourceVM)
				expectEqualTemplateAnnotations(targetVM, sourceVM)

				By("Making sure snapshot and restore objects are cleaned up")
				Expect(vmClone.Status.SnapshotName).To(BeNil())
				Expect(vmClone.Status.RestoreName).To(BeNil())

				err = virtClient.VirtualMachine(targetVM.Namespace).Delete(context.Background(), targetVM.Name, v1.DeleteOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				Eventually(func() error {
					_, err := virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, v1.GetOptions{})
					return err
				}, 120*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "VM clone should be successfully deleted")
			})

			It("simple clone with snapshot source, create clone before snapshot", func() {
				By("Creating a VM")
				sourceVM, err = createSourceVM(defaultVMIOptions...)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() virtv1.VirtualMachinePrintableStatus {
					sourceVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), sourceVM.Name, v1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return sourceVM.Status.PrintableStatus
				}, 30*time.Second, 1*time.Second).Should(Equal(virtv1.VirtualMachineStatusStopped))

				snapshot := generateSnapshot(sourceVM.Name, sourceVM.Namespace)
				By("Creating a clone before snapshot source created")
				vmClone = generateCloneFromSnapshot(snapshot.Name, snapshot.Namespace, targetVMName)
				vmClone = createClone(vmClone)

				events.ExpectEvent(vmClone, k8sv1.EventTypeNormal, "SourceDoesNotExist")

				By("Creating a snapshot from VM")
				snapshot, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitCloneSucceeded(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				By("Making sure snapshot source was not deleted when clone completed")
				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Get(context.Background(), snapshot.Name, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("clone with only some of labels/annotations", func() {
				sourceVM, err = createSourceVM(defaultVMIOptions...)
				Expect(err).ShouldNot(HaveOccurred())
				vmClone = generateCloneFromVM()

				vmClone.Spec.LabelFilters = []string{
					"*",
					"!" + key2,
				}
				vmClone.Spec.AnnotationFilters = []string{
					key1,
				}
				createCloneAndWaitForCompletion(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				expectSpecsToEqualExceptForFirmwareUUIDAndSerial(sourceVM, targetVM)
				expectEqualLabels(targetVM, sourceVM, key2)
				expectEqualAnnotations(targetVM, sourceVM, key2)
			})

			It("clone with only some of template.labels/template.annotations", func() {
				sourceVM, err = createSourceVM(defaultVMIOptions...)
				Expect(err).ShouldNot(HaveOccurred())
				vmClone = generateCloneFromVM()

				vmClone.Spec.Template.LabelFilters = []string{
					"*",
					"!" + key2,
				}
				vmClone.Spec.Template.AnnotationFilters = []string{
					key1,
				}
				createCloneAndWaitForCompletion(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				expectEqualTemplateLabels(targetVM, sourceVM, key2)
				expectEqualTemplateAnnotations(targetVM, sourceVM, key2)
			})

			It("clone with changed MAC address", func() {
				const newMacAddress = "BE-AD-00-00-BE-04"
				options := append(
					defaultVMIOptions,
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				)
				sourceVM, err = createSourceVM(options...)
				Expect(err).ShouldNot(HaveOccurred())

				srcInterfaces := sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces
				Expect(srcInterfaces).ToNot(BeEmpty())
				srcInterface := srcInterfaces[0]

				vmClone = generateCloneFromVM()
				vmClone.Spec.NewMacAddresses = map[string]string{
					srcInterface.Name: newMacAddress,
				}

				createCloneAndWaitForCompletion(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				By("Finding target interface with same name as original")
				var targetInterface *virtv1.Interface
				targetInterfaces := targetVM.Spec.Template.Spec.Domain.Devices.Interfaces
				for _, iface := range targetInterfaces {
					if iface.Name == srcInterface.Name {
						targetInterface = iface.DeepCopy()
						break
					}
				}
				Expect(targetInterface).ToNot(BeNil(), fmt.Sprintf("clone target does not have interface with name %s", srcInterface.Name))

				By("Making sure new mac address is applied to target VM")
				Expect(targetInterface.MacAddress).ToNot(Equal(srcInterface.MacAddress))

				expectVMsToEqualExcludingMACAndFirmwareIDs(targetVM, sourceVM)
				expectEqualLabels(targetVM, sourceVM)
				expectEqualAnnotations(targetVM, sourceVM)
				expectEqualTemplateLabels(targetVM, sourceVM)
				expectEqualTemplateAnnotations(targetVM, sourceVM)
			})

			Context("regarding domain Firmware", func() {
				It("clone with changed SMBios serial", func() {
					const sourceSerial = "source-serial"
					const targetSerial = "target-serial"

					options := append(
						defaultVMIOptions,
						withFirmware(&virtv1.Firmware{Serial: sourceSerial}),
					)
					sourceVM, err = createSourceVM(options...)
					Expect(err).ShouldNot(HaveOccurred())

					vmClone = generateCloneFromVM()
					vmClone.Spec.NewSMBiosSerial = pointer.P(targetSerial)

					createCloneAndWaitForCompletion(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Making sure target is runnable")
					targetVM = expectVMRunnable(targetVM)

					By("Making sure new smBios serial is applied to target VM")
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(sourceVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware.Serial).ToNot(Equal(sourceVM.Spec.Template.Spec.Domain.Firmware.Serial))

					expectEqualLabels(targetVM, sourceVM)
					expectEqualAnnotations(targetVM, sourceVM)
					expectEqualTemplateLabels(targetVM, sourceVM)
					expectEqualTemplateAnnotations(targetVM, sourceVM)
				})

				It("should strip firmware UUID", func() {
					const fakeFirmwareUUID = "fake-uuid"

					options := append(
						defaultVMIOptions,
						withFirmware(&virtv1.Firmware{UUID: fakeFirmwareUUID}),
					)
					sourceVM, err = createSourceVM(options...)
					Expect(err).ShouldNot(HaveOccurred())
					vmClone = generateCloneFromVM()

					createCloneAndWaitForCompletion(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Making sure target is runnable")
					targetVM = expectVMRunnable(targetVM)

					By("Making sure new smBios serial is applied to target VM")
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(sourceVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware.UUID).ToNot(Equal(sourceVM.Spec.Template.Spec.Domain.Firmware.UUID))
				})
			})

		})

		Context("[sig-storage]with more complicated VM", decorators.SigStorage, func() {

			expectVMRunnable := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
				return expectVMRunnable(vm, console.LoginToAlpine)
			}

			generateVMWithStorageClass := func(storageClass string, runStrategy virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachine {
				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(storageClass),
						libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine))),
					),
				)
				vm := libstorage.RenderVMWithDataVolumeTemplate(dv)
				vm.Spec.RunStrategy = &runStrategy
				return vm
			}

			createVM := func(vm *virtv1.VirtualMachine, storageClass string, runStrategy virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachine {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				if !(runStrategy == virtv1.RunStrategyAlways) && libstorage.IsStorageClassBindingModeWaitForFirstConsumer(storageClass) {
					return vm
				}

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					libstorage.EventuallyDVWith(vm.Namespace, dvt.Name, 180, HaveSucceeded())
				}

				return vm
			}

			createVMWithStorageClass := func(storageClass string, runStrategy virtv1.VirtualMachineRunStrategy) *virtv1.VirtualMachine {
				vm := generateVMWithStorageClass(storageClass, runStrategy)
				return createVM(vm, storageClass, runStrategy)
			}

			Context("and snapshot storage class", decorators.RequiresSnapshotStorageClass, func() {
				var (
					snapshotStorageClass string
				)

				BeforeEach(func() {
					snapshotStorageClass, err = libstorage.GetSnapshotStorageClass(virtClient, k8s.Client())
					Expect(err).ToNot(HaveOccurred())
					Expect(snapshotStorageClass).ToNot(BeEmpty(), "no storage class with snapshot support")
				})

				It("with a simple clone, create clone before VM", func() {
					runStrategy := virtv1.RunStrategyHalted
					if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(snapshotStorageClass) {
						// with wffc need to start the virtual machine
						// in order for the pvc to be populated
						runStrategy = virtv1.RunStrategyAlways
					}
					sourceVM = generateVMWithStorageClass(snapshotStorageClass, runStrategy)
					vmClone = generateCloneFromVM()
					vmClone = createClone(vmClone)

					events.ExpectEvent(vmClone, k8sv1.EventTypeNormal, "SourceDoesNotExist")
					createVM(sourceVM, snapshotStorageClass, runStrategy)

					waitCloneSucceeded(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Making sure target is runnable")
					targetVM = expectVMRunnable(targetVM)

					expectEqualLabels(targetVM, sourceVM)
					expectEqualAnnotations(targetVM, sourceVM)
					expectEqualTemplateLabels(targetVM, sourceVM)
					expectEqualTemplateAnnotations(targetVM, sourceVM)
				})

				Context("with instancetype and preferences", decorators.SigComputeInstancetype, func() {
					var (
						instancetype *instancetypev1beta1.VirtualMachineInstancetype
						preference   *instancetypev1beta1.VirtualMachinePreference
					)

					BeforeEach(func() {
						ns := testsuite.GetTestNamespace(nil)
						instancetype = &instancetypev1beta1.VirtualMachineInstancetype{
							ObjectMeta: v1.ObjectMeta{
								GenerateName: "vm-instancetype-",
								Namespace:    ns,
							},
							Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1beta1.CPUInstancetype{
									Guest: 1,
								},
								Memory: instancetypev1beta1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						}
						instancetype, err := virtClient.VirtualMachineInstancetype(ns).Create(context.Background(), instancetype, v1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						preferredCPUTopology := instancetypev1beta1.Sockets
						preference = &instancetypev1beta1.VirtualMachinePreference{
							ObjectMeta: v1.ObjectMeta{
								GenerateName: "vm-preference-",
								Namespace:    ns,
							},
							Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &preferredCPUTopology,
								},
							},
						}
						preference, err := virtClient.VirtualMachinePreference(ns).Create(context.Background(), preference, v1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						sourceVM = libvmi.NewVirtualMachine(libvmifact.NewGuestless())
						sourceVM.Spec.Template.Spec.Domain.Resources = virtv1.ResourceRequirements{}
						sourceVM.Spec.Instancetype = &virtv1.InstancetypeMatcher{
							Name: instancetype.Name,
							Kind: "VirtualMachineInstanceType",
						}
						sourceVM.Spec.Preference = &virtv1.PreferenceMatcher{
							Name: preference.Name,
							Kind: "VirtualMachinePreference",
						}
					})

					DescribeTable("should create new ControllerRevisions for cloned VM", Label("instancetype", "clone"), func(runStrategy virtv1.VirtualMachineRunStrategy) {
						sourceVM.Spec.RunStrategy = &runStrategy
						sourceVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), sourceVM, v1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						By("Waiting until the source VM has instancetype and preference RevisionNames")
						Eventually(matcher.ThisVM(sourceVM)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

						vmClone = generateCloneFromVM()
						createCloneAndWaitForCompletion(vmClone)

						By("Waiting until the targetVM has instancetype and preference RevisionNames")
						Eventually(matcher.ThisVMWith(testsuite.GetTestNamespace(sourceVM), targetVMName)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

						By("Asserting that the targetVM has new instancetype and preference controllerRevisions")
						sourceVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(sourceVM)).Get(context.Background(), sourceVM.Name, v1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						targetVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(sourceVM)).Get(context.Background(), targetVMName, v1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						Expect(targetVM.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(Equal(sourceVM.Status.InstancetypeRef.ControllerRevisionRef.Name), "source and target instancetype revision names should not be equal")
						Expect(targetVM.Status.PreferenceRef.ControllerRevisionRef.Name).ToNot(Equal(sourceVM.Status.PreferenceRef.ControllerRevisionRef.Name), "source and target preference revision names should not be equal")

						By("Asserting that the source and target ControllerRevisions contain the same Object")
						Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Status.InstancetypeRef.ControllerRevisionRef.Name, targetVM.Status.InstancetypeRef.ControllerRevisionRef.Name, k8s.Client())).To(BeTrue(), "source and target instance type controller revisions are expected to be equal")
						Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Status.PreferenceRef.ControllerRevisionRef.Name, targetVM.Status.PreferenceRef.ControllerRevisionRef.Name, k8s.Client())).To(BeTrue(), "source and target preference controller revisions are expected to be equal")
					},
						Entry("with a running VM", virtv1.RunStrategyAlways),
						Entry("with a stopped VM", virtv1.RunStrategyHalted),
					)
				})

				It("double cloning: clone target as a clone source", func() {
					addCloneAnnotationAndLabelFilters := func(vmClone *clone.VirtualMachineClone) {
						filters := []string{"somekey/*"}
						vmClone.Spec.LabelFilters = filters
						vmClone.Spec.AnnotationFilters = filters
						vmClone.Spec.Template.LabelFilters = filters
						vmClone.Spec.Template.AnnotationFilters = filters
					}
					generateCloneWithFilters := func(sourceVM *virtv1.VirtualMachine, targetVMName string) *clone.VirtualMachineClone {
						vmclone := generateCloneFromVMWithParams(sourceVM.Name, sourceVM.Namespace, targetVMName)
						addCloneAnnotationAndLabelFilters(vmclone)
						return vmclone
					}

					runStrategy := virtv1.RunStrategyHalted
					wffcSC := libstorage.IsStorageClassBindingModeWaitForFirstConsumer(snapshotStorageClass)
					if wffcSC {
						// with wffc need to start the virtual machine
						// in order for the pvc to be populated
						runStrategy = virtv1.RunStrategyAlways
					}
					sourceVM = createVMWithStorageClass(snapshotStorageClass, runStrategy)
					vmClone = generateCloneWithFilters(sourceVM, targetVMName)

					createCloneAndWaitForCompletion(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					if wffcSC {
						// run the virtual machine for the clone dv to be populated
						expectVMRunnable(targetVM)
					}

					By("Creating another clone from the target VM")
					const cloneFromCloneName = "vm-clone-from-clone"
					vmCloneFromClone := generateCloneWithFilters(targetVM, cloneFromCloneName)
					vmCloneFromClone.Name = "test-clone-from-clone"
					createCloneAndWaitForCompletion(vmCloneFromClone)

					By(fmt.Sprintf("Getting the target VM %s from clone", cloneFromCloneName))
					targetVMCloneFromClone, err := virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), cloneFromCloneName, v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					expectVMRunnable(targetVMCloneFromClone)
					expectEqualLabels(targetVMCloneFromClone, sourceVM)
					expectEqualAnnotations(targetVMCloneFromClone, sourceVM)
					expectEqualTemplateLabels(targetVMCloneFromClone, sourceVM, "name")
					expectEqualTemplateAnnotations(targetVMCloneFromClone, sourceVM)
				})

				Context("with WaitForFirstConsumer binding mode", decorators.RequiresWFFCStorageClass, func() {
					BeforeEach(func() {
						snapshotStorageClass, err = libstorage.GetWFFCStorageSnapshotClass(virtClient, k8s.Client())
						Expect(err).ToNot(HaveOccurred())
						if snapshotStorageClass == "" {
							Fail("Failing test, no storage class with snapshot support and wffc binding mode")
						}
					})

					It("should not delete the vmsnapshot and vmrestore until all the pvc(s) are bound", func() {
						addCloneAnnotationAndLabelFilters := func(vmClone *clone.VirtualMachineClone) {
							filters := []string{"somekey/*"}
							vmClone.Spec.LabelFilters = filters
							vmClone.Spec.AnnotationFilters = filters
							vmClone.Spec.Template.LabelFilters = filters
							vmClone.Spec.Template.AnnotationFilters = filters
						}
						generateCloneWithFilters := func(sourceVM *virtv1.VirtualMachine, targetVMName string) *clone.VirtualMachineClone {
							vmclone := generateCloneFromVMWithParams(sourceVM.Name, sourceVM.Namespace, targetVMName)
							addCloneAnnotationAndLabelFilters(vmclone)
							return vmclone
						}

						sourceVM = createVMWithStorageClass(snapshotStorageClass, virtv1.RunStrategyAlways)
						vmClone = generateCloneWithFilters(sourceVM, targetVMName)
						Expect(virtClient.VirtualMachine(sourceVM.Namespace).Stop(context.Background(), sourceVM.Name, &virtv1.StopOptions{})).To(Succeed())
						Eventually(ThisVMIWith(sourceVM.Namespace, sourceVM.Name), 300*time.Second, 1*time.Second).ShouldNot(Exist())
						Eventually(ThisVM(sourceVM), 300*time.Second, 1*time.Second).Should(Not(BeReady()))

						createCloneAndWaitForCompletion(vmClone)

						By(fmt.Sprintf("Getting the target VM %s", targetVMName))
						targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), targetVMName, v1.GetOptions{})
						Expect(err).ShouldNot(HaveOccurred())

						vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, v1.GetOptions{})
						Expect(err).ShouldNot(HaveOccurred())
						Expect(vmClone.Status.SnapshotName).ShouldNot(BeNil())
						vmSnapshotName := vmClone.Status.SnapshotName
						Expect(vmClone.Status.RestoreName).ShouldNot(BeNil())
						vmRestoreName := vmClone.Status.RestoreName
						Consistently(func(g Gomega) {
							vmSnapshot, err := virtClient.VirtualMachineSnapshot(vmClone.Namespace).Get(context.Background(), *vmSnapshotName, v1.GetOptions{})
							g.Expect(err).ShouldNot(HaveOccurred())
							g.Expect(vmSnapshot).ShouldNot(BeNil())
							vmRestore, err := virtClient.VirtualMachineRestore(vmClone.Namespace).Get(context.Background(), *vmRestoreName, v1.GetOptions{})
							g.Expect(err).ShouldNot(HaveOccurred())
							g.Expect(vmRestore).ShouldNot(BeNil())
						}, 30*time.Second).Should(Succeed(), "vmsnapshot and vmrestore should not be deleted until the pvc is bound")

						By(fmt.Sprintf("Starting the target VM %s", targetVMName))
						err = virtClient.VirtualMachine(testsuite.GetTestNamespace(targetVM)).Start(context.Background(), targetVMName, &virtv1.StartOptions{Paused: false})
						Expect(err).ToNot(HaveOccurred())
						Eventually(func(g Gomega) {
							_, err := virtClient.VirtualMachineSnapshot(vmClone.Namespace).Get(context.Background(), *vmSnapshotName, v1.GetOptions{})
							g.Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
							_, err = virtClient.VirtualMachineRestore(vmClone.Namespace).Get(context.Background(), *vmRestoreName, v1.GetOptions{})
							g.Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
						}, 1*time.Minute).Should(Succeed(), "vmsnapshot and vmrestore should be deleted once the pvc is bound")
					})

				})

			})
		})
	})
})

func withFirmware(firmware *virtv1.Firmware) libvmi.Option {
	return func(vmi *virtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Firmware = firmware
	}
}

func createSourceVM(options ...libvmi.Option) (*virtv1.VirtualMachine, error) {
	vmi := libvmifact.NewCirros(options...)
	vmi.Namespace = testsuite.GetTestNamespace(nil)
	vm := libvmi.NewVirtualMachine(vmi,
		libvmi.WithAnnotations(vmi.Annotations),
		libvmi.WithLabels(vmi.Labels))

	By(fmt.Sprintf("Creating VM %s", vm.Name))
	virtClient := kubevirt.Client()
	return virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
}

func generateSnapshot(vmName, vmNamespace string) *snapshotv1.VirtualMachineSnapshot {
	snapshot := &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: v1.ObjectMeta{
			Name:      "snapshot-" + vmName,
			Namespace: vmNamespace,
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: pointer.P(vmAPIGroup),
				Kind:     "VirtualMachine",
				Name:     vmName,
			},
		},
	}
	return snapshot
}

func generateCloneFromSnapshot(snapshotName, namespace, targetVMName string) *clone.VirtualMachineClone {
	vmClone := kubecli.NewMinimalCloneWithNS("testclone", namespace)

	cloneSourceRef := &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.P(virtsnapshot.GroupName),
		Kind:     "VirtualMachineSnapshot",
		Name:     snapshotName,
	}

	cloneTargetRef := &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.P(vmAPIGroup),
		Kind:     "VirtualMachine",
		Name:     targetVMName,
	}

	vmClone.Spec.Source = cloneSourceRef
	vmClone.Spec.Target = cloneTargetRef

	return vmClone
}

func generateCloneFromVMWithParams(sourceVMName, sourceVMNamespace, targetVMName string) *clone.VirtualMachineClone {
	vmClone := kubecli.NewMinimalCloneWithNS("testclone", sourceVMNamespace)

	cloneSourceRef := &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.P(vmAPIGroup),
		Kind:     "VirtualMachine",
		Name:     sourceVMName,
	}

	cloneTargetRef := cloneSourceRef.DeepCopy()
	cloneTargetRef.Name = targetVMName

	vmClone.Spec.Source = cloneSourceRef
	vmClone.Spec.Target = cloneTargetRef

	return vmClone
}

func filterOutIrrelevantKeys(in map[string]string) map[string]string {
	out := make(map[string]string)

	for key, val := range in {
		if !strings.Contains(key, "kubevirt.io") && !strings.Contains(key, "kubemacpool.io") {
			out[key] = val
		}
	}

	return out
}
