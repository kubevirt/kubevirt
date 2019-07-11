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
	"fmt"

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

	BeforeEach(func() {
		tests.SkipIfNoCmd("kubectl")
		tests.BeforeTestCleanup()
	})

	runCommandOnNode := func(command string, node string, args ...string) (err error) {
		cmdName := tests.GetK8sCmdClient()
		newArgs := make([]string, 0)
		if tests.IsOpenShift() {
			// if the cluster is openshift we need to append `adm` for the commands `drain` and `uncordon`
			// as the oc binary is used
			newArgs = append(newArgs, "adm")
		}
		newArgs = append(newArgs, command)
		newArgs = append(newArgs, node)
		newArgs = append(newArgs, args...)
		_, _, err = tests.RunCommandWithNS("", cmdName, newArgs...)
		return
	}

	uncordonNode := func(node string) (err error) {
		err = runCommandOnNode("uncordon", node)
		return
	}

	AfterEach(func() {
		err = uncordonNode("node01")
		Expect(err).ToNot(HaveOccurred())

		err = uncordonNode("node02")
		Expect(err).ToNot(HaveOccurred())
	})

	drainNode := func(node string, podSelector string) (err error) {
		err = runCommandOnNode("drain", node, DefaultTimeout, podSelector)
		return
	}

	drainNodesSelectingPods := func(podName string) {
		podSelector := fmt.Sprintf("--pod-selector=kubevirt.io=%s", podName)

		By("draining node01")
		err = drainNode("node01", podSelector)
		Expect(err).ToNot(HaveOccurred())

		By("draining node02 should fail, because the target pod is protected from voluntary evictions by pdb")
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
			drainNodesSelectingPods("virt-controller")
		})

		It("for virt-api", func() {
			drainNodesSelectingPods("virt-api")
		})

	})

})
