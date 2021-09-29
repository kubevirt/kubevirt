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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	vmsgen "kubevirt.io/kubevirt/tools/vms-generator/utils"
)

const (
	defaultNamePrefix = "testvm-"
	defaultCPUCores   = "2"
	defaultMemory     = "2Gi"
)

var _ = Describe("[Serial][sig-compute]Templates", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	var (
		templateParams map[string]string
		workDir        string
		templateFile   string
		vmName         string
	)

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.SkipIfNoCmd("oc")
		tests.BeforeTestCleanup()
		SetDefaultEventuallyTimeout(120 * time.Second)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		workDir, err = ioutil.TempDir("", tests.TempDirPrefix+"-")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if workDir != "" {
			err := os.RemoveAll(workDir)
			Expect(err).ToNot(HaveOccurred())
			workDir = ""
		}
	})

	Describe("Creating VM from Template", func() {

		AssertTestSetupSuccess := func() func() {
			return func() {
				templateParams = map[string]string{
					"NAME":      defaultNamePrefix + rand.String(12),
					"CPU_CORES": defaultCPUCores,
					"MEMORY":    defaultMemory,
				}
				templateFile = ""
				ExpectWithOffset(1, templateParams).To(HaveKeyWithValue("NAME", Not(BeEmpty())), "invalid NAME parameter: VirtualMachine name cannot be empty string")
				ExpectWithOffset(1, templateParams).To(HaveKeyWithValue("CPU_CORES", MatchRegexp(`^[0-9]+$`)), "invalid CPU_CORES parameter: %q is not unsigned integer", templateParams["CPU_CORES"])
				ExpectWithOffset(1, templateParams).To(HaveKeyWithValue("MEMORY", MatchRegexp(`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`)), "invalid MEMORY parameter: %q is not valid quantity", templateParams["MEMORY"])
				vmName = templateParams["NAME"]
				vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
				ExpectWithOffset(1, errors.IsNotFound(err) || vm.ObjectMeta.DeletionTimestamp != nil).To(BeTrue(), "invalid NAME parameter: VirtualMachine %q already exists", vmName)
			}
		}

		AssertTemplateSetupSuccess := func(template *vmsgen.Template, params map[string]string) func() {
			return func() {
				ExpectWithOffset(1, template).NotTo(BeNil(), "template object was not provided")
				By("Creating the Template JSON file")
				var err error
				templateFile, err = tests.GenerateTemplateJson(template, workDir)
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to write template JSON file: %v", err)
				ExpectWithOffset(1, templateFile).To(BeAnExistingFile(), "template JSON file %q was not created", templateFile)

				if params != nil {
					By("Validating template parameters")
					for param, value := range params {
						switch param {
						case "NAME":
							ExpectWithOffset(1, value).NotTo(BeEmpty(), "invalid NAME parameter: VirtualMachine name cannot be empty string")
							vmName = value
							vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
							ExpectWithOffset(1, errors.IsNotFound(err) || vm.ObjectMeta.DeletionTimestamp != nil).To(BeTrue(), "invalid NAME parameter: VirtualMachine %q already exists", vmName)
						case "CPU_CORES":
							ExpectWithOffset(1, templateParams).To(HaveKeyWithValue("CPU_CORES", MatchRegexp(`^[0-9]+$`)), "invalid CPU_CORES parameter: %q is not unsigned integer", templateParams["CPU_CORES"])
						case "MEMORY":
							ExpectWithOffset(1, templateParams).To(HaveKeyWithValue("MEMORY", MatchRegexp(`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`)), "invalid MEMORY parameter: %q is not valid quantity", templateParams["MEMORY"])
						}
						templateParams[param] = value
					}
				}
			}
		}

		AssertTestCleanupSuccess := func() func() {
			return func() {
				if vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{}); err == nil && vm.ObjectMeta.DeletionTimestamp == nil {
					By("Deleting the VirtualMachine")
					ExpectWithOffset(1, virtClient.VirtualMachine(util.NamespaceTestDefault).Delete(vmName, &metav1.DeleteOptions{})).To(Succeed(), "failed to delete VirtualMachine %q: %v", vmName, err)
					EventuallyWithOffset(1, func() bool {
						obj, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
						return errors.IsNotFound(err) || obj.ObjectMeta.DeletionTimestamp != nil
					}).Should(BeTrue(), "VirtualMachine %q still exists and the deletion timestamp was not set", vmName)
				}
				if templateFile != "" {
					if _, err := os.Stat(templateFile); !os.IsNotExist(err) {
						By("Deleting template JSON file")
						ExpectWithOffset(1, os.RemoveAll(filepath.Dir(templateFile))).To(Succeed(), "failed to remove template JSON file %q: %v", templateFile, err)
						ExpectWithOffset(1, templateFile).NotTo(BeAnExistingFile(), "template JSON file %q was not removed", templateFile)
					}
				}
			}
		}

		AssertVMCreationSuccess := func() func() {
			return func() {
				By("Creating VirtualMachine from Template via oc command")
				ocProcessCommand := []string{"oc", "process", "-f", templateFile}
				for param, value := range templateParams {
					ocProcessCommand = append(ocProcessCommand, "-p", fmt.Sprintf("%s=%s", param, value))
				}
				out, stderr, err := tests.RunCommandPipe(ocProcessCommand, []string{"oc", "create", "-f", "-"})
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to create VirtualMachine %q via command \"%s | oc create -f -\": %s: %v", vmName, strings.Join(ocProcessCommand, " "), out+stderr, err)
				ExpectWithOffset(1, out).To(MatchRegexp(`"?%s"? created\n`, vmName), "command \"%s | oc create -f -\" did not print expected message: %s", strings.Join(ocProcessCommand, " "), out+stderr)
				By("Checking if the VirtualMachine exists")
				EventuallyWithOffset(1, func() error {
					_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
					return err
				}).Should(Succeed(), "VirtualMachine %q still does not exist", vmName)
			}
		}

		AssertVMCreationFailure := func() func() {
			return func() {
				By("Creating VirtualMachine from Template via oc command")
				ocProcessCommand := []string{"oc", "process", "-f", templateFile}
				for param, value := range templateParams {
					ocProcessCommand = append(ocProcessCommand, "-p", fmt.Sprintf("%s=%s", param, value))
				}
				out, stderr, err := tests.RunCommandPipe(ocProcessCommand, []string{"oc", "create", "-f", "-"})
				ExpectWithOffset(1, err).To(HaveOccurred(), "creation of VirtualMachine %q via command \"%s | oc create -f -\" succeeded: %s: %v", vmName, strings.Join(ocProcessCommand, " "), out+stderr, err)
			}
		}

		AssertVMDeletionSuccess := func() func() {
			return func() {
				By("Deleting the VirtualMachine via oc command")
				out, stderr, err := tests.RunCommand("oc", "delete", "vm", vmName)
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to delete VirtualMachine via command \"oc delete vm %s\": %s: %v", vmName, out+stderr, err)
				ExpectWithOffset(1, out).To(MatchRegexp(`"?%s"? deleted\n`, vmName), "command \"oc delete vm %s\" did not print expected message: %s", vmName, out)

				By("Checking if the VM does not exist anymore")
				EventuallyWithOffset(1, func() bool {
					vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
					return errors.IsNotFound(err) || vm.ObjectMeta.DeletionTimestamp != nil
				}).Should(BeTrue(), "the VirtualMachine %q still exists and deletion timestamp was not set", vmName)
			}
		}

		AssertVMDeletionFailure := func() func() {
			return func() {
				By("Deleting the VirtualMachine via oc command")
				out, stderr, err := tests.RunCommand("oc", "delete", "vm", vmName)
				ExpectWithOffset(1, err).To(HaveOccurred(), "failed to delete VirtualMachine via command \"oc delete vm %s\": %s: %v", vmName, out+stderr, err)
			}
		}

		AssertVMStartSuccess := func(command string) func() {
			return func() {
				switch command {
				case "oc":
					By("Starting VirtualMachine via oc command")
					patch := `{"spec":{"running":true}}`
					out, stderr, err := tests.RunCommand("oc", "patch", "vm", vmName, "--type=merge", "-p", patch)
					ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed schedule VirtualMachine %q start via command \"oc patch vm %s --type=merge -p '%s'\": %s: %v", vmName, vmName, patch, out+stderr, err)
					ExpectWithOffset(1, out).To(MatchRegexp(`"?%s"? patched\n`, vmName), "command \"oc patch vm %s --type=merge -p '%s'\" did not print expected message: %s", vmName, patch, out+stderr)

				case "virtctl":
					By("Starting VirtualMachine via virtctl command")
					out, stderr, err := tests.RunCommand("virtctl", "start", vmName)
					ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to schedule VirtualMachine %q start via command \"virtctl start %s\": %s: %v", vmName, vmName, out+stderr, err)
					ExpectWithOffset(1, out).To(ContainSubstring("%s was scheduled to start\n", vmName), "command \"virtctl start %s\" did not print expected message: %s", vmName, out+stderr)
				}

				By("Checking if the VirtualMachineInstance was created")
				EventuallyWithOffset(1, func() error {
					_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
					return err
				}).Should(Succeed(), "the VirtualMachineInstance %q still does not exist", vmName)

				By("Checking if the VirtualMachine has status ready")
				EventuallyWithOffset(1, func() bool {
					vm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
					ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to fetch VirtualMachine %q: %v", vmName, err)
					return vm.Status.Ready
				}).Should(BeTrue(), "VirtualMachine %q still does not have status ready", vmName)

				By("Checking if the VirtualMachineInstance specs match Template parameters")
				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmName, &metav1.GetOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to fetch VirtualMachine %q: %v", vmName, err)
				vmiCPUCores := vmi.Spec.Domain.CPU.Cores
				templateParamCPUCores, err := strconv.ParseUint(templateParams["CPU_CORES"], 10, 32)
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "cannot parse CPU_CORES parameter: value %q: %v", templateParams["CPU_CORES"], err)
				ExpectWithOffset(1, vmiCPUCores).To(Equal(uint32(templateParamCPUCores)), "VirtualMachineInstance CPU cores (%d) does not match CPU_CORES parameter value: %s", vmiCPUCores, templateParams["CPU_CORES"])
				vmiMemory := vmi.Spec.Domain.Resources.Requests["memory"]
				templateParamMemory, err := resource.ParseQuantity(templateParams["MEMORY"])
				ExpectWithOffset(1, err).ToNot(HaveOccurred(), "cannot parse MEMORY parameter: value %q: %v", templateParams["MEMORY"], err)
				ExpectWithOffset(1, vmiMemory).To(Equal(templateParamMemory), "VirtualMachineInstance memory (%s) does not match MEMORY parameter value: %s", vmiMemory.String(), templateParams["MEMORY"])
			}
		}

		AssertTemplateTestSuccess := func() {
			It("[test_id:3292]should succeed to create VirtualMachine via oc command", AssertVMCreationSuccess())
			It("[test_id:3293]should fail to delete VirtualMachine via oc command", AssertVMDeletionFailure())

			When("the VirtualMachine was created", func() {
				BeforeEach(AssertVMCreationSuccess())
				It("[test_id:3294]should succeed to start the VirtualMachine via oc command", AssertVMStartSuccess("oc"))
				It("[test_id:3295]should succeed to delete VirtualMachine via oc command", AssertVMDeletionSuccess())
				It("[test_id:3308]should fail to create the same VirtualMachine via oc command", AssertVMCreationFailure())
			})
		}

		BeforeEach(AssertTestSetupSuccess())

		AfterEach(AssertTestCleanupSuccess())

		Context("with Fedora Template", func() {
			BeforeEach(func() {
				AssertTemplateSetupSuccess(vmsgen.GetTemplateFedoraWithContainerDisk(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)), nil)()
			})

			AssertTemplateTestSuccess()
		})

		Context("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]with RHEL Template", func() {
			BeforeEach(func() {
				tests.SkipIfNoRhelImage(virtClient)
				tests.CreatePVC(tests.OSRhel, "15Gi", tests.Config.StorageClassRhel, true)
				AssertTemplateSetupSuccess(vmsgen.GetTestTemplateRHEL7(), nil)()
			})

			AssertTemplateTestSuccess()
		})
	})
})
