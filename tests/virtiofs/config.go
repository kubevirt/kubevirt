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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virtiofs

import (
	"fmt"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[sig-compute] vitiofs config volumes", decorators.SigCompute, func() {
	BeforeEach(func() {
		checks.SkipTestIfNoFeatureGate(virtconfig.VirtIOFSGate)
	})

	Context("With a single ConfigMap volume", func() {
		var (
			configMapName string
			configMapPath string
		)

		BeforeEach(func() {
			// We use the ConfigMap name as mount `tag` for qemu, but the `tag` property must be 36 bytes or less
			configMapName = "configmap-" + uuid.NewString()[:6]
			configMapPath = config.GetConfigMapSourcePath(configMapName)

			data := map[string]string{
				"option1": "value1",
				"option2": "value2",
				"option3": "value3",
			}
			tests.CreateConfigMap(configMapName, testsuite.GetTestNamespace(nil), data)
		})

		It("Should be the mounted virtiofs layout the same for a pod and vmi", func() {
			expectedOutput := "value1value2value3"

			By("Running VMI")
			vmi := libvmi.NewFedora(
				libvmi.WithConfigMapFs(configMapName, configMapName),
			)
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			By("Logging into the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking if ConfigMap has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podOutput, err := exec.ExecuteCommandOnPod(
				vmiPod,
				fmt.Sprintf("virtiofs-%s", configMapName),
				[]string{"cat",
					configMapPath + "/option1",
					configMapPath + "/option2",
					configMapPath + "/option3",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(podOutput).To(Equal(expectedOutput))

			By("Checking mounted ConfigMap")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt \n", configMapName)},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/option1 /mnt/option2 /mnt/option3\n"},
				&expect.BExp{R: expectedOutput},
			}, 200)).To(Succeed())
		})
	})

	Context("With a single Secret volume", func() {
		var (
			secretName string
			secretPath string
		)

		BeforeEach(func() {
			// We use the Secret name as mount `tag` for qemu, but the `tag` property must be 36 bytes or less
			secretName = "secret-" + uuid.NewString()[:6]
			secretPath = config.GetSecretSourcePath(secretName)

			data := map[string]string{
				"user":     "admin",
				"password": "redhat",
			}
			tests.CreateSecret(secretName, testsuite.GetTestNamespace(nil), data)
		})

		It("Should be the mounted virtiofs layout the same for a pod and vmi", func() {
			expectedOutput := "adminredhat"

			By("Running VMI")
			vmi := libvmi.NewFedora(
				libvmi.WithSecretFs(secretName, secretName),
			)
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			By("Logging into the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking if Secret has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podOutput, err := exec.ExecuteCommandOnPod(
				vmiPod,
				fmt.Sprintf("virtiofs-%s", secretName),
				[]string{"cat",
					secretPath + "/user",
					secretPath + "/password",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(podOutput).To(Equal(expectedOutput))

			By("Checking mounted Secret")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt \n", secretName)},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/user /mnt/password\n"},
				&expect.BExp{R: expectedOutput},
			}, 200)).To(Succeed())
		})
	})

	Context("With a ServiceAccount defined", func() {

		serviceAccountPath := config.ServiceAccountSourceDir

		It("Should be the namespace and token the same for a pod and vmi with virtiofs", func() {
			serviceAccountVolumeName := "default-disk"

			By("Running VMI")
			vmi := libvmi.NewFedora(
				libvmi.WithServiceAccountFs("default", serviceAccountVolumeName),
			)
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			By("Logging into the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking if ServiceAccount has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			namespace, err := exec.ExecuteCommandOnPod(
				vmiPod,
				fmt.Sprintf("virtiofs-%s", serviceAccountVolumeName),
				[]string{"cat",
					serviceAccountPath + "/namespace",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(namespace).To(Equal(testsuite.GetTestNamespace(vmi)))

			token, err := exec.ExecuteCommandOnPod(
				vmiPod,
				fmt.Sprintf("virtiofs-%s", serviceAccountVolumeName),
				[]string{"tail", "-c", "20",
					serviceAccountPath + "/token",
				},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Checking mounted ServiceAccount")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// mount iso ConfigMap image
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt \n", serviceAccountVolumeName)},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/namespace\n"},
				&expect.BExp{R: testsuite.GetTestNamespace(vmi)},
				&expect.BSnd{S: "tail -c 20 /mnt/token\n"},
				&expect.BExp{R: token},
			}, 200)).To(Succeed())
		})
	})

	Context("With a DownwardAPI defined", func() {
		// We use the DownwardAPI name as mount `tag` for qemu, but the `tag` property must be 36 bytes or less
		downwardAPIName := "downwardapi-" + uuid.NewString()[:6]
		downwardAPIPath := config.GetDownwardAPISourcePath(downwardAPIName)

		testLabelKey := "kubevirt.io.testdownwardapi"
		testLabelVal := "downwardAPIValue"
		expectedOutput := testLabelKey + "=" + "\"" + testLabelVal + "\""

		It("Should be the namespace and token the same for a pod and vmi with virtiofs", func() {

			By("Running VMI")
			vmi := libvmi.NewFedora(
				libvmi.WithLabel(testLabelKey, testLabelVal),
				libvmi.WithDownwardAPIFs(downwardAPIName),
			)
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			By("Logging into the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Checking if DownwardAPI has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			podOutput, err := exec.ExecuteCommandOnPod(
				vmiPod,
				fmt.Sprintf("virtiofs-%s", downwardAPIName),
				[]string{"grep", testLabelKey,
					downwardAPIPath + "/labels",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(podOutput).To(Equal(expectedOutput + "\n"))

			By("Checking mounted DownwardAPI")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt \n", downwardAPIName)},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep --color=never " + testLabelKey + " /mnt/labels\n"},
				&expect.BExp{R: expectedOutput},
			}, 200)).To(Succeed())
		})
	})
})
