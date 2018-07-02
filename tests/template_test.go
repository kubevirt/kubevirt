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
	"io/ioutil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Templates", func() {
	flag.Parse()

	_, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var (
		template         *tests.Template
		templateParams   TemplateParams
		templateJsonFile string
		vmJsonFile       string
	)

	BeforeEach(func() {
		tests.SkipIfNoOc()
		tests.BeforeTestCleanup()
	})

	Describe("Launching VMI from VM Template", func() {

		assertGeneratedVMJson := func() func() {
			return func() {
				By("Generating VM JSON from the Template via oc-process command")
				_, err := runOcProcessCommand(templateJsonFile, templateParams, vmJsonFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(vmJsonFile).To(BeAnExistingFile())
			}
		}

		assertCreatedVM := func() func() {
			return func() {
				By("Creating VM via oc-create command")
				out, err := runOcCreateCommand(vmJsonFile)
				Expect(err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" created\n", templateParams.Name)
				Expect(out).To(Equal(message))

				By("Checking if the VM exists via oc-get command.")
				Eventually(func() bool {
					out, err := runOcGetCommand("vms")
					Expect(err).ToNot(HaveOccurred())
					return strings.Contains(out, templateParams.Name)
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VM to apppear")
			}
		}

		assertDeletedVM := func() func() {
			return func() {
				By("Deleting the VM via oc-delete command")
				out, err := runOcDeleteCommand(templateParams.Name)
				Expect(err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" deleted\n", templateParams.Name)
				Expect(out).To(Equal(message))

				By("Checking if the VM does not exist anymore via oc-get command.")
				Eventually(func() bool {
					out, err := runOcGetCommand("vms")
					Expect(err).ToNot(HaveOccurred())
					return out == "No resources found.\n"
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VM to disappear")
			}
		}

		assertLaunchedVMI := func() func() {
			return func() {
				By("Launching VMI via oc-patch command")
				out, err := runOcPatchCommand(templateParams.Name, "{\"spec\":{\"running\":true}}")
				Expect(err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" patched\n", templateParams.Name)
				Expect(out).To(Equal(message))

				By("Checking if the VMI does exist via oc-get command")
				Eventually(func() bool {
					out, err := runOcGetCommand("vmis")
					Expect(err).ToNot(HaveOccurred())
					return strings.Contains(out, templateParams.Name)
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VMI to appear")
			}
		}

		assertTerminatedVMI := func() func() {
			return func() {
				By("Terminating the VMI via oc-patch command")
				out, err := runOcPatchCommand(templateParams.Name, "{\"spec\":{\"running\":false}}")
				Expect(err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" patched\n", templateParams.Name)
				Expect(out).To(Equal(message))

				By("Checking if the VMI does not exist anymore via oc-get command")
				Eventually(func() bool {
					out, err := runOcGetCommand("vmis")
					Expect(err).ToNot(HaveOccurred())
					return out == "No resources found.\n"
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VMI to disappear")
			}
		}

		assertRemovedFile := func(file string) func() {
			return func() {
				if _, err := os.Stat(file); !os.IsNotExist(err) {
					err := os.Remove(file)
					Expect(err).ToNot(HaveOccurred())
				}
				Expect(file).NotTo(BeAnExistingFile())
			}
		}

		testGivenTemplate := func() {
			It("should succeed to generate a VM JSON file using oc-process command", assertGeneratedVMJson())

			Context("with given VM JSON from the Template", func() {
				JustBeforeEach(assertGeneratedVMJson())
				AfterEach(assertDeletedVM())

				It("should succeed to create a VM using oc-create command", assertCreatedVM())

				Context("with given VM from the VM JSON", func() {
					JustBeforeEach(assertCreatedVM())

					It("should succeed to launch a VMI using oc-patch command", assertLaunchedVMI())

					Context("with given VMI from the VM", func() {
						JustBeforeEach(assertLaunchedVMI())

						It("should succeed to terminate the VMI using oc-patch command", assertTerminatedVMI())
					})
				})
			})
		}

		BeforeEach(func() {
			templateParams = TemplateParams{
				Name:     "testvm",
				CpuCores: "2",
			}
			vmJsonFile = fmt.Sprintf("%s.json", templateParams.Name)
			Expect(vmJsonFile).NotTo(BeAnExistingFile())
		})

		JustBeforeEach(func() {
			var err error
			templateJsonFile, err = tests.GenerateTemplateJson(template)
			Expect(err).ToNot(HaveOccurred())
			Expect(templateJsonFile).To(BeAnExistingFile())
		})

		AfterEach(func() {
			assertRemovedFile(vmJsonFile)()
			assertRemovedFile(templateJsonFile)()
		})

		Context("with given Fedora Template", func() {
			BeforeEach(func() {
				template = newTemplate(TemplateMeta{
					Name:        "vm-template-fedora",
					Description: "OCP KubeVirt Fedora 27 VM template",
					Label:       "fedora27",
					IconClass:   "icon-fedora",
					Image:       "registry:5000/kubevirt/fedora-cloud-registry-disk-demo:latest",
					UserData:    "#cloud-config\npassword: fedora\nchpasswd: { expire: False }",
				})
			})

			testGivenTemplate()
		})
	})
})

type TemplateMeta struct {
	Name        string
	Description string
	Label       string
	IconClass   string
	Image       string
	UserData    string
}

type TemplateParams struct {
	Name     string
	CpuCores string
}

func runOcProcessCommand(templateJsonFile string, templateParams TemplateParams, vmJsonFile string) (string, error) {
	templateParamArgs := []string{"-p", "NAME=" + templateParams.Name, "-p", "CPU_CORES=" + templateParams.CpuCores}
	args := append([]string{"process", "-f", templateJsonFile}, templateParamArgs...)
	out, err := tests.RunOcCommand(args...)
	if err != nil {
		return out, err
	}
	err = ioutil.WriteFile(vmJsonFile, []byte(out), 0644)
	if err != nil {
		return out, fmt.Errorf("failed to write json file %s", vmJsonFile)
	}
	return out, err
}

func runOcCreateCommand(vmJsonFile string) (string, error) {
	out, err := tests.RunOcCommand("create", "-f", vmJsonFile)
	return out, err
}

func runOcPatchCommand(vmName string, patch string) (string, error) {
	out, err := tests.RunOcCommand("patch", "virtualmachine", vmName, "--type", "merge", "-p", patch)
	return out, err
}

func runOcDeleteCommand(vmName string) (string, error) {
	out, err := tests.RunOcCommand("delete", "vm", vmName)
	return out, err
}

func runOcGetCommand(resourceType string) (string, error) {
	out, err := tests.RunOcCommand("get", resourceType)
	return out, err
}

func newBaseTemplate(vm *v1.VirtualMachine, memory string, cores string) *tests.Template {
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

func templateParameters(memory string, cores string) []tests.Parameter {
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

func newTemplate(templateMeta TemplateMeta) *tests.Template {
	gracePeriod := int64(0)
	vmSpec := &v1.VirtualMachine{
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
									Image: templateMeta.Image,
								},
							},
						},
						{
							Name: "cloudinitvolume",
							VolumeSource: v1.VolumeSource{
								CloudInitNoCloud: &v1.CloudInitNoCloudSource{
									UserData: templateMeta.UserData,
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

	template := newBaseTemplate(vmSpec, "4096Mi", "4")
	template.ObjectMeta = metav1.ObjectMeta{
		Name: templateMeta.Name,
		Annotations: map[string]string{
			"description": templateMeta.Description,
			"tags":        "kubevirt,ocp,template,linux,virtualmachine",
			"iconClass":   templateMeta.IconClass,
		},
		Labels: map[string]string{
			"kubevirt.io/os":                        templateMeta.Label,
			"miq.github.io/kubevirt-is-vm-template": "true",
		},
	}
	return template
}
