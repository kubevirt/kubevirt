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
 */

package portforward_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
)

var _ = Describe("Port forward", func() {
	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind, expectedError string) {
		kind, namespace, name, err := portforward.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(expectedError))
		}
	},
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vmi with name and namespace", "vmi/testvmi/default", "default", "testvmi", "vmi", ""),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", ""),
		Entry("kind vm with name and namespace", "vm/testvm/default", "default", "testvm", "vm", ""),
		Entry("name with dots and namespace", "vmi/testvmi.with.dots/default", "default", "testvmi.with.dots", "vmi", ""),
		Entry("name and namespace with dots", "vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", ""),
		Entry("name with dots and namespace with dots", "vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", ""),
		Entry("no slash", "testvmi", "", "", "", "target must contain type and name separated by '/'"),
		Entry("empty namespace", "vmi/testvmi/", "", "", "", "namespace cannot be empty"),
		Entry("more than three slashes", "vmi/testvmi/default/something", "", "", "", "target is not valid with more than two '/'"),
		Entry("invalid type with name", "invalid/testvmi", "", "", "", "unsupported resource type 'invalid'"),
		Entry("invalid type with name and namespace", "invalid/testvmi/default", "", "", "", "unsupported resource type 'invalid'"),
		Entry("only valid kind", "vmi/", "", "", "", "name cannot be empty"),
		Entry("empty target", "", "", "", "", "target cannot be empty"),
		Entry("only slash", "/", "", "", "", "unsupported resource type ''"),
		Entry("two slashes", "//", "", "", "", "namespace cannot be empty"),
		Entry("only dot", ".", "", "", "", "target must contain type and name separated by '/'"),
		Entry("only separators", "/.", "", "", "", "unsupported resource type ''"),
		// Normalization of type
		Entry("kind vmi", "vmi/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vmis", "vmis/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind virtualmachineinstance", "virtualmachineinstance/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind virtualmachineinstances", "virtualmachineinstances/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vm", "vm/testvm", "", "testvm", "vm", ""),
		Entry("kind vms", "vms/testvm", "", "testvm", "vm", ""),
		Entry("kind virtualmachine", "virtualmachine/testvm", "", "testvm", "vm", ""),
		Entry("kind virtualmachines", "virtualmachines/testvm", "", "testvm", "vm", ""),
		// Legacy parsing
		Entry("name with dots", "vmi/testvmi.with.dots", "dots", "testvmi.with", "vmi", ""),
		Entry("kind vmi with name and namespace (legacy)", "vmi/testvmi.default", "default", "testvmi", "vmi", ""),
		Entry("kind vm with name and namespace (legacy)", "vm/testvm.default", "default", "testvm", "vm", ""),
	)
})
