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

package oci

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	artifactTypeVM            = "application/vnd.kubevirt.virtualmachine.v1"
	mediaTypeVMConfig         = "application/vnd.kubevirt.virtualmachine.config.v1+json"
	artifactTypeVMTemplate    = "application/vnd.kubevirt.virtualmachinetemplate.v1"
	mediaTypeVMTemplateConfig = "application/vnd.kubevirt.virtualmachinetemplate.config.v1+json"
	mediaTypeDiskRawZstd      = "application/vnd.kubevirt.disk.raw+zstd"

	annotationDiskName = "io.kubevirt.disk.name"
	annotationDiskSize = "io.kubevirt.disk.size"

	tarBlockSize    = 512
	tarEndOfArchive = 2 * tarBlockSize
)

// DiskInfo describes a PVC-backed disk to include in the OCI artifact.
type DiskInfo struct {
	VolumeName string
	FilePath   string
}

// Builder constructs an OCI image layout TAR archive using a two-pass strategy.
// Pass 1 (Prepare) computes digests and sizes. Pass 2 (WriteTar) streams the archive.
type Builder struct {
	disks           []DiskInfo
	configJSON      []byte
	architecture    string
	artifactType    string
	configMediaType string
	createdAt       time.Time

	configDesc   ocispec.Descriptor
	diskDescs    []ocispec.Descriptor
	manifestDesc ocispec.Descriptor

	layoutBlob   []byte
	manifestBlob []byte
	indexBlob    []byte

	ready atomic.Bool
}

func newBuilder(configJSON []byte, architecture string, disks []DiskInfo, artifactType, configMediaType string) *Builder {
	if architecture == "" {
		architecture = "amd64"
	}
	return &Builder{
		disks:           disks,
		configJSON:      configJSON,
		architecture:    architecture,
		artifactType:    artifactType,
		configMediaType: configMediaType,
		createdAt:       time.Now().UTC(),
	}
}

// NewVMBuilder creates a Builder for a VirtualMachine OCI artifact.
func NewVMBuilder(configJSON []byte, architecture string, disks []DiskInfo) *Builder {
	return newBuilder(configJSON, architecture, disks, artifactTypeVM, mediaTypeVMConfig)
}

// NewVMTemplateBuilder creates a Builder for a VirtualMachineTemplate OCI artifact.
func NewVMTemplateBuilder(configJSON []byte, architecture string, disks []DiskInfo) *Builder {
	return newBuilder(configJSON, architecture, disks, artifactTypeVMTemplate, mediaTypeVMTemplateConfig)
}

// Ready returns true after Prepare has completed successfully.
func (b *Builder) Ready() bool {
	return b.ready.Load()
}

// Prepare runs the first pass of the OCI export that computes SHA-256 digests and sizes for all blobs.
// After Prepare returns successfully, WriteTar and Size can be called.
func (b *Builder) Prepare(ctx context.Context) error {
	b.configDesc = descriptorFromBytes(b.configMediaType, b.configJSON)

	b.diskDescs = make([]ocispec.Descriptor, len(b.disks))
	g, gCtx := errgroup.WithContext(ctx)
	for i, disk := range b.disks {
		g.Go(func() error {
			desc, err := computeDiskDescriptor(gCtx, disk)
			if err != nil {
				return fmt.Errorf("failed to compute digest for disk %s: %w", disk.VolumeName, err)
			}
			b.diskDescs[i] = desc
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	manifestBlob, err := b.buildManifest()
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}
	b.manifestBlob = manifestBlob
	b.manifestDesc = descriptorFromBytes(ocispec.MediaTypeImageManifest, b.manifestBlob)

	indexBlob, err := b.buildIndex()
	if err != nil {
		return fmt.Errorf("failed to build index: %w", err)
	}
	b.indexBlob = indexBlob

	layoutBlob, err := json.Marshal(ocispec.ImageLayout{Version: ocispec.ImageLayoutVersion})
	if err != nil {
		return fmt.Errorf("failed to build layout: %w", err)
	}
	b.layoutBlob = layoutBlob

	b.ready.Store(true)
	return nil
}

// Size returns the total TAR archive size in bytes. Returns -1 if Prepare has not completed.
func (b *Builder) Size() int64 {
	if !b.ready.Load() {
		return -1
	}
	total := tarEntrySize(int64(len(b.layoutBlob)))
	total += tarEntrySize(int64(len(b.indexBlob)))
	total += tarEntrySize(b.manifestDesc.Size)
	total += tarEntrySize(b.configDesc.Size)
	for _, desc := range b.diskDescs {
		total += tarEntrySize(desc.Size)
	}
	total += tarEndOfArchive
	return total
}

// Each tar entry is a 512-byte header followed by data padded to a 512-byte boundary.
func tarEntrySize(dataSize int64) int64 {
	return tarBlockSize + ((dataSize+tarBlockSize-1)/tarBlockSize)*tarBlockSize
}

// WriteTar runs the second pass that streams the OCI image as a TAR archive.
func (b *Builder) WriteTar(ctx context.Context, w io.Writer) (retErr error) {
	if !b.ready.Load() {
		return fmt.Errorf("prepare must be called before WriteTar")
	}

	tw := tar.NewWriter(w)
	defer func() { retErr = errors.Join(retErr, tw.Close()) }()

	if err := writeTarBlob(tw, ocispec.ImageLayoutFile, b.layoutBlob); err != nil {
		return err
	}
	if err := writeTarBlob(tw, ocispec.ImageIndexFile, b.indexBlob); err != nil {
		return err
	}
	if err := writeTarBlob(tw, blobPath(b.manifestDesc.Digest), b.manifestBlob); err != nil {
		return err
	}
	if err := writeTarBlob(tw, blobPath(b.configDesc.Digest), b.configJSON); err != nil {
		return err
	}

	for i, disk := range b.disks {
		if err := streamDiskToTar(ctx, tw, blobPath(b.diskDescs[i].Digest), b.diskDescs[i].Size, disk.FilePath); err != nil {
			return fmt.Errorf("streaming disk %s: %w", disk.VolumeName, err)
		}
	}

	return nil
}

func (b *Builder) buildManifest() ([]byte, error) {
	return json.Marshal(ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: b.artifactType,
		Config:       b.configDesc,
		Layers:       b.diskDescs,
	})
}

func (b *Builder) buildIndex() ([]byte, error) {
	return json.Marshal(ocispec.Index{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageIndex,
		ArtifactType: b.artifactType,
		Manifests: []ocispec.Descriptor{
			{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    b.manifestDesc.Digest,
				Size:      b.manifestDesc.Size,
				Platform: &ocispec.Platform{
					Architecture: b.architecture,
					OS:           "linux",
				},
			},
		},
		Annotations: map[string]string{
			ocispec.AnnotationCreated: b.createdAt.Format(time.RFC3339),
		},
	})
}

func descriptorFromBytes(mediaType string, data []byte) ocispec.Descriptor {
	h := sha256.Sum256(data)
	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    digest.NewDigestFromBytes(digest.SHA256, h[:]),
		Size:      int64(len(data)),
	}
}

func computeDiskDescriptor(ctx context.Context, disk DiskInfo) (_ ocispec.Descriptor, retErr error) {
	f, err := os.Open(disk.FilePath)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()

	diskSize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to determine disk size: %w", err)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to reset file offset: %w", err)
	}

	h := sha256.New()
	counter := &countWriter{w: h}

	enc, err := newZstdEncoder(counter)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if _, err := io.Copy(enc, &ctxReader{ctx: ctx, r: f}); err != nil {
		return ocispec.Descriptor{}, errors.Join(err, enc.Close())
	}
	if err := enc.Close(); err != nil {
		return ocispec.Descriptor{}, err
	}

	annotations := map[string]string{
		annotationDiskName:      disk.VolumeName,
		annotationDiskSize:      resource.NewQuantity(diskSize, resource.BinarySI).String(),
		ocispec.AnnotationTitle: fmt.Sprintf("%s.raw.zst", disk.VolumeName),
	}

	return ocispec.Descriptor{
		MediaType:   mediaTypeDiskRawZstd,
		Digest:      digest.NewDigestFromBytes(digest.SHA256, h.Sum(nil)),
		Size:        counter.n,
		Annotations: annotations,
	}, nil
}

func streamDiskToTar(ctx context.Context, tw *tar.Writer, name string, size int64, filePath string) (retErr error) {
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     size,
		Mode:     0o644,
	}); err != nil {
		return err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()

	enc, err := newZstdEncoder(tw)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, enc.Close()) }()

	if _, err := io.Copy(enc, &ctxReader{ctx: ctx, r: f}); err != nil {
		return err
	}
	return nil
}

func writeTarBlob(tw *tar.Writer, name string, data []byte) error {
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(data)),
		Mode:     0o644,
	}); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func blobPath(d digest.Digest) string {
	return path.Join(ocispec.ImageBlobsDir, string(d.Algorithm()), d.Encoded())
}

func newZstdEncoder(w io.Writer) (*zstd.Encoder, error) {
	return zstd.NewWriter(w,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
	)
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *ctxReader) Read(p []byte) (int, error) {
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
		return cr.r.Read(p)
	}
}

type countWriter struct {
	w io.Writer
	n int64
}

func (cw *countWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}
