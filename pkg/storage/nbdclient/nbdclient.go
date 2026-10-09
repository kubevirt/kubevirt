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
	"context"
	"fmt"
	"io"
	"iter"
	"net"
	"time"

	"libguestfs.org/libnbd"

	"kubevirt.io/client-go/log"
)

const maxReadChunkSize uint64 = 256 * 1024

type NBDClient struct {
	socketPath string
}

func NewNBDClient(socketPath string) *NBDClient {
	return &NBDClient{socketPath: socketPath}
}

type Extent struct {
	Offset      uint64
	Length      uint64
	Flags       uint64
	Description string
}

type descFn func(uint64) string

// mapHandler consumes the extents of one BlockStatus64 call through
// HandleExtents, then Merge returns the extents completed so far and the
// offset up to which the map is now complete. The next call must resume
// from that offset.
type mapHandler interface {
	HandleExtents(metacontext string, offset uint64, entries []libnbd.LibnbdExtent)
	Merge() ([]Extent, uint64)
	Flush() (Extent, bool)
}

type extentCoalescer struct {
	endOffset uint64
	done      []Extent
	last      *Extent
	desc      descFn
}

func (c *extentCoalescer) coalesce(offset, length, flags uint64) {
	if c.last != nil && c.last.Flags == flags && c.last.Offset+c.last.Length == offset {
		c.last.Length += length
		return
	}
	if c.last != nil {
		c.done = append(c.done, *c.last)
	}
	c.last = &Extent{
		Offset:      offset,
		Length:      length,
		Flags:       flags,
		Description: c.desc(flags),
	}
}

func (c *extentCoalescer) takeDone() []Extent {
	done := c.done
	c.done = nil
	return done
}

func (c *extentCoalescer) Flush() (Extent, bool) {
	if c.last == nil {
		return Extent{}, false
	}
	last := *c.last
	c.last = nil
	return last, true
}

// singleContextMapper coalesces extents from a single base:allocation
// context (full backup path).
type singleContextMapper struct {
	extentCoalescer
	end uint64
}

func newSingleContextMapper(endOffset uint64) *singleContextMapper {
	return &singleContextMapper{
		extentCoalescer: extentCoalescer{
			endOffset: endOffset,
			desc:      allocDescription,
		},
	}
}

func (b *singleContextMapper) HandleExtents(_ string, offset uint64, entries []libnbd.LibnbdExtent) {
	localOffset := offset
	for _, e := range entries {
		if localOffset >= b.endOffset {
			break
		}
		length := e.Length
		if localOffset+length > b.endOffset {
			length = b.endOffset - localOffset
		}
		if length == 0 {
			continue
		}
		b.coalesce(localOffset, length, e.Flags)
		localOffset += length
	}
	b.end = max(b.end, localOffset)
}

func (b *singleContextMapper) Merge() ([]Extent, uint64) { return b.takeDone(), b.end }

// mergedContextMapper merges extents from base:allocation and qemu:dirty-bitmap
// contexts into a single stream with combined flags, replicating client-side
// what QEMU does internally for push-mode backups (block/backup.c).
//
// BlockStatus64 delivers both contexts' callbacks sequentially within a
// single call. The merger buffers each context's extents during the
// callbacks, then runs a two-pointer walk to merge at boundary splits.
//
// The server limits the number of extents per context independently, so
// one context may describe a longer range than the other. Only the range
// both describe can be merged, the rest is dropped and requested again.
//
// Flag remapping avoids the STATE_HOLE/STATE_DIRTY bit collision (both
// value 1 in different contexts) following the same approach as oVirt's
// ovirt-imageio client:
// https://github.com/oVirt/ovirt-imageio/blob/master/ovirt_imageio/_internal/nbdutil.py
// https://gitlab.com/qemu-project/qemu/-/blob/master/block/backup.c
type mergedContextMapper struct {
	extentCoalescer
	allocExtents []Extent
	dirtyExtents []Extent
	end          uint64
}

func newMergedContextMapper(endOffset uint64) *mergedContextMapper {
	return &mergedContextMapper{
		extentCoalescer: extentCoalescer{
			endOffset: endOffset,
			desc:      mergedDescription,
		},
	}
}

func (m *mergedContextMapper) HandleExtents(metacontext string, offset uint64, entries []libnbd.LibnbdExtent) {
	localOffset := offset
	for _, e := range entries {
		if localOffset >= m.endOffset {
			break
		}
		length := e.Length
		if localOffset+length > m.endOffset {
			length = m.endOffset - localOffset
		}
		if length == 0 {
			continue
		}
		if metacontext == libnbd.CONTEXT_BASE_ALLOCATION {
			var flags uint64
			if e.Flags&uint64(libnbd.STATE_ZERO) != 0 {
				flags = uint64(libnbd.STATE_ZERO)
			}
			m.allocExtents = append(m.allocExtents, Extent{Offset: localOffset, Length: length, Flags: flags})
		} else {
			m.dirtyExtents = append(m.dirtyExtents, Extent{Offset: localOffset, Length: length, Flags: e.Flags})
		}
		localOffset += length
	}
}

// Merge merges the range described by both contexts and returns its end.
// When a context returned no extents nothing is merged and the previous
// end is returned.
func (m *mergedContextMapper) Merge() ([]Extent, uint64) {
	defer func() {
		m.allocExtents = m.allocExtents[:0]
		m.dirtyExtents = m.dirtyExtents[:0]
	}()

	if len(m.allocExtents) == 0 || len(m.dirtyExtents) == 0 {
		return nil, m.end
	}

	a, b := 0, 0
	for a < len(m.allocExtents) && b < len(m.dirtyExtents) {
		alloc := &m.allocExtents[a]
		dirty := &m.dirtyExtents[b]
		n := min(alloc.Length, dirty.Length)

		m.coalesce(alloc.Offset, n, alloc.Flags|dirty.Flags)
		m.end = alloc.Offset + n

		alloc.Offset += n
		alloc.Length -= n
		if alloc.Length == 0 {
			a++
		}
		dirty.Offset += n
		dirty.Length -= n
		if dirty.Length == 0 {
			b++
		}
	}

	return m.takeDone(), m.end
}

func (c *NBDClient) connectForMap(exportName, bitmapName string) (*libnbd.Libnbd, error) {
	l, err := libnbd.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create libnbd handle: %w", err)
	}

	if err := l.AddMetaContext(libnbd.CONTEXT_BASE_ALLOCATION); err != nil {
		log.Log.Reason(err).Warningf("AddMetaContext(%s) failed", libnbd.CONTEXT_BASE_ALLOCATION)
	}

	if bitmapName != "" {
		bitmapContext := libnbd.CONTEXT_QEMU_DIRTY_BITMAP + bitmapName
		if err := l.AddMetaContext(bitmapContext); err != nil {
			log.Log.Reason(err).Warningf("AddMetaContext(%s) failed", bitmapContext)
		}
	}

	if err := c.connect(l, exportName); err != nil {
		l.Close()
		return nil, err
	}

	if err := verifyContexts(l, bitmapName); err != nil {
		l.Close()
		return nil, err
	}

	return l, nil
}

func verifyContexts(l *libnbd.Libnbd, bitmapName string) error {
	if can, err := l.CanMetaContext(libnbd.CONTEXT_BASE_ALLOCATION); err != nil || !can {
		return fmt.Errorf("server does not support requested context: %s", libnbd.CONTEXT_BASE_ALLOCATION)
	}
	if bitmapName != "" {
		ctx := libnbd.CONTEXT_QEMU_DIRTY_BITMAP + bitmapName
		if can, err := l.CanMetaContext(ctx); err != nil || !can {
			return fmt.Errorf("server does not support requested context: %s", ctx)
		}
	}
	return nil
}

// Map yields the extents of the export in [offset, offset+length), merged
// with the dirty bitmap when bitmapName is set. A zero length maps up to the
// end of the export.
//
// based on https://gitlab.com/nbdkit/libnbd/-/blob/master/info/map.c
func (c *NBDClient) Map(ctx context.Context, exportName, bitmapName string, offset, length uint64) iter.Seq2[Extent, error] {
	return func(yield func(Extent, error) bool) {
		l, err := c.connectForMap(exportName, bitmapName)
		if err != nil {
			yield(Extent{}, err)
			return
		}
		defer l.Close()

		size, err := l.GetSize()
		if err != nil {
			yield(Extent{}, fmt.Errorf("failed to get export size: %w", err))
			return
		}

		startOffset, endOffset, err := resolveRange(offset, length, size)
		if err != nil {
			yield(Extent{}, err)
			return
		}

		var handler mapHandler
		if bitmapName != "" {
			handler = newMergedContextMapper(endOffset)
		} else {
			handler = newSingleContextMapper(endOffset)
		}

		blockStatus := func(count, offset uint64, cb libnbd.Extent64Callback) error {
			return l.BlockStatus64(count, offset, cb, nil)
		}
		for extent, err := range mapExtents(ctx, blockStatus, handler, startOffset, endOffset) {
			if !yield(extent, err) {
				return
			}
		}
	}
}

type blockStatusFn func(count, offset uint64, cb libnbd.Extent64Callback) error

// mapExtents walks [startOffset, endOffset) with blockStatus, resuming each
// call where the handler's merged map ends. Extents are yielded between
// calls, never from within the libnbd callback.
func mapExtents(ctx context.Context, blockStatus blockStatusFn, handler mapHandler, startOffset, endOffset uint64) iter.Seq2[Extent, error] {
	return func(yield func(Extent, error) bool) {
		for currentOffset := startOffset; currentOffset < endOffset; {
			if err := ctx.Err(); err != nil {
				yield(Extent{}, err)
				return
			}
			if err := blockStatus(endOffset-currentOffset, currentOffset,
				func(metacontext string, offset uint64, entries []libnbd.LibnbdExtent, _ *int) int {
					handler.HandleExtents(metacontext, offset, entries)
					return 0
				}); err != nil {
				yield(Extent{}, fmt.Errorf("BlockStatus64 at offset %d: %w", currentOffset, err))
				return
			}
			extents, mergedEnd := handler.Merge()
			for _, extent := range extents {
				if !yield(extent, nil) {
					return
				}
			}
			if mergedEnd <= currentOffset {
				yield(Extent{}, fmt.Errorf("BlockStatus64 at offset %d did not advance the map, a metadata context returned no extents", currentOffset))
				return
			}
			currentOffset = mergedEnd
		}

		if last, ok := handler.Flush(); ok {
			yield(last, nil)
		}
	}
}

type readChunk struct {
	offset uint64
	length uint64
}

func computeChunks(offset, length, chunkSize uint64) []readChunk {
	chunks := make([]readChunk, 0, (length+chunkSize-1)/chunkSize)
	for remaining := length; remaining > 0; {
		size := chunkSize
		if remaining < size {
			size = remaining
		}
		chunks = append(chunks, readChunk{offset: offset, length: size})
		offset += size
		remaining -= size
	}
	return chunks
}

// Read copies [offset, offset+length) of the export into w, in chunks of at
// most maxReadChunkSize. A zero length reads up to the end of the export.
func (c *NBDClient) Read(ctx context.Context, exportName string, offset, length uint64, w io.Writer) error {
	l, err := libnbd.Create()
	if err != nil {
		return fmt.Errorf("failed to create libnbd handle: %w", err)
	}
	defer l.Close()

	if err := c.connect(l, exportName); err != nil {
		return err
	}

	size, err := l.GetSize()
	if err != nil {
		return fmt.Errorf("failed to get export size: %w", err)
	}
	length, err = clampLength(offset, length, size)
	if err != nil {
		return err
	}

	pread := func(buf []byte, offset uint64) error {
		return l.Pread(buf, offset, nil)
	}

	return readChunks(ctx, pread, w, computeChunks(offset, length, maxReadChunkSize))
}

type preadFn func(buf []byte, offset uint64) error

// readChunks copies the chunks into w, reusing one buffer across them.
func readChunks(ctx context.Context, pread preadFn, w io.Writer, chunks []readChunk) error {
	buf := make([]byte, maxReadChunkSize)
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return err
		}
		b := buf[:chunk.length]
		if err := pread(b, chunk.offset); err != nil {
			return fmt.Errorf("pread failed at offset %d: %w", chunk.offset, err)
		}
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("failed to write chunk at offset %d: %w", chunk.offset, err)
		}
	}

	return nil
}

// Serving reports whether an NBD server accepts connections on the socket.
// QEMU both creates and removes the socket around serving a backup, so
// connecting to it is enough to tell.
func (c *NBDClient) Serving(timeout time.Duration) error {
	conn, err := net.DialTimeout("unix", c.socketPath, timeout)
	if err != nil {
		return err
	}

	return conn.Close()
}

func (c *NBDClient) connect(l *libnbd.Libnbd, exportName string) error {
	if err := l.SetExportName(exportName); err != nil {
		return fmt.Errorf("failed to set export name: %w", err)
	}

	if err := l.ConnectUnix(c.socketPath); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.socketPath, err)
	}
	return nil
}

func allocDescription(flags uint64) string {
	switch flags {
	case 0:
		return "data"
	case uint64(libnbd.STATE_HOLE):
		return "hole"
	case uint64(libnbd.STATE_ZERO):
		return "zero"
	case uint64(libnbd.STATE_HOLE | libnbd.STATE_ZERO):
		return "hole,zero"
	default:
		return "unknown"
	}
}

func mergedDescription(flags uint64) string {
	switch flags {
	case 0:
		return "clean"
	case uint64(libnbd.STATE_DIRTY):
		return "dirty"
	case uint64(libnbd.STATE_ZERO):
		return "zero"
	case uint64(libnbd.STATE_DIRTY) | uint64(libnbd.STATE_ZERO):
		return "dirty,zero"
	default:
		return "unknown"
	}
}

func resolveRange(offset, length, size uint64) (uint64, uint64, error) {
	clamped, err := clampLength(offset, length, size)
	if err != nil {
		return 0, 0, err
	}
	return offset, offset + clamped, nil
}

func clampLength(offset, length, size uint64) (uint64, error) {
	if offset >= size {
		return 0, fmt.Errorf("offset %d is beyond export size %d", offset, size)
	}
	if length == 0 || offset+length > size {
		length = size - offset
	}
	return length, nil
}
