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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package velero

const (
	VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION  = "pre.hook.backup.velero.io/container"
	VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION    = "pre.hook.backup.velero.io/command"
	VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION = "post.hook.backup.velero.io/container"
	VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION   = "post.hook.backup.velero.io/command"
)
