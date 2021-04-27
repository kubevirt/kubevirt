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
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

const cloudinitHookSidecarImage = "example-cloudinit-hook-sidecar"

var _ = Describe("[sig-compute]CloudInitHookSidecars", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var vmi *v1.VirtualMachineInstance

	GetCloudInitHookSidecarLogs := func(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
		namespace := vmi.GetObjectMeta().GetNamespace()
		podName := tests.GetVmPodName(virtCli, vmi)

		var tailLines int64 = 100
		logsRaw, err := virtCli.CoreV1().
			Pods(namespace).
			GetLogs(podName, &k8sv1.PodLogOptions{
				TailLines: &tailLines,
				Container: "hook-sidecar-0",
			}).
			DoRaw(context.Background())
		Expect(err).To(BeNil())

		return string(logsRaw)
	}
	MountCloudInit := func(vmi *v1.VirtualMachineInstance) {
		cmdCheck := "mount $(blkid  -L cidata) /mnt/\n"
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "echo $?\n"},
			&expect.BExp{R: console.RetValue("0")},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	CheckCloudInitFile := func(vmi *v1.VirtualMachineInstance, testFile, testData string) {
		cmdCheck := "cat /mnt/" + testFile + "\n"
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: testData},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#FAKE")
		vmi.ObjectMeta.Annotations = map[string]string{
			"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(`[{"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`, flags.KubeVirtUtilityRepoPrefix, cloudinitHookSidecarImage, flags.KubeVirtUtilityVersionTag),
		}
	})

	Describe("VMI definition", func() {
		Context("with CloudInit hook sidecar", func() {
			It("[test_id:3167]should successfully start with hook sidecar annotation", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
			}, 300)

			It("[test_id:3168]should call Collect and PreCloudInitIso on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				logs := func() string { return GetCloudInitHookSidecarLogs(virtClient, vmi) }
				tests.WaitForSuccessfulVMIStart(vmi)
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's Info method has been called"))
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's PreCloudInitIso callback method has been called"))
			}, 300)

			It("[test_id:3169]should have cloud-init user-data from sidecar", func() {
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
				By("mouting cloudinit iso")
				MountCloudInit(vmi)
				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "user-data", "#cloud-config")
			}, 300)
		})
	})

})
