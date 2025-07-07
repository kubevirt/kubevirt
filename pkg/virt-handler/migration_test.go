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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	originalIP = "1.1.1.1"
)

var _ = Describe("virt-handler", func() {
	Context("findMigrationIp", func() {
		It("Should return the IP passed to it when no migration0 interface exists", func() {
			newIp, err := FindMigrationIP(originalIP)
			Expect(err).NotTo(HaveOccurred())
			Expect(newIp).To(Equal(originalIP))
		})
	})
})
