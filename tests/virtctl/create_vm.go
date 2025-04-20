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
 *
 */

package virtctl

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	"golang.org/x/crypto/ssh"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	"kubevirt.io/kubevirt/pkg/virtctl/create/vm"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libssh"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[sig-compute]create vm", decorators.SigCompute, func() {
	const (
		randNameTail         = 5
		size                 = "128Mi"
		importedVolumeRegexp = `imported-volume-\w{5}`
		sysprepDisk          = "sysprepdisk"
		cloudInitDisk        = "cloudinitdisk"
		cloudInitUserData    = `#cloud-config
user: user
password: password
chpasswd: { expire: False }`
	)

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("[test_id:9840]VM with random name and default settings", func() {
		out, err := runCreateVMCmd()
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Name).ToNot(BeEmpty())
		const defaultTerminationGracePeriod = int64(180)
		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(gstruct.PointTo(Equal(defaultTerminationGracePeriod)))
		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(v1.RunStrategyAlways)))
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))
	})

	It("[test_id:11647]Example with volume-import flag and PVC type", func() {
		const (
			runStrategy = v1.RunStrategyAlways
			volName     = "imported-volume"
		)
		instancetype := createInstancetype(virtClient)
		preference := createPreference(virtClient)
		pvc := createAnnotatedSourcePVC(instancetype.Name, preference.Name)

		out, err := runCreateVMCmd(
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:pvc,size:%s,src:%s/%s,name:%s", size, pvc.Namespace, pvc.Name, volName)),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

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

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(volName))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvc.Name))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))

		Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
			Name: volName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: volName,
				},
			},
		}))
	})

	It("[test_id:11648]Example with volume-import flag and Registry type", func() {
		const (
			runStrategy = v1.RunStrategyAlways
			volName     = "registry-source"
		)
		cdSource := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskAlpine)

		out, err := runCreateVMCmd(
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:registry,size:%s,url:%s,name:%s", size, cdSource, volName)),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))

		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(volName))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Registry).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Registry.URL).To(HaveValue(Equal(cdSource)))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(size)))

		Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
			Name: volName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: volName,
				},
			},
		}))
	})

	It("[test_id:11649]Example with volume-import flag and Blank type", func() {
		const runStrategy = v1.RunStrategyAlways

		out, err := runCreateVMCmd(
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.VolumeImportFlag, "type:blank,size:"+size),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))

		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
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
	})

	It("[test_id:9841]Complex example", func() {
		const (
			runStrategy                  = v1.RunStrategyManual
			terminationGracePeriod int64 = 123
			cdSource                     = "my.registry/my-image:my-tag"
			pvcBootOrder                 = 1
			blankSize                    = "10Gi"
			expectedDVTLen               = 3
			expectedVolumesLen           = 6
		)
		vmName := "vm-" + rand.String(randNameTail)
		instancetype := createInstancetype(virtClient)
		preference := createPreference(virtClient)
		dataSource := createAnnotatedDataSource(virtClient, "something", "something")
		pvc := libstorage.CreateFSPVC("vm-pvc-"+rand.String(randNameTail), testsuite.GetTestNamespace(nil), size, nil)
		userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

		out, err := runCreateVMCmd(
			setFlag(vm.NameFlag, vmName),
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
			setFlag(vm.InstancetypeFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularResourceName, instancetype.Name)),
			setFlag(vm.PreferenceFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularPreferenceResourceName, preference.Name)),
			setFlag(vm.ContainerdiskVolumeFlag, "src:"+cdSource),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:ds,src:%s/%s", dataSource.Namespace, dataSource.Name)),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:pvc,src:%s/%s", pvc.Namespace, pvc.Name)),
			setFlag(vm.PvcVolumeFlag, fmt.Sprintf("src:%s,bootorder:%d", pvc.Name, pvcBootOrder)),
			setFlag(vm.VolumeImportFlag, "type:blank,size:"+blankSize),
			setFlag(vm.CloudInitUserDataFlag, userDataB64),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Name).To(Equal(vmName))

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(gstruct.PointTo(Equal(terminationGracePeriod)))

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

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(expectedDVTLen))

		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(gstruct.PointTo(Equal(dataSource.Namespace)))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dataSource.Name))

		Expect(vm.Spec.DataVolumeTemplates[1].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Name).To(Equal(pvc.Name))

		Expect(vm.Spec.DataVolumeTemplates[2].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source.Blank).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(pvc.Name))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim.ClaimName).To(Equal(pvc.Name))

		Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))

		Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal(vm.Spec.DataVolumeTemplates[1].Name))
		Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[1].Name))

		Expect(vm.Spec.Template.Spec.Volumes[4].Name).To(Equal(vm.Spec.DataVolumeTemplates[2].Name))
		Expect(vm.Spec.Template.Spec.Volumes[4].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[4].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[2].Name))

		Expect(vm.Spec.Template.Spec.Volumes[5].Name).To(Equal(cloudInitDisk))
		Expect(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

		decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[5].VolumeSource.CloudInitNoCloud.UserDataBase64)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(decoded)).To(Equal(cloudInitUserData))

		Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
			Name:      pvc.Name,
			BootOrder: pointer.P(uint(pvcBootOrder)),
		}))
	})

	It("[test_id:9842]Complex example with inferred instancetype and preference", func() {
		const (
			runStrategy                  = v1.RunStrategyManual
			terminationGracePeriod int64 = 123
			pvcBootOrder                 = 1
			blankSize                    = "10Gi"
			expectedDVTLen               = 3
			expectedVolumesLen           = 4
		)
		vmName := "vm-" + rand.String(randNameTail)
		instancetype := createInstancetype(virtClient)
		preference := createPreference(virtClient)
		dataSource := createAnnotatedDataSource(virtClient, "something", preference.Name)
		dvtDsName := fmt.Sprintf("%s-ds-%s", vmName, dataSource.Name)
		pvc := createAnnotatedSourcePVC(instancetype.Name, "something")
		userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

		out, err := runCreateVMCmd(
			setFlag(vm.NameFlag, vmName),
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
			setFlag(vm.InferInstancetypeFlag, "true"),
			setFlag(vm.InferPreferenceFromFlag, dvtDsName),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:ds,src:%s/%s,name:%s", dataSource.Namespace, dataSource.Name, dvtDsName)),
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:pvc,src:%s/%s,bootorder:%d", pvc.Namespace, pvc.Name, pvcBootOrder)),
			setFlag(vm.VolumeImportFlag, "type:blank,size:"+blankSize),
			setFlag(vm.CloudInitUserDataFlag, userDataB64),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Name).To(Equal(vmName))

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(gstruct.PointTo(Equal(terminationGracePeriod)))

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

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(expectedDVTLen))

		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(dvtDsName))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Kind).To(Equal("DataSource"))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Namespace).To(gstruct.PointTo(Equal(dataSource.Namespace)))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.SourceRef.Name).To(Equal(dataSource.Name))

		Expect(vm.Spec.DataVolumeTemplates[1].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Source.PVC.Name).To(Equal(pvc.Name))

		Expect(vm.Spec.DataVolumeTemplates[2].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Source.Blank).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[2].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(vm.Spec.DataVolumeTemplates[1].Name))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[1].Name))

		Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(vm.Spec.DataVolumeTemplates[2].Name))
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[2].Name))

		Expect(vm.Spec.Template.Spec.Volumes[3].Name).To(Equal(cloudInitDisk))
		Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

		decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[3].VolumeSource.CloudInitNoCloud.UserDataBase64)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(decoded)).To(Equal(cloudInitUserData))

		Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(ConsistOf(v1.Disk{
			Name:      vm.Spec.DataVolumeTemplates[1].Name,
			BootOrder: pointer.P(uint(pvcBootOrder)),
		}))
	})

	It("[test_id:11650]Complex example with memory", func() {
		const (
			runStrategy                  = v1.RunStrategyManual
			terminationGracePeriod int64 = 123
			memory                       = "4Gi"
			cdSource                     = "my.registry/my-image:my-tag"
			blankSize                    = "10Gi"
			expectedVolumesLen           = 3
		)
		vmName := "vm-" + rand.String(randNameTail)
		preference := createPreference(virtClient)
		userDataB64 := base64.StdEncoding.EncodeToString([]byte(cloudInitUserData))

		out, err := runCreateVMCmd(
			setFlag(vm.NameFlag, vmName),
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
			setFlag(vm.MemoryFlag, memory),
			setFlag(vm.PreferenceFlag, fmt.Sprintf("%s/%s", apiinstancetype.SingularPreferenceResourceName, preference.Name)),
			setFlag(vm.ContainerdiskVolumeFlag, "src:"+cdSource),
			setFlag(vm.VolumeImportFlag, "type:blank,size:"+blankSize),
			setFlag(vm.CloudInitUserDataFlag, userDataB64),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Name).To(Equal(vmName))

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(gstruct.PointTo(Equal(terminationGracePeriod)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse(memory))))

		Expect(vm.Spec.Preference).ToNot(BeNil())
		Expect(vm.Spec.Preference.Kind).To(Equal(apiinstancetype.SingularPreferenceResourceName))
		Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
		Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
		Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))

		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(MatchRegexp(importedVolumeRegexp))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.Blank).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(blankSize)))

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.DataVolume.Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))

		Expect(vm.Spec.Template.Spec.Volumes[2].Name).To(Equal(cloudInitDisk))
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64).To(Equal(userDataB64))

		decoded, err := base64.StdEncoding.DecodeString(vm.Spec.Template.Spec.Volumes[2].VolumeSource.CloudInitNoCloud.UserDataBase64)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(decoded)).To(Equal(cloudInitUserData))
	})

	It("[test_id:11651]Complex example with sysprep volume", func() {
		const (
			runStrategy                  = v1.RunStrategyManual
			terminationGracePeriod int64 = 123
			cdSource                     = "my.registry/my-image:my-tag"
			expectedVolumesLen           = 2
		)

		cm := libconfigmap.New("cm-"+rand.String(randNameTail), map[string]string{"Autounattend.xml": "test"})
		cm, err := virtClient.CoreV1().ConfigMaps(testsuite.GetTestNamespace(nil)).Create(context.Background(), cm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		out, err := runCreateVMCmd(
			setFlag(vm.RunStrategyFlag, string(runStrategy)),
			setFlag(vm.TerminationGracePeriodFlag, fmt.Sprint(terminationGracePeriod)),
			setFlag(vm.ContainerdiskVolumeFlag, "src:"+cdSource),
			setFlag(vm.SysprepVolumeFlag, "src:"+cm.Name),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(runStrategy)))

		Expect(vm.Spec.Template.Spec.TerminationGracePeriodSeconds).To(gstruct.PointTo(Equal(terminationGracePeriod)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))

		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(BeEmpty())

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(sysprepDisk))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Sysprep).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Sysprep.ConfigMap).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Sysprep.ConfigMap.Name).To(Equal(cm.Name))
		Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Sysprep.Secret).To(BeNil())
	})

	It("[test_id:11652]Complex example with generated cloud-init config", func() {
		const (
			user               = "alpine"
			randPassLen        = 12
			expectedVolumesLen = 2
		)
		cdSource := cd.ContainerDiskFor(cd.ContainerDiskAlpineTestTooling)
		tmpDir := GinkgoT().TempDir()
		password := rand.String(randPassLen)

		path := filepath.Join(tmpDir, "pw")
		pwFile, err := os.Create(path)
		Expect(err).ToNot(HaveOccurred())
		_, err = pwFile.WriteString(password)
		Expect(err).ToNot(HaveOccurred())
		Expect(pwFile.Close()).To(Succeed())

		priv, pub, err := libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		keyFile := filepath.Join(tmpDir, "id_rsa")
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
		sshKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))

		out, err := runCreateVMCmd(
			setFlag(vm.ContainerdiskVolumeFlag, "src:"+cdSource),
			setFlag(vm.UserFlag, user),
			setFlag(vm.PasswordFileFlag, path), // This is required to unlock the alpine user
			setFlag(vm.SSHKeyFlag, sshKey),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(v1.RunStrategyAlways)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))

		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(BeEmpty())

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk.Image).To(Equal(cdSource))

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(cloudInitDisk))
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("user: " + user))
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).
			To(ContainSubstring("password: %s\nchpasswd: { expire: False }", password))
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).To(ContainSubstring("ssh_authorized_keys:\n  - " + sshKey))

		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		runSSHCommand(vm.Namespace, vm.Name, user, keyFile)
	})

	It("[test_id:11653]Complex example with access credentials", func() {
		const (
			user               = "fedora"
			expectedVolumesLen = 2
		)
		cdSource := cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)

		priv, pub, err := libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		keyFile := filepath.Join(GinkgoT().TempDir(), "id_rsa")
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
		sshKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))

		secret := libsecret.New("my-keys-"+rand.String(randNameTail), libsecret.DataString{"key1": sshKey})
		secret, err = kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), secret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		out, err := runCreateVMCmd(
			setFlag(vm.ContainerdiskVolumeFlag, "src:"+cdSource),
			setFlag(vm.AccessCredFlag, fmt.Sprintf("type:ssh,src:%s,method:ga,user:%s", secret.Name, user)),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		vm, err = virtClient.VirtualMachine(secret.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.RunStrategy).To(gstruct.PointTo(Equal(v1.RunStrategyAlways)))

		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))

		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(BeEmpty())

		Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(expectedVolumesLen))

		Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(vm.Name + "-containerdisk-0"))
		Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ContainerDisk).ToNot(BeNil())

		Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(cloudInitDisk))
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes[1].CloudInitNoCloud.UserData).
			To(Equal("#cloud-config\nruncmd:\n  - [ setsebool, -P, 'virt_qemu_ga_manage_ssh', 'on' ]"))

		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
		vmi, err := virtClient.VirtualMachineInstance(secret.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

		Eventually(func(g Gomega) {
			vmi, err := virtClient.VirtualMachineInstance(secret.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vmi).To(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAccessCredentialsSynchronized))
		}, 60*time.Second, 1*time.Second).Should(Succeed())

		runSSHCommand(secret.Namespace, vm.Name, user, keyFile)
	})

	It("[test_id:11654]Failure of implicit inference does not fail the VM creation", func() {
		By("Creating a PVC without annotation labels")
		pvc := libstorage.CreateFSPVC("vm-pvc-"+rand.String(randNameTail), testsuite.GetTestNamespace(nil), size, nil)
		volumeName := "imported-volume"

		By("Creating a VM with implicit inference (inference enabled by default)")
		out, err := runCreateVMCmd(
			setFlag(vm.VolumeImportFlag, fmt.Sprintf("type:pvc,size:%s,src:%s/%s,name:%s", size, pvc.Namespace, pvc.Name, volumeName)),
		)
		Expect(err).ToNot(HaveOccurred())
		vm, err := decodeVM(out)
		Expect(err).ToNot(HaveOccurred())

		By("Asserting that implicit inference is enabled")
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))
		Expect(vm.Spec.Instancetype).ToNot(BeNil())
		Expect(vm.Spec.Instancetype.InferFromVolume).To(Equal(volumeName))
		Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(gstruct.PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))
		Expect(vm.Spec.Preference).ToNot(BeNil())
		Expect(vm.Spec.Preference.InferFromVolume).To(Equal(volumeName))
		Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(gstruct.PointTo(Equal(v1.IgnoreInferFromVolumeFailure)))

		By("Performing dry run creation of the VM")
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vm, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		Expect(err).ToNot(HaveOccurred())

		By("Asserting that matchers were cleared and memory was kept")
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(gstruct.PointTo(Equal(resource.MustParse("512Mi"))))
		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Preference).To(BeNil())

		Expect(vm.Spec.DataVolumeTemplates).To(HaveLen(1))
		Expect(vm.Spec.DataVolumeTemplates[0].Name).To(Equal(volumeName))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC).ToNot(BeNil())
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Namespace).To(Equal(pvc.Namespace))
		Expect(vm.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal(pvc.Name))

		Expect(vm.Spec.Template.Spec.Volumes).To(ConsistOf(v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: volumeName,
				},
			},
		}))
	})
}))

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCreateVMCmd(args ...string) ([]byte, error) {
	_args := append([]string{create.CREATE, "vm"}, args...)
	return newRepeatableVirtctlCommandWithOut(_args...)()
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

func createInstancetype(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachineInstancetype {
	instancetype := builder.NewInstancetype(
		builder.WithCPUs(1),
		builder.WithMemory("128Mi"),
	)
	instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(nil)).
		Create(context.Background(), instancetype, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return instancetype
}

func createPreference(virtClient kubecli.KubevirtClient) *instancetypev1beta1.VirtualMachinePreference {
	preference := builder.NewPreference(
		builder.WithPreferredCPUTopology(instancetypev1beta1.Cores),
	)
	preference, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(nil)).
		Create(context.Background(), preference, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return preference
}

func createAnnotatedDataSource(virtClient kubecli.KubevirtClient, instancetypeName, preferenceName string) *cdiv1.DataSource {
	dataSource := &cdiv1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-datasource-",
			Labels: map[string]string{
				apiinstancetype.DefaultInstancetypeLabel:     instancetypeName,
				apiinstancetype.DefaultInstancetypeKindLabel: apiinstancetype.SingularResourceName,
				apiinstancetype.DefaultPreferenceLabel:       preferenceName,
				apiinstancetype.DefaultPreferenceKindLabel:   apiinstancetype.SingularPreferenceResourceName,
			},
		},
		Spec: cdiv1.DataSourceSpec{
			Source: cdiv1.DataSourceSource{},
		},
	}
	dataSource, err := virtClient.CdiClient().CdiV1beta1().DataSources(testsuite.GetTestNamespace(nil)).
		Create(context.Background(), dataSource, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return dataSource
}

func createAnnotatedSourcePVC(instancetypeName, preferenceName string) *k8sv1.PersistentVolumeClaim {
	const randNameTail = 5
	return libstorage.CreateFSPVC("vm-pvc-"+rand.String(randNameTail), testsuite.GetTestNamespace(nil), "128Mi", map[string]string{
		apiinstancetype.DefaultInstancetypeLabel:     instancetypeName,
		apiinstancetype.DefaultInstancetypeKindLabel: apiinstancetype.SingularResourceName,
		apiinstancetype.DefaultPreferenceLabel:       preferenceName,
		apiinstancetype.DefaultPreferenceKindLabel:   apiinstancetype.SingularPreferenceResourceName,
	})
}

func runSSHCommand(namespace, name, user, keyFile string) {
	libssh.DisableSSHAgent()
	// The virtctl binary needs to run here because of the way local SSH client wrapping works.
	// Running the command through newRepeatableVirtctlCommand does not suffice.
	_, cmd, err := clientcmd.CreateCommandWithNS(testsuite.GetTestNamespace(nil), "virtctl",
		"ssh",
		"--namespace", namespace,
		"--username", user,
		"--identity-file", keyFile,
		"--known-hosts=",
		"-t", "-o StrictHostKeyChecking=no",
		"-t", "-o UserKnownHostsFile=/dev/null",
		"--command", "true",
		"vm/"+name,
	)
	Expect(err).ToNot(HaveOccurred())
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	Expect(out).ToNot(BeEmpty())
}
