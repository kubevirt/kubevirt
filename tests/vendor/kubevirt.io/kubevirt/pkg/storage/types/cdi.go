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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package types

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	ConfigName        = "config"
	DefaultFSOverhead = cdiv1.Percent("0.055")
	FSOverheadMsg     = "Using default 5.5%% filesystem overhead for pvc size"
)

var ErrFailedToFindCdi error = errors.New("No CDI instances found")
var ErrMultipleCdiInstances error = errors.New("Detected more than one CDI instance")

func GetFilesystemOverhead(volumeMode *k8sv1.PersistentVolumeMode, storageClass *string, cdiConfig *cdiv1.CDIConfig) cdiv1.Percent {
	if IsPVCBlock(volumeMode) {
		return "0"
	}
	if storageClass == nil {
		return cdiConfig.Status.FilesystemOverhead.Global
	}
	fsOverhead, ok := cdiConfig.Status.FilesystemOverhead.StorageClass[*storageClass]
	if !ok {
		return cdiConfig.Status.FilesystemOverhead.Global
	}
	return fsOverhead
}

func roundUpToUnit(size, unit float64) float64 {
	if size < unit {
		return unit
	}
	return math.Ceil(size/unit) * unit
}

func alignSizeUpTo1MiB(size float64) float64 {
	return roundUpToUnit(size, float64(MiB))
}

func GetSizeIncludingGivenOverhead(size *resource.Quantity, overhead cdiv1.Percent) (*resource.Quantity, error) {
	fsOverhead, err := strconv.ParseFloat(string(overhead), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filesystem overhead as float: %v", err)
	}
	totalSize := (1 + fsOverhead) * size.AsApproximateFloat64()
	totalSize = alignSizeUpTo1MiB(totalSize)
	return resource.NewQuantity(int64(totalSize), size.Format), nil
}

func GetSizeIncludingDefaultFSOverhead(size *resource.Quantity) (*resource.Quantity, error) {
	return GetSizeIncludingGivenOverhead(size, DefaultFSOverhead)
}

func GetSizeIncludingFSOverhead(size *resource.Quantity, storageClass *string, volumeMode *k8sv1.PersistentVolumeMode, cdiConfig *cdiv1.CDIConfig) (*resource.Quantity, error) {
	cdiFSOverhead := GetFilesystemOverhead(volumeMode, storageClass, cdiConfig)
	return GetSizeIncludingGivenOverhead(size, cdiFSOverhead)
}
