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

package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/pod/annotations"
	"kubevirt.io/kubevirt/pkg/storage/velero"
)

var _ = Describe("Annotations Generator", func() {
	const (
		testNamespace = "testns"
		vmiName       = "testvmi"
	)

	const (
		expectedPreHookBackupCommand  = `["/usr/bin/virt-freezer", "--freeze", "--name", "testvmi", "--namespace", "testns"]`
		expectedPostHookBackupCommand = `["/usr/bin/virt-freezer", "--unfreeze", "--name", "testvmi", "--namespace", "testns"]`
	)

	expectedAnnotations := map[string]string{
		"pre.hook.backup.velero.io/container":  "compute",
		"pre.hook.backup.velero.io/command":    expectedPreHookBackupCommand,
		"post.hook.backup.velero.io/container": "compute",
		"post.hook.backup.velero.io/command":   expectedPostHookBackupCommand,
		"pre.hook.backup.velero.io/timeout":    "60s",
	}

	It("Should generate storage annotations", func() {
		generator := annotations.Generator{}
		annotations, err := generator.Generate(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName)))
		Expect(err).NotTo(HaveOccurred())
		Expect(annotations).To(Equal(expectedAnnotations))
	})

	DescribeTable("Should not generate storage annotations when skip annotation is set to true",
		func(skipValue string) {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName))
			vmi.Annotations = map[string]string{
				velero.SkipHooksAnnotation: skipValue,
			}

			generator := annotations.Generator{}
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(annotations).To(BeEmpty())
		},
		Entry("with lowercase 'true'", "true"),
		Entry("with capitalized 'True'", "True"),
		Entry("with uppercase 'TRUE'", "TRUE"),
		Entry("with '1'", "1"),
		Entry("with lowercase 't'", "t"),
		Entry("with uppercase 'T'", "T"),
	)

	DescribeTable("Should generate storage annotations when skip annotation is set to false",
		func(skipValue string) {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName))
			vmi.Annotations = map[string]string{
				velero.SkipHooksAnnotation: skipValue,
			}

			generator := annotations.Generator{}
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(annotations).To(Equal(expectedAnnotations))
		},
		Entry("with lowercase 'false'", "false"),
		Entry("with capitalized 'False'", "False"),
		Entry("with uppercase 'FALSE'", "FALSE"),
		Entry("with '0'", "0"),
		Entry("with lowercase 'f'", "f"),
		Entry("with uppercase 'F'", "F"),
		Entry("with invalid value", "invalid"),
	)
})
