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

package libclient

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

const waitingVMInstanceStart = "Waiting until the VirtualMachineInstance will start"

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By(waitingVMInstanceStart)
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectLaunchWithDataVolume(vmi *v1.VirtualMachineInstance, dv *cdiv1beta1.DataVolume, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By("Waiting until the DataVolume is ready")
	libstorage.EventuallyDV(dv, timeout, matcher.HaveSucceeded())
	By(waitingVMInstanceStart)
	warningsIgnoreList := []string{"didn't find PVC", "unable to find datavolume"}
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		libwait.WithWarningsIgnoreList(warningsIgnoreList),
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By(waitingVMInstanceStart)
	return libwait.WaitForSuccessfulVMIStart(vmi,
		libwait.WithFailOnWarnings(false),
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return RunVMIAndExpectSchedulingWithWarningPolicy(vmi, timeout, wp)
}

func RunVMIAndExpectSchedulingWithWarningPolicy(vmi *v1.VirtualMachineInstance, timeout int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By("Waiting until the VirtualMachineInstance will be scheduled")
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running},
		libwait.WithWarningsPolicy(&wp),
		libwait.WithTimeout(timeout),
	)
}
