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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/pod/annotations"
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

	It("Should generate storage annotations when config is nil", func() {
		generator := annotations.NewGenerator(nil)
		annotations, err := generator.Generate(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName)))
		Expect(err).NotTo(HaveOccurred())

		expectedAnnotations := map[string]string{
			"pre.hook.backup.velero.io/container":  "compute",
			"pre.hook.backup.velero.io/command":    expectedPreHookBackupCommand,
			"post.hook.backup.velero.io/container": "compute",
			"post.hook.backup.velero.io/command":   expectedPostHookBackupCommand,
		}

		Expect(annotations).To(Equal(expectedAnnotations))
	})

	It("Should generate storage annotations when DisableVeleroHooks is not set", func() {
		clusterConfig := &v1.KubeVirtConfiguration{}
		generator := annotations.NewGenerator(clusterConfig)
		annotations, err := generator.Generate(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName)))
		Expect(err).NotTo(HaveOccurred())

		expectedAnnotations := map[string]string{
			"pre.hook.backup.velero.io/container":  "compute",
			"pre.hook.backup.velero.io/command":    expectedPreHookBackupCommand,
			"post.hook.backup.velero.io/container": "compute",
			"post.hook.backup.velero.io/command":   expectedPostHookBackupCommand,
		}

		Expect(annotations).To(Equal(expectedAnnotations))
	})

	It("Should not generate storage annotations when DisableVeleroHooks is set", func() {
		clusterConfig := &v1.KubeVirtConfiguration{
			VirtualMachineOptions: &v1.VirtualMachineOptions{
				DisableVeleroHooks: &v1.DisableVeleroHooks{},
			},
		}
		generator := annotations.NewGenerator(clusterConfig)
		annotations, err := generator.Generate(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(vmiName)))
		Expect(err).NotTo(HaveOccurred())
		Expect(annotations).To(BeEmpty())
	})
})
