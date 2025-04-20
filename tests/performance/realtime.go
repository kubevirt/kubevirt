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

package performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const tuneAdminRealtimeCloudInitData = `#cloud-config
password: fedora
chpasswd: { expire: False }
bootcmd:
   - sudo tuned-adm profile realtime
`

func byStartingTheVMI(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient) {
	By("Starting a VirtualMachineInstance")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	libwait.WaitForSuccessfulVMIStart(vmi)
}

var _ = Describe(SIG("CPU latency tests for measuring realtime VMs performance", decorators.RequiresTwoWorkerNodesWithCPUManager, decorators.RequiresHugepages2Mi, func() {

	var (
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
		err        error
	)

	BeforeEach(func() {
		skipIfNoRealtimePerformanceTests()
		virtClient = kubevirt.Client()
	})

	It("running cyclictest and collecting results directly from VM", func() {
		const memory = "512Mi"
		const noMask = ""
		vmi = libvmi.New(
			libvmi.WithRng(),
			libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime)),
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(tuneAdminRealtimeCloudInitData)),
			libvmi.WithResourceCPU("2"),
			libvmi.WithLimitCPU("2"),
			libvmi.WithResourceMemory(memory),
			libvmi.WithLimitMemory(memory),
			libvmi.WithCPUModel(v1.CPUModeHostPassthrough),
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithRealtimeMask(noMask),
			libvmi.WithNUMAGuestMappingPassthrough(),
			libvmi.WithHugepages("2Mi"),
			libvmi.WithGuestMemory(memory),
		)
		byStartingTheVMI(vmi, virtClient)
		By("validating VMI is up and running")
		vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		By(fmt.Sprintf("running cyclictest for %d seconds", cyclicTestDurationInSeconds))
		cmd := fmt.Sprintf("sudo cyclictest --policy fifo --priority 95 -i 100 -H 1000 -D %ds -q |grep 'Max Latencies' |awk '{print $4}'\n", cyclicTestDurationInSeconds)
		res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
			&expect.BSnd{S: cmd},
			&expect.BExp{R: console.RetValue("[0-9]+")},
		}, int(5+cyclicTestDurationInSeconds))
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(HaveLen(1))
		sout := strings.Split(res[0].Output, "\r\n")
		Expect(sout).To(HaveLen(3))
		max, err := strconv.ParseInt(sout[1], 10, 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(max).NotTo(BeNumerically(">", realtimeThreshold), fmt.Sprintf("Maximum CPU latency of %d is greater than threshold %d", max, realtimeThreshold))
	})

}))

type psOutput struct {
	priority    int64
	processorID int64
}
