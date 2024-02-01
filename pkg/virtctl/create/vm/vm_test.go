package vm_test

import (
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/util"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/vm"
)

const (
	cloudInitUserData = `#cloud-config
user: user
password: password
chpasswd: { expire: False }`

	cloudInitNetworkData = `network:
  version: 1
  config:
  - type: physical
  name: eth0
  subnets:
    - type: dhcp`

	create     = "create"
	size       = "256Mi"
	certConfig = "my-cert"
	pullMethod = "pod"
	url        = "http://url.com"
	secretRef  = "secret-ref"
)

var _ = Describe("create vm", func() {
	ignoreInferFromVolumeFailure := v1.IgnoreInferFromVolumeFailure

	Context("Manifest is created successfully", func() {
		It("VM with random name", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			_ = unmarshalVM(out)
		})

		It("VM with specified namespace", func() {
			const namespace = "my-namespace"
			out, err := runCmd(setFlag("namespace", namespace))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Namespace).To(Equal(namespace))
		})

		It("VM with specified name", func() {
			const name = "my-vm"
			out, err := runCmd(setFlag(NameFlag, name))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Name).To(Equal(name))
		})

		It("RunStrategy is set to Always by default", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))
		})

		It("VM with specified run strategy", func() {
			const runStrategy = v1.RunStrategyManual
			out, err := runCmd(setFlag(RunStrategyFlag, string(runStrategy)))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))
		})

		It("Termination grace period defaults to 180", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)
			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(int64(180)))
		})

		It("VM with specified termination grace period", func() {
			const terminationGracePeriod int64 = 123
			out, err := runCmd(setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))
		})

		It("Memory is set to 512Mi by default", func() {
			const defaultMemory = "512Mi"
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(defaultMemory)))
		})

		It("VM with specified memory", func() {
			const memory = "1Gi"
			out, err := runCmd(setFlag(MemoryFlag, string(memory)))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(memory)))
		})

		DescribeTable("VM with specified instancetype", func(flag, name, kind string) {
			out, err := runCmd(setFlag(InstancetypeFlag, flag))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Name).To(Equal(name))
			Expect(vm.Spec.Instancetype.Kind).To(Equal(kind))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
		},
			Entry("Implicit cluster-wide", "my-instancetype", "my-instancetype", ""),
			Entry("Explicit cluster-wide", "virtualmachineclusterinstancetype/my-clusterinstancetype", "my-clusterinstancetype", instancetypeapi.ClusterSingularResourceName),
			Entry("Explicit namespaced", "virtualmachineinstancetype/my-instancetype", "my-instancetype", instancetypeapi.SingularResourceName),
		)

		DescribeTable("VM with inferred instancetype", func(args []string, inferFromVolume string, inferFromVolumePolicy *v1.InferFromVolumeFailurePolicy) {
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Name).To(BeEmpty())
			Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(inferFromVolume))
			if inferFromVolumePolicy == nil {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(*inferFromVolumePolicy))
			}
			if inferFromVolumePolicy != nil && *inferFromVolumePolicy == v1.IgnoreInferFromVolumeFailure {
				Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
				Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
			}
		},
			Entry("PvcVolumeFlag and implicit inference (enabled by default)", []string{setFlag(PvcVolumeFlag, "src:my-pvc")}, "my-pvc", &ignoreInferFromVolumeFailure),
			Entry("PvcVolumeFlag and explicit inference", []string{setFlag(PvcVolumeFlag, "src:my-pvc"), setFlag(InferInstancetypeFlag, "true")}, "my-pvc", nil),
			Entry("VolumeImportFlag and implicit inference (enabled by default)", []string{setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-namespace/my-volume,name:my-pvc")}, "my-pvc", &ignoreInferFromVolumeFailure),
			Entry("VolumeImportFlag and explicit inference", []string{setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-ns/my-volume,name:my-pvc"), setFlag(InferInstancetypeFlag, "true")}, "my-pvc", nil),
		)

		DescribeTable("VM with boot order and inferred instancetype", func(explicit bool) {
			args := []string{
				setFlag(DataSourceVolumeFlag, "src:my-ds-2,bootorder:2"),
				// This DS with bootorder 1 should be used to infer the instancetype, although it is defined second
				setFlag(DataSourceVolumeFlag, "src:my-ds-1,bootorder:1"),
			}
			if explicit {
				args = append(args, setFlag(InferInstancetypeFlag, "true"))
			}

			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Name).To(BeEmpty())
			Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(fmt.Sprintf("%s-ds-%s", vm.Name, "my-ds-1")))
			if explicit {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			}
			if explicit {
				Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
				Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))
			}
		},
			Entry("implicit (inference enabled by default)", false),
			Entry("explicit", true),
		)

		It("VM with inferred instancetype from specified volume", func() {
			out, err := runCmd(
				setFlag(DataSourceVolumeFlag, "src:my-ds-1,name:my-ds-1"),
				setFlag(DataSourceVolumeFlag, "src:my-ds-2,name:my-ds-2"),
				setFlag(InferInstancetypeFromFlag, "my-ds-2"))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Name).To(BeEmpty())
			Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal("my-ds-2"))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
		})

		It("VM with volume and without inferred instancetype", func() {
			out, err := runCmd(
				setFlag(DataSourceVolumeFlag, "src:my-ds"),
				setFlag(InferInstancetypeFlag, "false"))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Instancetype).To(BeNil())
		})

		It("VM with specified memory and volume and without implicitly inferred instancetype", func() {
			const memory = "1Gi"
			out, err := runCmd(
				setFlag(MemoryFlag, memory),
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:my-ds"))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(memory)))
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal("my-ds"))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
		})

		DescribeTable("VM with specified preference", func(flag, name, kind string) {
			out, err := runCmd(setFlag(PreferenceFlag, flag))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(Equal(name))
			Expect(vm.Spec.Preference.Kind).To(Equal(kind))
			Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
		},
			Entry("Implicit cluster-wide", "my-preference", "my-preference", ""),
			Entry("Explicit cluster-wide", "virtualmachineclusterpreference/my-clusterpreference", "my-clusterpreference", instancetypeapi.ClusterSingularPreferenceResourceName),
			Entry("Explicit namespaced", "virtualmachinepreference/my-preference", "my-preference", instancetypeapi.SingularPreferenceResourceName),
		)

		DescribeTable("VM with inferred preference", func(args []string, inferFromVolume string, inferFromVolumePolicy *v1.InferFromVolumeFailurePolicy) {
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(inferFromVolume))
			if inferFromVolumePolicy == nil {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(*inferFromVolumePolicy))
			}
		},
			Entry("PvcVolumeFlag and implicit inference (enabled by default)", []string{setFlag(PvcVolumeFlag, "src:my-pvc")}, "my-pvc", &ignoreInferFromVolumeFailure),
			Entry("PvcVolumeFlag and explicit inference", []string{setFlag(PvcVolumeFlag, "src:my-pvc"), setFlag(InferPreferenceFlag, "true")}, "my-pvc", nil),
			Entry("VolumeImportFlag and implicit inference (enabled by default)", []string{setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-namespace/my-pvc,name:volume-import")}, "volume-import", &ignoreInferFromVolumeFailure),
			Entry("VolumeImportFlag and explicit inference", []string{setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-namespace/my-pvc,name:volume-import"), setFlag(InferPreferenceFlag, "true")}, "volume-import", nil),
		)

		DescribeTable("VM with boot order and inferred preference", func(explicit bool) {
			args := []string{
				setFlag(DataSourceVolumeFlag, "src:my-ds-2,bootorder:2"),
				// This DS with bootorder 1 should be used to infer the preference, although it is defined second
				setFlag(DataSourceVolumeFlag, "src:my-ds-1,bootorder:1"),
			}
			if explicit {
				args = append(args, setFlag(InferPreferenceFlag, "true"))
			}

			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(fmt.Sprintf("%s-ds-%s", vm.Name, "my-ds-1")))
			if explicit {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			}
		},
			Entry("implicit (inference enabled by default)", false),
			Entry("explicit", true),
		)

		It("VM with inferred preference from specified volume", func() {
			out, err := runCmd(
				setFlag(DataSourceVolumeFlag, "src:my-ds-1,name:my-ds-1"),
				setFlag(DataSourceVolumeFlag, "src:my-ds-2,name:my-ds-2"),
				setFlag(InferPreferenceFromFlag, "my-ds-2"))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal("my-ds-2"))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
		})

		It("VM with volume and without inferred preference", func() {
			out, err := runCmd(
				setFlag(DataSourceVolumeFlag, "src:my-ds"),
				setFlag(InferPreferenceFlag, "false"))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Preference).To(BeNil())
		})

		DescribeTable("VM with specified containerdisk", func(containerdisk, volName string, bootOrder int, params string) {
			out, err := runCmd(setFlag(ContainerdiskVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			if volName == "" {
				volName = fmt.Sprintf("%s-containerdisk-0", vm.Name)
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(containerdisk))
			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(volName))
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(bootOrder)))
			}

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with src", "my.registry/my-image:my-tag", "", 0, "src:my.registry/my-image:my-tag"),
			Entry("with src and name", "my.registry/my-image:my-tag", "my-cd", 0, "src:my.registry/my-image:my-tag,name:my-cd"),
			Entry("with src and bootorder", "my.registry/my-image:my-tag", "", 1, "src:my.registry/my-image:my-tag,bootorder:1"),
			Entry("with src, name and bootorder", "my.registry/my-image:my-tag", "my-cd", 2, "src:my.registry/my-image:my-tag,name:my-cd,bootorder:2"),
		)

		DescribeTable("VM with specified datasource", func(dsNamespace, dsName, dvtName, dvtSize string, bootOrder int, params string) {
			out, err := runCmd(setFlag(DataSourceVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			if dvtName == "" {
				dvtName = fmt.Sprintf("%s-ds-%s", vm.Name, dsName)
			}
			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dsName))
			if dsNamespace != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).ToNot(BeNil())
				Expect(*vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(Equal(dsNamespace))
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(BeNil())
			}
			if dvtSize != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(dvtName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(dvtName))
			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(dvtName))
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(bootOrder)))
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(dvtName))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(dvtName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
		},
			Entry("without namespace", "", "my-dv", "", "", 0, "src:my-dv"),
			Entry("with namespace", "my-ns", "my-dv", "", "", 0, "src:my-ns/my-dv"),
			Entry("without namespace and with name", "", "my-dv", "my-dvt", "", 0, "src:my-dv,name:my-dvt"),
			Entry("with namespace and name", "my-ns", "my-dv", "my-dvt", "", 0, "src:my-ns/my-dv,name:my-dvt"),
			Entry("without namespace and with size", "", "my-dv", "", "10Gi", 0, "src:my-dv,size:10Gi"),
			Entry("with namespace and size", "my-ns", "my-dv", "", "10Gi", 0, "src:my-ns/my-dv,size:10Gi"),
			Entry("without namespace and with bootorder", "", "my-dv", "", "", 1, "src:my-dv,bootorder:1"),
			Entry("with namespace and bootorder", "my-ns", "my-dv", "", "", 2, "src:my-ns/my-dv,bootorder:2"),
			Entry("without namespace and with name and size", "", "my-dv", "my-dvt", "10Gi", 0, "src:my-dv,name:my-dvt,size:10Gi"),
			Entry("with namespace, name and size", "my-ns", "my-dv", "my-dvt", "10Gi", 0, "src:my-ns/my-dv,name:my-dvt,size:10Gi"),
			Entry("without namespace and with name and bootorder", "", "my-dv", "my-dvt", "", 3, "src:my-dv,name:my-dvt,bootorder:3"),
			Entry("with namespace, name and bootorder", "my-ns", "my-dv", "my-dvt", "", 4, "src:my-ns/my-dv,name:my-dvt,bootorder:4"),
			Entry("without namespace and with size and bootorder", "", "my-dv", "", "10Gi", 5, "src:my-dv,size:10Gi,bootorder:5"),
			Entry("with namespace, size and bootorder", "my-ns", "my-dv", "", "10Gi", 6, "src:my-ns/my-dv,size:10Gi,bootorder:6"),
			Entry("without namespace and with name, size and bootorder", "", "my-dv", "my-dvt", "10Gi", 7, "src:my-dv,name:my-dvt,size:10Gi,bootorder:7"),
			Entry("with namespace, name, size and bootorder", "my-ns", "my-dv", "my-dvt", "10Gi", 8, "src:my-ns/my-dv,name:my-dvt,size:10Gi,bootorder:8"),
		)

		DescribeTable("VM with specified volume source", func(params string, source *cdiv1.DataVolumeSource, inferVolume string) {
			out, err := runCmd(setFlag(VolumeImportFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(Equal(source))

			if source.PVC != nil {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).ToNot(BeNil())
				Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(inferVolume))
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
				Expect(vm.Spec.Preference).ToNot(BeNil())
				Expect(vm.Spec.Preference.InferFromVolume).To(Equal(inferVolume))
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			} else {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			}
		},
			Entry("with blank source", fmt.Sprintf("type:blank,size:%s", size), &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, ""),
			Entry("with http source", fmt.Sprintf("type:http,size:%s,url:%s", size, url), &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: url}}, ""),
			Entry("with imageio source", fmt.Sprintf("type:imageio,size:%s,url:%s,diskid:1,secretref:%s", size, url, secretRef), &cdiv1.DataVolumeSource{Imageio: &cdiv1.DataVolumeSourceImageIO{DiskID: "1", SecretRef: secretRef, URL: url}}, ""),
			Entry("with PVC source", fmt.Sprintf("type:pvc,size:%s,src:%s/pvc,name:imported-volume", size, util.NamespaceTestDefault), &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: util.NamespaceTestDefault}}, "imported-volume"),
			Entry("with registry source", fmt.Sprintf("type:registry,size:%s,certconfigmap:%s,pullmethod:%s,url:%s,secretref:%s", size, certConfig, pullMethod, url, secretRef), &cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{CertConfigMap: pointer.String(certConfig), PullMethod: (*cdiv1.RegistryPullMethod)(pointer.String(pullMethod)), URL: pointer.String(url), SecretRef: pointer.String(secretRef)}}, ""),
			Entry("with S3 source", fmt.Sprintf("type:s3,size:%s,url:%s,certconfigmap:%s,secretref:%s", size, url, certConfig, secretRef), &cdiv1.DataVolumeSource{S3: &cdiv1.DataVolumeSourceS3{CertConfigMap: certConfig, SecretRef: secretRef, URL: url}}, ""),
			Entry("with VDDK source", fmt.Sprintf("type:vddk,size:%s,backingfile:backing-file,initimageurl:%s,uuid:123e-11", size, url), &cdiv1.DataVolumeSource{VDDK: &cdiv1.DataVolumeSourceVDDK{BackingFile: "backing-file", InitImageURL: url, UUID: "123e-11"}}, ""),
			Entry("with Snapshot source", fmt.Sprintf("type:snapshot,size:%s,src:%s/snapshot,name:imported-volume", size, util.NamespaceTestDefault), &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: util.NamespaceTestDefault}}, "imported-volume"),
			Entry("with blank source and name", fmt.Sprintf("type:blank,size:%s,name:blank-name", size), &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, ""),
		)

		DescribeTable("VM with multiple volume-import sources and name", func(source1 *cdiv1.DataVolumeSource, source2 *cdiv1.DataVolumeSource, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(Equal(source1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("volume-source1"))

			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).To(Equal(source2))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal("volume-source2"))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with blank source", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s,name:volume-source1", size)), setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s,name:volume-source2", size))),
			Entry("with blank source and http source", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: url}}, setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s,name:volume-source1", size)), setFlag(VolumeImportFlag, fmt.Sprintf("type:http,size:%s,url:%s,name:volume-source2", size, url))),
		)

		DescribeTable("VM with specified clone pvc", func(pvcNamespace, pvcName, dvtName, dvtSize string, bootOrder int, params string) {
			out, err := runCmd(setFlag(ClonePvcVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			if dvtName == "" {
				dvtName = fmt.Sprintf("%s-pvc-%s", vm.Name, pvcName)
			}
			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvcNamespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvcName))
			if dvtSize != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(dvtName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(dvtName))
			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(dvtName))
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(bootOrder)))
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(dvtName))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(dvtName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
		},
			Entry("with src", "my-ns", "my-pvc", "", "", 0, "src:my-ns/my-pvc"),
			Entry("with src and name", "my-ns", "my-pvc", "my-dvt", "", 0, "src:my-ns/my-pvc,name:my-dvt"),
			Entry("with src and size", "my-ns", "my-pvc", "", "10Gi", 0, "src:my-ns/my-pvc,size:10Gi"),
			Entry("with src and bootorder", "my-ns", "my-pvc", "", "", 1, "src:my-ns/my-pvc,bootorder:1"),
			Entry("with src, name and size", "my-ns", "my-pvc", "my-dvt", "10Gi", 0, "src:my-ns/my-pvc,name:my-dvt,size:10Gi"),
			Entry("with src, name and bootorder", "my-ns", "my-pvc", "my-dvt", "", 2, "src:my-ns/my-pvc,name:my-dvt,bootorder:2"),
			Entry("with src, size and bootorder", "my-ns", "my-pvc", "", "10Gi", 3, "src:my-ns/my-pvc,size:10Gi,bootorder:3"),
			Entry("with src, name, size and bootorder", "my-ns", "my-pvc", "my-dvt", "10Gi", 4, "src:my-ns/my-pvc,name:my-dvt,size:10Gi,bootorder:4"),
		)

		DescribeTable("VM with specified pvc", func(pvcName, volName string, bootOrder int, params string) {
			out, err := runCmd(setFlag(PvcVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			if volName == "" {
				volName = pvcName
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(volName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvcName))
			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(volName))
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(bootOrder)))
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(volName))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(volName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
		},
			Entry("with src", "my-pvc", "", 0, "src:my-pvc"),
			Entry("with src and name", "my-pvc", "my-direct-pvc", 0, "src:my-pvc,name:my-direct-pvc"),
			Entry("with src and bootorder", "my-pvc", "", 1, "src:my-pvc,bootorder:1"),
			Entry("with src, name and bootorder", "my-pvc", "my-direct-pvc", 2, "src:my-pvc,name:my-direct-pvc,bootorder:2"),
		)

		DescribeTable("VM with blank disk", func(blankName, blankSize, params string) {
			out, err := runCmd(setFlag(BlankVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			if blankName == "" {
				blankName = fmt.Sprintf("%s-blank-0", vm.Name)
			}
			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(blankName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Blank).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(blankName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(blankName))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with size", "", "10Gi", "size:10Gi"),
			Entry("with size and name", "my-blank", "10Gi", "size:10Gi,name:my-blank"),
		)

		It("VM with specified cloud-init user data", func() {
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))
			out, err := runCmd(setFlag(CloudInitUserDataFlag, userDataB64))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		})

		It("VM with specified cloud-init network data", func() {
			networkDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))
			out, err := runCmd(setFlag(CloudInitNetworkDataFlag, networkDataB64))
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		})

		It("VM with specified cloud-init user and network data", func() {
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))
			networkDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))
			out, err := runCmd(
				setFlag(CloudInitUserDataFlag, userDataB64),
				setFlag(CloudInitNetworkDataFlag, networkDataB64),
			)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))
			decoded, err = base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		})

		It("Complex example", func() {
			const vmName = "my-vm"
			const runStrategy = v1.RunStrategyManual
			const terminationGracePeriod int64 = 123
			const instancetypeKind = "virtualmachineinstancetype"
			const instancetypeName = "my-instancetype"
			const dsNamespace = "my-ns"
			const dsName = "my-ds"
			const dvtSize = "10Gi"
			const pvcName = "my-pvc"
			const pvcBootOrder = 1
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

			out, err := runCmd(
				setFlag(NameFlag, vmName),
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
				setFlag(InstancetypeFlag, fmt.Sprintf("%s/%s", instancetypeKind, instancetypeName)),
				setFlag(InferPreferenceFromFlag, pvcName),
				setFlag(DataSourceVolumeFlag, fmt.Sprintf("src:%s/%s,size:%s", dsNamespace, dsName, dvtSize)),
				setFlag(PvcVolumeFlag, fmt.Sprintf("src:%s,bootorder:%d", pvcName, pvcBootOrder)),
				setFlag(CloudInitUserDataFlag, userDataB64),
			)
			Expect(err).ToNot(HaveOccurred())
			vm := unmarshalVM(out)

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(Equal(instancetypeKind))
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetypeName))
			Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(pvcName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

			dvtDsName := fmt.Sprintf("%s-ds-%s", vmName, dsName)
			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).ToNot(BeNil())
			Expect(*vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(Equal(dsNamespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dsName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(3))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal("cloudinitdisk"))
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(pvcName))
			Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(pvcBootOrder)))
		})
	})

	Describe("Manifest is not created successfully", func() {
		DescribeTable("Invalid values for RunStrategy", func(runStrategy string) {
			out, err := runCmd(setFlag(RunStrategyFlag, runStrategy))

			Expect(err).To(MatchError(fmt.Sprintf("failed to parse \"--run-strategy\" flag: invalid RunStrategy \"%s\", supported values are: Always, Manual, Halted, Once, RerunOnFailure", runStrategy)))
			Expect(out).To(BeEmpty())
		},
			Entry("some string", "not-a-bool"),
			Entry("float", "1.23"),
			Entry("bool", "true"),
		)

		DescribeTable("Invalid values for TerminationGracePeriodFlag", func(terminationGracePeriod string) {
			out, err := runCmd(setFlag(TerminationGracePeriodFlag, terminationGracePeriod))

			Expect(err).To(MatchError(fmt.Sprintf("invalid argument \"%s\" for \"--termination-grace-period\" flag: strconv.ParseInt: parsing \"%s\": invalid syntax", terminationGracePeriod, terminationGracePeriod)))
			Expect(out).To(BeEmpty())
		},
			Entry("string", "not-a-number"),
			Entry("float", "1.23"),
		)

		DescribeTable("Invalid arguments to MemoryFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(MemoryFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid number", "abc", "failed to parse \"--memory\" flag: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"),
			Entry("Invalid suffix", "512Gu", "failed to parse \"--memory\" flag: unable to parse quantity's suffix"),
		)

		DescribeTable("Invalid arguments to InstancetypeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(InstancetypeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid kind", "madethisup/my-instancetype", "failed to parse \"--instancetype\" flag: invalid instancetype kind \"madethisup\", supported values are: virtualmachineinstancetype, virtualmachineclusterinstancetype"),
			Entry("Invalid argument count", "virtualmachineinstancetype/my-instancetype/madethisup", "failed to parse \"--instancetype\" flag: invalid count 3 of slashes in prefix/name"),
			Entry("Empty name", "virtualmachineinstancetype/", "failed to parse \"--instancetype\" flag: name cannot be empty"),
		)

		It("Invalid argument to InferInstancetypeFlag", func() {
			out, err := runCmd(setFlag(InferInstancetypeFlag, "not-a-bool"))

			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--infer-instancetype\" flag: strconv.ParseBool: parsing \"not-a-bool\": invalid syntax"))
			Expect(out).To(BeEmpty())
		})

		It("InferInstancetypeFlag needs at least one volume", func() {
			out, err := runCmd(setFlag(InferInstancetypeFlag, "true"))

			Expect(err).To(MatchError("at least one volume is needed to infer an instance type or preference"))
			Expect(out).To(BeEmpty())
		})

		It("Volume specified in InferInstancetypeFromFlag should exist", func() {
			args := []string{
				setFlag(InferInstancetypeFromFlag, "does-not-exist"),
			}
			out, err := runCmd(args...)

			Expect(err).To(MatchError("there is no volume with name 'does-not-exist'"))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("MemoryFlag, InstancetypeFlag, InferInstancetypeFlag and InferInstancetypeFromFlag are mutually exclusive", func(flags []string, setFlags string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(fmt.Sprintf("if any flags in the group [memory instancetype infer-instancetype infer-instancetype-from] are set none of the others can be; [%s] were all set", setFlags)))
			Expect(out).To(BeEmpty())
		},
			Entry("MemoryFlag and InstancetypeFlag", []string{setFlag(MemoryFlag, "1Gi"), setFlag(InstancetypeFlag, "my-instancetype")}, "instancetype memory"),
			Entry("MemoryFlag and InferInstancetypeFlag", []string{setFlag(MemoryFlag, "1Gi"), setFlag(InferInstancetypeFlag, "true")}, "infer-instancetype memory"),
			Entry("MemoryFlag and InferInstancetypeFromFlag", []string{setFlag(MemoryFlag, "1Gi"), setFlag(InferInstancetypeFromFlag, "my-vol")}, "infer-instancetype-from memory"),
			Entry("InstancetypeFlag and InferInstancetypeFlag", []string{setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFlag, "true")}, "infer-instancetype instancetype"),
			Entry("InstancetypeFlag and InferInstancetypeFromFlag", []string{setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFromFlag, "my-vol")}, "infer-instancetype-from instancetype"),
			Entry("InferInstancetypeFlag and InferInstancetypeFromFlag", []string{setFlag(InferInstancetypeFlag, "true"), setFlag(InferInstancetypeFromFlag, "my-vol")}, "infer-instancetype infer-instancetype-from"),
			Entry("MemoryFlag, InstancetypeFlag, InferInstancetypeFlag and InferInstancetypeFromFlag", []string{setFlag(MemoryFlag, "1Gi"), setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")}, "infer-instancetype instancetype memory"),
		)

		DescribeTable("Invalid arguments to PreferenceFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(PreferenceFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid kind", "madethisup/my-preference", "failed to parse \"--instancetype\" flag: invalid preference kind \"madethisup\", supported values are: virtualmachinepreference, virtualmachineclusterpreference"),
			Entry("Invalid argument count", "virtualmachinepreference/my-preference/madethisup", "failed to parse \"--preference\" flag: invalid count 3 of slashes in prefix/name"),
			Entry("Empty name", "virtualmachinepreference/", "failed to parse \"--preference\" flag: name cannot be empty"),
		)

		It("Invalid argument to InferPreferenceFlag", func() {
			out, err := runCmd(setFlag(InferPreferenceFlag, "not-a-bool"))

			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--infer-preference\" flag: strconv.ParseBool: parsing \"not-a-bool\": invalid syntax"))
			Expect(out).To(BeEmpty())
		})

		It("InferPreferenceFlag needs at least one volume", func() {
			out, err := runCmd(setFlag(InferPreferenceFlag, "true"))

			Expect(err).To(MatchError("at least one volume is needed to infer an instance type or preference"))
			Expect(out).To(BeEmpty())
		})

		It("Volume specified in InferPreferenceFromFlag should exist", func() {
			args := []string{
				setFlag(InferPreferenceFromFlag, "does-not-exist"),
			}
			out, err := runCmd(args...)

			Expect(err).To(MatchError("there is no volume with name 'does-not-exist'"))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("PreferenceFlag, InferPreferenceFlag and InferPreferenceFromFlag are mutually exclusive", func(flags []string, setFlags string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(fmt.Sprintf("if any flags in the group [preference infer-preference infer-preference-from] are set none of the others can be; [%s] were all set", setFlags)))
			Expect(out).To(BeEmpty())
		},
			Entry("PreferenceFlag and InferPreferenceFlag", []string{setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFlag, "true")}, "infer-preference preference"),
			Entry("PreferenceFlag and InferPreferenceFromFlag", []string{setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFromFlag, "my-vol")}, "infer-preference-from preference"),
			Entry("InferPreference and InferPreferenceFromFlag", []string{setFlag(InferPreferenceFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")}, "infer-preference infer-preference-from"),
			Entry("PreferenceFlag, InferPreferenceFlag and InferPreferenceFromFlag", []string{setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")}, "infer-preference infer-preference-from preference"),
		)

		DescribeTable("Volume to explicitly infer from needs to be valid", func(args []string) {
			out, err := runCmd(args...)

			Expect(err).To(MatchError("inference of instancetype or preference works only with DataSources, DataVolumes or PersistentVolumeClaims"))
			Expect(out).To(BeEmpty())
		},
			Entry("explicit inference of instancetype with ContainerdiskVolumeFlag", []string{setFlag(InferInstancetypeFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")}),
			Entry("inference of instancetype from ContainerdiskVolumeFlag", []string{setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")}),
			Entry("explicit inference of preference with ContainerdiskVolumeFlag", []string{setFlag(InferPreferenceFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")}),
			Entry("inference of preference from ContainerdiskVolumeFlag", []string{setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")}),
			Entry("explicit inference of instancetype with VolumeImportFlag", []string{setFlag(InferInstancetypeFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")}),
			Entry("inference of instancetype from VolumeImportFlag", []string{setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")}),
			Entry("explicit inference of preference with VolumeImportFlag", []string{setFlag(InferPreferenceFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")}),
			Entry("inference of preference from VolumeImportFlag", []string{setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")}),
		)

		DescribeTable("Invalid arguments to DataSourceVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(DataSourceVolumeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-datasource\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-datasource\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-datasource\" flag: unknown param(s): test:test"),
			Entry("Missing src", "name:test", "failed to parse \"--volume-datasource\" flag: src must be specified"),
			Entry("Empty name in src", "src:my-ns/", "failed to parse \"--volume-datasource\" flag: src invalid: name cannot be empty"),
			Entry("Invalid slashes count in src", "src:my-ns/my-ds/madethisup", "failed to parse \"--volume-datasource\" flag: src invalid: invalid count 3 of slashes in prefix/name"),
			Entry("Invalid quantity in size", "size:10Gu", "failed to parse \"--volume-datasource\" flag: failed to parse param \"size\": unable to parse quantity's suffix"),
			Entry("Invalid number in bootorder", "bootorder:10Gu", "failed to parse \"--volume-datasource\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"),
			Entry("Negative number in bootorder", "bootorder:-1", "failed to parse \"--volume-datasource\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("Bootorder set to 0", "src:my-ds,bootorder:0", "failed to parse \"--volume-datasource\" flag: bootorder must be greater than 0"),
		)

		DescribeTable("Invalid arguments to VolumeImportFlag", func(errMsg string, flags ...string) {
			out, err := runCmd(flags...)

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Missing size with blank source", "size must be specified", setFlag(VolumeImportFlag, "type:blank")),
			Entry("Missing type value", "type must be specified", setFlag(VolumeImportFlag, "size:256Mi")),
			Entry("Missing url with http volume source", "failed to parse \"--volume-import\" flag: URL is required with http volume source", setFlag(VolumeImportFlag, "type:http,size:256Mi")),
			Entry("Missing url in imageIO volume source", "failed to parse \"--volume-import\" flag: URL and diskid are both required with imageIO volume source", setFlag(VolumeImportFlag, "type:imageio,diskid:0,size:256Mi")),
			Entry("Missing diskid in imageIO volume source", "failed to parse \"--volume-import\" flag: URL and diskid are both required with imageIO volume source", setFlag(VolumeImportFlag, "type:imageio,url:http://imageio.com,size:256Mi")),
			Entry("Missing src in pvc volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi")),
			Entry("Invalid src without slash in pvc volume source", "failed to parse \"--volume-import\" flag: namespace of pvc 'noslashingvalue' must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:noslashingvalue")),
			Entry("Invalid src in pvc volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:")),
			Entry("Missing src namespace in pvc volume source", "failed to parse \"--volume-import\" flag: namespace of pvc 'my-pvc' must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:/my-pvc")),
			Entry("Missing src name in pvc volume source", "failed to parse \"--volume-import\" flag: src invalid: name cannot be empty", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:default/")),
			Entry("Invalid src without slash in snapshot volume source", "failed to parse \"--volume-import\" flag: namespace of snapshot 'noslashingvalue' must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:noslashingvalue")),
			Entry("Missing src in snapshot volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi")),
			Entry("Invalid src in snapshot volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi")),
			Entry("Missing src namespace in snapshot volume source", "failed to parse \"--volume-import\" flag: namespace of snapshot 'my-snapshot' must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:/my-snapshot")),
			Entry("Missing src name in snapshot volume source", "failed to parse \"--volume-import\" flag: src invalid: name cannot be empty", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:default/")),
			Entry("Missing url in S3 volume source", "failed to parse \"--volume-import\" flag: URL is required with S3 volume source", setFlag(VolumeImportFlag, "type:s3,size:256Mi")),
			Entry("Unknown argument for blank source", fmt.Sprintf("failed to parse \"--volume-import\" flag: unknown param(s): %s", url), setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:256Mi,%s", url))),
			Entry("Invalid value for PullMethod", "failed to parse \"--volume-import\" flag: pullmethod must be set to pod or node", setFlag(VolumeImportFlag, fmt.Sprintf("type:registry,size:%s,pullmethod:invalid,imagestream:my-image", size))),
			Entry("Both url and imagestream defined in registry source", "failed to parse \"--volume-import\" flag: exactly one of url or imagestream must be defined", setFlag(VolumeImportFlag, fmt.Sprintf("type:registry,size:%s,pullmethod:node,imagestream:my-image,url:%s", size, url))),
			Entry("Missing url and imagestream in registry source", "failed to parse \"--volume-import\" flag: exactly one of url or imagestream must be defined", setFlag(VolumeImportFlag, fmt.Sprintf("type:registry,size:%s", size))),
			Entry("Volume already exists", "failed to parse \"--volume-import\" flag: there is already a volume with name 'duplicated'", setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s,name:duplicated", size)), setFlag(VolumeImportFlag, fmt.Sprintf("type:blank,size:%s,name:duplicated", size))),
		)

		DescribeTable("Invalid arguments to ContainerdiskVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(ContainerdiskVolumeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-containerdisk\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-containerdisk\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-containerdisk\" flag: unknown param(s): test:test"),
			Entry("Missing src", "name:test", "failed to parse \"--volume-containerdisk\" flag: src must be specified"),
			Entry("Invalid number in bootorder", "bootorder:10Gu", "failed to parse \"--volume-containerdisk\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"),
			Entry("Negative number in bootorder", "bootorder:-1", "failed to parse \"--volume-containerdisk\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("Bootorder set to 0", "src:my.registry/my-image:my-tag,bootorder:0", "failed to parse \"--volume-containerdisk\" flag: bootorder must be greater than 0"),
		)

		DescribeTable("Invalid arguments to ClonePvcVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(ClonePvcVolumeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-clone-pvc\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-clone-pvc\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-clone-pvc\" flag: unknown param(s): test:test"),
			Entry("Missing src", "name:test", "failed to parse \"--volume-clone-pvc\" flag: src must be specified"),
			Entry("Empty name in src", "src:my-ns/", "failed to parse \"--volume-clone-pvc\" flag: src invalid: name cannot be empty"),
			Entry("Invalid slashes count in src", "src:my-ns/my-pvc/madethisup", "failed to parse \"--volume-clone-pvc\" flag: src invalid: invalid count 3 of slashes in prefix/name"),
			Entry("Missing namespace in src", "src:my-pvc", "failed to parse \"--volume-clone-pvc\" flag: namespace of pvc 'my-pvc' must be specified"),
			Entry("Invalid quantity in size", "size:10Gu", "failed to parse \"--volume-clone-pvc\" flag: failed to parse param \"size\": unable to parse quantity's suffix"),
			Entry("Invalid number in bootorder", "bootorder:10Gu", "failed to parse \"--volume-clone-pvc\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"),
			Entry("Negative number in bootorder", "bootorder:-1", "failed to parse \"--volume-clone-pvc\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("Bootorder set to 0", "src:my-ns/my-pvc,bootorder:0", "failed to parse \"--volume-clone-pvc\" flag: bootorder must be greater than 0"),
		)

		DescribeTable("Invalid arguments to PvcVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(PvcVolumeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-pvc\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-pvc\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-pvc\" flag: unknown param(s): test:test"),
			Entry("Missing src", "name:test", "failed to parse \"--volume-pvc\" flag: src must be specified"),
			Entry("Empty name in src", "src:my-ns/", "failed to parse \"--volume-pvc\" flag: src invalid: name cannot be empty"),
			Entry("Invalid slashes count in src", "src:my-ns/my-pvc/madethisup", "failed to parse \"--volume-pvc\" flag: src invalid: invalid count 3 of slashes in prefix/name"),
			Entry("Namespace in src", "src:my-ns/my-pvc", "failed to parse \"--volume-pvc\" flag: not allowed to specify namespace of pvc 'my-pvc'"),
			Entry("Invalid number in bootorder", "bootorder:10Gu", "failed to parse \"--volume-pvc\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"),
			Entry("Negative number in bootorder", "bootorder:-1", "failed to parse \"--volume-pvc\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("Bootorder set to 0", "src:my-pvc,bootorder:0", "failed to parse \"--volume-pvc\" flag: bootorder must be greater than 0"),
		)

		DescribeTable("Invalid arguments to BlankVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(BlankVolumeFlag, flag))

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-blank\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-blank\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-blank\" flag: unknown param(s): test:test"),
			Entry("Missing size", "name:my-blank", "failed to parse \"--volume-blank\" flag: size must be specified"),
		)

		DescribeTable("Duplicate DataVolumeTemplates or Volumes are not allowed", func(errMsg string, flags ...string) {
			out, err := runCmd(flags...)

			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Duplicate Containerdisk", "failed to parse \"--volume-containerdisk\" flag: there is already a volume with name 'my-name'",
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
			),
			Entry("Duplicate DataSource", "failed to parse \"--volume-datasource\" flag: there is already a volume with name 'my-name'",
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:my-name"),
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:my-name"),
			),
			Entry("Duplicate ClonePvc", "failed to parse \"--volume-clone-pvc\" flag: there is already a volume with name 'my-name'",
				setFlag(ClonePvcVolumeFlag, "src:my-ns/my-pvc,name:my-name"),
				setFlag(ClonePvcVolumeFlag, "src:my-ns/my-pvc,name:my-name"),
			),
			Entry("Duplicate PVC", "failed to parse \"--volume-pvc\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
			),
			Entry("Duplicate blank volume", "failed to parse \"--volume-blank\" flag: there is already a volume with name 'my-name'",
				setFlag(BlankVolumeFlag, "size:10Gi,name:my-name"),
				setFlag(BlankVolumeFlag, "size:10Gi,name:my-name"),
			),
			Entry("Duplicate PVC and Containerdisk", "failed to parse \"--volume-pvc\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
			),
			Entry("Duplicate PVC and DataSource", "failed to parse \"--volume-pvc\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:my-name"),
			),
			Entry("Duplicate PVC and ClonePvc", "failed to parse \"--volume-pvc\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(ClonePvcVolumeFlag, "src:my-ns/my-pvc,name:my-name"),
			),
			Entry("Duplicate PVC and blank volume", "failed to parse \"--volume-blank\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(BlankVolumeFlag, "size:10Gi,name:my-name"),
			),
			Entry("There can only be one cloudInitDisk (UserData)", "failed to parse \"--cloud-init-user-data\" flag: there is already a volume with name 'cloudinitdisk'",
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitUserDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))),
			),
			Entry("There can only be one cloudInitDisk (NetworkData)", "failed to parse \"--cloud-init-network-data\" flag: there is already a volume with name 'cloudinitdisk'",
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitNetworkDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))),
			),
		)

		It("Duplicate boot orders are not allowed", func() {
			out, err := runCmd(
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,bootorder:1"),
				setFlag(DataSourceVolumeFlag, "src:my-ds,bootorder:1"),
			)

			Expect(err).To(MatchError("failed to parse \"--volume-datasource\" flag: bootorder 1 was specified multiple times"))
			Expect(out).To(BeEmpty())
		})
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(args ...string) ([]byte, error) {
	_args := append([]string{create, VM}, args...)
	return clientcmd.NewRepeatableVirtctlCommandWithOut(_args...)()
}

func unmarshalVM(bytes []byte) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{}
	Expect(yaml.Unmarshal(bytes, vm)).To(Succeed())
	Expect(vm.Kind).To(Equal("VirtualMachine"))
	Expect(vm.APIVersion).To(Equal("kubevirt.io/v1"))
	return vm
}
