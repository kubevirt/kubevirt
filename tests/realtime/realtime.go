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
 */

package realtime

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	memory                         = "512Mi"
	tuneAdminRealtimeCloudInitData = `#cloud-config
password: fedora
chpasswd: { expire: False }
bootcmd:
   - sudo tuned-adm profile realtime
`
)

func newFedoraRealtime(realtimeMask string) *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithRng(),
		libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime)),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(tuneAdminRealtimeCloudInitData)),
		libvmi.WithLimitMemory(memory),
		libvmi.WithLimitCPU("2"),
		libvmi.WithResourceMemory(memory),
		libvmi.WithResourceCPU("2"),
		libvmi.WithCPUModel(v1.CPUModeHostPassthrough),
		libvmi.WithDedicatedCPUPlacement(),
		libvmi.WithRealtimeMask(realtimeMask),
		libvmi.WithNUMAGuestMappingPassthrough(),
		libvmi.WithHugepages("2Mi"),
		libvmi.WithGuestMemory(memory),
	)
}

func byStartingTheVMI(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient) *v1.VirtualMachineInstance {
	By("Starting a VirtualMachineInstance")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return libwait.WaitForSuccessfulVMIStart(vmi)
}

var _ = Describe("[sig-compute-realtime]Realtime", Serial, decorators.SigComputeRealtime, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("should start the realtime VM", func() {

		It("when no mask is specified", func() {
			const noMask = ""
			vmi := byStartingTheVMI(newFedoraRealtime(noMask), virtClient)
			By("Validating VCPU scheduler placement information")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			emulator := filepath.Base(domSpec.Devices.Emulator)
			psOutput, err := exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", "ps -LC " + emulator + " -o policy,rtprio,psr|grep FF| awk '{print $2}'"},
			)
			Expect(err).ToNot(HaveOccurred())
			slice := strings.Split(strings.TrimSpace(psOutput), "\n")
			Expect(slice).To(HaveLen(2))
			for _, l := range slice {
				Expect(parsePriority(l)).To(BeEquivalentTo(1))
			}
			By("Validating that the memory lock limits are higher than the memory requested")
			psOutput, err = exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", "grep 'locked memory' /proc/$(ps -C " + emulator + " -o pid --noheader|xargs)/limits |tr -s ' '| awk '{print $4\" \"$5}'"},
			)
			Expect(err).ToNot(HaveOccurred())
			limits := strings.Split(strings.TrimSpace(psOutput), " ")
			softLimit, err := strconv.ParseInt(limits[0], 10, 64)
			Expect(err).ToNot(HaveOccurred())
			hardLimit, err := strconv.ParseInt(limits[1], 10, 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(softLimit).To(Equal(hardLimit))
			mustParse := resource.MustParse(memory)
			requested, canConvert := mustParse.AsInt64()
			Expect(canConvert).To(BeTrue())
			Expect(hardLimit).To(BeNumerically(">", requested))
			By("checking if the guest is still running")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
			Expect(console.LoginToFedora(vmi)).To(Succeed())
		})

		It("when realtime mask is specified", func() {
			vmi := byStartingTheVMI(newFedoraRealtime("0-1,^1"), virtClient)
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			By("Validating VCPU scheduler placement information")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			emulator := filepath.Base(domSpec.Devices.Emulator)
			psOutput, err := exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", "ps -LC " + emulator + " -o policy,rtprio,psr|grep FF| awk '{print $2}'"},
			)
			Expect(err).ToNot(HaveOccurred())
			slice := strings.Split(strings.TrimSpace(psOutput), "\n")
			Expect(slice).To(HaveLen(1))
			Expect(parsePriority(slice[0])).To(BeEquivalentTo(1))

			By("Validating the VCPU mask matches the scheduler profile for all cores")
			psOutput, err = exec.ExecuteCommandOnPod(
				pod,
				"compute",
				[]string{"/bin/bash", "-c", "ps -TcC " + emulator + " |grep CPU |awk '{print $3\" \" $8}'"},
			)
			Expect(err).ToNot(HaveOccurred())
			slice = strings.Split(strings.TrimSpace(psOutput), "\n")
			Expect(slice).To(HaveLen(2))
			Expect(slice[0]).To(Equal("FF 0/KVM"))
			Expect(slice[1]).To(Equal("TS 1/KVM"))

			By("checking if the guest is still running")
			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running))
			Expect(console.LoginToFedora(vmi)).To(Succeed())
		})
	})
})

func parsePriority(psLine string) int64 {
	s := strings.Split(psLine, " ")
	Expect(s).To(HaveLen(1))
	prio, err := strconv.ParseInt(s[0], 10, 8)
	Expect(err).NotTo(HaveOccurred())
	return prio
}
