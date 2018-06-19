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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/validating-webhook"
)

const (
	vmiEphemeral      = "vmi-ephemeral"
	vmiFlavorSmall    = "vmi-flavor-small"
	vmiSata           = "vmi-sata"
	vmiFedora         = "vmi-fedora"
	vmiNoCloud        = "vmi-nocloud"
	vmiPVC            = "vmi-pvc"
	vmiWindows        = "vmi-windows"
	vmTemplateFedora  = "vm-template-fedora"
	vmTemplateRHEL7   = "vm-template-rhel7"
	vmTemplateWindows = "vm-template-windows2012r2"
)

const (
	vmCirros         = "vm-cirros"
	vmAlpineMultiPvc = "vm-alpine-multipvc"
)

const vmiReplicaSetCirros = "vmi-replicaset-cirros"

const vmiPresetSmall = "vmi-preset-small"

const (
	busVirtio = "virtio"
	busSata   = "sata"
)

const (
	imageCirros = "cirros-registry-disk-demo"
	imageFedora = "fedora-cloud-registry-disk-demo"
)

const windowsFirmware = "5d307ca9-b3ef-428c-8861-06e72d69f223"

const apiVersion = "kubevirt.io/v1alpha2"

var dockerPrefix = "registry:5000/kubevirt"
var dockerTag = "devel"

var gracePeriod = int64(0)

func getBaseVMISpec() *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("64M"),
				},
			},
		},
	}
}

func getBaseVMI(name string) *v1.VirtualMachineInstance {
	baseVMISpec := getBaseVMISpec()

	return &v1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachineInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"special": name},
		},
		Spec: *baseVMISpec,
	}
}

func addRegistryDisk(spec *v1.VirtualMachineInstanceSpec, image string, bus string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices = v1.Devices{
		Disks: []v1.Disk{
			{
				Name:       "registrydisk",
				VolumeName: "registryvolume",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: bus,
					},
				},
			},
		},
	}
	spec.Volumes = []v1.Volume{
		{
			Name: "registryvolume",
			VolumeSource: v1.VolumeSource{
				RegistryDisk: &v1.RegistryDiskSource{
					Image: image,
				},
			},
		},
	}
	return spec
}

func addNoCloudDisk(spec *v1.VirtualMachineInstanceSpec) *v1.VirtualMachineInstanceSpec {
	return addNoCloudDiskWitUserData(spec, "#!/bin/sh\n\necho 'printed from cloud-init userdata'\n")
}

func addNoCloudDiskWitUserData(spec *v1.VirtualMachineInstanceSpec, data string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       "cloudinitdisk",
		VolumeName: "cloudinitvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "cloudinitvolume",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserData: data,
			},
		},
	})
	return spec
}

func addEmptyDisk(spec *v1.VirtualMachineInstanceSpec, size string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       "emptydisk",
		VolumeName: "emptydiskvolume",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busVirtio,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: "emptydiskvolume",
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(size),
			},
		},
	})
	return spec
}

func addPVCDisk(spec *v1.VirtualMachineInstanceSpec, claimName string, bus string, diskName string, volumeName string) *v1.VirtualMachineInstanceSpec {
	spec.Domain.Devices.Disks = append(spec.Domain.Devices.Disks, v1.Disk{
		Name:       diskName,
		VolumeName: volumeName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})

	spec.Volumes = append(spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return spec
}

func getVMIEphemeral() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiEphemeral)

	addRegistryDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vmi
}

func getVMISata() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiSata)

	addRegistryDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busSata)
	return vmi
}

func getVMIEphemeralFedora() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiFedora)
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1024M")

	addRegistryDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageFedora, dockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vmi.Spec, "#cloud-config\npassword: fedora\nchpasswd: { expire: False }")
	return vmi
}

func getVMINoCloud() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiNoCloud)

	addRegistryDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	addNoCloudDisk(&vmi.Spec)
	addEmptyDisk(&vmi.Spec, "2Gi")
	return vmi
}

func getVMIFlavorSmall() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiFlavorSmall)
	vmi.ObjectMeta.Labels = map[string]string{
		"kubevirt.io/flavor": "small",
	}

	addRegistryDisk(&vmi.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vmi
}

func getVMIPvc() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiPVC)

	addPVCDisk(&vmi.Spec, "disk-alpine", busVirtio, "pvcdisk", "pvcvolume")
	return vmi
}

func getVMIWindows() *v1.VirtualMachineInstance {
	vmi := getBaseVMI(vmiWindows)

	gracePeriod := int64(0)
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	_false := false
	vmi.Spec = v1.VirtualMachineInstanceSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			CPU: &v1.CPU{Cores: 2},
			Features: &v1.Features{
				ACPI: v1.FeatureState{},
				APIC: &v1.FeatureAPIC{},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:   &v1.FeatureState{},
					VAPIC:     &v1.FeatureState{},
					Spinlocks: &v1.FeatureSpinlocks{Retries: &spinlocks},
				},
			},
			Machine: v1.Machine{Type: "q35"},
			Clock: &v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: &_false},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			},
			Firmware: &v1.Firmware{UUID: firmware},
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
		},
	}

	// pick e1000 network model type for windows machines
	vmi.ObjectMeta.Annotations = map[string]string{v1.InterfaceModel: "e1000"}

	addPVCDisk(&vmi.Spec, "disk-windows", busSata, "pvcdisk", "pvcvolume")
	return vmi
}

func getBaseVM(name string, labels map[string]string) *v1.VirtualMachine {
	baseVMISpec := getBaseVMISpec()

	return &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.VirtualMachineSpec{
			Running: false,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: *baseVMISpec,
			},
		},
	}
}

func getVMCirros() *v1.VirtualMachine {
	vm := getBaseVM(vmCirros, map[string]string{
		"kubevirt.io/vm": vmCirros,
	})

	addRegistryDisk(&vm.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	addNoCloudDisk(&vm.Spec.Template.Spec)
	return vm
}

func getTemplateFedora() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}"})
	addRegistryDisk(&vm.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageFedora, dockerTag), busVirtio)
	addNoCloudDiskWitUserData(&vm.Spec.Template.Spec, "#cloud-config\npassword: fedora\nchpasswd: { expire: False }")

	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: vmTemplateFedora,
		Annotations: map[string]string{
			"description": "OCP KubeVirt Fedora 27 VM template",
			"tags":        "kubevirt,ocp,template,linux,virtualmachine",
			"iconClass":   "icon-fedora",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "fedora27",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	return template
}

func getTemplateRHEL7() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}"})
	addPVCDisk(&vm.Spec.Template.Spec, "linux-vm-pvc-${NAME}", busVirtio, "disk0", "disk0-pvc")

	pvc := getPVCForTemplate("linux-vm-pvc-${NAME}")

	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: vmTemplateRHEL7,
		Annotations: map[string]string{
			"iconClass":   "icon-rhel",
			"description": "OCP KubeVirt Red Hat Enterprise Linux 7.4 VM template",
			"tags":        "kubevirt,ocp,template,linux,virtualmachine",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "rhel-7.4",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	template.Objects = append(template.Objects, pvc)
	return template
}

func getTemplateWindows() *Template {
	vm := getBaseVM("", map[string]string{"kubevirt-vm": "vm-${NAME}"})
	windows := getVMIWindows()
	vm.Spec.Template.Spec = windows.Spec
	vm.Spec.Template.ObjectMeta.Annotations = windows.ObjectMeta.Annotations
	addPVCDisk(&vm.Spec.Template.Spec, "windows-vm-pvc-${NAME}", busVirtio, "disk0", "disk0-pvc")

	pvc := getPVCForTemplate("windows-vm-pvc-${NAME}")

	template := getBaseTemplate(vm, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: vmTemplateWindows,
		Annotations: map[string]string{
			"iconClass":   "icon-windows",
			"description": "OCP KubeVirt Microsoft Windows Server 2012 R2 VM template",
			"tags":        "kubevirt,ocp,template,windows,virtualmachine",
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        "win2k12r2",
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	template.Objects = append(template.Objects, pvc)
	return template
}

func getPVCForTemplate(name string) *k8sv1.PersistentVolumeClaim {

	return &k8sv1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
}

func getBaseTemplate(vm *v1.VirtualMachine, memory string, cores string) *Template {

	obj := toUnstructured(vm)
	unstructured.SetNestedField(obj.Object, "${{CPU_CORES}}", "spec", "template", "spec", "domain", "cpu")
	unstructured.SetNestedField(obj.Object, "${MEMORY}", "spec", "template", "spec", "domain", "resources", "requests", "memory")
	obj.SetName("${NAME}")

	return &Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "v1",
		},
		Objects: []runtime.Object{
			obj,
		},
		Parameters: templateParameters(memory, cores),
	}
}

func toUnstructured(object runtime.Object) *unstructured.Unstructured {
	raw, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	var objmap map[string]interface{}
	err = json.Unmarshal(raw, &objmap)

	return &unstructured.Unstructured{Object: objmap}
}

func templateParameters(memory string, cores string) []Parameter {
	return []Parameter{
		{
			Name:        "NAME",
			Description: "Name for the new VM",
		},
		{
			Name:        "MEMORY",
			Description: "Amount of memory",
			Value:       memory,
		},
		{
			Name:        "CPU_CORES",
			Description: "Amount of cores",
			Value:       cores,
		},
	}
}

func getVMMultiPvc() *v1.VirtualMachine {
	vm := getBaseVM(vmAlpineMultiPvc, map[string]string{
		"kubevirt.io/vm": vmAlpineMultiPvc,
	})

	addPVCDisk(&vm.Spec.Template.Spec, "disk-alpine", busVirtio, "pvcdisk1", "pvcvolume1")
	addPVCDisk(&vm.Spec.Template.Spec, "disk-custom", busVirtio, "pvcdisk2", "pvcvolume2")

	return vm
}

func getBaseVMIReplicaSet(name string, replicas int, selectorLabels map[string]string) *v1.VirtualMachineInstanceReplicaSet {
	baseVMISpec := getBaseVMISpec()
	replicasInt32 := int32(replicas)

	return &v1.VirtualMachineInstanceReplicaSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachineInstanceReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Replicas: &replicasInt32,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: *baseVMISpec,
			},
		},
	}
}

func getVMIReplicaSetCirros() *v1.VirtualMachineInstanceReplicaSet {
	vmReplicaSet := getBaseVMIReplicaSet(vmiReplicaSetCirros, 3, map[string]string{
		"kubevirt.io/vmReplicaSet": vmiReplicaSetCirros,
	})

	addRegistryDisk(&vmReplicaSet.Spec.Template.Spec, fmt.Sprintf("%s/%s:%s", dockerPrefix, imageCirros, dockerTag), busVirtio)
	return vmReplicaSet
}

func getBaseVMIPreset(name string, selectorLabels map[string]string) *v1.VirtualMachineInstancePreset {
	return &v1.VirtualMachineInstancePreset{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "VirtualMachineInstancePreset",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.VirtualMachineInstancePresetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
	}
}

func getVMIPresetSmall() *v1.VirtualMachineInstancePreset {
	vmPreset := getBaseVMIPreset(vmiPresetSmall, map[string]string{
		"kubevirt.io/vmPreset": vmiPresetSmall,
	})

	vmPreset.Spec.Domain = &v1.DomainPresetSpec{
		Resources: v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
		},
	}
	return vmPreset
}

func main() {
	flag.StringVar(&dockerPrefix, "docker-prefix", dockerPrefix, "")
	flag.StringVar(&dockerTag, "docker-tag", dockerTag, "")
	genDir := flag.String("generated-vms-dir", "", "")
	flag.Parse()

	var vms = map[string]*v1.VirtualMachine{
		vmCirros:         getVMCirros(),
		vmAlpineMultiPvc: getVMMultiPvc(),
	}

	var vmis = map[string]*v1.VirtualMachineInstance{
		vmiEphemeral:   getVMIEphemeral(),
		vmiFlavorSmall: getVMIFlavorSmall(),
		vmiSata:        getVMISata(),
		vmiFedora:      getVMIEphemeralFedora(),
		vmiNoCloud:     getVMINoCloud(),
		vmiPVC:         getVMIPvc(),
		vmiWindows:     getVMIWindows(),
	}

	var vmireplicasets = map[string]*v1.VirtualMachineInstanceReplicaSet{
		vmiReplicaSetCirros: getVMIReplicaSetCirros(),
	}

	var vmipresets = map[string]*v1.VirtualMachineInstancePreset{
		vmiPresetSmall: getVMIPresetSmall(),
	}

	var templates = map[string]*Template{
		vmTemplateFedora:  getTemplateFedora(),
		vmTemplateRHEL7:   getTemplateRHEL7(),
		vmTemplateWindows: getTemplateWindows(),
	}

	handleError := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			panic(err)
		}
	}

	handleCauses := func(causes []metav1.StatusCause, name string, objType string) {
		if len(causes) > 0 {
			for _, cause := range causes {
				fmt.Fprintf(
					os.Stderr,
					"Failed to validate %s spec: failed to admit yaml for %s: %s at %s: %s\n",
					objType, name, cause.Type, cause.Field, cause.Message)
			}
			panic(fmt.Errorf("Failed to admit %s of type %s", name, objType))
		}
	}

	dumpObject := func(name string, obj interface{}) error {
		data, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("Failed to generate yaml for %s: %s", name, err)
		}

		err = ioutil.WriteFile(filepath.Join(*genDir, fmt.Sprintf("%s.yaml", name)), data, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write yaml file: %s", err)
		}

		return nil
	}

	// Having no generics is lots of fun
	for name, obj := range vms {
		causes := validating_webhook.ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vm")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmis {
		causes := validating_webhook.ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmireplicasets {
		causes := validating_webhook.ValidateVMIRSSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi replica set")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmipresets {
		causes := validating_webhook.ValidateVMIPresetSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi preset")
		handleError(dumpObject(name, *obj))
	}

	// TODO:(ihar) how to validate templates?
	for name, obj := range templates {
		handleError(dumpObject(name, *obj))
	}
}
