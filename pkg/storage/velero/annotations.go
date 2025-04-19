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

package velero

// For additional information please see https://velero.io/docs/v1.14/backup-hooks/#specifying-hooks-as-pod-annotations
const (
	// PreBackupHookContainerAnnotation specifies the container where the command should be executed.
	PreBackupHookContainerAnnotation = "pre.hook.backup.velero.io/container"

	// PreBackupHookCommandAnnotation specifies the command to execute.
	PreBackupHookCommandAnnotation = "pre.hook.backup.velero.io/command"

	// PostBackupHookContainerAnnotation specifies the container where the command should be executed.
	PostBackupHookContainerAnnotation = "post.hook.backup.velero.io/container"

	// PostBackupHookCommandAnnotation specifies the command to execute.
	PostBackupHookCommandAnnotation = "post.hook.backup.velero.io/command"
)
