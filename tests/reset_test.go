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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func waitForVMIReset(vmi *v1.VirtualMachineInstance) error {
	By(fmt.Sprintf("Waiting for vmi %s reset", vmi.Name))
	if vmi.Namespace == "" {
		vmi.Namespace = testsuite.GetTestNamespace(vmi)
	}
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: ".*Detected virtualization kvm.*"},
	}, 300)
}

var _ = Describe("[level:component][sig-compute]Reset", decorators.SigCompute, func() {
	const vmiLaunchTimeout = 360

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("reset vmi with should succeed", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(), vmiLaunchTimeout)

		Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		errChan := make(chan error)
		go func() {
			time.Sleep(5)
			errChan <- virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Reset(context.Background(), vmi.Name)
		}()

		start := time.Now().UTC().Unix()
		err := waitForVMIReset(vmi)
		end := time.Now().UTC().Unix()

		if err != nil {
			err = fmt.Errorf("start [%d] end [%d] err: %v", int32(start), int32(end), err)
		}
		Expect(err).ToNot(HaveOccurred())

		select {
		case err := <-errChan:
			Expect(err).ToNot(HaveOccurred())
		}
	})

})
