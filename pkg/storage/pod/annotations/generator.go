/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package annotations

import (
	"fmt"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/velero"
)

const computeContainerName = "compute"

type kubeVirtCRProvider interface {
	GetConfigFromKubeVirtCR() *v1.KubeVirt
}

type Generator struct {
	clusterConfig kubeVirtCRProvider
}

func NewGenerator(clusterConfig kubeVirtCRProvider) Generator {
	return Generator{clusterConfig: clusterConfig}
}

func (g Generator) ManagedAnnotationKeys() []string {
	return []string{
		velero.PreBackupHookContainerAnnotation,
		velero.PreBackupHookCommandAnnotation,
		velero.PreBackupHookTimeoutAnnotation,
		velero.PostBackupHookContainerAnnotation,
		velero.PostBackupHookCommandAnnotation,
	}
}

func (g Generator) Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	// Check VMI annotation first, fallback to kubevirt CR if not set
	skipValue, hasSkipValue := vmi.Annotations[velero.SkipHooksAnnotation]
	if !hasSkipValue && g.clusterConfig != nil {
		kubeVirtCR := g.clusterConfig.GetConfigFromKubeVirtCR()
		if kubeVirtCR != nil {
			skipValue = kubeVirtCR.Annotations[velero.SkipHooksAnnotation]
		}
	}

	annotations := map[string]string{}

	skip, _ := strconv.ParseBool(skipValue)
	if !skip {
		annotations[velero.PreBackupHookContainerAnnotation] = computeContainerName
		annotations[velero.PreBackupHookCommandAnnotation] = fmt.Sprintf(
			"[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", %q, \"--namespace\", %q]",
			vmi.Name,
			vmi.Namespace,
		)
		annotations[velero.PreBackupHookTimeoutAnnotation] = "60s"
		annotations[velero.PostBackupHookContainerAnnotation] = computeContainerName
		annotations[velero.PostBackupHookCommandAnnotation] = fmt.Sprintf(
			"[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", %q, \"--namespace\", %q]",
			vmi.Name,
			vmi.Namespace,
		)
	}

	return annotations, nil
}
