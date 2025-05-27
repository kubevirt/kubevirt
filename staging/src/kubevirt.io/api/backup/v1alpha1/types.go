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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupMode is the const type for the backup possible modes
type BackupMode string

const (
	// PushMode defines backup which pushes the backup output
	// to a provided PVC - this is the default behavior
	PushMode BackupMode = "Push"
)

// BackupCmd is the const type for the backup possible commands
type BackupCmd string

const (
	Start BackupCmd = "Start"
)

// BackupOptions are options used to configure virtual machine backup job
type BackupOptions struct {
	BackupName      string       `json:"backupName,omitempty"`
	Cmd             BackupCmd    `json:"cmd,omitempty"`
	Mode            BackupMode   `json:"mode,omitempty"`
	BackupStartTime *metav1.Time `json:"backupStartTime,omitempty"`
	PushPath        *string      `json:"pushPath,omitempty"`
	SkipQuiesce     bool         `json:"skipQuiesce,omitempty"`
}
