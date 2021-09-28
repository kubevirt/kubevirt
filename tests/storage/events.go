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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	ioerrorPV  = "ioerror-pv"
	ioerrorPVC = "ioerror-pvc"
	deviceName = "errdev0"
	diskName   = "disk0"
)

var _ = SIGDescribe("[Serial]K8s IO events", func() {
	var (
		nodeName   string
		virtClient kubecli.KubevirtClient
		pv         *k8sv1.PersistentVolume
		pvc        *k8sv1.PersistentVolumeClaim
	)

	isExpectedIOEvent := func(e corev1.Event, vmiName string) bool {
		if e.Type == "Warning" &&
			e.Reason == "IOerror" &&
			e.Message == "VM Paused due to IO error at the volume: "+diskName &&
			e.InvolvedObject.Kind == "VirtualMachineInstance" &&
			e.InvolvedObject.Name == vmiName {
			return true
		}
		return false
	}

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		nodeName = tests.NodeNameWithHandler()
		tests.CreateFaultyDisk(nodeName, deviceName)
		pv, pvc, err = tests.CreatePVandPVCwithFaultyDisk(nodeName, "/dev/mapper/"+deviceName, util.NamespaceTestDefault)
		Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
	})
	AfterEach(func() {
		tests.RemoveFaultyDisk(nodeName, deviceName)

		err := virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})
	It("[test_id:6225]Should catch the IO error event", func() {
		By("Creating VMI with faulty disk")
		vmi := tests.NewRandomVMIWithPVC(pvc.Name)
		Eventually(func() error {
			var err error
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			return err
		}, 100*time.Second, time.Second).Should(BeNil(), "Failed to create vmi")

		tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi, 240)

		By("Expecting  paused event on VMI ")
		Eventually(func() bool {
			events, err := virtClient.CoreV1().Events(util.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, e := range events.Items {
				if isExpectedIOEvent(e, vmi.Name) {
					return true
				}
			}

			return false
		}, 30*time.Second, 5*time.Second).Should(BeTrue())
		err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
		Expect(err).To(BeNil(), "Failed to delete VMI")
		tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
	})
})
