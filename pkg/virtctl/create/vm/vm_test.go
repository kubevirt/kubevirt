package vm_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	. "kubevirt.io/kubevirt/pkg/virtctl/create/vm"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("create vm", func() {
	const (
		importedVolumeRegexp = `imported-volume-\w{5}`
		cloudInitUserData    = `#cloud-config
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
	)

	Context("Manifest is created successfully", func() {
		It("VM with random name", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			_, err = decodeVM(out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM with specified namespace", func() {
			const namespace = "my-namespace"
			out, err := runCmd(setFlag("namespace", namespace))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Namespace).To(Equal(namespace))
		})

		It("VM with specified name", func() {
			const name = "my-vm"
			out, err := runCmd(setFlag(NameFlag, name))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(name))
		})

		It("RunStrategy is set to Always by default", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))
		})

		It("VM with specified run strategy", func() {
			const runStrategy = v1.RunStrategyManual
			out, err := runCmd(setFlag(RunStrategyFlag, string(runStrategy)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))
		})

		It("Termination grace period defaults to 180", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(int64(180)))
		})

		It("VM with specified termination grace period", func() {
			const terminationGracePeriod int64 = 123
			out, err := runCmd(setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))
		})

		It("Memory is set to 512Mi by default", func() {
			const defaultMemory = "512Mi"
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(defaultMemory)))
		})

		It("VM with specified memory", func() {
			const memory = "1Gi"
			out, err := runCmd(setFlag(MemoryFlag, string(memory)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse(memory)))
		})

		DescribeTable("VM with specified instancetype", func(flag, name, kind string) {
			out, err := runCmd(setFlag(InstancetypeFlag, flag))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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

		DescribeTable("VM with inferred instancetype", func(inferFromVolume string, inferFromVolumePolicy *v1.InferFromVolumeFailurePolicy, args ...string) {
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			Entry("PvcVolumeFlag and implicit inference (enabled by default)", "my-pvc", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(PvcVolumeFlag, "src:my-pvc")),
			Entry("PvcVolumeFlag and explicit inference", "my-pvc", nil, setFlag(PvcVolumeFlag, "src:my-pvc"), setFlag(InferInstancetypeFlag, "true")),
			Entry("VolumeImportFlag and implicit inference with pvc source (enabled by default)", "my-pvc", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-namespace/my-volume,name:my-pvc")),
			Entry("VolumeImportFlag and explicit inference with pvc source", "my-pvc", nil, setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-ns/my-volume,name:my-pvc"), setFlag(InferInstancetypeFlag, "true")),
			Entry("VolumeImportFlag and implicit inference with registry source (enabled by default)", "my-containerdisk", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(VolumeImportFlag, "type:registry,size:1Gi,url:docker://my-containerdisk,name:my-containerdisk")),
			Entry("VolumeImportFlag and explicit inference with registry source", "my-containerdisk", nil, setFlag(VolumeImportFlag, "type:registry,size:1Gi,url:docker://my-containerdisk,name:my-containerdisk"), setFlag(InferInstancetypeFlag, "true")),
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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Instancetype).To(BeNil())
		})

		It("VM with specified memory and volume and without implicitly inferred instancetype", func() {
			const memory = "1Gi"
			out, err := runCmd(
				setFlag(MemoryFlag, memory),
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:my-ds"))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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

		DescribeTable("VM with inferred preference", func(inferFromVolume string, inferFromVolumePolicy *v1.InferFromVolumeFailurePolicy, args ...string) {
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			Entry("PvcVolumeFlag and implicit inference (enabled by default)", "my-pvc", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(PvcVolumeFlag, "src:my-pvc")),
			Entry("PvcVolumeFlag and explicit inference", "my-pvc", nil, setFlag(PvcVolumeFlag, "src:my-pvc"), setFlag(InferPreferenceFlag, "true")),
			Entry("VolumeImportFlag and implicit inference with pvc source (enabled by default)", "my-pvc", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-namespace/my-volume,name:my-pvc")),
			Entry("VolumeImportFlag and explicit inference with pvc source", "my-pvc", nil, setFlag(VolumeImportFlag, "type:pvc,size:1Gi,src:my-ns/my-volume,name:my-pvc"), setFlag(InferPreferenceFlag, "true")),
			Entry("VolumeImportFlag and implicit inference with registry source (enabled by default)", "my-containerdisk", pointer.P(v1.IgnoreInferFromVolumeFailure), setFlag(VolumeImportFlag, "type:registry,size:1Gi,url:docker://my-containerdisk,name:my-containerdisk")),
			Entry("VolumeImportFlag and explicit inference with registry source", "my-containerdisk", nil, setFlag(VolumeImportFlag, "type:registry,size:1Gi,url:docker://my-containerdisk,name:my-containerdisk"), setFlag(InferPreferenceFlag, "true")),
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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Preference).To(BeNil())
		})

		DescribeTable("VM with specified containerdisk", func(containerdisk, volName string, bootOrder int, params string) {
			out, err := runCmd(setFlag(ContainerdiskVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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

		DescribeTable("VM with specified imported volume", func(params, name, size string, bootOrder *int, source *cdiv1.DataVolumeSource, sourceRef *cdiv1.DataVolumeSourceRef) {
			out, err := runCmd(setFlag(VolumeImportFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			if source != nil {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(Equal(source))
			}
			if sourceRef != nil {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).To(Equal(sourceRef))
			}
			if name == "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(name))
			}
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage).ToNot(BeNil())
			if size != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
			}

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[0].DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))

			if bootOrder == nil {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).ToNot(BeNil())
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(*bootOrder)))
			}

			if (source != nil && (source.PVC != nil || source.Registry != nil || source.Snapshot != nil)) ||
				(sourceRef != nil && sourceRef.Kind == "DataSource") {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).ToNot(BeNil())
				Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.Template.Spec.Volumes[0].Name))
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
				Expect(vm.Spec.Preference).ToNot(BeNil())
				Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.Template.Spec.Volumes[0].Name))
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
				Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			} else {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			}
		},
			Entry("with blank source", "type:blank,size:256Mi", "", "256Mi", nil, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with blank source and bootorder", "type:blank,size:256Mi,bootorder:1", "", "256Mi", pointer.P(1), &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with blank source and name", "type:blank,size:256Mi,name:blank-name", "blank-name", "256Mi", nil, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with GCS source", "type:gcs,size:256Mi,url:http://url.com,secretref:test-credentials", "", "256Mi", nil, &cdiv1.DataVolumeSource{GCS: &cdiv1.DataVolumeSourceGCS{URL: "http://url.com", SecretRef: "test-credentials"}}, nil),
			Entry("with GCS source and bootorder", "type:gcs,size:256Mi,url:http://url.com,secretref:test-credentials,bootorder:2", "", "256Mi", pointer.P(2), &cdiv1.DataVolumeSource{GCS: &cdiv1.DataVolumeSourceGCS{URL: "http://url.com", SecretRef: "test-credentials"}}, nil),
			Entry("with http source", "type:http,size:256Mi,url:http://url.com", "", "256Mi", nil, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}, nil),
			Entry("with http source and bootorder", "type:http,size:256Mi,url:http://url.com,bootorder:3", "", "256Mi", pointer.P(3), &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}, nil),
			Entry("with imageio source", "type:imageio,size:256Mi,url:http://url.com,diskid:1,secretref:secret-ref", "", "256Mi", nil, &cdiv1.DataVolumeSource{Imageio: &cdiv1.DataVolumeSourceImageIO{DiskID: "1", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with imageio source and bootorder", "type:imageio,size:256Mi,url:http://url.com,diskid:1,secretref:secret-ref,bootorder:4", "", "256Mi", pointer.P(4), &cdiv1.DataVolumeSource{Imageio: &cdiv1.DataVolumeSourceImageIO{DiskID: "1", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with PVC source", "type:pvc,size:256Mi,src:default/pvc", "", "256Mi", nil, &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with PVC source and bootorder", "type:pvc,size:256Mi,src:default/pvc,name:imported-volume,bootorder:5", "imported-volume", "256Mi", pointer.P(5), &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with PVC source without size", "type:pvc,src:default/pvc,name:imported-volume", "imported-volume", "", nil, &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with registry source", "type:registry,size:256Mi,certconfigmap:my-cert,pullmethod:pod,url:http://url.com,secretref:secret-ref,name:imported-volume", "imported-volume", "256Mi", nil, &cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{CertConfigMap: pointer.P("my-cert"), PullMethod: pointer.P(cdiv1.RegistryPullMethod("pod")), URL: pointer.P("http://url.com"), SecretRef: pointer.P("secret-ref")}}, nil),
			Entry("with registry source and bootorder", "type:registry,size:256Mi,certconfigmap:my-cert,pullmethod:pod,url:http://url.com,secretref:secret-ref,name:imported-volume,bootorder:6", "imported-volume", "256Mi", pointer.P(6), &cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{CertConfigMap: pointer.P("my-cert"), PullMethod: pointer.P(cdiv1.RegistryPullMethod("pod")), URL: pointer.P("http://url.com"), SecretRef: pointer.P("secret-ref")}}, nil),
			Entry("with S3 source", "type:s3,size:256Mi,url:http://url.com,certconfigmap:my-cert,secretref:secret-ref", "", "256Mi", nil, &cdiv1.DataVolumeSource{S3: &cdiv1.DataVolumeSourceS3{CertConfigMap: "my-cert", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with S3 source and bootorder", "type:s3,size:256Mi,url:http://url.com,certconfigmap:my-cert,secretref:secret-ref,bootorder:7", "", "256Mi", pointer.P(7), &cdiv1.DataVolumeSource{S3: &cdiv1.DataVolumeSourceS3{CertConfigMap: "my-cert", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with VDDK source", "type:vddk,size:256Mi,backingfile:backing-file,initimageurl:http://url.com,uuid:123e-11,url:http://url.com,thumbprint:test-thumbprint,secretref:test-credentials", "", "256Mi", nil, &cdiv1.DataVolumeSource{VDDK: &cdiv1.DataVolumeSourceVDDK{BackingFile: "backing-file", InitImageURL: "http://url.com", UUID: "123e-11", URL: "http://url.com", Thumbprint: "test-thumbprint", SecretRef: "test-credentials"}}, nil),
			Entry("with VDDK source and bootorder", "type:vddk,size:256Mi,backingfile:backing-file,initimageurl:http://url.com,uuid:123e-11,url:http://url.com,thumbprint:test-thumbprint,secretref:test-credentials,bootorder:8", "", "256Mi", pointer.P(8), &cdiv1.DataVolumeSource{VDDK: &cdiv1.DataVolumeSourceVDDK{BackingFile: "backing-file", InitImageURL: "http://url.com", UUID: "123e-11", URL: "http://url.com", Thumbprint: "test-thumbprint", SecretRef: "test-credentials"}}, nil),
			Entry("with Snapshot source", "type:snapshot,size:256Mi,src:default/snapshot,name:imported-volume", "imported-volume", "256Mi", nil, &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with Snapshot source and bootorder", "type:snapshot,size:256Mi,src:default/snapshot,name:imported-volume,bootorder:9", "imported-volume", "256Mi", pointer.P(9), &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with Snapshot source without size", "type:snapshot,src:default/snapshot,name:imported-volume", "imported-volume", "", nil, &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with DataSource source", "type:ds,src:default/datasource,name:imported-ds", "imported-ds", "", nil, nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource", Namespace: pointer.P("default")}),
			Entry("with DataSource source without namespace", "type:ds,src:datasource", "", "", nil, nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource"}),
			Entry("with DataSource source and bootorder", "type:ds,src:default/datasource,name:imported-ds,bootorder:1", "imported-ds", "", pointer.P(1), nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource", Namespace: pointer.P("default")}),
		)

		DescribeTable("VM with multiple volume-import sources and name", func(source1 *cdiv1.DataVolumeSource, source2 *cdiv1.DataVolumeSource, size string, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			Entry("with blank source", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, "256Mi", setFlag(VolumeImportFlag, "type:blank,size:256Mi,name:volume-source1"), setFlag(VolumeImportFlag, "type:blank,size:256Mi,name:volume-source2")),
			Entry("with blank source and http source", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}, "256Mi", setFlag(VolumeImportFlag, "type:blank,size:256Mi,name:volume-source1"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com,name:volume-source2")),
		)

		DescribeTable("VM with specified clone pvc", func(pvcNamespace, pvcName, dvtName, dvtSize string, bootOrder int, params string) {
			out, err := runCmd(setFlag(ClonePvcVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			if dvtName == "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtName))
			}
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvcNamespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvcName))
			if dvtSize != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
				Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(bootOrder)))
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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

		DescribeTable("VM with specified sysprep volume", func(volSrc, volType, params string) {
			out, err := runCmd(setFlag(SysprepVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(SysprepDisk))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep).ToNot(BeNil())

			switch volType {
			case SysprepConfigMap:
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap.Name).To(Equal(volSrc))
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret).To(BeNil())
			case SysprepSecret:
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap).To(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret.Name).To(Equal(volSrc))
			default:
				Fail(fmt.Sprintf("invalid sysprep volume type %s", volType))

			}

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("ConfigMap with src (implicity default)", "my-src", SysprepConfigMap, "src:my-src"),
			Entry("ConfigMap with src and type", "my-src", SysprepConfigMap, "src:my-src,type:configMap"),
			Entry("Secret with src and type", "my-src", SysprepSecret, "src:my-src,type:secret"),
		)

		DescribeTable("VM with user specified in cloud-init user data", func(userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			const user = "my-user"
			args := append([]string{
				setFlag(UserFlag, user),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("user: " + user))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with password read from file in cloud-init user data", func(userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			password := rand.String(12)

			path := filepath.Join(GinkgoT().TempDir(), "pw")
			file, err := os.Create(path)
			Expect(err).ToNot(HaveOccurred())
			_, err = file.Write([]byte(password))
			Expect(err).ToNot(HaveOccurred())
			Expect(file.Close()).To(Succeed())

			args := append([]string{
				setFlag(PasswordFileFlag, path),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("password: %s\nchpasswd: { expire: False }", password))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with ssh key in cloud-init user data", func(argsFn func() ([]string, string), userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			args, keys := argsFn()
			args = append(args, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("ssh_authorized_keys:" + keys))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default) and single key", randomSingleKey, noCloudUserData),
			Entry("with CloudInitNoCloud (explicit) and single key", randomSingleKey, noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and single key", randomSingleKey, configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
			Entry("with CloudInitNoCLoud (default) and multiple keys in single flag", randomMultipleKeysSingleFlag, noCloudUserData),
			Entry("with CloudInitNoCLoud (explicit) and multiple keys in single flag", randomMultipleKeysSingleFlag, noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and multiple keys in single flag", randomMultipleKeysSingleFlag, configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
			Entry("with CloudInitNoCloud (default) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, noCloudUserData),
			Entry("with CloudInitNoCloud (explicit) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		It("VM with no generated cloud-init config while setting option", func() {
			out, err := runCmd(
				setFlag(CloudInitFlag, CloudInitNone),
				setFlag(GAManageSSHFlag, "true"),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
		})

		DescribeTable("VM with qemu-guest-agent managing SSH enabled in cloud-init user data", func(userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			args := append([]string{
				setFlag(GAManageSSHFlag, "true"),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("runcmd:\n  - [ setsebool, -P, 'virt_qemu_ga_manage_ssh', 'on' ]"))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with specified cloud-init user data", func(userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))
			args := append([]string{
				setFlag(CloudInitUserDataFlag, userDataB64),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(userDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserDataB64, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserDataB64, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with specified cloud-init network data", func(networkDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			networkDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))
			args := append([]string{
				setFlag(CloudInitNetworkDataFlag, networkDataB64),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(networkDataFn(vm)).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(networkDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudNetworkDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudNetworkDataB64, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveNetworkDataB64, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with specified cloud-init user and network data", func(userDataFn func(*v1.VirtualMachine) string, networkDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))
			networkDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))
			args := append([]string{
				setFlag(CloudInitUserDataFlag, userDataB64),
				setFlag(CloudInitNetworkDataFlag, networkDataB64),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(Equal(userDataB64))
			Expect(networkDataFn(vm)).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(userDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))
			decoded, err = base64.StdEncoding.DecodeString(networkDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserDataB64, noCloudNetworkDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserDataB64, noCloudNetworkDataB64, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserDataB64, configDriveNetworkDataB64, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

		DescribeTable("VM with generated cloud-init user and specified network data", func(userDataFn func(*v1.VirtualMachine) string, networkDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			const user = "my-user"
			networkDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))
			args := append([]string{
				setFlag(UserFlag, user),
				setFlag(CloudInitNetworkDataFlag, networkDataB64),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(CloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("user: " + user))
			Expect(networkDataFn(vm)).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(networkDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData, noCloudNetworkDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, noCloudNetworkDataB64, setFlag(CloudInitFlag, CloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, configDriveNetworkDataB64, setFlag(CloudInitFlag, CloudInitConfigDrive)),
		)

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
			const secretName = "my-secret"
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

			out, err := runCmd(
				setFlag(NameFlag, vmName),
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
				setFlag(InstancetypeFlag, fmt.Sprintf("%s/%s", instancetypeKind, instancetypeName)),
				setFlag(InferPreferenceFromFlag, pvcName),
				setFlag(DataSourceVolumeFlag, fmt.Sprintf("src:%s/%s,size:%s", dsNamespace, dsName, dvtSize)),
				setFlag(PvcVolumeFlag, fmt.Sprintf("src:%s,bootorder:%d", pvcName, pvcBootOrder)),
				setFlag(SysprepVolumeFlag, fmt.Sprintf("src:%s,type:%s", secretName, SysprepSecret)),
				setFlag(CloudInitUserDataFlag, userDataB64),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

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

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(4))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(dvtDsName))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(SysprepDisk))
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.ConfigMap).To(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.Secret).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.Secret.Name).To(Equal(secretName))
			Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal(CloudInitDisk))
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(pvcName))
			Expect(*vm.Spec.Template.Spec.Domain.Devices.Disks[0].BootOrder).To(Equal(uint(pvcBootOrder)))
		})

		It("Complex example with generated cloud-init config", func() {
			const vmName = "my-vm"
			const terminationGracePeriod int64 = 180
			const pvcNamespace = "my-ns"
			const pvcName = "my-ds"
			const dvtSize = "10Gi"
			const user = "my-user"
			const sshKey = "my-ssh-key"

			out, err := runCmd(
				setFlag(NameFlag, vmName),
				setFlag(VolumeImportFlag, fmt.Sprintf("type:pvc,src:%s/%s,size:%s", pvcNamespace, pvcName, dvtSize)),
				setFlag(UserFlag, user),
				setFlag(SSHKeyFlag, sshKey),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(terminationGracePeriod))

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(`imported-volume-\w{4}`))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvcName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvcNamespace))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(CloudInitDisk))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("user: " + user))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("ssh_authorized_keys:\n  - " + sshKey))

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
			Expect(vm.Spec.Instancetype.Name).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(resource.MustParse("512Mi")))

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).ToNot(BeNil())
			Expect(*vm.Spec.Preference.InferFromVolumeFailurePolicy).To(Equal(v1.IgnoreInferFromVolumeFailure))
		})
	})

	Describe("Manifest is not created successfully", func() {
		DescribeTable("Invalid arguments to RunStrategyFlag", func(runStrategy string) {
			out, err := runCmd(setFlag(RunStrategyFlag, runStrategy))
			Expect(err).To(MatchError(fmt.Sprintf("failed to parse \"--run-strategy\" flag: invalid RunStrategy \"%s\", supported values are: Always, Manual, Halted, Once, RerunOnFailure", runStrategy)))
			Expect(out).To(BeEmpty())
		},
			Entry("some string", "not-a-bool"),
			Entry("float", "1.23"),
			Entry("bool", "true"),
		)

		DescribeTable("Invalid arguments to TerminationGracePeriodFlag", func(terminationGracePeriod string) {
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
			out, err := runCmd(setFlag(InferInstancetypeFromFlag, "does-not-exist"))
			Expect(err).To(MatchError("there is no volume with name 'does-not-exist'"))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("MemoryFlag, InstancetypeFlag, InferInstancetypeFlag and InferInstancetypeFromFlag are mutually exclusive", func(setFlags string, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(fmt.Sprintf("if any flags in the group [memory instancetype infer-instancetype infer-instancetype-from] are set none of the others can be; [%s] were all set", setFlags)))
			Expect(out).To(BeEmpty())
		},
			Entry("MemoryFlag and InstancetypeFlag", "instancetype memory", setFlag(MemoryFlag, "1Gi"), setFlag(InstancetypeFlag, "my-instancetype")),
			Entry("MemoryFlag and InferInstancetypeFlag", "infer-instancetype memory", setFlag(MemoryFlag, "1Gi"), setFlag(InferInstancetypeFlag, "true")),
			Entry("MemoryFlag and InferInstancetypeFromFlag", "infer-instancetype-from memory", setFlag(MemoryFlag, "1Gi"), setFlag(InferInstancetypeFromFlag, "my-vol")),
			Entry("InstancetypeFlag and InferInstancetypeFlag", "infer-instancetype instancetype", setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFlag, "true")),
			Entry("InstancetypeFlag and InferInstancetypeFromFlag", "infer-instancetype-from instancetype", setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFromFlag, "my-vol")),
			Entry("InferInstancetypeFlag and InferInstancetypeFromFlag", "infer-instancetype infer-instancetype-from", setFlag(InferInstancetypeFlag, "true"), setFlag(InferInstancetypeFromFlag, "my-vol")),
			Entry("MemoryFlag, InstancetypeFlag, InferInstancetypeFlag and InferInstancetypeFromFlag", "infer-instancetype instancetype memory", setFlag(MemoryFlag, "1Gi"), setFlag(InstancetypeFlag, "my-instancetype"), setFlag(InferInstancetypeFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")),
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
			out, err := runCmd(setFlag(InferPreferenceFromFlag, "does-not-exist"))
			Expect(err).To(MatchError("there is no volume with name 'does-not-exist'"))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("PreferenceFlag, InferPreferenceFlag and InferPreferenceFromFlag are mutually exclusive", func(setFlags string, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(fmt.Sprintf("if any flags in the group [preference infer-preference infer-preference-from] are set none of the others can be; [%s] were all set", setFlags)))
			Expect(out).To(BeEmpty())
		},
			Entry("PreferenceFlag and InferPreferenceFlag", "infer-preference preference", setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFlag, "true")),
			Entry("PreferenceFlag and InferPreferenceFromFlag", "infer-preference-from preference", setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFromFlag, "my-vol")),
			Entry("InferPreference and InferPreferenceFromFlag", "infer-preference infer-preference-from", setFlag(InferPreferenceFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")),
			Entry("PreferenceFlag, InferPreferenceFlag and InferPreferenceFromFlag", "infer-preference infer-preference-from preference", setFlag(PreferenceFlag, "my-preference"), setFlag(InferPreferenceFlag, "true"), setFlag(InferPreferenceFromFlag, "my-vol")),
		)

		DescribeTable("Volume to explicitly infer from needs to be valid", func(errMsg string, args ...string) {
			out, err := runCmd(args...)
			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("explicit inference of instancetype with ContainerdiskVolumeFlag", InvalidInferenceVolumeError, setFlag(InferInstancetypeFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")),
			Entry("inference of instancetype from ContainerdiskVolumeFlag", InvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")),
			Entry("explicit inference of preference with ContainerdiskVolumeFlag", InvalidInferenceVolumeError, setFlag(InferPreferenceFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")),
			Entry("inference of preference from ContainerdiskVolumeFlag", InvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")),
			Entry("explicit inference of instancetype with VolumeImportFlag", DVInvalidInferenceVolumeError, setFlag(InferInstancetypeFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")),
			Entry("inference of instancetype from VolumeImportFlag", DVInvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")),
			Entry("explicit inference of preference with VolumeImportFlag", DVInvalidInferenceVolumeError, setFlag(InferPreferenceFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")),
			Entry("inference of preference from VolumeImportFlag", DVInvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")),
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

		DescribeTable("Invalid arguments to ClonePvcVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(ClonePvcVolumeFlag, flag))
			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-import\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-import\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-import\" flag: unknown param(s): test:test"),
			Entry("Missing src", "name:test", "failed to parse \"--volume-import\" flag: src must be specified"),
			Entry("Empty name in src", "src:my-ns/", "failed to parse \"--volume-import\" flag: src invalid: name cannot be empty"),
			Entry("Invalid slashes count in src", "src:my-ns/my-pvc/madethisup", "failed to parse \"--volume-import\" flag: src invalid: invalid count 3 of slashes in prefix/name"),
			Entry("Missing namespace in src", "src:my-pvc", "failed to parse \"--volume-import\" flag: namespace of pvc 'my-pvc' must be specified"),
			Entry("Invalid quantity in size", "size:10Gu", "failed to parse \"--volume-import\" flag: failed to parse param \"size\": unable to parse quantity's suffix"),
			Entry("Invalid number in bootorder", "bootorder:10Gu", "failed to parse \"--volume-import\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"),
			Entry("Negative number in bootorder", "bootorder:-1", "failed to parse \"--volume-import\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("Bootorder set to 0", "src:my-ns/my-pvc,bootorder:0", "failed to parse \"--volume-import\" flag: bootorder must be greater than 0"),
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

		DescribeTable("Invalid arguments to VolumeImportFlag", func(errMsg string, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Missing size with blank volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:blank")),
			Entry("Missing type value", "failed to parse \"--volume-import\" flag: type must be specified", setFlag(VolumeImportFlag, "size:256Mi")),
			Entry("Unknown param for blank volume source", "failed to parse \"--volume-import\" flag: unknown param(s): testparam:", setFlag(VolumeImportFlag, "type:blank,size:256Mi,testparam:")),
			Entry("Missing size with GCS volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:gcs,url:http://url.com")),
			Entry("Missing url with GCS volume source", "failed to parse \"--volume-import\" flag: URL is required with GCS volume source", setFlag(VolumeImportFlag, "type:gcs,size:256Mi")),
			Entry("Missing size with http volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:http,url:http://url.com")),
			Entry("Missing url with http volume source", "failed to parse \"--volume-import\" flag: URL is required with http volume source", setFlag(VolumeImportFlag, "type:http,size:256Mi")),
			Entry("Missing size with imageIO volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:imageio,url:http://imageio.com,diskid:0")),
			Entry("Missing url with imageIO volume source", "failed to parse \"--volume-import\" flag: URL and diskid are both required with imageIO volume source", setFlag(VolumeImportFlag, "type:imageio,diskid:0,size:256Mi")),
			Entry("Missing diskid with imageIO volume source", "failed to parse \"--volume-import\" flag: URL and diskid are both required with imageIO volume source", setFlag(VolumeImportFlag, "type:imageio,url:http://imageio.com,size:256Mi")),
			Entry("Missing src in pvc volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi")),
			Entry("Invalid src without slash in pvc volume source", "failed to parse \"--volume-import\" flag: namespace of pvc 'noslashingvalue' must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:noslashingvalue")),
			Entry("Invalid src in pvc volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:")),
			Entry("Missing src namespace in pvc volume source", "failed to parse \"--volume-import\" flag: namespace of pvc 'my-pvc' must be specified", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:/my-pvc")),
			Entry("Missing src name in pvc volume source", "failed to parse \"--volume-import\" flag: src invalid: name cannot be empty", setFlag(VolumeImportFlag, "type:pvc,size:256Mi,src:default/")),
			Entry("Missing src in snapshot volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi")),
			Entry("Invalid src without slash in snapshot volume source", "failed to parse \"--volume-import\" flag: namespace of snapshot 'noslashingvalue' must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:noslashingvalue")),
			Entry("Invalid src in snapshot volume source", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:")),
			Entry("Missing src namespace in snapshot volume source", "failed to parse \"--volume-import\" flag: namespace of snapshot 'my-snapshot' must be specified", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:/my-snapshot")),
			Entry("Missing src name in snapshot volume source", "failed to parse \"--volume-import\" flag: src invalid: name cannot be empty", setFlag(VolumeImportFlag, "type:snapshot,size:256Mi,src:default/")),
			Entry("Missing size with S3 volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:s3,url:http://url.com")),
			Entry("Missing url in S3 volume source", "failed to parse \"--volume-import\" flag: URL is required with S3 volume source", setFlag(VolumeImportFlag, "type:s3,size:256Mi")),
			Entry("Missing size with registry volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:registry,imagestream:my-image")),
			Entry("Invalid value for pullmethod with registry volume source", "failed to parse \"--volume-import\" flag: pullmethod must be set to pod or node", setFlag(VolumeImportFlag, "type:registry,size:256Mi,pullmethod:invalid,imagestream:my-image")),
			Entry("Both url and imagestream defined in registry volume source", "failed to parse \"--volume-import\" flag: exactly one of url or imagestream must be defined", setFlag(VolumeImportFlag, "type:registry,size:256Mi,pullmethod:node,imagestream:my-image,url:http://url.com")),
			Entry("Missing url and imagestream in registry volume source", "failed to parse \"--volume-import\" flag: exactly one of url or imagestream must be defined", setFlag(VolumeImportFlag, "type:registry,size:256Mi")),
			Entry("Missing size with vddk volume source", "failed to parse \"--volume-import\" flag: size must be specified", setFlag(VolumeImportFlag, "type:vddk,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid")),
			Entry("Missing backingfile with vddk volume source", "failed to parse \"--volume-import\" flag: BackingFile is required with VDDK volume source", setFlag(VolumeImportFlag, "type:vddk,size:256Mi,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid")),
			Entry("Missing secretref with vddk volume source", "failed to parse \"--volume-import\" flag: SecretRef is required with VDDK volume source", setFlag(VolumeImportFlag, "type:vddk,size:256Mi,backingfile:test-backingfile,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid")),
			Entry("Missing thumbprint with vddk volume source", "failed to parse \"--volume-import\" flag: ThumbPrint is required with VDDK volume source", setFlag(VolumeImportFlag, "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,url:http://url.com,uuid:test-uuid")),
			Entry("Missing url with vddk volume source", "failed to parse \"--volume-import\" flag: URL is required with VDDK volume source", setFlag(VolumeImportFlag, "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,uuid:test-uuid")),
			Entry("Missing uuid with vddk volume source", "failed to parse \"--volume-import\" flag: UUID is required with VDDK volume source", setFlag(VolumeImportFlag, "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com")),
			Entry("Missing src in ds volume source ref", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:ds,size:256Mi")),
			Entry("Volume already exists", "failed to parse \"--volume-import\" flag: there is already a volume with name 'duplicated'", setFlag(VolumeImportFlag, "type:blank,size:256Mi,name:duplicated"), setFlag(VolumeImportFlag, "type:blank,size:256Mi,name:duplicated")),
			Entry("Empty name in src", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,name:my-ns/")),
			Entry("Invalid slashes count in src", "failed to parse \"--volume-import\" flag: src must be specified", setFlag(VolumeImportFlag, "type:pvc,name:my-ns/my-pvc/madethisup")),
			Entry("Invalid quantity in size", "failed to parse \"--volume-import\" flag: failed to parse param \"size\": unable to parse quantity's suffix", setFlag(VolumeImportFlag, "type:blank,size:10Gu")),
			Entry("Invalid number in bootorder", "failed to parse \"--volume-import\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax", setFlag(VolumeImportFlag, "type:blank,size:256Mi,bootorder:10Gu")),
			Entry("Negative number in bootorder", "failed to parse \"--volume-import\" flag: failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax", setFlag(VolumeImportFlag, "type:blank,size:256Mi,bootorder:-1")),
			Entry("Bootorder set to 0", "failed to parse \"--volume-import\" flag: bootorder must be greater than 0", setFlag(VolumeImportFlag, "type:blank,size:256Mi,bootorder:0")),
		)

		DescribeTable("Invalid arguments to SysprepVolumeFlag", func(flag, errMsg string) {
			out, err := runCmd(setFlag(SysprepVolumeFlag, flag))
			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", "failed to parse \"--volume-sysprep\" flag: params may not be empty"),
			Entry("Invalid param", "test=test", "failed to parse \"--volume-sysprep\" flag: params need to have at least one colon: test=test"),
			Entry("Unknown param", "test:test", "failed to parse \"--volume-sysprep\" flag: unknown param(s): test:test"),
			Entry("Missing src", "type:configMap", "failed to parse \"--volume-sysprep\" flag: src must be specified"),
			Entry("Empty name in src", "src:my-ns/", "failed to parse \"--volume-sysprep\" flag: src invalid: name cannot be empty"),
			Entry("Invalid slashes count in src", "src:my-ns/my-src/madethisup", "failed to parse \"--volume-sysprep\" flag: src invalid: invalid count 3 of slashes in prefix/name"),
			Entry("Namespace in src", "src:my-ns/my-src", "failed to parse \"--volume-sysprep\" flag: not allowed to specify namespace of ConfigMap or Secret 'my-src'"),
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
			Entry("Duplicate imported PVC", "failed to parse \"--volume-import\" flag: there is already a volume with name 'my-name'",
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
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
			Entry("Duplicate PVC and imported PVC", "failed to parse \"--volume-import\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
			),
			Entry("Duplicate PVC and blank volume", "failed to parse \"--volume-blank\" flag: there is already a volume with name 'my-name'",
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(BlankVolumeFlag, "size:10Gi,name:my-name"),
			),
			Entry("There can only be one cloudInitDisk (UserData)", "there is already a volume with name 'cloudinitdisk'",
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitUserDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))),
			),
			Entry("There can only be one cloudInitDisk (NetworkData)", "there is already a volume with name 'cloudinitdisk'",
				setFlag(DataSourceVolumeFlag, "src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitNetworkDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))),
			),
			Entry("There can only be one sysprepDisk", "failed to parse \"--volume-sysprep\" flag: there is already a volume with name 'sysprepdisk'",
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:sysprepdisk"),
				setFlag(SysprepVolumeFlag, "src:my-src"),
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

		It("Invalid path to PasswordFileFlag", func() {
			out, err := runCmd(setFlag(PasswordFileFlag, "testpath/does/not/exist"))
			Expect(err).To(MatchError("failed to parse \"--password-file\" flag: open testpath/does/not/exist: no such file or directory"))
			Expect(out).To(BeEmpty())
		})

		It("Invalid argument to GAManageSSHFlag", func() {
			out, err := runCmd(setFlag(GAManageSSHFlag, "not-a-bool"))
			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--ga-manage-ssh\" flag: strconv.ParseBool: parsing \"not-a-bool\": invalid syntax"))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("Invalid arguments to CloudInitFlag", func(sourceType string) {
			out, err := runCmd(setFlag(CloudInitFlag, sourceType))
			Expect(err).To(MatchError(fmt.Sprintf("failed to parse \"--cloud-init\" flag: invalid cloud-init data source type \"%s\", supported values are: noCloud, configDrive, none", sourceType)))
			Expect(out).To(BeEmpty())
		},
			Entry("some string", "not-a-bool"),
			Entry("float", "1.23"),
			Entry("bool", "true"),
		)

		DescribeTable("CloudInitUserDataFlag and generated cloud-init config are mutually exclusive", func(flag string, arg string) {
			out, err := runCmd(
				setFlag(CloudInitUserDataFlag, "test"),
				arg,
			)
			Expect(err).To(MatchError(fmt.Sprintf("if any flags in the group [cloud-init-user-data %s] are set none of the others can be; [cloud-init-user-data %s] were all set", flag, flag)))
			Expect(out).To(BeEmpty())
		},
			Entry("CloudInitUserDataFlag and UserFlag", "user", setFlag(UserFlag, "test")),
			Entry("CloudInitUserDataFlag and PasswordFileFlag", "password-file", setFlag(PasswordFileFlag, "testpath")),
			Entry("CloudInitUserDataFlag and SSHKeyFlag", "ssh-key", setFlag(SSHKeyFlag, "test")),
			Entry("CloudInitUserDataFlag and GAManageSSHFlag", "ga-manage-ssh", setFlag(GAManageSSHFlag, "true")),
		)
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(args ...string) ([]byte, error) {
	_args := append([]string{create.CREATE, VM}, args...)
	return clientcmd.NewRepeatableVirtctlCommandWithOut(_args...)()
}

func decodeVM(bytes []byte) (*v1.VirtualMachine, error) {
	decoded, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	if err != nil {
		return nil, err
	}
	switch obj := decoded.(type) {
	case *v1.VirtualMachine:
		Expect(obj.Kind).To(Equal(v1.VirtualMachineGroupVersionKind.Kind))
		Expect(obj.APIVersion).To(Equal(v1.VirtualMachineGroupVersionKind.GroupVersion().String()))
		return obj, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", obj)
	}
}

func noCloudUserData(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData
}

func noCloudUserDataB64(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64
}

func noCloudNetworkDataB64(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkData).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.NetworkDataBase64
}

func configDriveUserData(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.UserDataBase64).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.UserData
}

func configDriveUserDataB64(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.UserData).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.UserDataBase64
}

func configDriveNetworkDataB64(vm *v1.VirtualMachine) string {
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitNoCloud).To(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive).ToNot(BeNil())
	Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.NetworkData).To(BeEmpty())
	return vm.Spec.Template.Spec.Volumes[0].VolumeSource.CloudInitConfigDrive.NetworkDataBase64
}

func randomSingleKey() ([]string, string) {
	key := rand.String(64)
	return []string{
		setFlag(SSHKeyFlag, key),
	}, "\n  - " + key
}

func randomMultipleKeysSingleFlag() ([]string, string) {
	var keys []string
	for range 5 {
		keys = append(keys, rand.String(64))
	}
	return []string{setFlag(SSHKeyFlag, strings.Join(keys, ","))},
		"\n  - " + strings.Join(keys, "\n  - ")
}

func randomMultipleKeysMultipleFlags() ([]string, string) {
	var args []string
	keys := ""
	for range 5 {
		key := rand.String(64)
		args = append(args, setFlag(SSHKeyFlag, key))
		keys += "\n  - " + key
	}
	return args, keys
}
