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
 * Copyright the KubeVirt Authors.
 *
 */

package libvmi

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type VMIRunner struct {
	VMI         *v1.VirtualMachineInstance
	runOptions  []VMIRunnerRunOption
	timeout     int
	waitOptions []libwait.Option
}

type VMIRunnerRunOption func(v *VMIRunner)

func WithTimeout(timeout int) VMIRunnerRunOption {
	return func(v *VMIRunner) {
		v.timeout = timeout
	}
}

func NewRunnerFor(vmi *v1.VirtualMachineInstance, options ...VMIRunnerRunOption) *VMIRunner {
	return &VMIRunner{VMI: vmi, runOptions: options}
}

func (v *VMIRunner) Run(expectOptions ...VMIRunnerExpectOption) *VMIRunner {
	for _, runOption := range v.runOptions {
		runOption(v)
	}

	ginkgo.By("Starting a VirtualMachineInstance")
	virtCli := kubevirt.Client()

	var obj *v1.VirtualMachineInstance
	var err error
	gomega.Eventually(func() error {
		obj, err = virtCli.VirtualMachineInstance(testsuite.GetTestNamespace(v.VMI)).Create(context.Background(), v.VMI)
		return err
	}, v.timeout, 1*time.Second).ShouldNot(gomega.HaveOccurred())
	v.VMI = obj

	for _, expectOption := range expectOptions {
		expectOption(v)
	}

	return v
}

type VMIRunnerExpectOption func(v *VMIRunner)

func WithDataVolumeSucceeded(dv *v1beta1.DataVolume, timeout int) VMIRunnerExpectOption {
	return func(v *VMIRunner) {
		ginkgo.By("Waiting until the DataVolume is ready")
		libstorage.EventuallyDV(dv, timeout, matcher.HaveSucceeded())
	}
}

func (v *VMIRunner) WithWaitOptions(options ...libwait.Option) *VMIRunner {
	v.waitOptions = options
	return v
}

func (v *VMIRunner) ThenWaitFor(vmiPhases ...v1.VirtualMachineInstancePhase) *VMIRunner {
	ginkgo.By(fmt.Sprintf("Waiting until the VirtualMachineInstance will reach any of %v", vmiPhases))
	v.VMI = libwait.WaitForVMIPhase(v.VMI, vmiPhases, v.waitOptions...)
	return v
}
