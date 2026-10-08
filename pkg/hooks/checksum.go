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

package hooks

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

const SHA256ChecksumAlgorithm = "sha256"

func ValidateChecksum(checksum Checksum) error {
	_, err := decodeChecksum(checksum)
	return err
}

func ValidateConfigMapChecksum(configMap ConfigMap) error {
	if configMap.Checksum == nil {
		return nil
	}

	if configMap.Name == "" {
		return errors.New("config map name is required when a checksum is specified")
	}
	if configMap.Key == "" {
		return errors.New("config map key is required when a checksum is specified")
	}
	if !IsSupportedHookPath(configMap.HookPath) {
		return fmt.Errorf("unsupported hook path %q", configMap.HookPath)
	}

	return ValidateChecksum(*configMap.Checksum)
}

func IsSupportedHookPath(path string) bool {
	return path == OnDefineDomainHookPath || path == PreCloudInitIsoHookPath
}

func VerifyChecksum(content []byte, checksum Checksum) error {
	expectedChecksum, err := decodeChecksum(checksum)
	if err != nil {
		return err
	}

	actualChecksum := sha256.Sum256(content)
	if subtle.ConstantTimeCompare(actualChecksum[:], expectedChecksum) != 1 {
		return errors.New("checksum mismatch")
	}

	return nil
}

func decodeChecksum(checksum Checksum) ([]byte, error) {
	if checksum.Algorithm != SHA256ChecksumAlgorithm {
		return nil, fmt.Errorf("unsupported checksum algorithm %q", checksum.Algorithm)
	}

	decodedChecksum, err := hex.DecodeString(checksum.Value)
	if err != nil {
		return nil, fmt.Errorf("decode %s checksum: %w", checksum.Algorithm, err)
	}

	if len(decodedChecksum) != sha256.Size {
		return nil, fmt.Errorf("invalid %s checksum length: expected %d bytes, got %d", checksum.Algorithm, sha256.Size, len(decodedChecksum))
	}

	return decodedChecksum, nil
}
