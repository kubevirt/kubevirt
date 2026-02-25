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

package storage

import (
	"fmt"
	"sort"
	"strings"

	"libguestfs.org/libnbd"

	"kubevirt.io/client-go/log"

	nbdv1 "kubevirt.io/kubevirt/pkg/storage/cbt/nbd/v1"
)

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

// mapBuilder accumulates, coalesces, and batches extents.
type mapBuilder struct {
	endOffset   uint64
	send        sendFn
	lastExtents map[string]*nbdv1.Extent
	batch       []*nbdv1.Extent
	batchSize   int
}

func newMapBuilder(endOffset uint64, batchSize int, send sendFn) *mapBuilder {
	return &mapBuilder{
		endOffset:   endOffset,
		send:        send,
		lastExtents: make(map[string]*nbdv1.Extent),
		batch:       make([]*nbdv1.Extent, 0, batchSize),
		batchSize:   batchSize,
	}
}

func (b *mapBuilder) HandleExtents(metacontext string, offset uint64, entries []libnbd.LibnbdExtent) (uint64, error) {
	localOffset := offset
	for _, e := range entries {
		length := e.Length
		if localOffset+length > b.endOffset {
			length = b.endOffset - localOffset
		}
		if length == 0 {
			continue
		}
		if err := b.coalesce(metacontext, localOffset, length, e.Flags); err != nil {
			return localOffset, err
		}
		localOffset += length
	}
	return localOffset, nil
}

func (b *mapBuilder) coalesce(metacontext string, offset, length, flags uint64) error {
	last := b.lastExtents[metacontext]
	if last != nil && last.Flags == flags && last.Offset+last.Length == offset {
		last.Length += length
		return nil
	}
	if err := b.flushContext(metacontext); err != nil {
		return err
	}
	b.lastExtents[metacontext] = &nbdv1.Extent{
		Offset:      offset,
		Length:      length,
		Flags:       flags,
		Description: getExtentDescription(metacontext, flags),
	}
	return nil
}

func (b *mapBuilder) flushContext(metacontext string) error {
	e, ok := b.lastExtents[metacontext]
	if !ok || e == nil {
		return nil
	}
	b.batch = append(b.batch, e)
	delete(b.lastExtents, metacontext)
	if len(b.batch) >= b.batchSize {
		return b.Flush()
	}
	return nil
}

func (b *mapBuilder) Flush() error {
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

func (b *mapBuilder) FlushAll() error {
	contexts := sortedContextsByOffset(b.lastExtents)
	for _, ctxName := range contexts {
		if err := b.flushContext(ctxName); err != nil {
			return fmt.Errorf("flushing final extent for context %s: %w", ctxName, err)
		}
	}
	return b.Flush()
}

func sortedContextsByOffset(lastExtents map[string]*nbdv1.Extent) []string {
	contexts := make([]string, 0, len(lastExtents))
	for k := range lastExtents {
		contexts = append(contexts, k)
	}
	sort.Slice(contexts, func(i, j int) bool {
		return lastExtents[contexts[i]].Offset < lastExtents[contexts[j]].Offset
	})
	return contexts
}

// based on https://gitlab.com/nbdkit/libnbd/-/blob/master/info/map.c
func (c *NBDClient) Map(req *nbdv1.MapRequest, stream nbdv1.NBD_MapServer) error {
	l, err := libnbd.Create()
	if err != nil {
		return fmt.Errorf("failed to create libnbd handle: %w", err)
	}
	defer l.Close()

	requestedContext := libnbd.CONTEXT_BASE_ALLOCATION
	if req.BitmapName != "" {
		requestedContext = libnbd.CONTEXT_QEMU_DIRTY_BITMAP + req.BitmapName
	}

	if err := l.AddMetaContext(requestedContext); err != nil {
		log.Log.Reason(err).Warningf("AddMetaContext(%s) failed: %v", requestedContext, err)
	}

	if err := c.connect(l, req.ExportName); err != nil {
		return err
	}

	// if the export lacks the requested context error out
	// falling back to base:allocation is misleading
	if can, err := l.CanMetaContext(requestedContext); err != nil || !can {
		return fmt.Errorf("server does not support requested context: %s", requestedContext)
	}

	size, err := l.GetSize()
	if err != nil {
		return fmt.Errorf("failed to get export size: %w", err)
	}

	currentOffset, endOffset, err := resolveRange(req.Offset, req.Length, size)
	if err != nil {
		return err
	}

	builder := newMapBuilder(endOffset, mapResponseBatchSize, stream.Send)

	for currentOffset < endOffset {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		prevOffset := currentOffset
		err := l.BlockStatus64(endOffset-currentOffset, currentOffset,
			func(metacontext string, offset uint64, entries []libnbd.LibnbdExtent, nbdErr *int) int {
				maxOffset, err := builder.HandleExtents(metacontext, offset, entries)
				if err != nil {
					*nbdErr = 1
					return -1
				}
				if maxOffset > currentOffset {
					currentOffset = maxOffset
				}
				return 0
			}, nil)
		if err != nil {
			return fmt.Errorf("BlockStatus64 at offset %d: %w", prevOffset, err)
		}
		if currentOffset <= prevOffset {
			return fmt.Errorf("BlockStatus64 returned no forward progress at offset %d for context %s", prevOffset, requestedContext)
		}
	}

	return builder.FlushAll()
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

func getExtentDescription(metacontext string, flags uint64) string {
	if metacontext == libnbd.CONTEXT_BASE_ALLOCATION {
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
	if strings.HasPrefix(metacontext, libnbd.CONTEXT_QEMU_DIRTY_BITMAP) {
		switch flags {
		case 0:
			return "clean"
		case uint64(libnbd.STATE_DIRTY):
			return "dirty"
		default:
			return "unknown"
		}
	}
	return "unknown"
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
