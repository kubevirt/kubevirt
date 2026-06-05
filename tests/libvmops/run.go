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

package libvmops

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"

	v1 "kubevirt.io/api/core/v1"
)

const (
	StartupTimeoutSecondsTiny   = 30
	StartupTimeoutSecondsSmall  = 60
	StartupTimeoutSecondsMedium = 90
	StartupTimeoutSecondsLarge  = 120
	StartupTimeoutSecondsXLarge = 180
	StartupTimeoutSecondsHuge   = 240
	StartupTimeoutSecondsXHuge  = 360
)

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, watcher.WarningsPolicy{FailOnWarnings: true}, timeout)
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, watcher.WarningsPolicy{FailOnWarnings: false}, timeout)
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running},
		watcher.WarningsPolicy{FailOnWarnings: true}, timeout)
}

func RunVMIAndExpectSchedulingWithWarningPolicy(vmi *v1.VirtualMachineInstance, timeout int, wp watcher.WarningsPolicy,
) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, wp, timeout)
}

func runVMI(vmi *v1.VirtualMachineInstance, phases []v1.VirtualMachineInstancePhase, wp watcher.WarningsPolicy, timeout int,
) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(
		testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	By("Waiting until the VirtualMachineInstance reaches the desired phase")
	return libwait.WaitForVMIPhase(vmi, phases, libwait.WithWarningsPolicy(&wp), libwait.WithTimeout(timeout))
}
