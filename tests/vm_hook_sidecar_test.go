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
	"time"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("HookSidecars", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vm *v1.VirtualMachine

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vm = tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
		vm.ObjectMeta.Annotations = map[string]string{
			"hooks.kubevirt.io/hookSidecars":              `[{"image": "registry:5000/kubevirt/example-hook-sidecar:devel"}]`,
			"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
		}
	})

	Describe("VM definition", func() {
		Context("with SM BIOS hook sidecar", func() {
			It("should successfully start with hook sidecar annotation", func() {
				By("Starting a VM")
				vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(vm)
			}, 300)

			It("should call Collect on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				logs := func() string { return getHookSidecarLogs(virtClient, vm) }
				tests.WaitForSuccessfulVMStart(vm)
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's Info method has been called"))
			}, 300)

			It("should call OnDefineDomain on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				logs := func() string { return getHookSidecarLogs(virtClient, vm) }
				tests.WaitForSuccessfulVMStart(vm)
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's OnDefineDomain callback method has been called"))
			}, 300)

			It("should update domain XML with SM BIOS properties", func() {
				By("Reading domain XML using virsh")
				tests.SkipIfNoKubectl()
				vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				tests.WaitForSuccessfulVMStart(vm)
				domainXml := getVmDomainXml(virtClient, vm)
				Expect(domainXml).Should(ContainSubstring("<sysinfo type='smbios'>"))
				Expect(domainXml).Should(ContainSubstring("<smbios mode='sysinfo'/>"))
				Expect(domainXml).Should(ContainSubstring("<entry name='manufacturer'>Radical Edward</entry>"))
			}, 300)
		})
	})

})

func getHookSidecarLogs(virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine) string {
	namespace := vm.GetObjectMeta().GetNamespace()
	podName := getVmPodName(virtCli, vm)

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: "hook-sidecar-0",
		}).
		DoRaw()
	Expect(err).To(BeNil())

	return string(logsRaw)
}

func getVmDomainXml(virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine) string {
	podName := getVmPodName(virtCli, vm)

	vmNameListRaw, err := tests.RunKubectlCommand("exec", "-ti", "--namespace", vm.GetObjectMeta().GetNamespace(), podName, "--container", "compute", "--", "virsh", "list", "--name")
	Expect(err).ToNot(HaveOccurred())

	vmName := strings.Split(vmNameListRaw, "\n")[0]
	vmDomainXML, err := tests.RunKubectlCommand("exec", "-ti", "--namespace", vm.GetObjectMeta().GetNamespace(), podName, "--container", "compute", "--", "virsh", "dumpxml", vmName)
	Expect(err).ToNot(HaveOccurred())

	return vmDomainXML
}

func getVmPodName(virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine) string {
	namespace := vm.GetObjectMeta().GetNamespace()
	domain := vm.GetObjectMeta().GetName()
	labelSelector := fmt.Sprintf("kubevirt.io/domain in (%s)", domain)

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty())

	return podName
}
