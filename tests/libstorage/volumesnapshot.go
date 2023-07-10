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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libstorage

import (
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewVolumeSnapshot initializes a VolumeSnapshot struct
func NewVolumeSnapshot(name, namespace, sourcePvcName string, snapshotClassName *string) *vsv1.VolumeSnapshot {
	return &vsv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vsv1.VolumeSnapshotSpec{
			Source: vsv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &sourcePvcName,
			},
			VolumeSnapshotClassName: snapshotClassName,
		},
	}
}
