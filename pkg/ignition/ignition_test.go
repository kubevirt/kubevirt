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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package ignition

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var tmpDir string

var _ = BeforeSuite(func() {
	var err error
	tmpDir, err = os.MkdirTemp("", "ignitiontest")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(os.RemoveAll, tmpDir)

	err = SetLocalDirectory(tmpDir)
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Ignition", func() {
	const vmName = "my-vm"
	const namespace = "my-namespace"

	// const ignitionLocalDir = "/var/run/libvirt/ignition-dir"
	var vmi *v1.VirtualMachineInstance

	Describe("A new VirtualMachineInstance definition", func() {
		Context("with ignition data", func() {
			It("should success", func() {
				data := "{ \"ignition\": { \"config\": {}, \"version\": \"2.2.0\" }, \"networkd\": {}, \"storage\": { \"files\": [ { \"contents\": { \"source\": \"data:,test\", \"verification\": {} }, \"filesystem\": \"root\", \"mode\": 420, \"path\": \"/etc/hostname\" } ] }, \"systemd\": {} }"
				vmi = libvmi.New(
					libvmi.WithNamespace(namespace),
					libvmi.WithName(vmName),
					libvmi.WithAnnotation(v1.IgnitionAnnotation, data),
				)
				err := GenerateIgnitionLocalData(vmi, namespace)
				Expect(err).ToNot(HaveOccurred())
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s/%s", tmpDir, namespace, vmName, IgnitionFile))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
