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

package tests_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/tests"
)

// template parameters
const (
	vmName     = "testvm"
	vmCpuCores = "2"
)

const (
	vmStartPatch = "{\"spec\":{\"running\":true}}"
	vmStopPatch  = "{\"spec\":{\"running\":false}}"
)

var _ = Describe("VM Template", func() {
	flag.Parse()

	vmTemplateFedora := getVmTemplateFedora()

	generateVmJsonFromTemplate := func(vmName string, vmTemplateJsonFile string) string {
		By("Converting VirtualMachine Template into JSON file via oc-process command")
		vmTemplateParams := []string{"-p", "NAME=" + vmName, "-p", "CPU_CORES=" + vmCpuCores}
		args := append([]string{"process", "-f", vmTemplateJsonFile}, vmTemplateParams...)
		out, err := tests.RunOcCommand(args...)
		Expect(err).ToNot(HaveOccurred())
		vmJsonFile, err := tests.WriteJson(vmName, out)
		Expect(err).ToNot(HaveOccurred())
		return vmJsonFile
	}

	createVmFromJson := func(vmName string, vmJsonFile string) {
		var message = ""
		By("Creating VirtualMachine from JSON file via oc-create command")
		out, err := tests.RunOcCommand("create", "-f", vmJsonFile)
		Expect(err).ToNot(HaveOccurred())
		message = fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" created\n", vmName)
		Expect(out).To(Equal(message))

		Eventually(func() bool {
			out, err := tests.RunOcCommand("get", "vms")
			Expect(err).ToNot(HaveOccurred())
			return strings.Contains(out, vmName)
		}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for vm to appear")
	}

	patchVm := func(vmName string, patch string) {
		var message = ""
		By("Starting VirtualMachine via oc-patch command")
		out, err := tests.RunOcCommand("patch", "virtualmachine", vmName, "--type", "merge", "-p", patch)
		Expect(err).ToNot(HaveOccurred())
		message = fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" patched\n", vmName)
		Expect(out).To(Equal(message))
	}

	deleteVm := func(vmName string) {
		var message = ""
		By("Deleting the VirtualMachine via oc-delete command")
		out, err := tests.RunOcCommand("delete", "vm", vmName)
		Expect(err).ToNot(HaveOccurred())
		message = fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" deleted\n", vmName)
		Expect(out).To(Equal(message))

		Eventually(func() bool {
			out, err := tests.RunOcCommand("get", "vms")
			Expect(err).ToNot(HaveOccurred())
			return out == "No resources found.\n"
		}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for vm to disappear")
	}

	Context("with oc command", func() {
		vmTemplateJsonFile := ""
		vmJsonFile := ""

		BeforeEach(func() {
			tests.SkipIfNoOc()
			// write testing VirtualMachine Template into JSON file
			var err error
			vmTemplateJsonFile, err = tests.GenerateVmTemplateJson(vmTemplateFedora)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if vmTemplateJsonFile != "" {
				// remove testing VirtualMachine Template JSON file
				err := os.Remove(vmTemplateJsonFile)
				Expect(err).ToNot(HaveOccurred())
				vmTemplateJsonFile = ""
			}
			if vmJsonFile != "" {
				// remove testing VirtualMachine JSON file
				err := os.Remove(vmJsonFile)
				Expect(err).ToNot(HaveOccurred())
				vmJsonFile = ""
			}
		})

		It("should generate a vm JSON from a template", func() {
			generateVmJsonFromTemplate(vmName, vmTemplateJsonFile)
		})

		It("should create a vm from template", func() {
			vmJsonFile = generateVmJsonFromTemplate(vmName, vmTemplateJsonFile)
			createVmFromJson(vmName, vmJsonFile)
			deleteVm(vmName)
		})

		It("should succeed to start a vmi from template", func() {
			vmJsonFile = generateVmJsonFromTemplate(vmName, vmTemplateJsonFile)
			createVmFromJson(vmName, vmJsonFile)

			patchVm(vmName, vmStartPatch)

			Eventually(func() bool {
				out, err := tests.RunOcCommand("get", "vmis")
				Expect(err).ToNot(HaveOccurred())
				return strings.Contains(out, vmName)
			}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for vmi to appear")

			patchVm(vmName, vmStopPatch)

			Eventually(func() bool {
				out, err := tests.RunOcCommand("get", "vmis")
				Expect(err).ToNot(HaveOccurred())
				return out == "No resources found.\n"
			}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for vmi to disappear")

			deleteVm(vmName)
		})
	})
})

func getBaseVmTemplate(vm *v1.VirtualMachine, memory string, cores string) *tests.Template {

	obj := toUnstructured(vm)
	unstructured.SetNestedField(obj.Object, "${{CPU_CORES}}", "spec", "template", "spec", "domain", "cpu", "cores")
	unstructured.SetNestedField(obj.Object, "${MEMORY}", "spec", "template", "spec", "domain", "resources", "requests", "memory")
	obj.SetName("${NAME}")

	return &tests.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "v1",
		},
		Objects: []runtime.Object{
			obj,
		},
		Parameters: vmTemplateParameters(memory, cores),
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

func vmTemplateParameters(memory string, cores string) []tests.Parameter {
	return []tests.Parameter{
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

func getVmTemplateFedora() *tests.Template {
	gracePeriod := int64(0)
	vmForTemplate := &v1.VirtualMachine{
		Spec: v1.VirtualMachineSpec{
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: &gracePeriod,
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "registrydisk",
									VolumeName: "registryvolume",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name:       "cloudinitdisk",
									VolumeName: "cloudinitvolume",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "registryvolume",
							VolumeSource: v1.VolumeSource{
								RegistryDisk: &v1.RegistryDiskSource{
									Image: "registry:5000/kubevirt/fedora-cloud-registry-disk-demo:latest",
								},
							},
						},
						{
							Name: "cloudinitvolume",
							VolumeSource: v1.VolumeSource{
								CloudInitNoCloud: &v1.CloudInitNoCloudSource{
									UserData: "#cloud-config\npassword: fedora\nchpasswd: { expire: False }",
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt-vm": "vm-${NAME}",
					},
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubevirt.io/v1alpha2",
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"kubevirt-vm": "vm-${NAME}",
			},
			Name: "${NAME}",
		},
	}

	vmTemplate := getBaseVmTemplate(vmForTemplate, "4096Mi", "4")
	vmTemplate.ObjectMeta = metav1.ObjectMeta{
		Name: "vm-template-fedora",
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
	return vmTemplate
}
