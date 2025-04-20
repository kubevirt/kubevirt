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

package certificates_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/certificates"
)

var _ = Describe("Certificates", func() {

	var certDir string

	BeforeEach(func() {
		var err error
		certDir, err = os.MkdirTemp("", "certsdir")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should be generated in temporary directory", func() {
		store, err := certificates.GenerateSelfSignedCert(certDir, "testname", "testnamespace")
		Expect(err).ToNot(HaveOccurred())
		_, err = store.Current()
		Expect(err).ToNot(HaveOccurred())
		Expect(store.CurrentPath()).To(ContainSubstring(certDir))
	})

	AfterEach(func() {
		os.RemoveAll(certDir)
	})
})
