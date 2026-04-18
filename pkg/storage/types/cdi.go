/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	ConfigName        = "config"
	DefaultFSOverhead = virtv1.Percent("0.055")
	FSOverheadMsg     = "Using default 5.5%% filesystem overhead for pvc size"

	// LabelApplyStorageProfile is a label used by the CDI mutating webhook
	// to modify the PVC according to the storage profile.
	LabelApplyStorageProfile = "cdi.kubevirt.io/applyStorageProfile"
)

var ErrFailedToFindCdi error = errors.New("No CDI instances found")
var ErrMultipleCdiInstances error = errors.New("Detected more than one CDI instance")

func GetFilesystemOverhead(volumeMode *k8sv1.PersistentVolumeMode, storageClass *string, cdiConfig *cdiv1.CDIConfig) (virtv1.Percent, error) {
	if IsPVCBlock(volumeMode) {
		return "0", nil
	}
	if cdiConfig.Status.FilesystemOverhead == nil {
		return "0", errors.New("CDI config not initialized")
	}
	if storageClass == nil {
		return virtv1.Percent(cdiConfig.Status.FilesystemOverhead.Global), nil
	}
	fsOverhead, ok := cdiConfig.Status.FilesystemOverhead.StorageClass[*storageClass]
	if !ok {
		return virtv1.Percent(cdiConfig.Status.FilesystemOverhead.Global), nil
	}
	return virtv1.Percent(fsOverhead), nil
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

func GetSizeIncludingGivenOverhead(size *resource.Quantity, overhead virtv1.Percent) (*resource.Quantity, error) {
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
	cdiFSOverhead, err := GetFilesystemOverhead(volumeMode, storageClass, cdiConfig)
	if err != nil {
		return nil, err
	}
	return GetSizeIncludingGivenOverhead(size, cdiFSOverhead)
}
