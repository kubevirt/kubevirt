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
 * Copyright The KubeVirt Authors.
 */

//nolint:dupl,gocritic,lll
package vm_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

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
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

const runCmdGAManageSSH = "runcmd:\n  - [ setsebool, -P, 'virt_qemu_ga_manage_ssh', 'on' ]"

var _ = Describe("create vm", func() {
	const (
		cloudInitNoCloud     = "nocloud"
		cloudInitConfigDrive = "configdrive"
		cloudInitNone        = "none"
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
		const (
			importedVolumeRegexp  = `imported-volume-\w{5}`
			sysprepDisk           = "sysprepdisk"
			sysprepConfigMap      = "configMap"
			sysprepSecret         = "secret"
			cloudInitDisk         = "cloudinitdisk"
			cloudInitConfigHeader = "#cloud-config"
		)

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
			Expect(vm.Spec.RunStrategy).To(PointTo(Equal(v1.RunStrategyAlways)))
		})

		It("VM with specified run strategy", func() {
			const runStrategy = v1.RunStrategyManual

			out, err := runCmd(setFlag(RunStrategyFlag, string(runStrategy)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).To(PointTo(Equal(runStrategy)))
		})

		It("Termination grace period defaults to 180", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(PointTo(Equal(int64(180))))
		})

		It("VM with specified termination grace period", func() {
			const terminationGracePeriod int64 = 123

			out, err := runCmd(setFlag(TerminationGracePeriodFlag, strconv.FormatInt(terminationGracePeriod, 10)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(PointTo(Equal(terminationGracePeriod)))
		})

		It("Memory is set to 512Mi by default", func() {
			const defaultMemory = "512Mi"

			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse(defaultMemory))))
		})

		It("VM with specified memory", func() {
			const memory = "1Gi"

			out, err := runCmd(setFlag(MemoryFlag, string(memory)))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse(memory))))
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
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(*inferFromVolumePolicy)))
			}
			if inferFromVolumePolicy != nil && *inferFromVolumePolicy == v1.IgnoreInferFromVolumeFailure {
				Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse("512Mi"))))
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
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-2,bootorder:2"),
				// This DS with bootorder 1 should be used to infer the instancetype, although it is defined second
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-1,bootorder:1"),
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
			Expect(vm.Spec.Instancetype.InferFromVolume).To(MatchRegexp(importedVolumeRegexp))
			if explicit {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			}
			if explicit {
				Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse("512Mi"))))
			}
		},
			Entry("implicit (inference enabled by default)", false),
			Entry("explicit", true),
		)

		It("VM with inferred instancetype from specified volume", func() {
			out, err := runCmd(
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-1,name:my-ds-1"),
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-2,name:my-ds-2"),
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
				setFlag(VolumeImportFlag, "type:ds,src:my-ds"),
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
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:my-ds"))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse(memory))))
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal("my-ds"))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
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
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(*inferFromVolumePolicy)))
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
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-2,bootorder:2"),
				// This DS with bootorder 1 should be used to infer the preference, although it is defined second
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-1,bootorder:1"),
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
			Expect(vm.Spec.Preference.InferFromVolume).To(MatchRegexp(importedVolumeRegexp))
			if explicit {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())
			} else {
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			}
		},
			Entry("implicit (inference enabled by default)", false),
			Entry("explicit", true),
		)

		It("VM with inferred preference from specified volume", func() {
			out, err := runCmd(
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-1,name:my-ds-1"),
				setFlag(VolumeImportFlag, "type:ds,src:my-ds-2,name:my-ds-2"),
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
				setFlag(VolumeImportFlag, "type:ds,src:my-ds"),
				setFlag(InferPreferenceFlag, "false"))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Preference).To(BeNil())
		})

		DescribeTable("VM with specified containerdisk", func(params, volName string, bootOrder int) {
			const cdSource = "my.registry/my-image:my-tag"

			out, err := runCmd(setFlag(ContainerdiskVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			if volName == "" {
				volName = vm.Name + "-containerdisk-0"
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: volName,
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{
						Image: cdSource,
					},
				},
			}))

			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
					Name:      volName,
					BootOrder: pointer.P(uint(bootOrder)),
				}))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
			}

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with src", "src:my.registry/my-image:my-tag", "", 0),
			Entry("with src and name", "src:my.registry/my-image:my-tag,name:my-cd", "my-cd", 0),
			Entry("with src and bootorder", "src:my.registry/my-image:my-tag,bootorder:1", "", 1),
			Entry("with src, name and bootorder", "src:my.registry/my-image:my-tag,name:my-cd,bootorder:2", "my-cd", 2),
		)

		DescribeTable("VM with specified datasource", func(params, dsNamespace, dvtName, dvtSize string, bootOrder int) {
			const dsName = "my-ds"

			out, err := runCmd(setFlag(DataSourceVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			if dvtName == "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtName))
			}
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dsName))
			if dsNamespace == "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(BeNil())
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(PointTo(Equal(dsNamespace)))
			}
			if dvtSize != "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))
			}

			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: vm.Spec.DataVolumeTemplates[0].Name,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: vm.Spec.DataVolumeTemplates[0].Name,
					},
				},
			}))

			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
					Name:      vm.Spec.DataVolumeTemplates[0].Name,
					BootOrder: pointer.P(uint(bootOrder)),
				}))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
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
			Entry("without namespace", "src:my-ds", "", "", "", 0),
			Entry("with namespace", "src:my-ns/my-ds", "my-ns", "", "", 0),
			Entry("without namespace and with name", "src:my-ds,name:my-dvt", "", "my-dvt", "", 0),
			Entry("with namespace and name", "src:my-ns/my-ds,name:my-dvt", "my-ns", "my-dvt", "", 0),
			Entry("without namespace and with size", "src:my-ds,size:10Gi", "", "", "10Gi", 0),
			Entry("with namespace and size", "src:my-ns/my-ds,size:10Gi", "my-ns", "", "10Gi", 0),
			Entry("without namespace and with bootorder", "src:my-ds,bootorder:1", "", "", "", 1),
			Entry("with namespace and bootorder", "src:my-ns/my-ds,bootorder:2", "my-ns", "", "", 2),
			Entry("without namespace and with name and size", "src:my-ds,name:my-dvt,size:10Gi", "", "my-dvt", "10Gi", 0),
			Entry("with namespace, name and size", "src:my-ns/my-ds,name:my-dvt,size:10Gi", "my-ns", "my-dvt", "10Gi", 0),
			Entry("without namespace and with name and bootorder", "src:my-ds,name:my-dvt,bootorder:3", "", "my-dvt", "", 3),
			Entry("with namespace, name and bootorder", "src:my-ns/my-ds,name:my-dvt,bootorder:4", "my-ns", "my-dvt", "", 4),
			Entry("without namespace and with size and bootorder", "src:my-ds,size:10Gi,bootorder:5", "", "", "10Gi", 5),
			Entry("with namespace, size and bootorder", "src:my-ns/my-ds,size:10Gi,bootorder:6", "my-ns", "", "10Gi", 6),
			Entry("without namespace and with name, size and bootorder", "src:my-ds,name:my-dvt,size:10Gi,bootorder:7", "", "my-dvt", "10Gi", 7),
			Entry("with namespace, name, size and bootorder", "src:my-ns/my-ds,name:my-dvt,size:10Gi,bootorder:8", "my-ns", "my-dvt", "10Gi", 8),
		)

		DescribeTable("VM with specified imported volume", func(params, name, size string, bootOrder int, source *cdiv1.DataVolumeSource, sourceRef *cdiv1.DataVolumeSourceRef) {
			out, err := runCmd(setFlag(VolumeImportFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			if source == nil {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(BeNil())
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(Equal(source))
			}
			if sourceRef == nil {
				Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).To(BeNil())
			} else {
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

			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: vm.Spec.DataVolumeTemplates[0].Name,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: vm.Spec.DataVolumeTemplates[0].Name,
					},
				},
			}))

			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
					Name:      vm.Spec.DataVolumeTemplates[0].Name,
					BootOrder: pointer.P(uint(bootOrder)),
				}))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
			}

			if (source != nil && (source.PVC != nil || source.Registry != nil || source.Snapshot != nil)) ||
				(sourceRef != nil && sourceRef.Kind == "DataSource") {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).ToNot(BeNil())
				Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.Template.Spec.Volumes[0].Name))
				Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
				Expect(vm.Spec.Preference).ToNot(BeNil())
				Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.Template.Spec.Volumes[0].Name))
				Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			} else {
				// In this case inference should be possible
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			}
		},
			Entry("with blank source", "type:blank,size:256Mi", "", "256Mi", 0, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with blank source and bootorder", "type:blank,size:256Mi,bootorder:1", "", "256Mi", 1, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with blank source and name", "type:blank,size:256Mi,name:blank-name", "blank-name", "256Mi", 0, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil),
			Entry("with GCS source", "type:gcs,size:256Mi,url:http://url.com,secretref:test-credentials", "", "256Mi", 0, &cdiv1.DataVolumeSource{GCS: &cdiv1.DataVolumeSourceGCS{URL: "http://url.com", SecretRef: "test-credentials"}}, nil),
			Entry("with GCS source and bootorder", "type:gcs,size:256Mi,url:http://url.com,secretref:test-credentials,bootorder:2", "", "256Mi", 2, &cdiv1.DataVolumeSource{GCS: &cdiv1.DataVolumeSourceGCS{URL: "http://url.com", SecretRef: "test-credentials"}}, nil),
			Entry("with http source", "type:http,size:256Mi,url:http://url.com", "", "256Mi", 0, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}, nil),
			Entry("with http source and bootorder", "type:http,size:256Mi,url:http://url.com,bootorder:3", "", "256Mi", 3, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}, nil),
			Entry("with imageio source", "type:imageio,size:256Mi,url:http://url.com,diskid:1,secretref:secret-ref", "", "256Mi", 0, &cdiv1.DataVolumeSource{Imageio: &cdiv1.DataVolumeSourceImageIO{DiskID: "1", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with imageio source and bootorder", "type:imageio,size:256Mi,url:http://url.com,diskid:1,secretref:secret-ref,bootorder:4", "", "256Mi", 4, &cdiv1.DataVolumeSource{Imageio: &cdiv1.DataVolumeSourceImageIO{DiskID: "1", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with PVC source", "type:pvc,size:256Mi,src:default/pvc", "", "256Mi", 0, &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with PVC source and bootorder", "type:pvc,size:256Mi,src:default/pvc,name:imported-volume,bootorder:5", "imported-volume", "256Mi", 5, &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with PVC source without size", "type:pvc,src:default/pvc,name:imported-volume", "imported-volume", "", 0, &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Name: "pvc", Namespace: "default"}}, nil),
			Entry("with registry source", "type:registry,size:256Mi,certconfigmap:my-cert,pullmethod:pod,url:http://url.com,secretref:secret-ref,name:imported-volume", "imported-volume", "256Mi", 0, &cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{CertConfigMap: pointer.P("my-cert"), PullMethod: pointer.P(cdiv1.RegistryPullMethod("pod")), URL: pointer.P("http://url.com"), SecretRef: pointer.P("secret-ref")}}, nil),
			Entry("with registry source and bootorder", "type:registry,size:256Mi,certconfigmap:my-cert,pullmethod:pod,url:http://url.com,secretref:secret-ref,name:imported-volume,bootorder:6", "imported-volume", "256Mi", 6, &cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{CertConfigMap: pointer.P("my-cert"), PullMethod: pointer.P(cdiv1.RegistryPullMethod("pod")), URL: pointer.P("http://url.com"), SecretRef: pointer.P("secret-ref")}}, nil),
			Entry("with S3 source", "type:s3,size:256Mi,url:http://url.com,certconfigmap:my-cert,secretref:secret-ref", "", "256Mi", 0, &cdiv1.DataVolumeSource{S3: &cdiv1.DataVolumeSourceS3{CertConfigMap: "my-cert", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with S3 source and bootorder", "type:s3,size:256Mi,url:http://url.com,certconfigmap:my-cert,secretref:secret-ref,bootorder:7", "", "256Mi", 7, &cdiv1.DataVolumeSource{S3: &cdiv1.DataVolumeSourceS3{CertConfigMap: "my-cert", SecretRef: "secret-ref", URL: "http://url.com"}}, nil),
			Entry("with VDDK source", "type:vddk,size:256Mi,backingfile:backing-file,initimageurl:http://url.com,uuid:123e-11,url:http://url.com,thumbprint:test-thumbprint,secretref:test-credentials", "", "256Mi", 0, &cdiv1.DataVolumeSource{VDDK: &cdiv1.DataVolumeSourceVDDK{BackingFile: "backing-file", InitImageURL: "http://url.com", UUID: "123e-11", URL: "http://url.com", Thumbprint: "test-thumbprint", SecretRef: "test-credentials"}}, nil),
			Entry("with VDDK source and bootorder", "type:vddk,size:256Mi,backingfile:backing-file,initimageurl:http://url.com,uuid:123e-11,url:http://url.com,thumbprint:test-thumbprint,secretref:test-credentials,bootorder:8", "", "256Mi", 8, &cdiv1.DataVolumeSource{VDDK: &cdiv1.DataVolumeSourceVDDK{BackingFile: "backing-file", InitImageURL: "http://url.com", UUID: "123e-11", URL: "http://url.com", Thumbprint: "test-thumbprint", SecretRef: "test-credentials"}}, nil),
			Entry("with Snapshot source", "type:snapshot,size:256Mi,src:default/snapshot,name:imported-volume", "imported-volume", "256Mi", 0, &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with Snapshot source and bootorder", "type:snapshot,size:256Mi,src:default/snapshot,name:imported-volume,bootorder:9", "imported-volume", "256Mi", 9, &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with Snapshot source without size", "type:snapshot,src:default/snapshot,name:imported-volume", "imported-volume", "", 0, &cdiv1.DataVolumeSource{Snapshot: &cdiv1.DataVolumeSourceSnapshot{Name: "snapshot", Namespace: "default"}}, nil),
			Entry("with DataSource source", "type:ds,src:default/datasource,name:imported-ds", "imported-ds", "", 0, nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource", Namespace: pointer.P("default")}),
			Entry("with DataSource source without namespace", "type:ds,src:datasource", "", "", 0, nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource"}),
			Entry("with DataSource source and bootorder", "type:ds,src:default/datasource,name:imported-ds,bootorder:1", "imported-ds", "", 1, nil, &cdiv1.DataVolumeSourceRef{Kind: "DataSource", Name: "datasource", Namespace: pointer.P("default")}),
		)

		DescribeTable("VM with multiple volume-import sources and name", func(params1, params2 string, src1, src2 *cdiv1.DataVolumeSource) {
			const (
				size  = "256Mi"
				name1 = "volume-source1"
				name2 = "volume-source2"
			)

			out, err := runCmd(
				setFlag(VolumeImportFlag, params1),
				setFlag(VolumeImportFlag, params2),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(2))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(name1))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).To(Equal(src1))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))
			Expect(vm.Spec.DataVolumeTemplates[1].Name).To(Equal(name2))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).To(Equal(src2))
			Expect(vm.Spec.DataVolumeTemplates[1].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(name1))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(name2))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with blank source", "type:blank,size:256Mi,name:volume-source1", "type:blank,size:256Mi,name:volume-source2", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}),
			Entry("with blank source and http source", "type:blank,size:256Mi,name:volume-source1", "type:http,size:256Mi,url:http://url.com,name:volume-source2", &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, &cdiv1.DataVolumeSource{HTTP: &cdiv1.DataVolumeSourceHTTP{URL: "http://url.com"}}),
		)

		DescribeTable("VM with specified clone pvc", func(params, dvtName, dvtSize string, bootOrder int) {
			const (
				pvcNamespace = "my-ns"
				pvcName      = "my-pvc"
			)

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

			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: vm.Spec.DataVolumeTemplates[0].Name,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: vm.Spec.DataVolumeTemplates[0].Name,
					},
				},
			}))

			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
					Name:      vm.Spec.DataVolumeTemplates[0].Name,
					BootOrder: pointer.P(uint(bootOrder)),
				}))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
		},
			Entry("with src", "src:my-ns/my-pvc", "", "", 0),
			Entry("with src and name", "src:my-ns/my-pvc,name:my-dvt", "my-dvt", "", 0),
			Entry("with src and size", "src:my-ns/my-pvc,size:10Gi", "", "10Gi", 0),
			Entry("with src and bootorder", "src:my-ns/my-pvc,bootorder:1", "", "", 1),
			Entry("with src, name and size", "src:my-ns/my-pvc,name:my-dvt,size:10Gi", "my-dvt", "10Gi", 0),
			Entry("with src, name and bootorder", "src:my-ns/my-pvc,name:my-dvt,bootorder:2", "my-dvt", "", 2),
			Entry("with src, size and bootorder", "src:my-ns/my-pvc,size:10Gi,bootorder:3", "", "10Gi", 3),
			Entry("with src, name, size and bootorder", "src:my-ns/my-pvc,name:my-dvt,size:10Gi,bootorder:4", "my-dvt", "10Gi", 4),
		)

		DescribeTable("VM with specified pvc", func(params, volName string, bootOrder int) {
			const pvcName = "my-pvc"

			out, err := runCmd(setFlag(PvcVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			if volName == "" {
				volName = pvcName
			}
			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: volName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			}))

			if bootOrder > 0 {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
					Name:      volName,
					BootOrder: pointer.P(uint(bootOrder)),
				}))
			} else {
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(BeEmpty())
			}

			// In this case inference should be possible
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(volName))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(volName))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
		},
			Entry("with src", "src:my-pvc", "", 0),
			Entry("with src and name", "src:my-pvc,name:my-direct-pvc", "my-direct-pvc", 0),
			Entry("with src and bootorder", "src:my-pvc,bootorder:1", "", 1),
			Entry("with src, name and bootorder", "src:my-pvc,name:my-direct-pvc,bootorder:2", "my-direct-pvc", 2),
		)

		DescribeTable("VM with blank disk", func(params, blankName string) {
			const size = "10Gi"

			out, err := runCmd(setFlag(BlankVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			if blankName == "" {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
			} else {
				Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(blankName))
			}
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Blank).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))

			Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
				Name: vm.Spec.DataVolumeTemplates[0].Name,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: vm.Spec.DataVolumeTemplates[0].Name,
					},
				},
			}))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with size", "size:10Gi", ""),
			Entry("with size and name", "size:10Gi,name:my-blank", "my-blank"),
		)

		DescribeTable("VM with specified sysprep volume", func(params, volType string) {
			const src = "my-src"

			out, err := runCmd(setFlag(SysprepVolumeFlag, params))
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(sysprepDisk))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep).ToNot(BeNil())

			switch volType {
			case sysprepConfigMap:
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap.Name).To(Equal(src))
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret).To(BeNil())
			case sysprepSecret:
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.ConfigMap).To(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret).ToNot(BeNil())
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.Sysprep.Secret.Name).To(Equal(src))
			default:
				Fail("invalid sysprep volume type " + volType)

			}

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("ConfigMap with src (implicitly default)", "src:my-src", sysprepConfigMap),
			Entry("ConfigMap with src and type", "src:my-src,type:configMap", sysprepConfigMap),
			Entry("Secret with src and type", "src:my-src,type:secret", sysprepSecret),
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("user: " + user))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("password: %s\nchpasswd: { expire: False }", password))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTable("VM with ssh key in cloud-init user data", func(argsFn func() ([]string, string), userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			args, keys := argsFn()
			args = append(args, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring("ssh_authorized_keys:" + keys))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default) and single key", randomSingleKey, noCloudUserData),
			Entry("with CloudInitNoCloud (explicit) and single key", randomSingleKey, noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and single key", randomSingleKey, configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
			Entry("with CloudInitNoCLoud (default) and multiple keys in single flag", randomMultipleKeysSingleFlag, noCloudUserData),
			Entry("with CloudInitNoCLoud (explicit) and multiple keys in single flag", randomMultipleKeysSingleFlag, noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and multiple keys in single flag", randomMultipleKeysSingleFlag, configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
			Entry("with CloudInitNoCloud (default) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, noCloudUserData),
			Entry("with CloudInitNoCloud (explicit) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit) and multiple keys in multiple flags", randomMultipleKeysMultipleFlags, configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTable("VM with no generated cloud-init config while setting option", func(args ...string) {
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
		},
			Entry("with type CloudInitNone", setFlag(CloudInitFlag, cloudInitNone), setFlag(GAManageSSHFlag, "true")),
			Entry("with default type and GAManageSSHFlag set to false", setFlag(GAManageSSHFlag, "false")),
		)

		DescribeTable("VM with qemu-guest-agent managing SSH enabled in cloud-init user data", func(userDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
			args := append([]string{
				setFlag(GAManageSSHFlag, "true"),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(ContainSubstring(runCmdGAManageSSH))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserData),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, setFlag(CloudInitFlag, cloudInitConfigDrive)),
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(userDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserDataB64, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserDataB64, setFlag(CloudInitFlag, cloudInitConfigDrive)),
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(networkDataFn(vm)).To(Equal(networkDataB64))

			decoded, err := base64.StdEncoding.DecodeString(networkDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudNetworkDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudNetworkDataB64, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveNetworkDataB64, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTable("VM with specified cloud-init user and network data", func(userDataFn, networkDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
			Expect(userDataFn(vm)).To(Equal(userDataB64))
			Expect(networkDataFn(vm)).To(Equal(networkDataB64))

			userData, err := base64.StdEncoding.DecodeString(userDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(userData)).To(Equal(cloudInitUserData))

			networkData, err := base64.StdEncoding.DecodeString(networkDataFn(vm))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(networkData)).To(Equal(cloudInitNetworkData))

			// No inference possible in this case
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
		},
			Entry("with CloudInitNoCloud (default)", noCloudUserDataB64, noCloudNetworkDataB64),
			Entry("with CloudInitNoCloud (explicit)", noCloudUserDataB64, noCloudNetworkDataB64, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserDataB64, configDriveNetworkDataB64, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTable("VM with generated cloud-init user and specified network data", func(userDataFn, networkDataFn func(*v1.VirtualMachine) string, extraArgs ...string) {
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
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cloudInitDisk))
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
			Entry("with CloudInitNoCloud (explicit)", noCloudUserData, noCloudNetworkDataB64, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("with CloudInitConfigDrive (explicit)", configDriveUserData, configDriveNetworkDataB64, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTableSubtree("VM with access credentials with type ssh and method ga and with", func(params string) {
			testFn := func(params string, verifyFn func(*v1.VirtualMachine), extraArgs ...string) {
				args := append([]string{
					setFlag(AccessCredFlag, params),
				}, extraArgs...)
				out, err := runCmd(args...)
				Expect(err).ToNot(HaveOccurred())
				vm, err := decodeVM(out)
				Expect(err).ToNot(HaveOccurred())

				Expect(vm.Spec.Template.Spec.AccessCredentials).To(ConsistOf(
					v1.AccessCredential{
						SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
							Source: v1.SSHPublicKeyAccessCredentialSource{
								Secret: &v1.AccessCredentialSecretSource{
									SecretName: "my-keys",
								},
							},
							PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
								QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
									Users: []string{"myuser"},
								},
							},
						},
					},
				))

				verifyFn(vm)
			}

			DescribeTable("user from param", func(verifyFn func(*v1.VirtualMachine), extraArgs ...string) {
				params += ",user:myuser"
				testFn(params, verifyFn, extraArgs...)
			},
				Entry("nocloud default", verifyNoCloudGAManageSSH),
				Entry("nocloud explicit", verifyNoCloudGAManageSSH, setFlag(CloudInitFlag, cloudInitNoCloud)),
				Entry("configdrive explicit", verifyConfigDriveGAManageSSH, setFlag(CloudInitFlag, cloudInitConfigDrive)),
				Entry("cloud-init none", verifyCloudInitNone, setFlag(CloudInitFlag, cloudInitNone)),
				Entry("GAManageSSH false", verifyCloudInitNone, setFlag(GAManageSSHFlag, "false")),
			)

			DescribeTable("user from flag", func(verifyFn func(*v1.VirtualMachine), extraArgs ...string) {
				extraArgs = append(extraArgs, setFlag(UserFlag, "myuser"))
				testFn(params, verifyFn, extraArgs...)
			},
				Entry("nocloud default", verifyNoCloudGAManageSSHAndUser),
				Entry("nocloud explicit", verifyNoCloudGAManageSSHAndUser, setFlag(CloudInitFlag, cloudInitNoCloud)),
				Entry("configdrive explicit", verifyConfigDriveGAManageSSHAndUser, setFlag(CloudInitFlag, cloudInitConfigDrive)),
				Entry("cloud-init none", verifyCloudInitNone, setFlag(CloudInitFlag, cloudInitNone)),
				Entry("GAManageSSH false", verifyNoCloudUser, setFlag(GAManageSSHFlag, "false")),
			)
		},
			Entry("default type and method", "src:my-keys"),
			Entry("explicit type and default method", "type:ssh,src:my-keys"),
			Entry("explicit type and method", "type:ssh,src:my-keys,method:ga"),
		)

		DescribeTable("VM with access credentials with type ssh and method nocloud and with", func(params string, noCloud bool, extraArgs ...string) {
			args := append([]string{
				setFlag(AccessCredFlag, params),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(ConsistOf(
				v1.AccessCredential{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-keys",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			))

			if noCloud {
				Expect(noCloudUserData(vm)).To(Equal(cloudInitConfigHeader))
			} else {
				Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
			}
		},
			Entry("default type and explicit method, nocloud default", "src:my-keys,method:nocloud", true),
			Entry("default type and explicit method, nocloud explicit", "src:my-keys,method:nocloud", true, setFlag(CloudInitFlag, cloudInitNoCloud)),
			Entry("explicit type and method, nocloud default", "type:ssh,src:my-keys,method:nocloud", true),
			Entry("explicit type and method, nocloud explicit", "type:ssh,src:my-keys,method:nocloud", true, setFlag(CloudInitFlag, cloudInitNoCloud)),
		)

		DescribeTable("VM with access credentials with type ssh and method configdrive and with", func(params string, configDrive bool, extraArgs ...string) {
			args := append([]string{
				setFlag(AccessCredFlag, params),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(ConsistOf(
				v1.AccessCredential{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-keys",
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
						},
					},
				},
			))

			if configDrive {
				Expect(configDriveUserData(vm)).To(Equal(cloudInitConfigHeader))
			} else {
				Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
			}
		},
			Entry("default type and explicit method, configdrive default", "src:my-keys,method:configdrive", true),
			Entry("default type and explicit method, configdrive explicit", "src:my-keys,method:configdrive", true, setFlag(CloudInitFlag, cloudInitConfigDrive)),
			Entry("explicit type and method, configdrive default", "type:ssh,src:my-keys,method:configdrive", true),
			Entry("explicit type and method, configdrive explicit", "type:ssh,src:my-keys,method:configdrive", true, setFlag(CloudInitFlag, cloudInitConfigDrive)),
		)

		DescribeTable("VM with access credentials with type password and with", func(params string) {
			out, err := runCmd(
				setFlag(AccessCredFlag, params),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(ConsistOf(
				v1.AccessCredential{
					UserPassword: &v1.UserPasswordAccessCredential{
						Source: v1.UserPasswordAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: "my-pws",
							},
						},
						PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
						},
					},
				},
			))

			Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
		},
			Entry("explicit type and default method ga", "type:password,src:my-pws"),
			Entry("explicit type and method", "type:password,src:my-pws,method:ga"),
		)

		It("Complex example", func() {
			const (
				vmName                       = "my-vm"
				runStrategy                  = v1.RunStrategyManual
				terminationGracePeriod int64 = 123
				instancetypeKind             = "virtualmachineinstancetype"
				instancetypeName             = "my-instancetype"
				dsNamespace                  = "my-ns"
				dsName                       = "my-ds"
				dvtSize                      = "10Gi"
				pvcName                      = "my-pvc"
				pvcBootOrder                 = 1
				secretName                   = "my-secret"
			)
			userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

			out, err := runCmd(
				setFlag(NameFlag, vmName),
				setFlag(RunStrategyFlag, string(runStrategy)),
				setFlag(TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
				setFlag(InstancetypeFlag, fmt.Sprintf("%s/%s", instancetypeKind, instancetypeName)),
				setFlag(InferPreferenceFromFlag, pvcName),
				setFlag(VolumeImportFlag, fmt.Sprintf("type:ds,src:%s/%s,size:%s", dsNamespace, dsName, dvtSize)),
				setFlag(PvcVolumeFlag, fmt.Sprintf("src:%s,bootorder:%d", pvcName, pvcBootOrder)),
				setFlag(SysprepVolumeFlag, fmt.Sprintf("src:%s,type:%s", secretName, sysprepSecret)),
				setFlag(CloudInitUserDataFlag, userDataB64),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.Running).To(BeNil())
			Expect(vm.Spec.RunStrategy).To(PointTo(Equal(runStrategy)))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(PointTo(Equal(terminationGracePeriod)))

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

			Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(PointTo(Equal(dsNamespace)))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dsName))
			Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(dvtSize)))

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(4))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvcName))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(sysprepDisk))
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.ConfigMap).To(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.Secret).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.Sysprep.Secret.Name).To(Equal(secretName))
			Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal(cloudInitDisk))
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

			decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(decoded)).To(Equal(cloudInitUserData))

			Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
				Name:      pvcName,
				BootOrder: pointer.P(uint(pvcBootOrder)),
			}))
		})

		It("Complex example with generated cloud-init config", func() {
			const (
				vmName                       = "my-vm"
				terminationGracePeriod int64 = 180
				pvcNamespace                 = "my-ns"
				pvcName                      = "my-ds"
				dvtSize                      = "10Gi"
				user                         = "my-user"
				sshKey                       = "my-ssh-key"
			)

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
			Expect(vm.Spec.RunStrategy).To(PointTo(Equal(v1.RunStrategyAlways)))

			Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(PointTo(Equal(terminationGracePeriod)))

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
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(cloudInitDisk))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("user: " + user))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("ssh_authorized_keys:\n  - " + sshKey))

			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Kind).To(BeEmpty())
			Expect(vm.Spec.Instancetype.Name).To(BeEmpty())
			Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse("512Mi"))))

			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Kind).To(BeEmpty())
			Expect(vm.Spec.Preference.Name).To(BeEmpty())
			Expect(vm.Spec.Preference.InferFromVolume).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
			Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
		})

		It("Complex example with access credentials", func() {
			const (
				vmName     = "my-vm"
				cdSource   = "src:my.registry/my-image:my-tag"
				user       = "my-user"
				secretName = "my-keys"
			)

			out, err := runCmd(
				setFlag(NameFlag, vmName),
				setFlag(ContainerdiskVolumeFlag, "src:"+cdSource),
				setFlag(UserFlag, user),
				setFlag(AccessCredFlag, "type:ssh,src:"+secretName),
			)
			Expect(err).ToNot(HaveOccurred())
			vm, err := decodeVM(out)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Name).To(Equal(vmName))

			Expect(vm.Spec.DataVolumeTemplates).To(BeEmpty())

			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(cloudInitDisk))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("user: " + user))
			Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring(runCmdGAManageSSH))

			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(PointTo(Equal(resource.MustParse("512Mi"))))

			Expect(vm.Spec.Preference).To(BeNil())

			Expect(vm.Spec.Template.Spec.AccessCredentials).To(ConsistOf(
				v1.AccessCredential{
					SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
						Source: v1.SSHPublicKeyAccessCredentialSource{
							Secret: &v1.AccessCredentialSecretSource{
								SecretName: secretName,
							},
						},
						PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
							QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
								Users: []string{user},
							},
						},
					},
				},
			))
		})
	})

	Describe("Manifest is not created successfully", func() {
		const (
			paramsEmptyError       = "params may not be empty"
			paramsInvalidError     = "params need to have at least one colon: test=test"
			paramsUnknownError     = "unknown param(s): test:test"
			emptyNameError         = "name cannot be empty"
			invalidSlashCountError = "invalid count 2 of slashes in prefix/name"

			boolInvalidError          = "strconv.ParseBool: parsing \"not-a-bool\": invalid syntax"
			bootOrderInvalidError     = "failed to parse param \"bootorder\": strconv.ParseUint: parsing \"10Gu\": invalid syntax"
			bootOrderNegativeError    = "failed to parse param \"bootorder\": strconv.ParseUint: parsing \"-1\": invalid syntax"
			bootOrderZeroError        = "bootorder must be greater than 0"
			sizeInvalidError          = "failed to parse param \"size\": unable to parse quantity's suffix"
			sizeMissingError          = "size must be specified"
			srcEmptyNameError         = "src invalid: name cannot be empty"
			srcInvalidSlashCountError = "src invalid: invalid count 2 of slashes in prefix/name"
			srcMissingError           = "src must be specified"

			nameDotsError          = "invalid name \"name.with.dot\": must not contain dots"
			nameTooLongError       = "invalid name \"somanycharactersthatthedisksnameislooooongerthantheallowedlength\": must be no more than 63 characters"
			dns1123LabelError      = "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"
			nameUpperCaseError     = "invalid name \"NOTALLOWED\": " + dns1123LabelError
			nameDashBeginningError = "invalid name \"-notallowed\": " + dns1123LabelError

			inferenceNoVolumeError = "at least one volume is needed to infer an instance type or preference"
			noVolumeError          = "there is no volume with name \"does-not-exist\""

			pvcMissingNamespaceError      = "namespace of pvc \"my-pvc\" must be specified"
			snapshotMissingNamespaceError = "namespace of snapshot \"my-snapshot\" must be specified"

			volumeImportDuplicateError  = "failed to parse \"--volume-import\" flag: there is already a volume with name \"my-name\""
			volumePvcDuplicateError     = "failed to parse \"--volume-pvc\" flag: there is already a volume with name \"my-name\""
			cloudInitDiskDuplicateError = "there is already a volume with name \"cloudinitdisk\""

			userMissingError    = "user must be specified with access credential ssh method ga (\"--user\" flag or param \"user\")"
			userNotAllowedError = "user cannot be specified with selected access credential type and method"

			invalidInferenceVolumeError   = "inference of instancetype or preference works only with datasources, datavolumes or pvcs"
			dvInvalidInferenceVolumeError = "this datavolume is not valid to infer an instancetype or preference from (source needs to be PVC, Registry or Snapshot, sourceRef needs to be DataSource)"
		)

		DescribeTable("Invalid parameter to NameFlag when a volume name is derived from it", func(param, errMsg string) {
			out, err := runCmd(
				setFlag(NameFlag, param),
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag"),
			)
			Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to parse \"--volume-containerdisk\" flag: invalid name \"%s-containerdisk-0\": %s", param, errMsg))))
			Expect(out).To(BeEmpty())
		},
			Entry("invalid character (dot)", "name.with.dot", "must not contain dots"),
			Entry("derived name will have more than 63 characters", "manycharacterssothatthedisknamewillbetoolongerthantheallowedlength", "must be no more than 63 characters"),
			Entry("upper case", "NOTALLOWED", dns1123LabelError),
			Entry("dash at the beginning", "-notallowed", dns1123LabelError),
		)

		DescribeTable("Invalid parameter to RunStrategyFlag", func(param string) {
			out, err := runCmd(setFlag(RunStrategyFlag, param))
			Expect(err).To(MatchError(fmt.Sprintf("failed to parse \"--run-strategy\" flag: invalid run strategy \"%s\", supported values are: Always, Manual, Halted, Once, RerunOnFailure", param)))
			Expect(out).To(BeEmpty())
		},
			Entry("some string", "not-a-bool"),
			Entry("float", "1.23"),
			Entry("bool", "true"),
		)

		DescribeTable("Invalid parameter to TerminationGracePeriodFlag", func(param string) {
			out, err := runCmd(setFlag(TerminationGracePeriodFlag, param))
			Expect(err).To(MatchError(fmt.Sprintf("invalid argument \"%s\" for \"--termination-grace-period\" flag: strconv.ParseInt: parsing \"%s\": invalid syntax", param, param)))
			Expect(out).To(BeEmpty())
		},
			Entry("string", "not-a-number"),
			Entry("float", "1.23"),
		)

		DescribeTable("Invalid parameter to MemoryFlag", func(param, errMsg string) {
			out, err := runCmd(setFlag(MemoryFlag, param))
			Expect(err).To(MatchError("failed to parse \"--memory\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid number", "abc", "quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"),
			Entry("Invalid suffix", "512Gu", "unable to parse quantity's suffix"),
		)

		DescribeTable("Invalid parameter to InstancetypeFlag", func(param, errMsg string) {
			out, err := runCmd(setFlag(InstancetypeFlag, param))
			Expect(err).To(MatchError("failed to parse \"--instancetype\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid kind", "madethisup/my-instancetype", "invalid instancetype kind \"madethisup\", supported values are: virtualmachineinstancetype, virtualmachineclusterinstancetype"),
			Entry("Invalid argument count", "virtualmachineinstancetype/my-instancetype/madethisup", invalidSlashCountError),
			Entry("Empty name", "virtualmachineinstancetype/", emptyNameError),
		)

		It("Invalid parameter to InferInstancetypeFlag", func() {
			out, err := runCmd(setFlag(InferInstancetypeFlag, "not-a-bool"))
			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--infer-instancetype\" flag: " + boolInvalidError))
			Expect(out).To(BeEmpty())
		})

		It("InferInstancetypeFlag needs at least one volume", func() {
			out, err := runCmd(setFlag(InferInstancetypeFlag, "true"))
			Expect(err).To(MatchError(inferenceNoVolumeError))
			Expect(out).To(BeEmpty())
		})

		It("Volume specified in InferInstancetypeFromFlag should exist", func() {
			out, err := runCmd(setFlag(InferInstancetypeFromFlag, "does-not-exist"))
			Expect(err).To(MatchError(noVolumeError))
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

		DescribeTable("Invalid parameter to PreferenceFlag", func(param, errMsg string) {
			out, err := runCmd(setFlag(PreferenceFlag, param))
			Expect(err).To(MatchError("failed to parse \"--preference\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Invalid kind", "madethisup/my-preference", "invalid preference kind \"madethisup\", supported values are: virtualmachinepreference, virtualmachineclusterpreference"),
			Entry("Invalid argument count", "virtualmachinepreference/my-preference/madethisup", invalidSlashCountError),
			Entry("Empty name", "virtualmachinepreference/", emptyNameError),
		)

		It("Invalid argument to InferPreferenceFlag", func() {
			out, err := runCmd(setFlag(InferPreferenceFlag, "not-a-bool"))
			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--infer-preference\" flag: " + boolInvalidError))
			Expect(out).To(BeEmpty())
		})

		It("InferPreferenceFlag needs at least one volume", func() {
			out, err := runCmd(setFlag(InferPreferenceFlag, "true"))
			Expect(err).To(MatchError(inferenceNoVolumeError))
			Expect(out).To(BeEmpty())
		})

		It("Volume specified in InferPreferenceFromFlag should exist", func() {
			out, err := runCmd(setFlag(InferPreferenceFromFlag, "does-not-exist"))
			Expect(err).To(MatchError(noVolumeError))
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
			Entry("explicit inference of instancetype with ContainerdiskVolumeFlag", invalidInferenceVolumeError, setFlag(InferInstancetypeFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")),
			Entry("inference of instancetype from ContainerdiskVolumeFlag", invalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")),
			Entry("explicit inference of preference with ContainerdiskVolumeFlag", invalidInferenceVolumeError, setFlag(InferPreferenceFlag, "true"), setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag")),
			Entry("inference of preference from ContainerdiskVolumeFlag", invalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(ContainerdiskVolumeFlag, "name:my-vol,src:my.registry/my-image:my-tag")),
			Entry("explicit inference of instancetype with VolumeImportFlag", dvInvalidInferenceVolumeError, setFlag(InferInstancetypeFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")),
			Entry("inference of instancetype from VolumeImportFlag", dvInvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")),
			Entry("explicit inference of preference with VolumeImportFlag", dvInvalidInferenceVolumeError, setFlag(InferPreferenceFlag, "true"), setFlag(VolumeImportFlag, "type:http,size:256Mi,url:http://url.com")),
			Entry("inference of preference from VolumeImportFlag", dvInvalidInferenceVolumeError, setFlag(InferInstancetypeFromFlag, "my-vol"), setFlag(VolumeImportFlag, "name:my-vol,type:http,size:256Mi,url:http://url.com")),
		)

		DescribeTable("Invalid parameters to ContainerdiskVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(ContainerdiskVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-containerdisk\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "name:test", srcMissingError),
			Entry("Invalid number in bootorder", "bootorder:10Gu", bootOrderInvalidError),
			Entry("Negative number in bootorder", "bootorder:-1", bootOrderNegativeError),
			Entry("Bootorder set to 0", "src:my.registry/my-image:my-tag,bootorder:0", bootOrderZeroError),
			Entry("invalid character (dot)", "src:my.registry/my-image:my-tag,name:name.with.dot", nameDotsError),
			Entry("name has more than 63 characters", "src:my.registry/my-image:my-tag,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("upper case", "src:my.registry/my-image:my-tag,name:NOTALLOWED", nameUpperCaseError),
			Entry("dash at the beginning", "src:my.registry/my-image:my-tag,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to DataSourceVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(DataSourceVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-import\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "name:test", srcMissingError),
			Entry("Empty name in src", "src:my-ns/", srcEmptyNameError),
			Entry("Invalid slashes count in src", "src:my-ns/my-ds/madethisup", srcInvalidSlashCountError),
			Entry("Invalid quantity in size", "size:10Gu", sizeInvalidError),
			Entry("Invalid number in bootorder", "bootorder:10Gu", bootOrderInvalidError),
			Entry("Negative number in bootorder", "bootorder:-1", bootOrderNegativeError),
			Entry("Bootorder set to 0", "src:my-ds,bootorder:0", bootOrderZeroError),
			Entry("invalid character (dot)", "src:my-ds,name:name.with.dot", nameDotsError),
			Entry("name has more than 63 characters", "src:my-ds,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("upper case", "src:my-ds,name:NOTALLOWED", nameUpperCaseError),
			Entry("dash at the beginning", "src:my-ds,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to ClonePvcVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(ClonePvcVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-import\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "name:test", srcMissingError),
			Entry("Empty name in src", "src:my-ns/", srcEmptyNameError),
			Entry("Invalid slashes count in src", "src:my-ns/my-pvc/madethisup", srcInvalidSlashCountError),
			Entry("Missing namespace in src", "src:my-pvc", pvcMissingNamespaceError),
			Entry("Invalid quantity in size", "size:10Gu", sizeInvalidError),
			Entry("Invalid number in bootorder", "bootorder:10Gu", bootOrderInvalidError),
			Entry("Negative number in bootorder", "bootorder:-1", bootOrderNegativeError),
			Entry("Bootorder set to 0", "src:my-ns/my-pvc,bootorder:0", bootOrderZeroError),
			Entry("invalid character (dot)", "src:my-ns/my-pvc,name:name.with.dot", nameDotsError),
			Entry("name has more than 63 characters", "src:my-ns/my-pvc,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("upper case", "src:my-ns/my-pvc,name:NOTALLOWED", nameUpperCaseError),
			Entry("dash at the beginning", "src:my-ns/my-pvc,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to PvcVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(PvcVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-pvc\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "name:test", srcMissingError),
			Entry("Empty name in src", "src:my-ns/", srcEmptyNameError),
			Entry("Invalid slashes count in src", "src:my-ns/my-pvc/madethisup", srcInvalidSlashCountError),
			Entry("Namespace in src", "src:my-ns/my-pvc", "not allowed to specify namespace of pvc \"my-pvc\""),
			Entry("Invalid number in bootorder", "bootorder:10Gu", bootOrderInvalidError),
			Entry("Negative number in bootorder", "bootorder:-1", bootOrderNegativeError),
			Entry("Bootorder set to 0", "src:my-pvc,bootorder:0", bootOrderZeroError),
			Entry("invalid character (dot)", "src:my-pvc,name:name.with.dot", nameDotsError),
			Entry("name has more than 63 characters", "src:my-pvc,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("upper case", "src:my-pvc,name:NOTALLOWED", nameUpperCaseError),
			Entry("dash at the beginning", "src:my-pvc,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to BlankVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(BlankVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-import\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing size", "name:my-blank", sizeMissingError),
			Entry("invalid character (dot)", "size:256Mi,name:name.with.dot", nameDotsError),
			Entry("name has more than 63 characters", "size:256Mi,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("upper case", "size:256Mi,name:NOTALLOWED", nameUpperCaseError),
			Entry("dash at the beginning", "size:256Mi,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to VolumeImportFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(VolumeImportFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-import\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Missing size with blank volume source", "type:blank", sizeMissingError),
			Entry("Missing type value", "size:256Mi", "type must be specified"),
			Entry("Invalid type value", "type:madeup", "invalid volume import type \"madeup\", see help for supported values"),
			Entry("Unknown param for blank volume source", "type:blank,size:256Mi,test:test", paramsUnknownError),
			Entry("Missing size with GCS volume source", "type:gcs,url:http://url.com", sizeMissingError),
			Entry("Missing url with GCS volume source", "type:gcs,size:256Mi", "url is required with gcs volume source"),
			Entry("Missing size with http volume source", "type:http,url:http://url.com", sizeMissingError),
			Entry("Missing url with http volume source", "type:http,size:256Mi", "url is required with http volume source"),
			Entry("Missing size with imageIO volume source", "type:imageio,url:http://imageio.com,diskid:0", sizeMissingError),
			Entry("Missing url with imageIO volume source", "type:imageio,diskid:0,size:256Mi", "url and diskid are both required with imageio volume source"),
			Entry("Missing diskid with imageIO volume source", "type:imageio,url:http://imageio.com,size:256Mi", "url and diskid are both required with imageio volume source"),
			Entry("Missing src in pvc volume source", "type:pvc,size:256Mi", srcMissingError),
			Entry("Invalid src without slash in pvc volume source", "type:pvc,size:256Mi,src:my-pvc", pvcMissingNamespaceError),
			Entry("Invalid src in pvc volume source", "type:pvc,size:256Mi,src:", srcMissingError),
			Entry("Missing src namespace in pvc volume source", "type:pvc,size:256Mi,src:/my-pvc", pvcMissingNamespaceError),
			Entry("Missing src name in pvc volume source", "type:pvc,size:256Mi,src:default/", srcEmptyNameError),
			Entry("Missing src in snapshot volume source", "type:snapshot,size:256Mi", srcMissingError),
			Entry("Invalid src without slash in snapshot volume source", "type:snapshot,size:256Mi,src:my-snapshot", snapshotMissingNamespaceError),
			Entry("Invalid src in snapshot volume source", "type:snapshot,size:256Mi,src:", srcMissingError),
			Entry("Missing src namespace in snapshot volume source", "type:snapshot,size:256Mi,src:/my-snapshot", snapshotMissingNamespaceError),
			Entry("Missing src name in snapshot volume source", "type:snapshot,size:256Mi,src:default/", srcEmptyNameError),
			Entry("Missing size with S3 volume source", "type:s3,url:http://url.com", sizeMissingError),
			Entry("Missing url in S3 volume source", "type:s3,size:256Mi", "url is required with s3 volume source"),
			Entry("Missing size with registry volume source", "type:registry,imagestream:my-image", sizeMissingError),
			Entry("Invalid value for pullmethod with registry volume source", "type:registry,size:256Mi,pullmethod:invalid,imagestream:my-image", "pullmethod must be set to pod or node"),
			Entry("Both url and imagestream defined in registry volume source", "type:registry,size:256Mi,pullmethod:node,imagestream:my-image,url:http://url.com", "exactly one of url or imagestream must be defined"),
			Entry("Missing url and imagestream in registry volume source", "type:registry,size:256Mi", "exactly one of url or imagestream must be defined"),
			Entry("Missing size with vddk volume source", "type:vddk,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid", sizeMissingError),
			Entry("Missing backingfile with vddk volume source", "type:vddk,size:256Mi,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid", "backingfile is required with vddk volume source"),
			Entry("Missing secretref with vddk volume source", "type:vddk,size:256Mi,backingfile:test-backingfile,thumbprint:test-thumb,url:http://url.com,uuid:test-uuid", "secretref is required with vddk volume source"),
			Entry("Missing thumbprint with vddk volume source", "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,url:http://url.com,uuid:test-uuid", "thumbprint is required with vddk volume source"),
			Entry("Missing url with vddk volume source", "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,uuid:test-uuid", "url is required with vddk volume source"),
			Entry("Missing uuid with vddk volume source", "type:vddk,size:256Mi,backingfile:test-backingfile,secretref:test-credentials,thumbprint:test-thumb,url:http://url.com", "uuid is required with vddk volume source"),
			Entry("Missing src in ds volume source ref", "type:ds,size:256Mi", srcMissingError),
			Entry("Empty name in src", "type:pvc,name:my-ns/", srcMissingError),
			Entry("Invalid slashes count in src", "type:pvc,name:my-ns/my-pvc/madethisup", srcMissingError),
			Entry("Invalid quantity in size", "type:blank,size:10Gu", sizeInvalidError),
			Entry("Invalid number in bootorder", "type:blank,size:256Mi,bootorder:10Gu", bootOrderInvalidError),
			Entry("Negative number in bootorder", "type:blank,size:256Mi,bootorder:-1", bootOrderNegativeError),
			Entry("Bootorder set to 0", "type:blank,size:256Mi,bootorder:0", bootOrderZeroError),
			Entry("Name has invalid character (dot)", "type:blank,size:256Mi,name:name.with.dot", nameDotsError),
			Entry("Name has more than 63 characters", "type:blank,size:256Mi,name:somanycharactersthatthedisksnameislooooongerthantheallowedlength", nameTooLongError),
			Entry("Name has upper case character", "type:blank,size:256Mi,name:NOTALLOWED", nameUpperCaseError),
			Entry("Name has dash at the beginning", "type:blank,size:256Mi,name:-notallowed", nameDashBeginningError),
		)

		DescribeTable("Invalid parameters to SysprepVolumeFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(SysprepVolumeFlag, params))
			Expect(err).To(MatchError("failed to parse \"--volume-sysprep\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "type:configMap", srcMissingError),
			Entry("Invalid type", "type:madeup,src:my-src", "invalid sysprep source type \"madeup\", supported values are: configmap, secret"),
			Entry("Empty name in src", "src:my-ns/", srcEmptyNameError),
			Entry("Invalid slashes count in src", "src:my-ns/my-src/madethisup", srcInvalidSlashCountError),
			Entry("Namespace in src", "src:my-ns/my-src", "not allowed to specify namespace of configmap or secret \"my-src\""),
		)

		DescribeTable("Duplicate DataVolumeTemplates or Volumes are not allowed", func(errMsg string, flags ...string) {
			out, err := runCmd(flags...)
			Expect(err).To(MatchError(errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Duplicate Containerdisk", "failed to parse \"--volume-containerdisk\" flag: there is already a volume with name \"my-name\"",
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
			),
			Entry("Duplicate DataSource", volumeImportDuplicateError,
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:my-name"),
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:my-name"),
			),
			Entry("Duplicate imported PVC", volumeImportDuplicateError,
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
			),
			Entry("Duplicate PVC", volumePvcDuplicateError,
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
			),
			Entry("Duplicate blank volume", volumeImportDuplicateError,
				setFlag(VolumeImportFlag, "type:blank,size:10Gi,name:my-name"),
				setFlag(VolumeImportFlag, "type:blank,size:10Gi,name:my-name"),
			),
			Entry("Duplicate PVC and Containerdisk", volumePvcDuplicateError,
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,name:my-name"),
			),
			Entry("Duplicate PVC and DataSource", volumeImportDuplicateError,
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:my-name"),
			),
			Entry("Duplicate PVC and imported PVC", volumeImportDuplicateError,
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:my-name"),
			),
			Entry("Duplicate PVC and blank volume", volumeImportDuplicateError,
				setFlag(PvcVolumeFlag, "src:my-pvc,name:my-name"),
				setFlag(VolumeImportFlag, "type:blank,size:10Gi,name:my-name"),
			),
			Entry("There can only be one cloudInitDisk (UserData)", cloudInitDiskDuplicateError,
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitUserDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))),
			),
			Entry("There can only be one cloudInitDisk (NetworkData)", cloudInitDiskDuplicateError,
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,name:cloudinitdisk"),
				setFlag(CloudInitNetworkDataFlag, base64.StdEncoding.EncodeToString([]byte(cloudInitNetworkData))),
			),
			Entry("There can only be one sysprepDisk", "failed to parse \"--volume-sysprep\" flag: there is already a volume with name \"sysprepdisk\"",
				setFlag(VolumeImportFlag, "type:pvc,src:my-ns/my-pvc,name:sysprepdisk"),
				setFlag(SysprepVolumeFlag, "src:my-src"),
			),
		)

		It("Duplicate boot orders are not allowed", func() {
			out, err := runCmd(
				setFlag(ContainerdiskVolumeFlag, "src:my.registry/my-image:my-tag,bootorder:1"),
				setFlag(VolumeImportFlag, "type:ds,src:my-ds,bootorder:1"),
			)
			Expect(err).To(MatchError("failed to parse \"--volume-import\" flag: bootorder 1 was specified multiple times"))
			Expect(out).To(BeEmpty())
		})

		It("Invalid path to PasswordFileFlag", func() {
			out, err := runCmd(setFlag(PasswordFileFlag, "testpath/does/not/exist"))
			Expect(err).To(MatchError("failed to parse \"--password-file\" flag: open testpath/does/not/exist: no such file or directory"))
			Expect(out).To(BeEmpty())
		})

		It("Invalid parameter to GAManageSSHFlag", func() {
			out, err := runCmd(setFlag(GAManageSSHFlag, "not-a-bool"))
			Expect(err).To(MatchError("invalid argument \"not-a-bool\" for \"--ga-manage-ssh\" flag: " + boolInvalidError))
			Expect(out).To(BeEmpty())
		})

		DescribeTable("Invalid parameter to CloudInitFlag", func(param string) {
			out, err := runCmd(setFlag(CloudInitFlag, param))
			Expect(err).To(MatchError(fmt.Sprintf("failed to parse \"--cloud-init\" flag: invalid cloud-init data source type \"%s\", supported values are: nocloud, configdrive, none", param)))
			Expect(out).To(BeEmpty())
		},
			Entry("some string", "not-a-bool"),
			Entry("float", "1.23"),
			Entry("bool", "true"),
		)

		DescribeTable("CloudInitUserDataFlag and generated cloud-init config are mutually exclusive", func(flag, arg string) {
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

		DescribeTable("Invalid parameters to AccessCredFlag", func(params, errMsg string) {
			out, err := runCmd(setFlag(AccessCredFlag, params))
			Expect(err).To(MatchError("failed to parse \"--access-cred\" flag: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("Empty params", "", paramsEmptyError),
			Entry("Invalid param", "test=test", paramsInvalidError),
			Entry("Unknown param", "test:test", paramsUnknownError),
			Entry("Missing src", "type:ssh", srcMissingError),
			Entry("Empty name in src", "src:my-ns/", srcEmptyNameError),
			Entry("Invalid slashes count in src", "src:my-ns/my-src/madethisup", srcInvalidSlashCountError),
			Entry("Namespace in src", "src:my-ns/my-src", "not allowed to specify namespace of secret \"my-src\""),
			Entry("Invalid type", "type:madeup,src:my-src", "invalid access credential type \"madeup\", supported values are: ssh, password"),
			Entry("Invalid method with type ssh", "type:ssh,src:my-src,method:madeup", "invalid access credentials ssh method \"madeup\", supported values are: ga, nocloud, configdrive"),
			Entry("No user with type ssh and method ga (default)", "type:ssh,src:my-src", userMissingError),
			Entry("No user with type ssh and method ga (explicit)", "type:ssh,src:my-src,method:ga", userMissingError),
			Entry("User with type ssh and method nocloud", "type:ssh,src:my-src,method:nocloud,user:myuser", userNotAllowedError),
			Entry("User with type ssh and method configdrive", "type:ssh,src:my-src,method:configdrive,user:myuser", userNotAllowedError),
			Entry("Invalid method with type password", "type:password,src:my-src,method:madeup", "invalid access credentials password method \"madeup\", supported values are: ga"),
			Entry("User with type password and method ga (default)", "type:password,src:my-src,user:myuser", userNotAllowedError),
			Entry("User with type password and method ga (explicit)", "type:password,src:my-src,method:ga,user:myuser", userNotAllowedError),
		)

		DescribeTable("Cloud-init source type mismatch with AccessCredFlag", func(params, cloudInit, errMsg string) {
			out, err := runCmd(
				setFlag(AccessCredFlag, params),
				setFlag(CloudInitFlag, cloudInit),
			)
			Expect(err).To(MatchError("failed to parse \"--access-cred\" flag: method param and value passed to --cloud-init have to match: " + errMsg))
			Expect(out).To(BeEmpty())
		},
			Entry("type ssh (default) and nocloud vs configdrive", "src:my-src,method:nocloud", cloudInitConfigDrive, "nocloud vs configdrive"),
			Entry("type ssh (default) and nocloud vs none", "src:my-src,method:nocloud", cloudInitNone, "nocloud vs none"),
			Entry("type ssh (default) and configdrive vs nocloud", "src:my-src,method:configdrive", cloudInitNoCloud, "configdrive vs nocloud"),
			Entry("type ssh (default) and configdrive vs none", "src:my-src,method:configdrive", cloudInitNone, "configdrive vs none"),
			Entry("type ssh (explicit) and nocloud vs configdrive", "type:ssh,src:my-src,method:nocloud", cloudInitConfigDrive, "nocloud vs configdrive"),
			Entry("type ssh (explicit) and nocloud vs none", "type:ssh,src:my-src,method:nocloud", cloudInitNone, "nocloud vs none"),
			Entry("type ssh (explicit) and configdrive vs nocloud", "type:ssh,src:my-src,method:configdrive", cloudInitNoCloud, "configdrive vs nocloud"),
			Entry("type ssh (explicit) and configdrive vs none", "type:ssh,src:my-src,method:configdrive", cloudInitNone, "configdrive vs none"),
		)
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{create.CREATE, "vm"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
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

func verifyNoCloudGAManageSSH(vm *v1.VirtualMachine) {
	Expect(noCloudUserData(vm)).To(ContainSubstring(runCmdGAManageSSH))
}

func verifyNoCloudUser(vm *v1.VirtualMachine) {
	Expect(noCloudUserData(vm)).To(ContainSubstring("user: myuser"))
}

func verifyNoCloudGAManageSSHAndUser(vm *v1.VirtualMachine) {
	userData := noCloudUserData(vm)
	Expect(userData).To(ContainSubstring(runCmdGAManageSSH))
	Expect(userData).To(ContainSubstring("user: myuser"))
}

func verifyConfigDriveGAManageSSH(vm *v1.VirtualMachine) {
	Expect(configDriveUserData(vm)).To(ContainSubstring(runCmdGAManageSSH))
}

func verifyConfigDriveGAManageSSHAndUser(vm *v1.VirtualMachine) {
	userData := configDriveUserData(vm)
	Expect(userData).To(ContainSubstring(runCmdGAManageSSH))
	Expect(userData).To(ContainSubstring("user: myuser"))
}

func verifyCloudInitNone(vm *v1.VirtualMachine) {
	Expect(vm.Spec.Template.Spec.Volumes).To(BeEmpty())
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
