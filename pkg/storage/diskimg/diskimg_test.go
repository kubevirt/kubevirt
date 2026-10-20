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

package diskimg

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/libvmi"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Creator", func() {
	const (
		size64Mi  = int64(64 << 20)
		size80Mi  = int64(80 << 20)
		size128Mi = int64(128 << 20)
	)

	var (
		tempDir            string
		recorder           *record.FakeRecorder
		creator            Creator
		creatorWithReserve Creator
	)

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		root, err := safepath.JoinAndResolveWithRelativeRoot("/")
		Expect(err).NotTo(HaveOccurred())

		creator = NewCreator(recorder, 0, 0, root)
		creatorWithReserve = NewCreator(recorder, 10, 1048576, root)
	})

	image := func(volumeName string, size int64) Image {
		dir := filepath.Join(tempDir, volumeName)
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())
		return Image{Dir: dir, Name: "disk.img", Size: size}
	}

	imagePath := func(img Image) string {
		return filepath.Join(img.Dir, img.Name)
	}

	Context("with non existing images", func() {
		It("should create images if there is enough space", func() {
			images := []Image{image("volume1", size64Mi), image("volume2", size128Mi), image("volume3", size80Mi)}

			Expect(creator.Create(libvmi.New(), images)).To(Succeed())

			for _, img := range images {
				info, err := os.Stat(imagePath(img))
				Expect(err).NotTo(HaveOccurred())
				Expect(info.Size()).To(Equal(img.Size))
			}
		})

		It("should stop creating images if there is not enough space and return an error", func() {
			images := []Image{image("volume1", size64Mi), image("volume2", 1<<60), image("volume3", size128Mi)}

			Expect(creator.Create(libvmi.New(), images)).NotTo(Succeed())

			info, err := os.Stat(imagePath(images[0]))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Size()).To(Equal(size64Mi))
			Expect(imagePath(images[1])).NotTo(BeAnExistingFile())
			Expect(imagePath(images[2])).NotTo(BeAnExistingFile())
		})

		It("should not subtract reserve if there is enough space for the requested size", func() {
			img := image("volume1", size64Mi)

			Expect(creatorWithReserve.Create(libvmi.New(), []Image{img})).To(Succeed())

			info, err := os.Stat(imagePath(img))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Size()).To(Equal(size64Mi))
		})

		It("should subtract reserve if there is not enough space for the requested size", func() {
			img := image("volume1", size64Mi)
			dirAvailable := uint64(size64Mi)
			creatorWithReserve.dirBytesAvailableFunc = func(_ *safepath.Path, reserve uint64) (uint64, error) {
				return dirAvailable - reserve, nil
			}

			Expect(creatorWithReserve.Create(libvmi.New(), []Image{img})).To(Succeed())

			info, err := os.Stat(imagePath(img))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Size()).To(BeNumerically("==", dirAvailable-creatorWithReserve.minimumPVCReserveBytes))
		})

		It("should refuse to create an image if reserve exceeds lessPVCSpaceToleration", func() {
			img := image("volume1", size64Mi)
			dirAvailable := uint64(size64Mi)
			creatorWithReserve.dirBytesAvailableFunc = func(_ *safepath.Path, reserve uint64) (uint64, error) {
				return dirAvailable - reserve, nil
			}
			creatorWithReserve.lessPVCSpaceToleration = 1

			err := creatorWithReserve.Create(libvmi.New(), []Image{img})
			Expect(err).To(MatchError(ContainSubstring("unable to create")))
			Expect(imagePath(img)).NotTo(BeAnExistingFile())
		})

		It("should refuse to create an image if reserve exceeds free space", func() {
			root, err := safepath.JoinAndResolveWithRelativeRoot("/")
			Expect(err).NotTo(HaveOccurred())
			img := image("volume1", size64Mi)

			c := NewCreator(recorder, 10, math.MaxUint64, root)
			err = c.Create(libvmi.New(), []Image{img})
			Expect(err).To(MatchError(ContainSubstring("not enough space")))
			Expect(imagePath(img)).NotTo(BeAnExistingFile())
		})

		It("should take lessPVCSpaceToleration into account", func() {
			images := []Image{image("volume1", size64Mi), image("volume2", size64Mi), image("volume3", size64Mi)}
			const toleration = 5
			creator.lessPVCSpaceToleration = toleration

			toleratedSize := func(diff int) uint64 {
				return uint64(size64Mi) * (100 - toleration + uint64(diff)) / 100
			}
			creator.dirBytesAvailableFunc = func(dir *safepath.Path, _ uint64) (uint64, error) {
				switch base, _ := dir.Base(); base {
				case "volume1":
					return toleratedSize(1), nil
				case "volume2":
					return toleratedSize(0), nil
				case "volume3":
					return toleratedSize(-1), nil
				}
				return 0, fmt.Errorf("unexpected dir %s", dir)
			}

			Expect(creator.Create(libvmi.New(), images)).NotTo(Succeed())

			info, err := os.Stat(imagePath(images[0]))
			Expect(err).NotTo(HaveOccurred())
			Expect(uint64(info.Size())).To(Equal(toleratedSize(1)))
			info, err = os.Stat(imagePath(images[1]))
			Expect(err).NotTo(HaveOccurred())
			Expect(uint64(info.Size())).To(Equal(toleratedSize(0)))
			Expect(imagePath(images[2])).NotTo(BeAnExistingFile())

			testutils.ExpectEvent(recorder, "PV size too small")
		})
	})

	Context("with an existing image", func() {
		AfterEach(func() {
			ephemeraldiskutils.MockDefaultOwnershipManager()
		})

		It("should not re-create or chown it", func() {
			ephemeraldiskutils.MockDefaultOwnershipManagerWithFailure()
			existing := image("volume1", size64Mi)
			dir, err := safepath.JoinAndResolveWithRelativeRoot(existing.Dir)
			Expect(err).NotTo(HaveOccurred())
			_, err = createSparseRaw(dir, existing.Name, existing.Size)
			Expect(err).NotTo(HaveOccurred())
			before, err := os.Stat(imagePath(existing))
			Expect(err).NotTo(HaveOccurred())

			Expect(creator.Create(libvmi.New(), []Image{{Dir: existing.Dir, Name: existing.Name, Size: size128Mi}})).To(Succeed())

			after, err := os.Stat(imagePath(existing))
			Expect(err).NotTo(HaveOccurred())
			Expect(after.ModTime()).To(Equal(before.ModTime()))
			Expect(after.Size()).To(Equal(size64Mi))
		})
	})
})
