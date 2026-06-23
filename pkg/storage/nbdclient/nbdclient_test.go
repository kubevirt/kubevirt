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

package nbdclient

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libguestfs.org/libnbd"

	nbdv1 "kubevirt.io/kubevirt/pkg/storage/cbt/nbd/v1"
)

var _ = Describe("NBDClient", func() {
	Context("allocDescription", func() {
		DescribeTable("should return correct description",
			func(flags uint64, expected string) {
				Expect(allocDescription(flags)).To(Equal(expected))
			},
			Entry("data", uint64(0), "data"),
			Entry("hole", uint64(libnbd.STATE_HOLE), "hole"),
			Entry("zero", uint64(libnbd.STATE_ZERO), "zero"),
			Entry("hole,zero", uint64(libnbd.STATE_HOLE|libnbd.STATE_ZERO), "hole,zero"),
			Entry("unknown flags", uint64(99), "unknown"),
		)
	})

	Context("mergedDescription", func() {
		DescribeTable("should return correct description",
			func(flags uint64, expected string) {
				Expect(mergedDescription(flags)).To(Equal(expected))
			},
			Entry("clean", uint64(0), "clean"),
			Entry("dirty", uint64(libnbd.STATE_DIRTY), "dirty"),
			Entry("zero", uint64(libnbd.STATE_ZERO), "zero"),
			Entry("dirty,zero", uint64(libnbd.STATE_DIRTY)|uint64(libnbd.STATE_ZERO), "dirty,zero"),
			Entry("unknown flags", uint64(99), "unknown"),
		)
	})

	Context("clampLength", func() {
		DescribeTable("should calculate length",
			func(offset, length, size, expectedLength uint64) {
				got, err := clampLength(offset, length, size)
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(expectedLength))
			},
			Entry("with range within export",
				uint64(0), uint64(512), uint64(1024), uint64(512)),
			Entry("with zero length",
				uint64(256), uint64(0), uint64(1024), uint64(768)),
			Entry("with length overshooting size",
				uint64(768), uint64(512), uint64(1024), uint64(256)),
			Entry("with full export from start",
				uint64(0), uint64(1024), uint64(1024), uint64(1024)),
		)

		DescribeTable("should error out",
			func(offset, length, size uint64) {
				_, err := clampLength(offset, length, size)
				Expect(err).To(HaveOccurred())
			},
			Entry("with offset equals size", uint64(1024), uint64(0), uint64(1024)),
			Entry("with offset beyond size", uint64(2000), uint64(10), uint64(1024)),
		)
	})

	Context("computeChunks", func() {
		It("should produce a single chunk when length <= chunkSize", func() {
			chunks := computeChunks(0, 100, 256)
			Expect(chunks).To(HaveLen(1))
			Expect(chunks[0]).To(Equal(readChunk{offset: 0, length: 100}))
		})

		It("should split length evenly into multiple chunks", func() {
			chunks := computeChunks(0, 1024, 256)
			Expect(chunks).To(HaveLen(4))
			for i, c := range chunks {
				Expect(c.offset).To(Equal(uint64(i * 256)))
				Expect(c.length).To(Equal(uint64(256)))
			}
		})

		It("should handle a remainder in the last chunk", func() {
			chunks := computeChunks(0, 300, 256)
			Expect(chunks).To(HaveLen(2))
			Expect(chunks[0]).To(Equal(readChunk{offset: 0, length: 256}))
			Expect(chunks[1]).To(Equal(readChunk{offset: 256, length: 44}))
		})

		It("should respect a non-zero starting offset", func() {
			chunks := computeChunks(512, 256, 256)
			Expect(chunks).To(HaveLen(1))
			Expect(chunks[0]).To(Equal(readChunk{offset: 512, length: 256}))
		})

		It("should return no chunks for zero length", func() {
			Expect(computeChunks(0, 0, 256)).To(BeEmpty())
		})
	})

	Context("singleContextMapper", func() {
		var (
			sent    []*nbdv1.MapResponse
			builder *singleContextMapper
		)

		sendFn := func(r *nbdv1.MapResponse) error {
			extCopy := make([]*nbdv1.Extent, len(r.Extents))
			copy(extCopy, r.Extents)
			sent = append(sent, &nbdv1.MapResponse{Extents: extCopy, NextOffset: r.NextOffset})
			return nil
		}

		BeforeEach(func() {
			sent = nil
		})

		Context("HandleExtents and coalescing", func() {
			It("should coalesce adjacent extents with the same flags", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(builder.batch).To(BeEmpty(), "coalesced extent should not be flushed yet")

				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(HaveLen(1))
				Expect(sent[0].Extents[0].Length).To(Equal(uint64(512)))
			})

			It("should not coalesce adjacent extents with different flags", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: uint64(libnbd.STATE_HOLE)},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(HaveLen(1))
				Expect(sent[0].Extents).To(HaveLen(2))
				Expect(sent[0].Extents[0].Flags).To(Equal(uint64(0)))
				Expect(sent[0].Extents[1].Flags).To(Equal(uint64(libnbd.STATE_HOLE)))
			})

			It("should clip extents that extend beyond endOffset", func() {
				builder = newSingleContextMapper(300, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(builder.Flush()).To(Succeed())
				Expect(sent[0].Extents[0].Length).To(Equal(uint64(300)))
			})

			It("should skip zero-length extents after clipping", func() {
				builder = newSingleContextMapper(256, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 256, []libnbd.LibnbdExtent{
					{Length: 128, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(BeEmpty())
			})

			It("should return the highest offset advanced by entries", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				maxOffset, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: uint64(libnbd.STATE_HOLE)},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(maxOffset).To(Equal(uint64(512)))
			})
		})

		Context("Flush and batching", func() {
			It("should send a batch when batchSize is reached and flush the trailing extent", func() {
				builder = newSingleContextMapper(4096, 2, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: uint64(libnbd.STATE_HOLE)},
					{Length: 256, Flags: uint64(libnbd.STATE_ZERO)},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(sent).To(HaveLen(1), "one batch should already have been sent")
				Expect(sent[0].Extents).To(HaveLen(2))

				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(HaveLen(2), "trailing extent should be flushed")
				Expect(sent[1].Extents).To(HaveLen(1))
				Expect(sent[1].Extents[0].Flags).To(Equal(uint64(libnbd.STATE_ZERO)))
			})

			It("should be a no-op when the batch is empty", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(BeEmpty())
			})

			It("should send remaining extents", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(sent).To(BeEmpty(), "nothing sent before Flush")

				Expect(builder.Flush()).To(Succeed())
				Expect(sent).To(HaveLen(1))
			})

			It("should set NextOffset to end of last extent in the batch", func() {
				builder = newSingleContextMapper(1024, 512, sendFn)
				ctx := libnbd.CONTEXT_BASE_ALLOCATION

				_, err := builder.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 512, Flags: uint64(libnbd.STATE_HOLE)},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(builder.Flush()).To(Succeed())
				Expect(sent[0].NextOffset).To(Equal(uint64(768)))
			})
		})
	})

	Context("mergedContextMapper", func() {
		var (
			sent   []*nbdv1.MapResponse
			merger *mergedContextMapper
		)

		mergeSendFn := func(r *nbdv1.MapResponse) error {
			extCopy := make([]*nbdv1.Extent, len(r.Extents))
			copy(extCopy, r.Extents)
			sent = append(sent, &nbdv1.MapResponse{Extents: extCopy, NextOffset: r.NextOffset})
			return nil
		}

		BeforeEach(func() {
			sent = nil
		})

		allExtents := func() []*nbdv1.Extent {
			var result []*nbdv1.Extent
			for _, r := range sent {
				result = append(result, r.Extents...)
			}
			return result
		}

		Context("HandleExtents", func() {
			It("should buffer allocation extents separately from dirty extents", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(merger.allocExtents).To(HaveLen(1))
				Expect(merger.dirtyExtents).To(HaveLen(1))
			})

			It("should clip extents beyond endOffset", func() {
				merger = newMergedContextMapper(300, 512, mergeSendFn)

				maxOff, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(maxOff).To(Equal(uint64(300)))
				Expect(merger.allocExtents[0].Length).To(Equal(uint64(300)))
			})

			It("should skip zero-length extents after clipping", func() {
				merger = newMergedContextMapper(256, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 256, []libnbd.LibnbdExtent{
					{Length: 128, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.allocExtents).To(BeEmpty())
			})
		})

		Context("Merge", func() {
			It("should merge aligned extents with combined flags", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
					{Length: 512, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(2))
				Expect(extents[0].Flags).To(Equal(uint64(libnbd.STATE_DIRTY)))
				Expect(extents[0].Description).To(Equal("dirty"))
				Expect(extents[0].Length).To(Equal(uint64(512)))
				Expect(extents[1].Flags).To(Equal(uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO)))
				Expect(extents[1].Description).To(Equal("dirty,zero"))
				Expect(extents[1].Length).To(Equal(uint64(512)))
			})

			It("should split at misaligned boundaries", func() {
				merger = newMergedContextMapper(16384, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 8192, Flags: 0},
					{Length: 8192, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 16384, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(2))
				Expect(extents[0].Offset).To(Equal(uint64(0)))
				Expect(extents[0].Length).To(Equal(uint64(8192)))
				Expect(extents[0].Flags).To(Equal(uint64(libnbd.STATE_DIRTY)))
				Expect(extents[0].Description).To(Equal("dirty"))
				Expect(extents[1].Offset).To(Equal(uint64(8192)))
				Expect(extents[1].Length).To(Equal(uint64(8192)))
				Expect(extents[1].Flags).To(Equal(uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO)))
				Expect(extents[1].Description).To(Equal("dirty,zero"))
			})

			It("should coalesce adjacent merged extents with the same flags", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: 0},
					{Length: 512, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(1))
				Expect(extents[0].Length).To(Equal(uint64(1024)))
				Expect(extents[0].Flags).To(Equal(uint64(libnbd.STATE_DIRTY)))
			})

			It("should produce clean extents for non-dirty allocated data", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(1))
				Expect(extents[0].Flags).To(Equal(uint64(0)))
				Expect(extents[0].Description).To(Equal("clean"))
			})

			It("should produce zero extents for non-dirty holes", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(1))
				Expect(extents[0].Flags).To(Equal(uint64(libnbd.STATE_ZERO)))
				Expect(extents[0].Description).To(Equal("zero"))
			})

			It("should correctly handle a block discard", func() {
				merger = newMergedContextMapper(32768, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 4096, Flags: 0},
					{Length: 24576, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
					{Length: 4096, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 32768, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(3))

				Expect(extents[0].Offset).To(Equal(uint64(0)))
				Expect(extents[0].Length).To(Equal(uint64(4096)))
				Expect(extents[0].Description).To(Equal("dirty"))

				Expect(extents[1].Offset).To(Equal(uint64(4096)))
				Expect(extents[1].Length).To(Equal(uint64(24576)))
				Expect(extents[1].Description).To(Equal("dirty,zero"))

				Expect(extents[2].Offset).To(Equal(uint64(28672)))
				Expect(extents[2].Length).To(Equal(uint64(4096)))
				Expect(extents[2].Description).To(Equal("dirty"))
			})

			It("should trigger batch send when batchSize is reached", func() {
				merger = newMergedContextMapper(4096, 2, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
					{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
					{Length: 1024, Flags: 0},
					{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 4096, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(merger.Merge()).To(Succeed())
				Expect(sent).To(HaveLen(1), "one batch should have been sent mid-merge")
				Expect(sent[0].Extents).To(HaveLen(2))

				Expect(merger.Flush()).To(Succeed())
				extents := allExtents()
				Expect(extents).To(HaveLen(4))
			})

			It("should be a no-op when one context is empty", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})

				Expect(merger.Merge()).To(Succeed())
				Expect(merger.Flush()).To(Succeed())
				Expect(sent).To(BeEmpty())
			})

			It("should be a no-op when both contexts are empty", func() {
				merger = newMergedContextMapper(1024, 512, mergeSendFn)
				Expect(merger.Merge()).To(Succeed())
			})

			It("should handle multiple Merge calls with coalescing across calls", func() {
				merger = newMergedContextMapper(2048, 512, mergeSendFn)

				_, err := merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(merger.Merge()).To(Succeed())

				_, err = merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 1024, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = merger.HandleExtents(libnbd.CONTEXT_QEMU_DIRTY_BITMAP+"checkpoint", 1024, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(merger.Merge()).To(Succeed())

				Expect(merger.Flush()).To(Succeed())

				extents := allExtents()
				Expect(extents).To(HaveLen(1))
				Expect(extents[0].Length).To(Equal(uint64(2048)))
				Expect(extents[0].Description).To(Equal("dirty"))
			})
		})
	})

	Context("readProcessor", func() {
		It("should read each chunk and send it in order", func() {
			var sentChunks []*nbdv1.DataChunk
			proc := &readProcessor{
				pread: func(buf []byte, offset uint64) error {
					for i := range buf {
						buf[i] = byte(offset)
					}
					return nil
				},
				send: func(c *nbdv1.DataChunk) error {
					cp := &nbdv1.DataChunk{Offset: c.Offset, Data: append([]byte(nil), c.Data...)}
					sentChunks = append(sentChunks, cp)
					return nil
				},
			}

			chunks := []readChunk{{offset: 0, length: 4}, {offset: 4, length: 4}}
			Expect(proc.Process(chunks)).To(Succeed())
			Expect(sentChunks).To(HaveLen(2))
			Expect(sentChunks[0].Offset).To(Equal(uint64(0)))
			Expect(sentChunks[1].Offset).To(Equal(uint64(4)))
			Expect(sentChunks[0].Data).To(Equal([]byte{0, 0, 0, 0}))
			Expect(sentChunks[1].Data).To(Equal([]byte{4, 4, 4, 4}))
		})

		It("should return an error when pread fails", func() {
			proc := &readProcessor{
				pread: func(buf []byte, offset uint64) error {
					return fmt.Errorf("disk error at %d", offset)
				},
				send: func(*nbdv1.DataChunk) error { return nil },
			}
			err := proc.Process([]readChunk{{offset: 0, length: 4}})
			Expect(err).To(MatchError(ContainSubstring("pread failed at offset 0")))
		})

		It("should return an error when send fails", func() {
			proc := &readProcessor{
				pread: func(buf []byte, offset uint64) error { return nil },
				send:  func(*nbdv1.DataChunk) error { return errors.New("stream closed") },
			}
			err := proc.Process([]readChunk{{offset: 0, length: 4}})
			Expect(err).To(MatchError(ContainSubstring("failed to send chunk at offset 0")))
		})
	})
})
