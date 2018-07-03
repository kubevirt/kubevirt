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

package network

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Common Methods", func() {
	Context("Functions Read and Write from cache", func() {
		It("should persist interface payload", func() {
			tmpDir, _ := ioutil.TempDir("", "commontest")
			setInterfaceCacheFile(tmpDir + "/cache-%s.json")

			ifaceName := "iface_name"
			iface := api.Interface{Type: "fake_type", Source: api.InterfaceSource{Bridge: "fake_br"}}
			err := writeToCachedFile(&iface, interfaceCacheFile, ifaceName)
			Expect(err).ToNot(HaveOccurred())

			var cached_iface api.Interface
			isExist, err := readFromCachedFile(ifaceName, interfaceCacheFile, &cached_iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(isExist).To(Equal(true))

			Expect(iface).To(Equal(cached_iface))
		})
		It("should persist qemu arg payload", func() {
			tmpDir, _ := ioutil.TempDir("", "commontest")
			setInterfaceCacheFile(tmpDir + "/cache-%s.json")

			qemuArgName := "iface_name"
			qemuArg := api.Arg{Value: "test_value"}
			err := writeToCachedFile(&qemuArg, interfaceCacheFile, qemuArgName)
			Expect(err).ToNot(HaveOccurred())

			var cached_qemuArg api.Arg
			isExist, err := readFromCachedFile(qemuArgName, interfaceCacheFile, &cached_qemuArg)
			Expect(err).ToNot(HaveOccurred())
			Expect(isExist).To(Equal(true))

			Expect(qemuArg).To(Equal(cached_qemuArg))
		})
	})
})
