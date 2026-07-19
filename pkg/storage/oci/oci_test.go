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
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("OCI Builder", func() {
	const (
		architecture  = "amd64"
		diskVolume    = "disk"
		osName        = "linux"
		schemaVersion = 2
	)

	emptyConfigJSON := func() []byte { return []byte("{}") }

	It("should produce a valid OCI image layout TAR", func() {
		const volumeName = "rootdisk"

		diskPath := createTestDisk()
		configJSON := []byte(`{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachine"}`)
		builder := NewVMBuilder(configJSON, architecture, []DiskInfo{
			{
				VolumeName: volumeName,
				FilePath:   diskPath,
			},
		})

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		files := readTarFiles(&buf)

		Expect(files).To(HaveKey(ocispec.ImageLayoutFile))
		var layout ocispec.ImageLayout
		Expect(json.Unmarshal(files[ocispec.ImageLayoutFile], &layout)).To(Succeed())
		Expect(layout.Version).To(Equal(ocispec.ImageLayoutVersion))

		Expect(files).To(HaveKey(ocispec.ImageIndexFile))
		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		Expect(index.MediaType).To(Equal(ocispec.MediaTypeImageIndex))
		Expect(index.SchemaVersion).To(Equal(schemaVersion))
		Expect(index.ArtifactType).To(Equal(artifactTypeVM))
		Expect(index.Manifests).To(HaveLen(1))
		Expect(index.Manifests[0].Platform.Architecture).To(Equal(architecture))
		Expect(index.Manifests[0].Platform.OS).To(Equal(osName))
		Expect(index.Annotations).To(HaveKey(ocispec.AnnotationCreated))

		manifestPath := blobPath(index.Manifests[0].Digest)
		Expect(files).To(HaveKey(manifestPath))

		var manifest ocispec.Manifest
		Expect(json.Unmarshal(files[manifestPath], &manifest)).To(Succeed())
		Expect(manifest.MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		Expect(manifest.SchemaVersion).To(Equal(schemaVersion))
		Expect(manifest.ArtifactType).To(Equal(artifactTypeVM))
		Expect(manifest.Config.MediaType).To(Equal(mediaTypeVMConfig))
		Expect(manifest.Layers).To(HaveLen(1))
		Expect(manifest.Layers[0].MediaType).To(Equal(mediaTypeDiskRawZstd))
		Expect(manifest.Layers[0].Annotations).To(HaveKeyWithValue(annotationDiskName, volumeName))
		Expect(manifest.Layers[0].Annotations).To(HaveKey(annotationDiskSize))
		Expect(manifest.Layers[0].Annotations).To(HaveKeyWithValue(ocispec.AnnotationTitle, volumeName+".raw.zst"))

		configDigest := manifest.Config.Digest
		configPath := blobPath(configDigest)
		Expect(files).To(HaveKey(configPath))
		Expect(files[configPath]).To(Equal(configJSON))

		diskDigest := manifest.Layers[0].Digest
		diskBlobPath := blobPath(diskDigest)
		Expect(files).To(HaveKey(diskBlobPath))
		Expect(files[diskBlobPath]).ToNot(BeEmpty())
	})

	It("should set disk size annotation from file size", func() {
		diskPath := createTestDisk()
		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: diskVolume, FilePath: diskPath},
		})

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		fi, err := os.Stat(diskPath)
		Expect(err).ToNot(HaveOccurred())
		expectedSize := resource.NewQuantity(fi.Size(), resource.BinarySI).String()

		files := readTarFiles(&buf)
		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		var manifest ocispec.Manifest
		Expect(json.Unmarshal(files[blobPath(index.Manifests[0].Digest)], &manifest)).To(Succeed())
		Expect(manifest.Layers).To(HaveLen(1))
		Expect(manifest.Layers[0].Annotations).To(HaveKeyWithValue(annotationDiskName, diskVolume))
		Expect(manifest.Layers[0].Annotations).To(HaveKeyWithValue(annotationDiskSize, expectedSize))
	})

	It("should default architecture to amd64 when empty", func() {
		diskPath := createTestDisk()
		builder := NewVMBuilder(emptyConfigJSON(), "", []DiskInfo{
			{VolumeName: diskVolume, FilePath: diskPath},
		})

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		files := readTarFiles(&buf)
		Expect(files).To(HaveKey(ocispec.ImageIndexFile))
		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		Expect(index.Manifests[0].Platform.Architecture).To(Equal(architecture))
	})

	It("should produce deterministic digests across runs", func() {
		diskPath := createTestDisk()
		configJSON := []byte(`{"name":"test"}`)

		disks := []DiskInfo{{VolumeName: "vol", FilePath: diskPath}}

		b1 := NewVMBuilder(configJSON, architecture, disks)
		Expect(b1.Prepare(context.Background())).To(Succeed())
		var buf1 bytes.Buffer
		Expect(b1.WriteTar(context.Background(), &buf1)).To(Succeed())

		b2 := NewVMBuilder(configJSON, architecture, disks)
		b2.createdAt = b1.createdAt
		Expect(b2.Prepare(context.Background())).To(Succeed())
		var buf2 bytes.Buffer
		Expect(b2.WriteTar(context.Background(), &buf2)).To(Succeed())

		tr1 := tar.NewReader(&buf1)
		tr2 := tar.NewReader(&buf2)
		for {
			hdr1, err1 := tr1.Next()
			hdr2, err2 := tr2.Next()
			if err1 == io.EOF && err2 == io.EOF {
				break
			}
			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())
			Expect(hdr1.Name).To(Equal(hdr2.Name))
			Expect(hdr1.Size).To(Equal(hdr2.Size))

			if strings.HasPrefix(hdr1.Name, ocispec.ImageBlobsDir) || hdr1.Name == ocispec.ImageLayoutFile || hdr1.Name == ocispec.ImageIndexFile {
				data1, _ := io.ReadAll(tr1)
				data2, _ := io.ReadAll(tr2)
				Expect(data1).To(Equal(data2), "content mismatch for %s", hdr1.Name)
			}
		}
	})

	It("should handle multiple disks", func() {
		const (
			vol1Name = "vol1"
			vol2Name = "vol2"
		)

		disk1 := createTestDisk()
		disk2 := createTestDisk()
		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: vol1Name, FilePath: disk1},
			{VolumeName: vol2Name, FilePath: disk2},
		})

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		files := readTarFiles(&buf)

		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		var manifest ocispec.Manifest
		Expect(json.Unmarshal(files[blobPath(index.Manifests[0].Digest)], &manifest)).To(Succeed())
		Expect(manifest.Layers).To(HaveLen(2))
		Expect(manifest.Layers[0].Annotations[annotationDiskName]).To(Equal(vol1Name))
		Expect(manifest.Layers[1].Annotations[annotationDiskName]).To(Equal(vol2Name))
	})

	It("should handle zero disks", func() {
		configJSON := []byte(`{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachine"}`)
		builder := NewVMBuilder(configJSON, architecture, nil)

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		files := readTarFiles(&buf)

		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		var manifest ocispec.Manifest
		Expect(json.Unmarshal(files[blobPath(index.Manifests[0].Digest)], &manifest)).To(Succeed())
		Expect(manifest.Layers).To(BeEmpty())
		Expect(manifest.Config.MediaType).To(Equal(mediaTypeVMConfig))
		Expect(files[blobPath(manifest.Config.Digest)]).To(Equal(configJSON))
	})

	It("should return -1 from TotalTarSize before Prepare", func() {
		builder := NewVMBuilder(emptyConfigJSON(), architecture, nil)
		Expect(builder.Size()).To(Equal(int64(-1)))
	})

	It("should return TotalTarSize matching actual TAR length", func() {
		diskPath := createTestDisk()
		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: diskVolume, FilePath: diskPath},
		})
		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		Expect(int64(buf.Len())).To(Equal(builder.Size()))
	})

	It("should return TotalTarSize matching actual TAR length with multiple disks", func() {
		disk1 := createTestDisk()
		disk2 := createTestDisk()
		builder := NewVMBuilder([]byte(`{"name":"test"}`), architecture, []DiskInfo{
			{VolumeName: "vol1", FilePath: disk1},
			{VolumeName: "vol2", FilePath: disk2},
		})
		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		Expect(int64(buf.Len())).To(Equal(builder.Size()))
	})

	It("should return TotalTarSize matching actual TAR length with zero disks", func() {
		builder := NewVMBuilder(emptyConfigJSON(), architecture, nil)
		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		Expect(int64(buf.Len())).To(Equal(builder.Size()))
	})

	It("should fail WriteTar before Prepare", func() {
		builder := NewVMBuilder(emptyConfigJSON(), architecture, nil)
		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(MatchError(ContainSubstring("prepare must be called")))
	})

	It("should fail Prepare with non-existent disk path", func() {
		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: "bad", FilePath: "/nonexistent/disk.img"},
		})
		Expect(builder.Prepare(context.Background())).To(MatchError(ContainSubstring("failed to compute digest for disk bad")))
	})

	It("should fail WriteTar when context is canceled", func() {
		diskPath := createTestDisk()
		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: diskVolume, FilePath: diskPath},
		})
		Expect(builder.Prepare(context.Background())).To(Succeed())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var buf bytes.Buffer
		Expect(builder.WriteTar(ctx, &buf)).To(MatchError(ContainSubstring("context canceled")))
	})

	It("should fail Prepare when context is canceled", func() {
		diskPath := createTestDisk()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		builder := NewVMBuilder(emptyConfigJSON(), architecture, []DiskInfo{
			{VolumeName: diskVolume, FilePath: diskPath},
		})
		Expect(builder.Prepare(ctx)).To(MatchError(ContainSubstring("context canceled")))
	})

	It("should use VMTemplate artifact type when using NewVMTemplateBuilder", func() {
		diskPath := createTestDisk()
		configJSON := []byte(`{"apiVersion":"template.kubevirt.io/v1beta1","kind":"VirtualMachineTemplate"}`)
		builder := NewVMTemplateBuilder(configJSON, architecture, []DiskInfo{
			{VolumeName: "rootdisk", FilePath: diskPath},
		})

		Expect(builder.Prepare(context.Background())).To(Succeed())

		var buf bytes.Buffer
		Expect(builder.WriteTar(context.Background(), &buf)).To(Succeed())

		files := readTarFiles(&buf)

		var index ocispec.Index
		Expect(json.Unmarshal(files[ocispec.ImageIndexFile], &index)).To(Succeed())
		Expect(index.ArtifactType).To(Equal(artifactTypeVMTemplate))

		var manifest ocispec.Manifest
		Expect(json.Unmarshal(files[blobPath(index.Manifests[0].Digest)], &manifest)).To(Succeed())
		Expect(manifest.ArtifactType).To(Equal(artifactTypeVMTemplate))
		Expect(manifest.Config.MediaType).To(Equal(mediaTypeVMTemplateConfig))
	})
})

func createTestDisk() string {
	p := filepath.Join(GinkgoT().TempDir(), "disk.img")
	f, err := os.Create(p)
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()
	_, err = io.CopyN(f, rand.Reader, 64*1024)
	Expect(err).ToNot(HaveOccurred())
	return p
}

func readTarFiles(r io.Reader) map[string][]byte {
	tr := tar.NewReader(r)
	files := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		Expect(err).ToNot(HaveOccurred())
		data, err := io.ReadAll(tr)
		Expect(err).ToNot(HaveOccurred())
		files[hdr.Name] = data
	}
	return files
}
