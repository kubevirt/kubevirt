package virtctl

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/tests/clientcmd"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/vm"
)

const (
	cloudInitUserData = `#cloud-config
user: user
password: password
chpasswd: { expire: False }`

	create = "create"
	size   = "128Mi"
)

var _ = Describe("[sig-compute][virtctl]create vm", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("should create a valid VM manifest", func() {
		It("[test_id:9840]VM with random name and default settings", func() {
			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM)()

			Expect(err).ToNot(HaveOccurred())
			vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), unmarshalVM(out))
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).ToNot(BeEmpty())
			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(int64(180)))
			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
		})

		It("Example with volume-import flag and PVC type", func() {
			const runStrategy = v1.RunStrategyAlways
			volumeName := "imported-volume"
			instancetype := createInstancetype(virtClient)
			preference := createPreference(virtClient)
			pvc := createAnnotatedSourcePVC(instancetype.Name, preference.Name)

			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(VolumeImportFlag, fmt.Sprintf("type:pvc,size:%s,src:%s/%s,name:%s", size, pvc.Namespace, pvc.Name, volumeName)),
			)()
			Expect(err).ToNot(HaveOccurred())

			vm := createVMWithRWOVolume(out, virtClient)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(Equal(apiinstancetype.SingularResourceName))
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(Equal(apiinstancetype.SingularPreferenceResourceName))
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volumeName))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(volumeName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvc.Name))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
		})

		It("Example with volume-import flag and Registry type", func() {
			const volName = "registry-source"
			const runStrategy = v1.RunStrategyAlways
			cdSource := cd.ContainerDiskFor(cd.ContainerDiskAlpine)

			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(VolumeImportFlag, fmt.Sprintf("type:registry,size:%s,url:docker://%s,name:%s", size, cdSource, volName)),
			)()
			Expect(err).ToNot(HaveOccurred())

			vm := createVMWithRWOVolume(out, virtClient)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volName))
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Registry).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Registry.URL).To(HaveValue(Equal("docker://" + cdSource)))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
		})

		It("Example with volume-import flag and Blank type", func() {
			const runStrategy = v1.RunStrategyAlways

			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s", size)),
			)()
			Expect(err).ToNot(HaveOccurred())

			vm := createVMWithRWOVolume(out, virtClient)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Blank).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
		})

		It("[test_id:9841]Complex example", func() {
			const runStrategy = v1.RunStrategyManual
			const terminationGracePeriod int64 = 123
			const cdSource = "my.registry/my-image:my-tag"
			const blankSize = "10Gi"
			const pvcBootOrder = 1
			vmName := "vm-" + rand.String(5)
			instancetype := createInstancetype(virtClient)
			preference := createPreference(virtClient)
			dataSource := createAnnotatedDataSource(virtClient, "something", "something")
			pvc := libstorage.CreateFSPVC("vm-pvc-"+rand.String(5), util.NamespaceTestDefault, size, nil)
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(NameFlag, vmName),
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
				setFlag(InstancetypeFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularResourceName, instancetype.Name)),
				setFlag(PreferenceFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularPreferenceResourceName, preference.Name)),
				setFlag(ContainerdiskVolumeFlag, fmt.Sprintf("src:%s", cdSource)),
				setFlag(DataSourceVolumeFlag, fmt.Sprintf("src:%s/%s", dataSource.Namespace, dataSource.Name)),
				setFlag(ClonePvcVolumeFlag, fmt.Sprintf("src:%s/%s", pvc.Namespace, pvc.Name)),
				setFlag(PvcVolumeFlag, fmt.Sprintf("src:%s,bootorder:%d", pvc.Name, pvcBootOrder)),
				setFlag(BlankVolumeFlag, fmt.Sprintf("size:%s", blankSize)),
				setFlag(CloudInitUserDataFlag, userDataB64),
			)()

			Expect(err).ToNot(HaveOccurred())
			vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), unmarshalVM(out))
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))

			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(Equal(apiinstancetype.SingularResourceName))
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(Equal(apiinstancetype.SingularPreferenceResourceName))
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
			Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(3))

			dvtDsName := fmt.Sprintf("%s-ds-%s", vmName, dataSource.Name)
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).ToNot(BeNil())
			Expect(*vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(Equal(dataSource.Namespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dataSource.Name))

			dvtPvcName := fmt.Sprintf("%s-pvc-%s", vmName, pvc.Name)
			Expect(vm.Spec.DataVolumeTemplates[1].Name).To(Equal(dvtPvcName))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Name).To(Equal(pvc.Name))

			dvtBlankName := fmt.Sprintf("%s-blank-0", vmName)
			Expect(vm.Spec.DataVolumeTemplates[2].Name).To(Equal(dvtBlankName))
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source.Blank).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(6))

			volCdName := fmt.Sprintf("%s-containerdisk-0", vm.Name)
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volCdName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(dvtDsName))

			Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(dvtPvcName))
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume.Name).To(Equal(dvtPvcName))

			Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal(pvc.Name))
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvc.Name))

			Expect(vm.Spec.Template.Spec.Volumes[4].Name).To(Equal(dvtBlankName))
			Expect(vm.Spec.Template.Spec.Volumes[4].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[4].VolumeSource.DataVolume.Name).To(Equal(dvtBlankName))

			Expect(vm.Spec.Template.Spec.Volumes[5].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(pvc.Name))
			Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(pvcBootOrder)))
		})

		It("[test_id:9842]Complex example with inferred instancetype and preference", func() {
			const runStrategy = v1.RunStrategyManual
			const terminationGracePeriod int64 = 123
			const blankSize = "10Gi"
			const pvcBootOrder = 1
			vmName := "vm-" + rand.String(5)
			instancetype := createInstancetype(virtClient)
			preference := createPreference(virtClient)
			dataSource := createAnnotatedDataSource(virtClient, "something", preference.Name)
			dvtDsName := fmt.Sprintf("%s-ds-%s", vmName, dataSource.Name)
			pvc := createAnnotatedSourcePVC(instancetype.Name, "something")
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))
			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(NameFlag, vmName),
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
				setFlag(InferInstancetypeFlag, "true"),
				setFlag(InferPreferenceFromFlag, dvtDsName),
				setFlag(DataSourceVolumeFlag, fmt.Sprintf("src:%s/%s", dataSource.Namespace, dataSource.Name)),
				setFlag(ClonePvcVolumeFlag, fmt.Sprintf("src:%s/%s,bootorder:%d", pvc.Namespace, pvc.Name, pvcBootOrder)),
				setFlag(BlankVolumeFlag, fmt.Sprintf("size:%s", blankSize)),
				setFlag(CloudInitUserDataFlag, userDataB64),
			)()

			Expect(err).ToNot(HaveOccurred())
			vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), unmarshalVM(out))
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))

			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(Equal(apiinstancetype.SingularResourceName))
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(Equal(apiinstancetype.SingularPreferenceResourceName))
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
			Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(3))

			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).ToNot(BeNil())
			Expect(*vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(Equal(dataSource.Namespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dataSource.Name))

			dvtPvcName := fmt.Sprintf("%s-pvc-%s", vmName, pvc.Name)
			Expect(vm.Spec.DataVolumeTemplates[1].Name).To(Equal(dvtPvcName))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Name).To(Equal(pvc.Name))

			dvtBlankName := fmt.Sprintf("%s-blank-0", vmName)
			Expect(vm.Spec.DataVolumeTemplates[2].Name).To(Equal(dvtBlankName))
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source.Blank).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[2].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(4))

			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(dvtDsName))

			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(dvtPvcName))
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(dvtPvcName))

			Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(dvtBlankName))
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume.Name).To(Equal(dvtBlankName))

			Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(dvtPvcName))
			Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(pvcBootOrder)))
		})

		It("Failure of implicit inference does not fail the VM creation", func() {
			By("Creating a PVC without annotation labels")
			pvc := libstorage.CreateFSPVC("vm-pvc-"+rand.String(5), testsuite.GetTestNamespace(nil), size, nil)
			volumeName := "imported-volume"

			By("Creating a VM with implicit inference (inference enabled by default)")
			out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
				setFlag(VolumeImportFlag, fmt.Sprintf("type:pvc,size:%s,src:%s/%s,name:%s", size, pvc.Namespace, pvc.Name, volumeName)),
			)()
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			By("Asserting that implicit inference is enabled")
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(volumeName))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(volumeName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))

			By("Creating the VM")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Asserting that matchers were cleared and memory was kept")
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		})
	})

	It("Complex example with memory", func() {
		const runStrategy = v1.RunStrategyManual
		const terminationGracePeriod int64 = 123
		const memory = "4Gi"
		const cdSource = "my.registry/my-image:my-tag"
		const blankSize = "10Gi"
		vmName := "vm-" + rand.String(5)
		preference := createPreference(virtClient)
		userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

		out, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, VM,
			setFlag(NameFlag, vmName),
			setFlag(RunStrategyFlag, string(runStrategy)),
			setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
			setFlag(MemoryFlag, memory),
			setFlag(PreferenceFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularPreferenceResourceName, preference.Name)),
			setFlag(ContainerdiskVolumeFlag, fmt.Sprintf("src:%s", cdSource)),
			setFlag(BlankVolumeFlag, fmt.Sprintf("size:%s", blankSize)),
			setFlag(CloudInitUserDataFlag, userDataB64),
		)()

		Expect(err).ToNot(HaveOccurred())
		vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), unmarshalVM(out))
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Name).To(Equal(vmName))

		Expect(vm.Spec.Running).To(BeNil())
		Expect(vm.Spec.RunStrategy).ToNot(BeNil())
		Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
		Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(memory)))

		Expect(vm.Spec.Preference).ToNot(BeNil())
		Expect(vm.Spec.Preference.Kind).To(Equal(apiinstancetype.SingularPreferenceResourceName))
		Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
		Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
		Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))

		dvtBlankName := fmt.Sprintf("%s-blank-0", vmName)
		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtBlankName))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Blank).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(3))

		volCdName := fmt.Sprintf("%s-containerdisk-0", vm.Name)
		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volCdName))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(dvtBlankName))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(dvtBlankName))

		Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal("cloudinitdisk"))
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

		decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(decoded)).To(Equal(cloudInitUserData))
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func unmarshalVM(bytes []byte) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{}
	Expect(yaml.Unmarshal(bytes, vm)).To(Succeed())
	Expect(vm.Kind).To(Equal("VirtualMachine"))
	Expect(vm.APIVersion).To(Equal("kubevirt.io/v1"))
	return vm
}

func createInstancetype(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachineInstancetype {
	instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-instancetype-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: uint32(1),
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest: resource.MustParse(size),
			},
		},
	}
	instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Create(context.Background(), instancetype, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return instancetype
}

func createPreference(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachinePreference {
	preferredCPUTopology := instancetypev1beta1.PreferCores
	preference := &instancetypev1beta1.VirtualMachinePreference{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-preference-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: &preferredCPUTopology,
			},
		},
	}
	preference, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).Create(context.Background(), preference, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return preference
}

func createAnnotatedDataSource(virtClient kubecli.KubevirtClient, instancetypeName, preferenceName string) *v1beta1.DataSource {
	dataSource := &v1beta1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-datasource-",
			Labels: map[string]string{
				apiinstancetype.DefaultInstancetypeLabel:     instancetypeName,
				apiinstancetype.DefaultInstancetypeKindLabel: apiinstancetype.SingularResourceName,
				apiinstancetype.DefaultPreferenceLabel:       preferenceName,
				apiinstancetype.DefaultPreferenceKindLabel:   apiinstancetype.SingularPreferenceResourceName,
			},
		},
		Spec: v1beta1.DataSourceSpec{
			Source: v1beta1.DataSourceSource{},
		},
	}
	dataSource, err := virtClient.CdiClient().CdiV1beta1().DataSources(util.NamespaceTestDefault).Create(context.Background(), dataSource, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return dataSource
}

func createAnnotatedSourcePVC(instancetypeName, preferenceName string) *k8sv1.PersistentVolumeClaim {
	pvcLabels := map[string]string{
		apiinstancetype.DefaultInstancetypeLabel:     instancetypeName,
		apiinstancetype.DefaultInstancetypeKindLabel: apiinstancetype.SingularResourceName,
		apiinstancetype.DefaultPreferenceLabel:       preferenceName,
		apiinstancetype.DefaultPreferenceKindLabel:   apiinstancetype.SingularPreferenceResourceName,
	}
	pvc := libstorage.CreateFSPVC("vm-pvc-"+rand.String(5), util.NamespaceTestDefault, size, pvcLabels)
	return pvc
}

func createVMWithRWOVolume(vmSpec []byte, virtClient kubecli.KubevirtClient) *v1.VirtualMachine {
	unmarshaledVm := unmarshalVM(vmSpec)
	// AccessMode needs to be set explicitly, because kubevirtci storage class
	// does not support automatically deriving
	unmarshaledVm.Spec.DataVolumeTemplates[0].Spec.Storage.AccessModes = []k8sv1.PersistentVolumeAccessMode{
		k8sv1.ReadWriteOnce,
	}

	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), unmarshaledVm)
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(HaveConditionTrue(v1.VirtualMachineReady))

	return vm
}
