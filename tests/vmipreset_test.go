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

package tests_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VMIPreset", decorators.SigCompute, func() {
	const (
		flavorKey    = core.GroupName + "/flavor"
		memoryFlavor = "memory-test"
		memoryPrefix = "test-memory-"
		cpuPrefix    = "test-cpu"
		cpuFlavor    = "cpu-test"
		cores        = 7
	)

	var (
		err          error
		virtClient   kubecli.KubevirtClient
		vmi          *v1.VirtualMachineInstance
		memoryPreset *v1.VirtualMachineInstancePreset
		cpuPreset    *v1.VirtualMachineInstancePreset
		memory       resource.Quantity
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		vmi = libvmifact.NewAlpine(libvmi.WithLabel(flavorKey, memoryFlavor))

		selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: memoryFlavor}}
		memory = resource.MustParse("128M")
		memoryPreset = &v1.VirtualMachineInstancePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: memoryPrefix},
			Spec: v1.VirtualMachineInstancePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
						"memory": memory}},
				},
			},
		}

		selector = k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: cpuFlavor}}
		cpuPreset = &v1.VirtualMachineInstancePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: cpuPrefix},
			Spec: v1.VirtualMachineInstancePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					CPU: &v1.CPU{Cores: uint32(cores)},
				},
			},
		}
	})

	Context("CRD Validation", func() {
		It("[test_id:1595]Should reject POST if schema is invalid", func() {
			// Preset with missing selector should fail CRD validation
			jsonString := fmt.Sprintf("{\"kind\":\"VirtualMachineInstancePreset\",\"apiVersion\":\"%s\",\"metadata\":{\"generateName\":\"test-memory-\",\"creationTimestamp\":null},\"spec\":{}}", v1.StorageGroupVersion.String())

			result := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))
		})

		It("[test_id:1596]should reject POST if validation webhoook deems the spec is invalid", func() {
			preset := &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: "fake"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"fake": "fake"}},
					Domain:   &v1.DomainSpec{},
				},
			}
			// disk with two targets is invalid
			preset.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{},
				},
			})
			result := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(preset).Do(context.Background())
			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &k8smetav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).ToNot(HaveOccurred())

			Expect(reviewResponse.Details.Causes).To(HaveLen(1))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1]"))
		})
	})

	Context("Preset Matching", func() {
		It("[test_id:1597]Should be accepted on POST", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1598]Should reject a second submission of a VMIPreset", func() {
			// This test requires an explicit name or the resources won't conflict
			presetName := "test-preset"
			memoryPreset.Name = presetName
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			b, err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(memoryPreset).DoRaw(context.Background())
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1599]Should return 404 if VMIPreset does not exist", func() {
			b, err := virtClient.RestClient().Get().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Name("wrong").DoRaw(context.Background())
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("[test_id:1600]Should reject presets that conflict with VirtualMachineInstance settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			newPreset, err := getPreset(virtClient, memoryPrefix)
			Expect(err).ToNot(HaveOccurred())

			newVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(newPreset)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(newVMI)

			Expect(newVMI.Labels).To(HaveKeyWithValue(flavorKey, memoryFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels).To(HaveKeyWithValue(flavorKey, memoryFlavor))

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", core.GroupName, newPreset.Name)
			Expect(newVMI.Annotations).ToNot(HaveKey(annotationKey))
		})

		It("[test_id:1601]Should accept presets that don't conflict with VirtualMachineInstance settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(cpuPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			newPreset, err := getPreset(virtClient, cpuPrefix)
			Expect(err).ToNot(HaveOccurred())

			vmi = libvmifact.NewAlpine(libvmi.WithLabel(flavorKey, cpuFlavor))

			newVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(newVMI)

			Expect(newVMI.Labels).To(HaveKeyWithValue(flavorKey, cpuFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels).To(HaveKeyWithValue(flavorKey, cpuFlavor))

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", core.GroupName, newPreset.Name)
			Expect(newVMI.Annotations).To(HaveKeyWithValue(annotationKey, fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))

			// check a setting from the preset itself to show it was applied
			Expect(int(newVMI.Spec.Domain.CPU.Cores)).To(Equal(cores))
		})

		It("[test_id:1602]Should ignore VMIs that don't match", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(vmi)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			newPreset, err := getPreset(virtClient, memoryPrefix)
			Expect(err).ToNot(HaveOccurred())

			// reset the label so it will not match
			vmi = libvmifact.NewAlpine()
			newVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			libwait.WaitForSuccessfulVMIStart(newVMI)

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", core.GroupName, newPreset.Name)
			Expect(newVMI.Annotations).ToNot(HaveKey(annotationKey))
			Expect(newVMI.Status.Phase).ToNot(Equal(v1.Failed))
		})

		It("[test_id:1603]Should not be applied to existing VMIs", func() {
			// create the VirtualMachineInstance first
			newVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(newVMI)

			err = virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(newVMI)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			newPreset, err := getPreset(virtClient, memoryPrefix)
			Expect(err).ToNot(HaveOccurred())

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", core.GroupName, newPreset.Name)
			Expect(newVMI.Annotations).ToNot(HaveKey(annotationKey))
			Expect(newVMI.Status.Phase).ToNot(Equal(v1.Failed))
		})
	})

	Context("Exclusions", func() {
		It("[test_id:1604]Should not apply presets to VirtualMachineInstance's with the exclusion marking", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(cpuPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			newPreset, err := getPreset(virtClient, cpuPrefix)
			Expect(err).ToNot(HaveOccurred())

			exclusionMarking := "virtualmachineinstancepresets.admission.kubevirt.io/exclude"
			vmi = libvmifact.NewAlpine(libvmi.WithLabel(flavorKey, cpuFlavor), libvmi.WithAnnotation(exclusionMarking, "true"))

			newVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(newVMI)

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", core.GroupName, newPreset.Name)
			Expect(newVMI.Annotations).ToNot(HaveKey(annotationKey), "Preset should not have been applied due to exclusion")

			// check a setting from the preset itself to show it was applied
			Expect(newVMI.Spec.Domain.CPU.Cores).NotTo(Equal(newPreset.Spec.Domain.CPU.Cores),
				"CPU should still have been the default value (not defined in spec)")
		})
	})

	Context("Conflict", func() {
		const (
			conflictFlavor = "conflict-test"
			conflictPrefix = "test-conflict-"
			conflictKey    = core.GroupName + "/conflict"
		)

		var (
			conflictPreset *v1.VirtualMachineInstancePreset
			conflictMemory resource.Quantity
		)

		BeforeEach(func() {
			conflictMemory = resource.MustParse("256M")
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{conflictKey: conflictFlavor}}

			conflictPreset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: conflictPrefix},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": conflictMemory}},
					},
				},
			}
		})

		It("[test_id:1605]should denied to start the VMI", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(conflictPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(memoryPreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			vmi.Labels = map[string]string{flavorKey: memoryFlavor, conflictKey: conflictFlavor}
			By("creating the VirtualMachineInstance")
			_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Override", func() {
		var (
			overridePreset *v1.VirtualMachineInstancePreset
		)

		const (
			overrideKey    = "kubevirt.io/vmPreset"
			overrideFlavor = "vmi-preset-small"
			overridePrefix = "test-override-"
		)

		BeforeEach(func() {
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{overrideKey: overrideFlavor}}
			overridePreset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: overridePrefix},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": resource.MustParse("64M")}},
					},
				},
			}
		})

		It("[test_id:644][rfe_id:609] should override presets", func() {
			By("Creating preset with 64M")
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(overridePreset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			By("Creating VMI with 128M")
			vmi = libvmi.New(
				libvmi.WithLabel(overrideKey, overrideFlavor),
				libvmi.WithMemoryRequest("128M"),
			)

			newVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying VMI")
			Expect(newVmi.Annotations).To(Equal(vmi.Annotations))
			Expect(vmi.Labels).To(HaveKeyWithValue(overrideKey, overrideFlavor))

			vmiMemory := resource.MustParse("128M")
			Expect(newVmi.Spec.Domain.Resources.Requests["memory"]).To(Equal(vmiMemory))
		})
	})

	Context("Preset Lifecycle", func() {
		var preset *v1.VirtualMachineInstancePreset
		presetNamePrefix := "vmi-preset-small-"
		selectorKey := "kubevirt.io/vmPreset"
		selectorLabel := "vmi-preset-small"

		BeforeEach(func() {
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{selectorKey: selectorLabel}}
			memory, _ := resource.ParseQuantity("64M")
			preset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: presetNamePrefix},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": memory}},
					},
				},
			}
		})

		It("[test_id:617][rfe_id:609] should create and delete preset", func() {
			By("Creating preset")
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Body(preset).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			By("Checking that preset was created")
			newPreset, err := getPreset(virtClient, presetNamePrefix)
			Expect(err).ToNot(HaveOccurred())

			By("Deleting preset")
			err = virtClient.RestClient().Delete().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Name(newPreset.GetName()).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred())

			By("Checking preset was deleted")
			waitForPresetDeletion(virtClient, newPreset.GetName())
		})
	})

	Context("Match Expressions", func() {
		var (
			preset   *v1.VirtualMachineInstancePreset
			vmiWin7  *v1.VirtualMachineInstance
			vmiWin10 *v1.VirtualMachineInstance
		)

		const (
			labelKey   = "kubevirt.io/os"
			win7Label  = "win7"
			win10Label = "win10"
		)

		BeforeEach(func() {
			selector := k8smetav1.LabelSelector{
				MatchExpressions: []k8smetav1.LabelSelectorRequirement{
					{
						Key:      labelKey,
						Operator: k8smetav1.LabelSelectorOpIn,
						Values:   []string{win10Label, win7Label},
					},
				},
			}

			preset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: memoryPrefix},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": memory}},
					},
				},
			}

			vmiWin7 = libvmi.New(libvmi.WithLabel(labelKey, win7Label))
			vmiWin10 = libvmi.New(libvmi.WithLabel(labelKey, win10Label))
		})

		It("[test_id:726] Should match multiple VMs via MatchExpression", func() {
			By("Creating preset with MatchExpression")
			_, err := virtClient.VirtualMachineInstancePreset(testsuite.GetTestNamespace(nil)).Create(context.Background(), preset, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			By("Creating first VirtualMachineInstance")
			newVmi7, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmiWin7, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating second VirtualMachineInstance")
			newVmi10, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmiWin10, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that preset matched bot VMs")
			Expect(newVmi7.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(newVmi10.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
		})
	})

	Context("[rfe_id:613]MatchLabels", func() {
		var (
			preset   *v1.VirtualMachineInstancePreset
			vmiWin7  *v1.VirtualMachineInstance
			vmiWin10 *v1.VirtualMachineInstance
		)

		const (
			labelKey        = "kubevirt.io/cpu"
			labelValue      = "dodecacore"
			numCores        = uint32(12)
			presetName      = "twelve-cores"
			annotationLabel = "virtualmachinepreset.kubevirt.io/" + presetName
		)

		var annotationVal string

		BeforeEach(func() {
			selector := k8smetav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: labelValue,
				},
			}

			preset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: presetName},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						CPU: &v1.CPU{Cores: numCores},
					},
				},
			}

			// The actual type of machine is unimportant here. This test is about the label
			vmiWin7 = libvmi.New(libvmi.WithLabel(labelKey, labelValue), libvmi.WithMemoryRequest("1Mi"))
			vmiWin10 = libvmi.New(libvmi.WithLabel(labelKey, labelValue), libvmi.WithMemoryRequest("1Mi"))

			annotationVal = v1.GroupVersion.String()
		})

		It("[test_id:672] Should match multiple VMs via MatchLabel", func() {
			By("Creating preset with MatchExpression")
			_, err := virtClient.VirtualMachineInstancePreset(testsuite.GetTestNamespace(nil)).Create(context.Background(), preset, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Give virt-api's cache time to sync before proceeding
			time.Sleep(3 * time.Second)

			By("Creating first VirtualMachineInstance")
			newVmi7, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmiWin7, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating second VirtualMachineInstance")
			newVmi10, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmiWin10, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that preset matched the first VMI")
			Expect(newVmi7.Annotations).To(HaveKeyWithValue(annotationLabel, annotationVal))
			By("Checking that preset matched the second VMI")
			Expect(newVmi10.Annotations).To(HaveKeyWithValue(annotationLabel, annotationVal))

			By("Checking that both VMs have 12 cores")
			Expect(newVmi7.Spec.Domain.CPU.Cores).To(Equal(numCores))
			Expect(newVmi10.Spec.Domain.CPU.Cores).To(Equal(numCores))
		})
	})
})

func getPreset(virtClient kubecli.KubevirtClient, prefix string) (*v1.VirtualMachineInstancePreset, error) {
	presetList := v1.VirtualMachineInstancePresetList{}
	err := virtClient.RestClient().Get().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Do(context.Background()).Into(&presetList)
	Expect(err).ToNot(HaveOccurred())
	for _, thisPreset := range presetList.Items {
		if strings.HasPrefix(thisPreset.Name, prefix) {
			return &thisPreset, nil
		}
	}
	return nil, fmt.Errorf("preset with prefix '%s' not found", prefix)
}

func waitForPresetDeletion(virtClient kubecli.KubevirtClient, presetName string) {
	Eventually(func() error {
		_, err := virtClient.RestClient().Get().Resource("virtualmachineinstancepresets").Namespace(testsuite.GetTestNamespace(nil)).Name(presetName).Do(context.Background()).Get()
		return err
	}).WithTimeout(60*time.Second).WithPolling(1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "timed out waiting for VMI preset to be deleted")
}
