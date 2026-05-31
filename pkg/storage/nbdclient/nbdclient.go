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
	"fmt"

	"google.golang.org/grpc"
	"libguestfs.org/libnbd"

	"kubevirt.io/client-go/log"

	nbdv1 "kubevirt.io/kubevirt/pkg/storage/cbt/nbd/v1"
)

// RegisterNBDServer registers an NBDClient as the NBD gRPC server
// implementation on the given gRPC server for the specified socket path.
func RegisterNBDServer(srv *grpc.Server, socketPath string) {
	nbdv1.RegisterNBDServer(srv, NewNBDClient(socketPath))
}

const (
	maxReadChunkSize     uint64 = 256 * 1024
	mapResponseBatchSize        = 512
)

type NBDClient struct {
	socketPath string
}

func NewNBDClient(socketPath string) *NBDClient {
	return &NBDClient{socketPath: socketPath}
}

type sendFn func(*nbdv1.MapResponse) error
type descFn func(uint64) string

type mapHandler interface {
	HandleExtents(metacontext string, offset uint64, entries []libnbd.LibnbdExtent) (uint64, error)
	Merge() error
	Flush() error
}

type extentBatcher struct {
	endOffset uint64
	batch     []*nbdv1.Extent
	batchSize int
	last      *nbdv1.Extent
	send      sendFn
	desc      descFn
}

func (b *extentBatcher) coalesce(offset, length, flags uint64) error {
	if b.last != nil && b.last.Flags == flags && b.last.Offset+b.last.Length == offset {
		b.last.Length += length
		return nil
	}
	if err := b.flushLast(); err != nil {
		return err
	}
	b.last = &nbdv1.Extent{
		Offset:      offset,
		Length:      length,
		Flags:       flags,
		Description: b.desc(flags),
	}
	return nil
}

func (b *extentBatcher) flushLast() error {
	if b.last == nil {
		return nil
	}
	e := b.last
	b.last = nil
	b.batch = append(b.batch, e)
	if len(b.batch) >= b.batchSize {
		return b.sendBatch()
	}
	return nil
}

func (b *extentBatcher) sendBatch() error {
	if len(b.batch) == 0 {
		return nil
	}
	last := b.batch[len(b.batch)-1]
	err := b.send(&nbdv1.MapResponse{
		Extents:    b.batch,
		NextOffset: last.Offset + last.Length,
	})
	b.batch = b.batch[:0]
	return err
}

func (b *extentBatcher) Flush() error {
	if err := b.flushLast(); err != nil {
		return err
	}
	return b.sendBatch()
}

// singleContextMapper accumulates, coalesces, and batches extents from a single
// base:allocation context (full backup path).
type singleContextMapper struct {
	extentBatcher
}

func newSingleContextMapper(endOffset uint64, batchSize int, send sendFn) *singleContextMapper {
	return &singleContextMapper{
		extentBatcher: extentBatcher{
			endOffset: endOffset,
			batch:     make([]*nbdv1.Extent, 0, batchSize),
			batchSize: batchSize,
			send:      send,
			desc:      allocDescription,
		},
	}
}

func (b *singleContextMapper) HandleExtents(_ string, offset uint64, entries []libnbd.LibnbdExtent) (uint64, error) {
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
		if err := b.coalesce(localOffset, length, e.Flags); err != nil {
			return localOffset, err
		}
		localOffset += length
	}
	return localOffset, nil
}

func (b *singleContextMapper) Merge() error { return nil }

// mergedContextMapper merges extents from base:allocation and qemu:dirty-bitmap
// contexts into a single stream with combined flags, replicating client-side
// what QEMU does internally for push-mode backups (block/backup.c).
//
// BlockStatus64 delivers both contexts' callbacks sequentially within a
// single call. The merger buffers each context's extents during the
// callbacks, then runs a two-pointer walk to merge at boundary splits.
//
// Flag remapping avoids the STATE_HOLE/STATE_DIRTY bit collision (both
// value 1 in different contexts) following the same approach as oVirt's
// ovirt-imageio client:
// https://github.com/oVirt/ovirt-imageio/blob/master/ovirt_imageio/_internal/nbdutil.py
// https://gitlab.com/qemu-project/qemu/-/blob/master/block/backup.c
type mergedContextMapper struct {
	extentBatcher
	allocExtents []nbdv1.Extent
	dirtyExtents []nbdv1.Extent
}

func newMergedContextMapper(endOffset uint64, batchSize int, send sendFn) *mergedContextMapper {
	return &mergedContextMapper{
		extentBatcher: extentBatcher{
			endOffset: endOffset,
			batch:     make([]*nbdv1.Extent, 0, batchSize),
			batchSize: batchSize,
			send:      send,
			desc:      mergedDescription,
		},
	}
}

func (m *mergedContextMapper) HandleExtents(metacontext string, offset uint64, entries []libnbd.LibnbdExtent) (uint64, error) {
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
			m.allocExtents = append(m.allocExtents, nbdv1.Extent{Offset: localOffset, Length: length, Flags: flags})
		} else {
			m.dirtyExtents = append(m.dirtyExtents, nbdv1.Extent{Offset: localOffset, Length: length, Flags: e.Flags})
		}
		localOffset += length
	}
	return localOffset, nil
}

func (m *mergedContextMapper) Merge() error {
	defer func() {
		m.allocExtents = m.allocExtents[:0]
		m.dirtyExtents = m.dirtyExtents[:0]
	}()

	if len(m.allocExtents) == 0 || len(m.dirtyExtents) == 0 {
		return nil
	}

	a, b := 0, 0
	for a < len(m.allocExtents) && b < len(m.dirtyExtents) {
		alloc := &m.allocExtents[a]
		dirty := &m.dirtyExtents[b]
		n := min(alloc.Length, dirty.Length)

		if err := m.coalesce(alloc.Offset, n, alloc.Flags|dirty.Flags); err != nil {
			return err
		}

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

	return nil
}

func (c *NBDClient) connectForMap(req *nbdv1.MapRequest) (*libnbd.Libnbd, bool, error) {
	l, err := libnbd.Create()
	if err != nil {
		return nil, false, fmt.Errorf("failed to create libnbd handle: %w", err)
	}

	if err := l.AddMetaContext(libnbd.CONTEXT_BASE_ALLOCATION); err != nil {
		log.Log.Reason(err).Warningf("AddMetaContext(%s) failed", libnbd.CONTEXT_BASE_ALLOCATION)
	}

	incremental := req.BitmapName != ""
	if incremental {
		bitmapContext := libnbd.CONTEXT_QEMU_DIRTY_BITMAP + req.BitmapName
		if err := l.AddMetaContext(bitmapContext); err != nil {
			log.Log.Reason(err).Warningf("AddMetaContext(%s) failed", bitmapContext)
		}
	}

	if err := c.connect(l, req.ExportName); err != nil {
		l.Close()
		return nil, false, err
	}

	if err := verifyContexts(l, req.BitmapName); err != nil {
		l.Close()
		return nil, false, err
	}

	return l, incremental, nil
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

// based on https://gitlab.com/nbdkit/libnbd/-/blob/master/info/map.c
func (c *NBDClient) Map(req *nbdv1.MapRequest, stream nbdv1.NBD_MapServer) error {
	l, incremental, err := c.connectForMap(req)
	if err != nil {
		return err
	}
	defer l.Close()

	size, err := l.GetSize()
	if err != nil {
		return fmt.Errorf("failed to get export size: %w", err)
	}

	currentOffset, endOffset, err := resolveRange(req.Offset, req.Length, size)
	if err != nil {
		return err
	}

	var handler mapHandler
	if incremental {
		handler = newMergedContextMapper(endOffset, mapResponseBatchSize, stream.Send)
	} else {
		handler = newSingleContextMapper(endOffset, mapResponseBatchSize, stream.Send)
	}

	for currentOffset < endOffset {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		prevOffset := currentOffset
		if err := l.BlockStatus64(endOffset-currentOffset, currentOffset,
			func(metacontext string, offset uint64, entries []libnbd.LibnbdExtent, nbdErr *int) int {
				maxOff, err := handler.HandleExtents(metacontext, offset, entries)
				if err != nil {
					*nbdErr = 1
					return -1
				}
				if maxOff > currentOffset {
					currentOffset = maxOff
				}
				return 0
			}, nil); err != nil {
			return fmt.Errorf("BlockStatus64 at offset %d: %w", prevOffset, err)
		}
		if err := handler.Merge(); err != nil {
			return err
		}
		if currentOffset <= prevOffset {
			return fmt.Errorf("BlockStatus64 returned no forward progress at offset %d", prevOffset)
		}
	}

	return handler.Flush()
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

type readProcessor struct {
	pread func(buf []byte, offset uint64) error
	send  func(*nbdv1.DataChunk) error
}

func (p *readProcessor) Process(chunks []readChunk) error {
	for _, c := range chunks {
		buf := make([]byte, c.length)
		if err := p.pread(buf, c.offset); err != nil {
			return fmt.Errorf("pread failed at offset %d: %w", c.offset, err)
		}
		if err := p.send(&nbdv1.DataChunk{Offset: c.offset, Data: buf}); err != nil {
			return fmt.Errorf("failed to send chunk at offset %d: %w", c.offset, err)
		}
	}
	return nil
}

func (c *NBDClient) Read(req *nbdv1.ReadRequest, stream nbdv1.NBD_ReadServer) error {
	l, err := libnbd.Create()
	if err != nil {
		return fmt.Errorf("failed to create libnbd handle: %w", err)
	}
	defer l.Close()

	if err := c.connect(l, req.ExportName); err != nil {
		return err
	}

	size, err := l.GetSize()
	if err != nil {
		return fmt.Errorf("failed to get export size: %w", err)
	}

	length, err := clampLength(req.Offset, req.Length, size)
	if err != nil {
		return err
	}

	chunks := computeChunks(req.Offset, length, maxReadChunkSize)

	p := &readProcessor{
		pread: func(buf []byte, offset uint64) error {
			return l.Pread(buf, offset, nil)
		},
		send: stream.Send,
	}

	for _, chunk := range chunks {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		if err := p.Process([]readChunk{chunk}); err != nil {
			return err
		}
	}
	return nil
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
