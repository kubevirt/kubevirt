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
	"bytes"
	"context"
	"errors"
	"io"
	"iter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libguestfs.org/libnbd"
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

	flushed := func(handler mapHandler) Extent {
		last, ok := handler.Flush()
		Expect(ok).To(BeTrue(), "expected a trailing extent")
		return last
	}

	drain := func(handler mapHandler) []Extent {
		extents, _ := handler.Merge()
		if last, ok := handler.Flush(); ok {
			extents = append(extents, last)
		}
		return extents
	}

	Context("singleContextMapper", func() {
		const ctx = libnbd.CONTEXT_BASE_ALLOCATION

		Context("HandleExtents and coalescing", func() {
			It("should coalesce adjacent extents with the same flags", func() {
				mapper := newSingleContextMapper(1024)

				mapper.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: 0},
				})
				extents, _ := mapper.Merge()
				Expect(extents).To(BeEmpty(), "coalesced extent should not be completed yet")

				Expect(flushed(mapper)).To(Equal(Extent{Offset: 0, Length: 512, Flags: 0, Description: "data"}))
			})

			It("should not coalesce adjacent extents with different flags", func() {
				mapper := newSingleContextMapper(1024)

				mapper.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: uint64(libnbd.STATE_HOLE)},
				})

				Expect(drain(mapper)).To(Equal([]Extent{
					{Offset: 0, Length: 256, Flags: 0, Description: "data"},
					{Offset: 256, Length: 256, Flags: uint64(libnbd.STATE_HOLE), Description: "hole"},
				}))
			})

			It("should clip extents that extend beyond endOffset", func() {
				mapper := newSingleContextMapper(300)

				mapper.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})

				extents := drain(mapper)
				Expect(extents).To(HaveLen(1))
				Expect(extents[0].Length).To(Equal(uint64(300)))
			})

			It("should skip zero-length extents after clipping", func() {
				mapper := newSingleContextMapper(256)

				mapper.HandleExtents(ctx, 256, []libnbd.LibnbdExtent{
					{Length: 128, Flags: 0},
				})

				Expect(drain(mapper)).To(BeEmpty())
			})

			It("should report the highest offset advanced by entries on Merge", func() {
				mapper := newSingleContextMapper(1024)

				mapper.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: uint64(libnbd.STATE_HOLE)},
				})

				_, end := mapper.Merge()
				Expect(end).To(Equal(uint64(512)))
			})
		})

		Context("Flush", func() {
			It("should return nothing when there are no extents", func() {
				_, ok := newSingleContextMapper(1024).Flush()
				Expect(ok).To(BeFalse())
			})

			It("should return the trailing extent only once", func() {
				mapper := newSingleContextMapper(1024)
				mapper.HandleExtents(ctx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})

				_, ok := mapper.Flush()
				Expect(ok).To(BeTrue())
				_, ok = mapper.Flush()
				Expect(ok).To(BeFalse())
			})
		})
	})

	Context("mergedContextMapper", func() {
		const dirtyCtx = libnbd.CONTEXT_QEMU_DIRTY_BITMAP + "checkpoint"

		Context("HandleExtents", func() {
			It("should buffer allocation extents separately from dirty extents", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(merger.allocExtents).To(HaveLen(1))
				Expect(merger.dirtyExtents).To(HaveLen(1))
			})

			It("should clip extents beyond endOffset", func() {
				merger := newMergedContextMapper(300)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
				})
				Expect(merger.allocExtents[0].Length).To(Equal(uint64(300)))
			})

			It("should skip zero-length extents after clipping", func() {
				merger := newMergedContextMapper(256)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 256, []libnbd.LibnbdExtent{
					{Length: 128, Flags: 0},
				})

				Expect(merger.allocExtents).To(BeEmpty())
			})
		})

		Context("Merge", func() {
			It("should merge aligned extents with combined flags", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: 0},
					{Length: 512, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
					{Length: 512, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 512, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
					{Offset: 512, Length: 512, Flags: uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO), Description: "dirty,zero"},
				}))
			})

			It("should split at misaligned boundaries", func() {
				merger := newMergedContextMapper(16384)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 8192, Flags: 0},
					{Length: 8192, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 16384, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 8192, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
					{Offset: 8192, Length: 8192, Flags: uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO), Description: "dirty,zero"},
				}))
			})

			It("should coalesce adjacent merged extents with the same flags", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 256, Flags: 0},
					{Length: 256, Flags: 0},
					{Length: 512, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 1024, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
				}))
			})

			It("should produce clean extents for non-dirty allocated data", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 1024, Flags: 0, Description: "clean"},
				}))
			})

			It("should produce zero extents for non-dirty holes", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 1024, Flags: uint64(libnbd.STATE_ZERO), Description: "zero"},
				}))
			})

			It("should correctly handle a block discard", func() {
				merger := newMergedContextMapper(32768)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 4096, Flags: 0},
					{Length: 24576, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
					{Length: 4096, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 32768, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 4096, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
					{Offset: 4096, Length: 24576, Flags: uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO), Description: "dirty,zero"},
					{Offset: 28672, Length: 4096, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
				}))
			})

			It("should complete every extent but the trailing one", func() {
				merger := newMergedContextMapper(4096)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
					{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)},
					{Length: 1024, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 3072, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				extents, _ := merger.Merge()
				Expect(extents).To(HaveLen(2))
				Expect(flushed(merger)).To(HaveField("Offset", uint64(2048)))
			})

			It("should merge nothing and report no progress when one context is empty", func() {
				merger := newMergedContextMapper(1024)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})

				extents, end := merger.Merge()
				Expect(extents).To(BeEmpty())
				Expect(end).To(BeZero())
				_, ok := merger.Flush()
				Expect(ok).To(BeFalse())
			})

			It("should be a no-op when both contexts are empty", func() {
				extents, end := newMergedContextMapper(1024).Merge()
				Expect(extents).To(BeEmpty())
				Expect(end).To(BeZero())
			})

			DescribeTable("should only merge the range both contexts describe",
				func(allocLength, dirtyLength, expectedEnd uint64) {
					merger := newMergedContextMapper(4096)

					merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
						{Length: allocLength, Flags: 0},
					})
					merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
						{Length: dirtyLength, Flags: uint64(libnbd.STATE_DIRTY)},
					})

					_, end := merger.Merge()
					Expect(end).To(Equal(expectedEnd))
					Expect(merger.allocExtents).To(BeEmpty(), "the undescribed rest must be dropped, not carried over")
					Expect(merger.dirtyExtents).To(BeEmpty(), "the undescribed rest must be dropped, not carried over")
					Expect(flushed(merger)).To(Equal(Extent{Offset: 0, Length: expectedEnd, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"}))
				},
				Entry("when the dirty bitmap context is shorter", uint64(4096), uint64(1024), uint64(1024)),
				Entry("when the allocation context is shorter", uint64(1024), uint64(4096), uint64(1024)),
			)

			It("should handle multiple Merge calls with coalescing across calls", func() {
				merger := newMergedContextMapper(2048)

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 0, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})
				extents, _ := merger.Merge()
				Expect(extents).To(BeEmpty())

				merger.HandleExtents(libnbd.CONTEXT_BASE_ALLOCATION, 1024, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: 0},
				})
				merger.HandleExtents(dirtyCtx, 1024, []libnbd.LibnbdExtent{
					{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)},
				})

				Expect(drain(merger)).To(Equal([]Extent{
					{Offset: 0, Length: 2048, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
				}))
			})
		})
	})

	Context("mapExtents", func() {
		const dirtyCtx = libnbd.CONTEXT_QEMU_DIRTY_BITMAP + "checkpoint"

		var requests []uint64

		type reply struct {
			alloc []libnbd.LibnbdExtent
			dirty []libnbd.LibnbdExtent
		}

		fakeServer := func(replies ...reply) blockStatusFn {
			return func(_, offset uint64, cb libnbd.Extent64Callback) error {
				Expect(len(requests)).To(BeNumerically("<", len(replies)), "unexpected request at offset %d", offset)
				r := replies[len(requests)]
				requests = append(requests, offset)
				var nbdErr int
				if r.alloc != nil {
					Expect(cb(libnbd.CONTEXT_BASE_ALLOCATION, offset, r.alloc, &nbdErr)).To(BeZero())
				}
				if r.dirty != nil {
					Expect(cb(dirtyCtx, offset, r.dirty, &nbdErr)).To(BeZero())
				}
				return nil
			}
		}

		collect := func(extents iter.Seq2[Extent, error]) ([]Extent, error) {
			var collected []Extent
			for extent, err := range extents {
				if err != nil {
					return collected, err
				}
				collected = append(collected, extent)
			}
			return collected, nil
		}

		BeforeEach(func() {
			requests = nil
		})

		DescribeTable("should resume from where the merged map ends when a context is truncated",
			func(first, second reply, expected []Extent) {
				merger := newMergedContextMapper(4096)

				extents, err := collect(mapExtents(context.Background(), fakeServer(first, second), merger, 0, 4096))
				Expect(err).ToNot(HaveOccurred())

				Expect(requests).To(Equal([]uint64{0, 1024}))
				Expect(extents).To(Equal(expected))
			},
			Entry("when the dirty bitmap context is truncated",
				reply{
					alloc: []libnbd.LibnbdExtent{{Length: 4096, Flags: 0}},
					dirty: []libnbd.LibnbdExtent{{Length: 1024, Flags: uint64(libnbd.STATE_DIRTY)}},
				},
				reply{
					alloc: []libnbd.LibnbdExtent{{Length: 3072, Flags: 0}},
					dirty: []libnbd.LibnbdExtent{{Length: 3072, Flags: 0}},
				},
				[]Extent{
					{Offset: 0, Length: 1024, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
					{Offset: 1024, Length: 3072, Flags: 0, Description: "clean"},
				},
			),
			Entry("when the allocation context is truncated",
				reply{
					alloc: []libnbd.LibnbdExtent{{Length: 1024, Flags: uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO)}},
					dirty: []libnbd.LibnbdExtent{{Length: 4096, Flags: uint64(libnbd.STATE_DIRTY)}},
				},
				reply{
					alloc: []libnbd.LibnbdExtent{{Length: 3072, Flags: 0}},
					dirty: []libnbd.LibnbdExtent{{Length: 3072, Flags: uint64(libnbd.STATE_DIRTY)}},
				},
				[]Extent{
					{Offset: 0, Length: 1024, Flags: uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO), Description: "dirty,zero"},
					{Offset: 1024, Length: 3072, Flags: uint64(libnbd.STATE_DIRTY), Description: "dirty"},
				},
			),
		)

		It("should fail instead of skipping the range when a context returns no extents", func() {
			merger := newMergedContextMapper(4096)
			server := fakeServer(reply{alloc: []libnbd.LibnbdExtent{{Length: 4096, Flags: 0}}})

			extents, err := collect(mapExtents(context.Background(), server, merger, 0, 4096))

			Expect(err).To(MatchError(ContainSubstring("BlockStatus64 at offset 0 did not advance the map")))
			Expect(extents).To(BeEmpty())
		})

		It("should fail when BlockStatus64 fails", func() {
			failing := func(_, _ uint64, _ libnbd.Extent64Callback) error { return errors.New("connection reset") }

			_, err := collect(mapExtents(context.Background(), failing, newSingleContextMapper(4096), 0, 4096))

			Expect(err).To(MatchError(ContainSubstring("BlockStatus64 at offset 0: connection reset")))
		})

		It("should stop when the context is canceled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := collect(mapExtents(ctx, fakeServer(), newMergedContextMapper(4096), 0, 4096))

			Expect(err).To(MatchError(context.Canceled))
			Expect(requests).To(BeEmpty())
		})

		It("should stop requesting extents when the consumer stops", func() {
			server := fakeServer(reply{alloc: []libnbd.LibnbdExtent{
				{Length: 1024, Flags: 0},
				{Length: 1024, Flags: uint64(libnbd.STATE_HOLE)},
			}})

			for extent, err := range mapExtents(context.Background(), server, newSingleContextMapper(4096), 0, 4096) {
				Expect(err).ToNot(HaveOccurred())
				Expect(extent).To(Equal(Extent{Offset: 0, Length: 1024, Flags: 0, Description: "data"}))
				break
			}

			Expect(requests).To(Equal([]uint64{0}))
		})
	})

	Context("readChunks", func() {
		fill := func(buf []byte, offset uint64) error {
			for i := range buf {
				buf[i] = byte(offset)
			}
			return nil
		}
		chunks := []readChunk{{offset: 0, length: 4}, {offset: 4, length: 4}}

		It("should write the chunks in order", func() {
			var out bytes.Buffer

			Expect(readChunks(context.Background(), fill, &out, chunks)).To(Succeed())

			Expect(out.Bytes()).To(Equal([]byte{0, 0, 0, 0, 4, 4, 4, 4}))
		})

		It("should return an error when pread fails", func() {
			failing := func([]byte, uint64) error { return errors.New("disk error") }

			err := readChunks(context.Background(), failing, io.Discard, chunks)

			Expect(err).To(MatchError(ContainSubstring("pread failed at offset 0")))
		})

		It("should return an error when the write fails", func() {
			err := readChunks(context.Background(), fill, failingWriter{}, chunks)

			Expect(err).To(MatchError(ContainSubstring("failed to write chunk at offset 0")))
		})

		It("should stop when the context is canceled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var out bytes.Buffer

			Expect(readChunks(ctx, fill, &out, chunks)).To(MatchError(context.Canceled))

			Expect(out.Len()).To(BeZero())
		})
	})
})

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stream closed") }
