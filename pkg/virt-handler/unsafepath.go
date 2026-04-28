/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import (
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

func passtSocketDirOnHost(path isolation.IsolationResult) (string, error) {
	root, err := path.MountRoot()
	if err != nil {
		return "", err
	}

	const repairSocketDir = "var/run/libvirt/qemu/run/passt"
	safePath, err := root.AppendAndResolveWithRelativeRoot(repairSocketDir)
	if err != nil {
		return "", err
	}

	return unsafepath.UnsafeAbsolute(safePath.Raw()), nil
}
