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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package virthandler

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	originalIP  = "1.1.1.1"
	migrationIP = "2.2.2.2"

	mainNetwork = `{
    "name": "k8s-pod-network",
    "ips": [
        "` + originalIP + `"
    ],
    "default": true,
    "dns": {}
}`

	migrationNetwork = `{
    "name": "migration-bridge",
    "interface": "migration0",
    "ips": [
        "` + migrationIP + `"
    ],
    "mac": "ae:33:70:a7:3a:8c",
    "dns": {}
}`
)

var _ = Describe("virt-handler", func() {
	Context("findMigrationIp", func() {
		It("Should error on missing file", func() {
			_, err := FindMigrationIP("/not-a-real-file", originalIP)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read network status from downwards API"))
		})
		It("Should handle the empty file case", func() {
			file, err := ioutil.TempFile("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(file.Name())
			newIP, err := FindMigrationIP(file.Name(), originalIP)
			Expect(err).ToNot(HaveOccurred())
			Expect(newIP).To(Equal(originalIP))
		})
		It("Should return the original IP if migration0 doesn't exist", func() {
			file, err := ioutil.TempFile("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(file.Name())
			err = os.WriteFile(file.Name(), []byte(`[`+mainNetwork+`]`), 0644)
			Expect(err).ToNot(HaveOccurred())
			newIP, err := FindMigrationIP(file.Name(), originalIP)
			Expect(err).ToNot(HaveOccurred())
			Expect(newIP).To(Equal(originalIP))
		})
		It("Should return the migration IP if migration0 exists", func() {
			file, err := ioutil.TempFile("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(file.Name())
			err = os.WriteFile(file.Name(), []byte(`[`+mainNetwork+`,`+migrationNetwork+`]`), 0644)
			Expect(err).ToNot(HaveOccurred())
			newIP, err := FindMigrationIP(file.Name(), originalIP)
			Expect(err).ToNot(HaveOccurred())
			Expect(newIP).To(Equal(migrationIP))
		})
	})
})
