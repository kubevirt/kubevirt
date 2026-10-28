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

package virthandler

import (
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/kubevirt/pkg/safepath"
)

var _ = Describe("prepareVFIO", func() {
	var (
		controller *BaseController
	)

	BeforeEach(func() {
		controller = &BaseController{}
	})

	mountRootOf := func(dir string) *safepath.Path {
		root, err := safepath.JoinAndResolveWithRelativeRoot(dir)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	It("succeeds when the VFIO directory does not exist", func() {
		Expect(controller.prepareVFIO(mountRootOf(GinkgoT().TempDir()))).To(Succeed())
	})

	It("succeeds when the VFIO control device does not exist", func() {
		tempDir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(tempDir, "dev", "vfio"), 0777)).To(Succeed())
		Expect(controller.prepareVFIO(mountRootOf(tempDir))).To(Succeed())
	})

	It("returns the error when the VFIO directory is not a directory", func() {
		tempDir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(tempDir, "dev"), 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tempDir, "dev", "vfio"), nil, 0666)).To(Succeed())
		Expect(controller.prepareVFIO(mountRootOf(tempDir))).To(MatchError(syscall.ENOTDIR))
	})
})
