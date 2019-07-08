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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests"
)

const (
	DefaultTimeout = "--timeout=60s"
)

var _ = Describe("KubeVirt control plane resilience", func() {

	var err error

	tests.FlagParse()

	tests.BeforeAll(func() {
		tests.SkipIfNoCmd("kubectl")
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	AfterEach(func() {
		_, _, err = tests.RunCommand("kubectl", "uncordon", "node01")
		Expect(err).ToNot(HaveOccurred())

		_, _, err = tests.RunCommand("kubectl", "uncordon", "node02")
		Expect(err).ToNot(HaveOccurred())
	})

	drainNode := func(node string, podSelector string) error {
		_, _, err := tests.RunCommandWithNS("", "kubectl", "drain", node, DefaultTimeout, podSelector)
		return err
	}

	uncordonNode := func(node string) error {
		_, _, err = tests.RunCommandWithNS("", "kubectl", "uncordon", node)
		return err
	}

	drainNodesSelectingPods := func(podSelector string) {
		By("draining node01")
		err = drainNode("node01", podSelector)
		Expect(err).ToNot(HaveOccurred())

		By("draining node02 should fail")
		err = drainNode("node02", podSelector)
		Expect(err).To(HaveOccurred())

		By("uncordoning node01")
		err = uncordonNode("node01")
		Expect(err).ToNot(HaveOccurred())

		By("draining node02 should not fail")
		err = drainNode("node02", podSelector)
		Expect(err).ToNot(HaveOccurred())
	}

	Context("should fail to drain the second node at first, then after uncordoning the first draining the second should succeed", func() {

		It("for virt-controller", func() {
			drainNodesSelectingPods("--pod-selector=kubevirt.io=virt-controller")
		})

		It("for virt-api", func() {
			drainNodesSelectingPods("--pod-selector=kubevirt.io=virt-api")
		})

	})

})
