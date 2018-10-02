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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
	vmsgen "kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var _ = Describe("Templates", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var (
		template         *vmsgen.Template
		parameters       templateParams
		templateJsonFile string
		vmJsonFile       string
	)

	BeforeEach(func() {
		tests.SkipIfNoCmd("oc")
		tests.BeforeTestCleanup()
	})

	Describe("Creating VM from Template", func() {

		assertGeneratedVMJson := func() func() {
			return func() {
				By("Generating VM JSON from the Template via oc-process command")
				_, err := runOcProcessCommand(templateJsonFile, parameters, vmJsonFile)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				ExpectWithOffset(1, vmJsonFile).To(BeAnExistingFile())
			}
		}

		assertCreatedVM := func() func() {
			return func() {
				By("Creating VM via oc-create command")
				out, err := runOcCreateCommand(vmJsonFile)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" created\n", parameters.name)
				ExpectWithOffset(1, out).To(ContainSubstring(message))

				By("Checking if the VM exists via oc-get command.")
				EventuallyWithOffset(1, func() bool {
					out, err := runOcGetCommand("vms")
					ExpectWithOffset(1, err).ToNot(HaveOccurred())
					return strings.Contains(out, parameters.name)
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VM to apppear")
			}
		}

		assertDeletedVM := func() func() {
			return func() {
				By("Deleting the VM via oc-delete command")
				out, err := runOcDeleteCommand(parameters.name)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" deleted\n", parameters.name)
				ExpectWithOffset(1, out).To(ContainSubstring(message))

				By("Checking if the VM does not exist anymore via oc-get command.")
				EventuallyWithOffset(1, func() bool {
					out, err := runOcGetCommand("vms")
					ExpectWithOffset(1, err).ToNot(HaveOccurred())
					return out == "No resources found.\n"
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VM to disappear")
			}
		}

		assertStartedVM := func() func() {
			return func() {
				By("Starting VM via oc-patch command")
				out, err := runOcPatchCommand(parameters.name, "{\"spec\":{\"running\":true}}")
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" patched\n", parameters.name)
				ExpectWithOffset(1, out).To(ContainSubstring(message))

				By("Checking if the VMI does exist via oc-get command")
				EventuallyWithOffset(1, func() bool {
					out, err := runOcGetCommand("vmis")
					ExpectWithOffset(1, err).ToNot(HaveOccurred())
					return strings.Contains(out, parameters.name)
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VMI to appear")
			}
		}

		assertStoppedVM := func() func() {
			return func() {
				By("Stopping the VM via oc-patch command")
				out, err := runOcPatchCommand(parameters.name, "{\"spec\":{\"running\":false}}")
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				message := fmt.Sprintf("virtualmachine.kubevirt.io \"%s\" patched\n", parameters.name)
				ExpectWithOffset(1, out).To(ContainSubstring(message))

				By("Checking if the VMI does not exist anymore via oc-get command")
				EventuallyWithOffset(1, func() bool {
					out, err := runOcGetCommand("vmis")
					ExpectWithOffset(1, err).ToNot(HaveOccurred())
					return out == "No resources found.\n"
				}, time.Duration(60)*time.Second).Should(BeTrue(), "Timed out waiting for VMI to disappear")
			}
		}

		assertRemovedFile := func(file string) func() {
			return func() {
				if _, err := os.Stat(file); !os.IsNotExist(err) {
					err := os.Remove(file)
					ExpectWithOffset(1, err).ToNot(HaveOccurred())
				}
				ExpectWithOffset(1, file).NotTo(BeAnExistingFile())
			}
		}

		testGivenTemplate := func() {
			It("should succeed to generate a VM JSON file using oc-process command", assertGeneratedVMJson())

			Context("with the given VM JSON", func() {
				JustBeforeEach(assertGeneratedVMJson())
				AfterEach(assertDeletedVM())

				It("should succeed to create a VM using oc-create command", assertCreatedVM())

				Context("with the given created VM", func() {
					JustBeforeEach(assertCreatedVM())

					It("should succeed to start the VM using oc-patch command", assertStartedVM())

					Context("with the given running VM", func() {
						JustBeforeEach(assertStartedVM())

						It("should succeed to stop the VM using oc-patch command", assertStoppedVM())
					})
				})
			})
		}

		BeforeEach(func() {
			parameters = templateParams{
				name:     "testvm",
				cpuCores: "2",
			}
			vmJsonFile = fmt.Sprintf("%s.json", parameters.name)
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
				template = vmsgen.GetTestTemplateFedora()
			})

			testGivenTemplate()
		})

		Context("with given RHEL Template", func() {
			BeforeEach(func() {
				tests.SkipIfNoRhelImage(virtClient)
				template = vmsgen.GetTestTemplateRHEL7()
			})

			testGivenTemplate()
		})
	})
})

type templateParams struct {
	name     string
	cpuCores string
	memory   string
}

func runOcProcessCommand(templateJsonFile string, parameters templateParams, vmJsonFile string) (string, error) {
	parameterArgs := []string{"-p", "NAME=" + parameters.name, "-p", "CPU_CORES=" + parameters.cpuCores}
	args := append([]string{"process", "-f", templateJsonFile}, parameterArgs...)
	out, err := tests.RunCommand("oc", args...)
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
	return tests.RunCommand("oc", "create", "-f", vmJsonFile)
}

func runOcPatchCommand(vmName string, patch string) (string, error) {
	return tests.RunCommand("oc", "patch", "virtualmachine", vmName, "--type", "merge", "-p", patch)
}

func runOcDeleteCommand(vmName string) (string, error) {
	return tests.RunCommand("oc", "delete", "vm", vmName)
}

func runOcGetCommand(resourceType string) (string, error) {
	return tests.RunCommand("oc", "get", resourceType)
}
