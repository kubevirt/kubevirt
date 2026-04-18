/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package velero

// For additional information please see https://velero.io/docs/v1.14/backup-hooks/#specifying-hooks-as-pod-annotations
const (
	// PreBackupHookContainerAnnotation specifies the container where the command should be executed.
	PreBackupHookContainerAnnotation = "pre.hook.backup.velero.io/container"

	// PreBackupHookCommandAnnotation specifies the command to execute.
	PreBackupHookCommandAnnotation = "pre.hook.backup.velero.io/command"

	// PreBackupHookTimeoutAnnotation specifies how long to wait for the pre-hook to complete.
	PreBackupHookTimeoutAnnotation = "pre.hook.backup.velero.io/timeout"

	// PostBackupHookContainerAnnotation specifies the container where the command should be executed.
	PostBackupHookContainerAnnotation = "post.hook.backup.velero.io/container"

	// PostBackupHookCommandAnnotation specifies the command to execute.
	PostBackupHookCommandAnnotation = "post.hook.backup.velero.io/command"

	// SkipHooksAnnotation signals that Velero backup freeze/unfreeze hooks should not be injected in virt-launcher.
	// Can be set on VM or VMI. Value must be "true" to skip hook injection.
	SkipHooksAnnotation = "kubevirt.io/skip-backup-hooks"
)
