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

// Package devicemetadata provides consumer-side utilities for reading and
// decoding DRA device metadata files. The package-level functions mirror
// k8s.io/dynamic-resource-allocation/devicemetadata so that switching to the
// upstream module is a matter of changing import paths.
package devicemetadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/dra/metadata"
)

// The functions in this file match the API from
// https://pkg.go.dev/k8s.io/dynamic-resource-allocation@v0.36.0-rc.0/devicemetadata
//
// They will be removed when the 1.36 packages are available for vendoring.

// DecodeMetadataFromStream reads the first compatible object from a DRA
// device metadata JSON stream. Unknown API versions are skipped so that a
// driver upgrade does not break older consumers.
//
// dest must be a *metadata.DeviceMetadata.
func DecodeMetadataFromStream(decoder *json.Decoder, dest runtime.Object) error {
	md, ok := dest.(*metadata.DeviceMetadata)
	if !ok {
		return fmt.Errorf("dest must be *metadata.DeviceMetadata, got %T", dest)
	}

	var skippedErrors []string
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("read metadata object from stream: %w", err)
		}

		var peek struct {
			APIVersion string `json:"apiVersion"`
		}
		if err := json.Unmarshal(raw, &peek); err != nil {
			skippedErrors = append(skippedErrors, err.Error())
			continue
		}

		if peek.APIVersion != metadata.SupportedAPIVersion {
			skippedErrors = append(skippedErrors, peek.APIVersion)
			continue
		}

		if err := json.Unmarshal(raw, md); err != nil {
			skippedErrors = append(skippedErrors, fmt.Sprintf("%s: %v", peek.APIVersion, err))
			continue
		}
		return nil
	}
	if len(skippedErrors) > 0 {
		return fmt.Errorf("no compatible metadata version found in stream (errors: %s)", strings.Join(skippedErrors, "; "))
	}
	return fmt.Errorf("no metadata objects found in stream")
}

// ReadResourceClaimMetadataWithDriverName reads and decodes the metadata file
// for a directly referenced ResourceClaim from a specific driver.
func ReadResourceClaimMetadataWithDriverName(driverName, claimName, requestName string) (*metadata.DeviceMetadata, error) {
	path := filepath.Join(metadata.ContainerDir, metadata.ResourceClaimsSubDir, claimName, requestName, metadata.MetadataFileName(driverName))
	return readMetadata(path)
}

// ReadResourceClaimTemplateMetadataWithDriverName reads and decodes the
// metadata file for a template-generated claim from a specific driver.
func ReadResourceClaimTemplateMetadataWithDriverName(driverName, podClaimName, requestName string) (*metadata.DeviceMetadata, error) {
	path := filepath.Join(metadata.ContainerDir, metadata.ResourceClaimTemplatesSubDir, podClaimName, requestName, metadata.MetadataFileName(driverName))
	return readMetadata(path)
}

// ReadResourceClaimMetadata reads and decodes all metadata files for a
// directly referenced ResourceClaim request, merging results from multiple
// drivers into a single DeviceMetadata.
func ReadResourceClaimMetadata(claimName, requestName string) (*metadata.DeviceMetadata, error) {
	dir := filepath.Join(metadata.ContainerDir, metadata.ResourceClaimsSubDir, claimName, requestName)
	return ReadRequestDir(dir)
}

// ReadResourceClaimTemplateMetadata reads and decodes all metadata files for
// a template-generated claim request, merging results from multiple drivers.
func ReadResourceClaimTemplateMetadata(podClaimName, requestName string) (*metadata.DeviceMetadata, error) {
	dir := filepath.Join(metadata.ContainerDir, metadata.ResourceClaimTemplatesSubDir, podClaimName, requestName)
	return ReadRequestDir(dir)
}

// ReadRequestDir reads and merges all *-metadata.json files from a single
// request directory. This is exported so callers with a custom base path
// (e.g. tests, virt-launcher) can use it directly.
func ReadRequestDir(dir string) (*metadata.DeviceMetadata, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*"+metadata.MetadataFileSuffix))
	if err != nil {
		return nil, fmt.Errorf("glob metadata files in %s: %w", dir, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no metadata files found in %s", dir)
	}

	var merged *metadata.DeviceMetadata
	for _, path := range matches {
		dm, err := readMetadata(path)
		if err != nil {
			return nil, err
		}
		if merged == nil {
			merged = dm
			continue
		}
		merged.Requests = append(merged.Requests, dm.Requests...)
	}
	return merged, nil
}

func readMetadata(path string) (*metadata.DeviceMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open metadata file: %w", err)
	}

	var dm metadata.DeviceMetadata
	decodeErr := DecodeMetadataFromStream(json.NewDecoder(f), &dm)
	if closeErr := f.Close(); closeErr != nil && decodeErr == nil {
		return nil, fmt.Errorf("close metadata file: %w", closeErr)
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("%s: %w", path, decodeErr)
	}
	return &dm, nil
}
