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
 *
 */

package storage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	deviceName = "errdev0"
)

var _ = Describe(SIG("K8s IO events", Serial, func() {
	var (
		nodeName   string
		virtClient kubecli.KubevirtClient
		pv         *k8sv1.PersistentVolume
		pvc        *k8sv1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodeName = libnode.GetNodeNameWithHandler()
		createFaultyDisk(nodeName, deviceName)
		var err error
		pv, pvc, err = CreatePVandPVCwithFaultyDisk(nodeName, "/dev/mapper/"+deviceName, testsuite.GetTestNamespace(nil))
		Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
	})
	AfterEach(func() {
		removeFaultyDisk(nodeName, deviceName)

		err := virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})
	It("[test_id:6225]Should catch the IO error event", func() {
		By("Creating VMI with faulty disk")
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithPersistentVolumeClaim("disk0", pvc.Name),
			libvmi.WithResourceMemory("128Mi"),
		)

		Eventually(func() error {
			var err error
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			return err
		}, 100*time.Second, time.Second).Should(BeNil(), "Failed to create vmi")

		libwait.WaitForSuccessfulVMIStart(vmi,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(240),
		)

		By("Expecting  paused event on VMI ")
		events.ExpectEvent(vmi, k8sv1.EventTypeWarning, "IOerror")

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete VMI")
		libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
	})
}))

func createFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Creating faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "create", deviceName, "--table", "0 204791 error"}
	executeDeviceMapperOnNode(nodeName, args)
}

func removeFaultyDisk(nodeName, deviceName string) {
	By(fmt.Sprintf("Removing faulty disk %s on %s node", deviceName, nodeName))
	args := []string{"dmsetup", "remove", deviceName}
	executeDeviceMapperOnNode(nodeName, args)
}

func executeDeviceMapperOnNode(nodeName string, cmd []string) {
	virtClient := kubevirt.Client()

	// Image that happens to have dmsetup
	image := fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtRepoPrefix, flags.KubeVirtVersionTag)
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "device-mapper-pod-",
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    "launcher",
					Image:   image,
					Command: cmd,
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: pointer.P(true),
						RunAsUser:  pointer.P(int64(0)),
					},
				},
			},
			NodeSelector: map[string]string{
				k8sv1.LabelHostname: nodeName,
			},
		},
	}
	pod, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(matcher.ThisPod(pod), 30).Should(matcher.HaveSucceeded())
}
