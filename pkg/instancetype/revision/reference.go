/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package revision

import virtv1 "kubevirt.io/api/core/v1"

func HasControllerRevisionRef(ref *virtv1.InstancetypeStatusRef) bool {
	return ref != nil && ref.ControllerRevisionRef != nil && ref.ControllerRevisionRef.Name != ""
}
