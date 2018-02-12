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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Storage", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		Skip("Direct ISCSI storage access is not supported at the moment.")
		tests.BeforeTestCleanup()
	})

	getTargetLogs := func(tailLines int64) string {
		pods, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{LabelSelector: v1.AppLabel + " in (iscsi-demo-target)"})
		Expect(err).ToNot(HaveOccurred())

		//FIXME Sometimes pods hang in terminating state, select the pod which does not have a deletion timestamp
		podName := ""
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				podName = pod.ObjectMeta.Name
				break
			}
		}
		Expect(podName).ToNot(BeEmpty())

		logsRaw, err := virtClient.CoreV1().
			Pods(metav1.NamespaceSystem).
			GetLogs(podName,
				&k8sv1.PodLogOptions{TailLines: &tailLines}).
			DoRaw()
		Expect(err).To(BeNil())

		return string(logsRaw)
	}

	BeforeEach(func() {
		// Wait until there is no connection
		logs := func() string { return getTargetLogs(70) }
		Eventually(logs,
			11*time.Second,
			500*time.Millisecond).
			Should(ContainSubstring("I_T nexus information:\n    LUN information:"))
	})

	RunVMAndExpectLaunch := func(vm *v1.VirtualMachine, withAuth bool) {
		obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMStart(obj)

		if withAuth == false {
			// Periodically check if we now have a connection on the target
			// We don't check against the actual IP, since depending on the kubernetes proxy mode, and the network provider
			// we will see different IPs here. The BeforeEach function makes sure that no other connections exist.
			Eventually(func() string { return getTargetLogs(70) },
				11*time.Second,
				500*time.Millisecond).
				Should(
					MatchRegexp(fmt.Sprintf("IP Address: [0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+")),
				)
		}
	}

	Context("Given a fresh iSCSI target", func() {

		It("should be available and ready", func() {
			logs := getTargetLogs(75)
			Expect(logs).To(ContainSubstring("Target 1: iqn.2017-01.io.kubevirt:sn.42"))
			Expect(logs).To(ContainSubstring("Driver: iscsi"))
			Expect(logs).To(ContainSubstring("State: ready"))
		})

		It("should not have any connections", func() {
			logs := getTargetLogs(70)
			// Ensure that no connections are listed
			Expect(logs).To(ContainSubstring("I_T nexus information:\n    LUN information:"))
		})
	})

	Context("Given a VM and a directly connected Alpine LUN", func() {

		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the LUN attached
			vm := tests.NewRandomVMWithDirectLun(2, false)
			RunVMAndExpectLaunch(vm, false)
			close(done)
		}, 30)
	})

	Context("Given a VM and a directly connected Alpine LUN with CHAP auth", func() {

		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the LUN attached
			vm := tests.NewRandomVMWithDirectLun(2, true)
			RunVMAndExpectLaunch(vm, true)
			close(done)
		}, 30)
	})

	Context("Given a VM and an Alpine PVC", func() {
		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the PVC attached
			vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
			RunVMAndExpectLaunch(vm, false)
			close(done)
		}, 30)
	})

	Context("Given a VM and an Alpine PVC with CHAP auth", func() {
		It("should be successfully started by libvirt", func(done Done) {
			// Start the VM with the PVC attached
			vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSIWithAuth)
			RunVMAndExpectLaunch(vm, true)
			close(done)
		}, 30)

		It("should not modify the VM spec on status update", func() {
			vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSIWithAuth)
			v1.SetObjectDefaults_VirtualMachine(vm)
			vm, err := virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStartWithTimeout(vm, 60)
			startedVM, err := virtClient.VM(tests.NamespaceTestDefault).Get(vm.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(startedVM.Spec).To(Equal(vm.Spec))
		})
	})
})
