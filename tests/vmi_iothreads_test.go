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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("IOThreads", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
	})

	Context("CRD Validation", func() {

		It("Should reject POST if ioThreadsPolicy is invalid", func() {
			policy := "invalid_policy"
			vmi.Spec.Domain.IOThreadsPolicy = &policy

			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			errorMessage := fmt.Sprintf("Invalid IOThreadsPolicy (%s)", policy)
			Expect(strings.Contains(err.Error(), errorMessage), "Expected error message to say: %s", errorMessage)
		})

	})
})
