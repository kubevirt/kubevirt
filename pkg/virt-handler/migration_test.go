/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
