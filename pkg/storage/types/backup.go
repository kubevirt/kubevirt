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

package types

import (
	"encoding/json"
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

// BackupCheckpointReferenceAnnotation carries the controller's checkpoint references to the launcher.
const BackupCheckpointReferenceAnnotation = "backup.kubevirt.io/checkpoint-reference"

// BackupCheckpointReference identifies a promoted backup and the checkpoints still in use.
// It is an internal controller-to-launcher message.
type BackupCheckpointReference struct {
	VMIUID                    types.UID    `json:"vmiUID"`
	BackupName                string       `json:"backupName"`
	StartTimestamp            *metav1.Time `json:"startTimestamp"`
	CheckpointName            string       `json:"checkpointName"`
	ReferencedCheckpointNames []string     `json:"checkpointNames"`
}

func GetBackupCheckpointReference(vmi *v1.VirtualMachineInstance) (*BackupCheckpointReference, error) {
	encoded := vmi.Annotations[BackupCheckpointReferenceAnnotation]
	if encoded == "" {
		return nil, nil
	}
	var reference BackupCheckpointReference
	if err := json.Unmarshal([]byte(encoded), &reference); err != nil {
		return nil, fmt.Errorf("invalid backup checkpoint reference: %w", err)
	}
	if reference.VMIUID == "" || reference.VMIUID != vmi.UID || reference.BackupName == "" ||
		reference.StartTimestamp == nil || reference.CheckpointName == "" ||
		!slices.Contains(reference.ReferencedCheckpointNames, reference.CheckpointName) {
		return nil, fmt.Errorf("backup checkpoint reference is incomplete or belongs to another VMI")
	}
	return &reference, nil
}
