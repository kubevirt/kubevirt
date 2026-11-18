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
	"errors"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("prepareVFIO", func() {
	var (
		res        *isolation.MockIsolationResult
		controller *BaseController
	)

	BeforeEach(func() {
		res = isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
		controller = &BaseController{}
	})

	mountRootOf := func(dir string) *safepath.Path {
		root, err := safepath.JoinAndResolveWithRelativeRoot(dir)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	It("returns the error when the mount root cannot be resolved", func() {
		res.EXPECT().MountRoot().Return(nil, errors.New("mount root is unavailable"))

		Expect(controller.prepareVFIO(res)).To(MatchError("mount root is unavailable"))
	})

	It("succeeds when the VFIO directory does not exist", func() {
		res.EXPECT().MountRoot().Return(mountRootOf(GinkgoT().TempDir()), nil)

		Expect(controller.prepareVFIO(res)).To(Succeed())
	})

	It("succeeds when the VFIO control device does not exist", func() {
		tempDir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(tempDir, "dev", "vfio"), 0777)).To(Succeed())
		res.EXPECT().MountRoot().Return(mountRootOf(tempDir), nil)

		Expect(controller.prepareVFIO(res)).To(Succeed())
	})

	It("returns the error when the VFIO directory is not a directory", func() {
		tempDir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(tempDir, "dev"), 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tempDir, "dev", "vfio"), nil, 0666)).To(Succeed())
		res.EXPECT().MountRoot().Return(mountRootOf(tempDir), nil)

		Expect(controller.prepareVFIO(res)).To(MatchError(syscall.ENOTDIR))
	})
})
