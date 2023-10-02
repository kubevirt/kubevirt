package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/libvmifact"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

type vmStatePersistenceOption struct {
	WithTPM           bool
	WithEFI           bool
	BlockStorageClass *bool
	VMDisk            *string
	OrderedOperations []string
}

var (
	vmStateBlockDVDisk       = "dv-block-order-1"
	vmStateFilesystemPVCDisk = "pvc-fs-order"
)

var _ = Describe("[sig-storage]VM state", decorators.SigStorage, func() {
	var virtClient kubecli.KubevirtClient
	var err error
	var dv *cdiv1.DataVolume
	var pvc *k8sv1.PersistentVolumeClaim

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		// We create an empty DV and PVC which will be added to the VMs later.
		// This is to test the auto creation of the persistent state PVC with
		// proper storage class and volume mode.
		dv, pvc, err = prepareVMStateDVAndPVC(virtClient, util.NamespaceTestDefault)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := cleanupVMStateEnv(virtClient, util.NamespaceTestDefault, dv, pvc)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with persistent VM state enabled", func() {
		stopVM := func(vm *v1.VirtualMachine) {
			By("Stopping the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			EventuallyWithOffset(1, func() error {
				_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(Succeed())
		}
		startVM := func(vm *v1.VirtualMachine) {
			By("Starting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			var vmi *v1.VirtualMachineInstance
			EventuallyWithOffset(1, func() error {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring the firmware is done so we don't send any keystroke to it")
			err = console.LinuxExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Logging in as root")
			err = console.LoginToFedora(vmi)
			Expect(err).ToNot(HaveOccurred())
		}

		migrateVMI := func(vmi *v1.VirtualMachineInstance) {
			By("Migrating the VMI")
			checks.SkipIfMigrationIsNotPossible()
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

		}

		addDataToTPM := func(vmi *v1.VirtualMachineInstance) {
			By("Storing a secret into the TPM")
			// https://www.intel.com/content/www/us/en/developer/articles/code-sample/protecting-secret-data-and-keys-using-intel-platform-trust-technology.html
			// Not sealing against a set of PCRs, out of scope here, but should work with a carefully selected set (at least PCR1 was seen changing accross reboots)
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_createprimary -Q --hierarchy=o --key-context=prim.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo MYSECRET | tpm2_create --hash-algorithm=sha256 --public=seal.pub --private=seal.priv --sealing-input=- --parent-context=prim.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_load -Q --parent-context=prim.ctx --public=seal.pub --private=seal.priv --name=seal.name --key-context=seal.ctx\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_evictcontrol --hierarchy=o --object-context=seal.ctx 0x81010002\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
				&expect.BExp{R: "MYSECRET"},
			}, 300)).To(Succeed(), "failed to store secret into the TPM")
		}

		checkTPM := func(vmi *v1.VirtualMachineInstance) {
			By("Ensuring the TPM is still functional and its state carried over")
			ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
				&expect.BExp{R: "MYSECRET"},
			}, 300)).To(Succeed(), "the state of the TPM did not persist")
		}

		addDataToEFI := func(vmi *v1.VirtualMachineInstance) {
			By("Creating an efivar")
			cmd := `printf "\x07\x00\x00\x00\x42" > /sys/firmware/efi/efivars/kvtest-12345678-1234-1234-1234-123456789abc`
			err = console.RunCommand(vmi, cmd, 10*time.Second)
			Expect(err).NotTo(HaveOccurred())
		}

		checkEFI := func(vmi *v1.VirtualMachineInstance) {
			By("Ensuring the efivar is present")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "hexdump /sys/firmware/efi/efivars/kvtest-12345678-1234-1234-1234-123456789abc\n"},
				&expect.BExp{R: "0042"},
			}, 10)).To(Succeed(), "expected efivar is missing")
		}

		DescribeTable("should persist VM state of", decorators.RequiresTwoSchedulableNodes, Serial, func(option vmStatePersistenceOption) {
			blockStorageClass, filesystemStorageClass, skip := getVMStateStorageClasses(&option)
			if skip {
				Skip(fmt.Sprintf("No storage class available. Block: %s, Filesystem: %s", blockStorageClass, filesystemStorageClass))
			}

			if option.BlockStorageClass != nil {
				var storageClass string
				var volumeMode k8sv1.PersistentVolumeMode
				if *option.BlockStorageClass {
					storageClass = blockStorageClass
					volumeMode = k8sv1.PersistentVolumeBlock
				} else {
					storageClass = filesystemStorageClass
					volumeMode = k8sv1.PersistentVolumeFilesystem
				}
				By(fmt.Sprintf("Using the storage class %s", storageClass))
				kv := util.GetCurrentKv(virtClient)
				kv.Spec.Configuration.VMStateStorageClass = storageClass
				kv.Spec.Configuration.VMStateVolumeMode = &volumeMode
				tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
			}

			By("Creating a migratable Fedora VM with UEFI")
			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(cloudinit.CreateDefaultCloudInitNetworkData())),
				libvmi.WithUefi(false),
				libvmi.WithResourceMemory("1Gi"),
			)
			vmi.Namespace = util.NamespaceTestDefault
			err := addDiskFromVMStatePersistenceOption(vmi, &option, dv, pvc)
			Expect(err).ToNot(HaveOccurred())

			if option.WithTPM {
				By("with persistent TPM enabled")
				vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
					Persistent: pointer.BoolPtr(true),
				}
			}
			if option.WithEFI {
				By("with persistent EFI enabled")
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{SecureBoot: pointer.BoolPtr(false), Persistent: pointer.BoolPtr(true)},
					},
				}
			}
			vm := libvmi.NewVirtualMachine(vmi)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			startVM(vm)

			if option.WithTPM {
				addDataToTPM(vmi)
			}
			if option.WithEFI {
				addDataToEFI(vmi)
			}

			for _, op := range option.OrderedOperations {
				switch op {
				case "migrate":
					migrateVMI(vmi)
				case "restart":
					stopVM(vm)
					startVM(vm)
				}
				if option.WithTPM {
					checkTPM(vmi)
				}
				if option.WithEFI {
					checkEFI(vmi)
				}
			}

			By("Stopping and removing the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, k8smetav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[test_id:10818]TPM across migration and restart with filesystem backend determined from disk config", vmStatePersistenceOption{WithTPM: true, WithEFI: false, BlockStorageClass: nil, VMDisk: &vmStateFilesystemPVCDisk, OrderedOperations: []string{"migrate", "restart"}}),
			Entry("[test_id:10819]TPM across restart and migration with filesystem backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: true, WithEFI: false, BlockStorageClass: pointer.Bool(false), OrderedOperations: []string{"restart", "migrate"}}),
			Entry("[test_id:10820]EFI across migration and restart with filesystem backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: false, WithEFI: true, BlockStorageClass: pointer.Bool(false), OrderedOperations: []string{"migrate", "restart"}}),
			Entry("[test_id:10821]TPM+EFI across migration and restart with filesystem backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: true, WithEFI: true, BlockStorageClass: pointer.Bool(false), OrderedOperations: []string{"migrate", "restart"}}),
			Entry("TPM across migration and restart with block backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: true, WithEFI: false, BlockStorageClass: pointer.Bool(true), OrderedOperations: []string{"migrate", "restart"}}),
			Entry("TPM across restart and migration with block backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: true, WithEFI: false, BlockStorageClass: pointer.Bool(true), VMDisk: &vmStateFilesystemPVCDisk, OrderedOperations: []string{"restart", "migrate"}}),
			Entry("EFI across migration and restart with block backend determined from kubevirt config", vmStatePersistenceOption{WithTPM: true, WithEFI: true, BlockStorageClass: pointer.Bool(true), OrderedOperations: []string{"migrate", "restart"}}),
			Entry("TPM+EFI across migration and restart with block backend determined from disk config", vmStatePersistenceOption{WithTPM: true, WithEFI: true, BlockStorageClass: nil, VMDisk: &vmStateBlockDVDisk, OrderedOperations: []string{"migrate", "restart"}}),
		)
		DescribeTable("should remove persistent storage PVC if VMI is not owned by a VM", Serial, func(storageVolumeMode k8sv1.PersistentVolumeMode) {
			By("Setting the backend storage class to the default for RWX FS")
			var storageClass string
			var exists bool
			if storageVolumeMode == k8sv1.PersistentVolumeBlock {
				storageClass, exists = libstorage.GetRWXBlockStorageClass()
			} else {
				storageClass, exists = libstorage.GetRWXFileSystemStorageClass()
			}
			if !exists {
				Skip(fmt.Sprintf("No storage class available for %s mode", storageVolumeMode))
			}
			By(fmt.Sprintf("Using the storage class %s", storageClass))
			kv := util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.VMStateStorageClass = storageClass
			kv.Spec.Configuration.VMStateVolumeMode = &storageVolumeMode
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			By("Creating a VMI with persistent TPM enabled")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
				Persistent: pointer.BoolPtr(true),
			}
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to start")
			Eventually(func() error {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Removing the VMI")
			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, k8smetav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring the PVC gets deleted")
			Eventually(func() error {
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
				if !errors.IsNotFound(err) {
					return fmt.Errorf("VM %s not removed: %v", vmi.Name, err)
				}
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), backendstorage.PVCForVMI(vmi), k8smetav1.GetOptions{})
				if !errors.IsNotFound(err) {
					return fmt.Errorf("PVC %s not removed: %v", backendstorage.PVCForVMI(vmi), err)
				}
				return nil
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		},
			Entry("with block storage", k8sv1.PersistentVolumeBlock),
			Entry("with filesystem storage", k8sv1.PersistentVolumeFilesystem),
		)
	})
})

func getVMStateStorageClasses(o *vmStatePersistenceOption) (blockStorageClass string, filesystemStorageClass string, skip bool) {
	var exist bool
	if (o.BlockStorageClass != nil && *o.BlockStorageClass) || (o.VMDisk != nil && *o.VMDisk == vmStateBlockDVDisk) {
		blockStorageClass, exist = libstorage.GetRWXBlockStorageClass()
		skip = skip || !exist
	} else {
		filesystemStorageClass, exist = libstorage.GetRWXFileSystemStorageClass()
		skip = skip || !exist
	}
	return
}

func addDataVolumeDisk(vmi *v1.VirtualMachineInstance, diskName, dataVolumeName string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusVirtio,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})

	return vmi
}

func addDiskFromVMStatePersistenceOption(vmi *v1.VirtualMachineInstance, o *vmStatePersistenceOption, dv *cdiv1.DataVolume, pvc *k8sv1.PersistentVolumeClaim) error {
	if o.VMDisk == nil {
		return nil
	}
	if *o.VMDisk == vmStateBlockDVDisk {
		if dv == nil {
			return fmt.Errorf("VMI uses block storage DV but this DV does not exists")
		}
		addDataVolumeDisk(vmi, "empty-disk", dv.Name)
		addBootOrderToDisk(vmi, "disk0", pointer.Uint(1))
		addBootOrderToDisk(vmi, "empty-disk", pointer.Uint(2))
	} else if *o.VMDisk == vmStateFilesystemPVCDisk {
		if pvc == nil {
			return fmt.Errorf("VMI uses filesystem storage PVC but this PVC does not exists")
		}
		libvmi.WithPersistentVolumeClaim("empty-disk", pvc.Name)(vmi)
	}
	return nil
}

// prepareVMStateDVAndPVC creates an empty block mode DV and a filesystem mode
// PVC.
func prepareVMStateDVAndPVC(virtClient kubecli.KubevirtClient, namespace string) (*cdiv1.DataVolume, *k8sv1.PersistentVolumeClaim, error) {
	var dv *cdiv1.DataVolume
	var pvc *k8sv1.PersistentVolumeClaim
	var err error

	blockStorageClass, exists := libstorage.GetRWXBlockStorageClass()
	if exists {
		dv = libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithPVC(
				libdv.PVCWithAccessMode(k8sv1.ReadWriteMany),
				libdv.PVCWithBlockVolumeMode(),
				libdv.PVCWithStorageClass(blockStorageClass),
				libdv.PVCWithVolumeSize("16Mi"),
			))
		dv.Namespace = namespace
		dv.Annotations = map[string]string{"cdi.kubevirt.io/storage.deleteAfterCompletion": "false"}
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, k8smetav1.CreateOptions{})
		if err != nil {
			return nil, nil, err
		}
	}

	filesystemStorageClass, exists := libstorage.GetRWXFileSystemStorageClass()
	if exists {
		m := k8sv1.PersistentVolumeFilesystem
		quantity, err := resource.ParseQuantity("12Mi")
		if err != nil {
			return nil, nil, err
		}
		pvc = &k8sv1.PersistentVolumeClaim{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      "test-pvc-" + rand.String(5),
				Namespace: namespace,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				AccessModes:      []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
				StorageClassName: &filesystemStorageClass,
				VolumeMode:       &m,
				Resources: k8sv1.VolumeResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": quantity,
					},
				},
			},
		}
		pvc, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, k8smetav1.CreateOptions{})
		if err != nil {
			return nil, nil, err
		}
	}
	return dv, pvc, nil
}

func cleanupVMStateEnv(virtClient kubecli.KubevirtClient, namespace string, dv *cdiv1.DataVolume, pvc *k8sv1.PersistentVolumeClaim) error {
	if dv != nil {
		if err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Delete(context.Background(), dv.Name, k8smetav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	if pvc != nil {
		if err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Delete(context.Background(), pvc.Name, k8smetav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	kv := util.GetCurrentKv(virtClient)
	kv.Spec.Configuration.VMStateStorageClass = ""
	kv.Spec.Configuration.VMStateVolumeMode = nil
	tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
	return nil
}
