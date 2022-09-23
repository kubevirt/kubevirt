package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	vmAPIGroup = "kubevirt.io"
)

type loginFunction func(*virtv1.VirtualMachineInstance) error

var _ = SIGDescribe("[Serial]VirtualMachineClone Tests", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.EnableFeatureGate(virtconfig.SnapshotGate)

		format.MaxLength = 0
	})

	createVM := func(options ...libvmi.Option) (vm *virtv1.VirtualMachine) {
		vmi := libvmi.NewCirros(options...)
		vmi.Namespace = util.NamespaceTestDefault
		vm = tests.NewRandomVirtualMachine(vmi, false)
		vm.Annotations = vmi.Annotations
		vm.Labels = vmi.Labels

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
		Expect(err).ShouldNot(HaveOccurred())

		return
	}

	generateClone := func(sourceVM *virtv1.VirtualMachine, targetVMName string) *clonev1alpha1.VirtualMachineClone {
		vmClone := kubecli.NewMinimalCloneWithNS("testclone", util.NamespaceTestDefault)

		cloneSourceRef := &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.String(vmAPIGroup),
			Kind:     "VirtualMachine",
			Name:     sourceVM.Name,
		}

		cloneTargetRef := cloneSourceRef.DeepCopy()
		cloneTargetRef.Name = targetVMName

		vmClone.Spec.Source = cloneSourceRef
		vmClone.Spec.Target = cloneTargetRef

		return vmClone
	}

	createCloneAndWaitForFinish := func(vmClone *clonev1alpha1.VirtualMachineClone) {
		By(fmt.Sprintf("Creating clone object %s", vmClone.Name))
		vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Create(context.Background(), vmClone, v1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		By(fmt.Sprintf("Waiting for the clone %s to finish", vmClone.Name))
		Eventually(func() clonev1alpha1.VirtualMachineClonePhase {
			vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, v1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			return vmClone.Status.Phase
		}, 3*time.Minute, 3*time.Second).Should(Equal(clonev1alpha1.Succeeded), "clone should finish successfully")
	}

	expectVMRunnable := func(vm *virtv1.VirtualMachine, login loginFunction) *virtv1.VirtualMachine {
		By(fmt.Sprintf("Starting VM %s", vm.Name))
		vm = tests.StartVirtualMachine(vm)
		targetVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		err = login(targetVMI)
		Expect(err).ShouldNot(HaveOccurred())

		vm = tests.StopVirtualMachine(vm)

		return vm
	}

	filterOutIrrelevantKeys := func(in map[string]string) map[string]string {
		out := make(map[string]string)

		for key, val := range in {
			if !strings.Contains(key, "kubevirt.io") && !strings.Contains(key, "kubemacpool.io") {
				out[key] = val
			}
		}

		return out
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
			vmClone            *clonev1alpha1.VirtualMachineClone
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

		expectEqualAnnotations := func(targetVM, sourceVM *virtv1.VirtualMachine, keysToExclude ...string) {
			expectEqualStrMap(targetVM.Annotations, sourceVM.Annotations, fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "annotations"), keysToExclude...)
		}

		expectSpecsToEqualExceptForMacAddress := func(vm1, vm2 *virtv1.VirtualMachine) {
			vm1Spec := vm1.Spec.DeepCopy()
			vm2Spec := vm2.Spec.DeepCopy()

			for _, spec := range []*virtv1.VirtualMachineSpec{vm1Spec, vm2Spec} {
				for i := range spec.Template.Spec.Domain.Devices.Interfaces {
					spec.Template.Spec.Domain.Devices.Interfaces[i].MacAddress = ""
				}
			}

			Expect(vm1Spec).To(Equal(vm2Spec), fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "spec not including mac adresses"))
		}

		createVM := func(options ...libvmi.Option) (vm *virtv1.VirtualMachine) {
			defaultOptions := []libvmi.Option{
				libvmi.WithLabel(key1, value1),
				libvmi.WithLabel(key2, value2),
				libvmi.WithAnnotation(key1, value1),
				libvmi.WithAnnotation(key2, value2),
			}

			options = append(options, defaultOptions...)
			return createVM(options...)
		}

		generateClone := func() *clonev1alpha1.VirtualMachineClone {
			return generateClone(sourceVM, targetVMName)
		}

		Context("simple VM and cloning operations", func() {

			expectVMRunnable := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
				return expectVMRunnable(vm, console.LoginToCirros)
			}

			It("simple default clone", func() {
				sourceVM = createVM()
				vmClone = generateClone()

				createCloneAndWaitForFinish(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				Expect(targetVM.Spec).To(Equal(sourceVM.Spec), fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "spec"))
				expectEqualLabels(targetVM, sourceVM)
				expectEqualAnnotations(targetVM, sourceVM)

				By("Making sure snapshot and restore objects are cleaned up")
				Expect(vmClone.Status.SnapshotName).To(BeNil())
				Expect(vmClone.Status.RestoreName).To(BeNil())
			})

			It("clone with only some of labels/annotations", func() {
				sourceVM = createVM()
				vmClone = generateClone()

				vmClone.Spec.LabelFilters = []string{
					"*",
					"!" + key2,
				}
				vmClone.Spec.AnnotationFilters = []string{
					key1,
				}
				createCloneAndWaitForFinish(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				Expect(targetVM.Spec).To(Equal(sourceVM.Spec), fmt.Sprintf(cloneShouldEqualSourceMsgPattern, "spec"))
				expectEqualLabels(targetVM, sourceVM, key2)
				expectEqualAnnotations(targetVM, sourceVM, key2)
			})

			It("clone with changed MAC address", func() {
				const newMacAddress = "BE-AD-00-00-BE-04"
				sourceVM = createVM(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				)

				srcInterfaces := sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces
				Expect(srcInterfaces).ToNot(BeEmpty())
				srcInterface := srcInterfaces[0]

				vmClone = generateClone()
				vmClone.Spec.NewMacAddresses = map[string]string{
					srcInterface.Name: newMacAddress,
				}

				createCloneAndWaitForFinish(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
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

				expectSpecsToEqualExceptForMacAddress(targetVM, sourceVM)
				expectEqualLabels(targetVM, sourceVM)
				expectEqualAnnotations(targetVM, sourceVM)
			})

			Context("regarding domain Firmware", func() {
				It("clone with changed SMBios serial", func() {
					const sourceSerial = "source-serial"
					const targetSerial = "target-serial"

					sourceVM = createVM(
						func(vmi *virtv1.VirtualMachineInstance) {
							vmi.Spec.Domain.Firmware = &virtv1.Firmware{Serial: sourceSerial}
						},
					)

					vmClone = generateClone()
					vmClone.Spec.NewSMBiosSerial = pointer.String(targetSerial)

					createCloneAndWaitForFinish(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Making sure target is runnable")
					targetVM = expectVMRunnable(targetVM)

					By("Making sure new smBios serial is applied to target VM")
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(sourceVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
					Expect(targetVM.Spec.Template.Spec.Domain.Firmware.Serial).ToNot(Equal(sourceVM.Spec.Template.Spec.Domain.Firmware.Serial))

					expectEqualLabels(targetVM, sourceVM)
					expectEqualAnnotations(targetVM, sourceVM)
				})

				It("[QUARANTINE] should strip firmware UUID", func() {
					const fakeFirmwareUUID = "fake-uuid"

					sourceVM = createVM(
						func(vmi *virtv1.VirtualMachineInstance) {
							vmi.Spec.Domain.Firmware = &virtv1.Firmware{UUID: fakeFirmwareUUID}
						},
					)
					vmClone = generateClone()

					createCloneAndWaitForFinish(vmClone)

					By(fmt.Sprintf("Getting the target VM %s", targetVMName))
					targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
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

		Context("with more complicated VM", func() {

			expectVMRunnable := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
				return expectVMRunnable(vm, console.LoginToAlpine)
			}

			createVMWithStorageClass := func(storageClass string, running bool) *virtv1.VirtualMachine {
				vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					util.NamespaceTestDefault,
					storageClass,
					k8sv1.ReadWriteOnce,
				)
				vm.Spec.Running = pointer.Bool(running)

				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					libstorage.EventuallyDVWith(vm.Namespace, dvt.Name, 180, HaveSucceeded())
				}

				return vm
			}

			getDumbStorageClass := func(preference string) string {
				scs, err := virtClient.StorageV1().StorageClasses().List(context.Background(), v1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				vscs, err := virtClient.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotClasses().List(context.Background(), v1.ListOptions{})
				if errors.IsNotFound(err) {
					return ""
				}
				Expect(err).ToNot(HaveOccurred())

				var candidates []string
				for _, sc := range scs.Items {
					found := false
					for _, vsc := range vscs.Items {
						if sc.Provisioner == vsc.Driver {
							found = true
							break
						}
					}
					if !found {
						candidates = append(candidates, sc.Name)
					}
				}

				if len(candidates) == 0 {
					return ""
				}

				if len(candidates) > 1 {
					for _, c := range candidates {
						if c == preference {
							return c
						}
					}
				}

				return candidates[0]
			}

			It("with a simple clone", func() {
				snapshotStorageClass, err := getSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())

				if snapshotStorageClass == "" {
					Skip("Skipping test, no VolumeSnapshot support")
				}

				sourceVM = createVMWithStorageClass(snapshotStorageClass, false)
				vmClone = generateClone()

				createCloneAndWaitForFinish(vmClone)

				By(fmt.Sprintf("Getting the target VM %s", targetVMName))
				targetVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(targetVMName, &v1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By("Making sure target is runnable")
				targetVM = expectVMRunnable(targetVM)

				expectEqualLabels(targetVM, sourceVM)
				expectEqualAnnotations(targetVM, sourceVM)
			})

			It("should reject clone with non snapshotable volume", func() {
				sc := getDumbStorageClass("local")
				if sc == "" {
					Skip("Skipping test, no storage class without snapshot support")
				}

				// create running in case storage is WFFC (local storage)
				sourceVM = createVMWithStorageClass(sc, true)
				sourceVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(sourceVM.Name, &v1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				sourceVM = tests.StopVirtualMachine(sourceVM)

				vmClone = generateClone()
				vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Create(context.Background(), vmClone, v1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("does not support snapshots"))
			})
		})
	})
})
