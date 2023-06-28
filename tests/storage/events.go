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

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	ioerrorPV  = "ioerror-pv"
	ioerrorPVC = "ioerror-pvc"
	deviceName = "errdev0"
	diskName   = "disk0"
)

var _ = SIGDescribe("[Serial]K8s IO events", Serial, func() {
	var (
		nodeName   string
		virtClient kubecli.KubevirtClient
		pv         *k8sv1.PersistentVolume
		pvc        *k8sv1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodeName = tests.NodeNameWithHandler()
		tests.CreateFaultyDisk(nodeName, deviceName)
		var err error
		pv, pvc, err = tests.CreatePVandPVCwithFaultyDisk(nodeName, "/dev/mapper/"+deviceName, testsuite.GetTestNamespace(nil))
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
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			return err
		}, 100*time.Second, time.Second).Should(BeNil(), "Failed to create vmi")

		libwait.WaitForSuccessfulVMIStart(vmi,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(240),
		)

		By("Expecting  paused event on VMI ")
		events.ExpectEvent(vmi, k8sv1.EventTypeWarning, "IOerror")

		err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete VMI")
		libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
	})
})
