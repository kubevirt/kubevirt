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
 */

package pdbs

import (
	"strings"

	policyv1 "k8s.io/api/policy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
)

func PDBsForVMI(vmi *virtv1.VirtualMachineInstance, pdbIndexer cache.Indexer) ([]*policyv1.PodDisruptionBudget, error) {
	pbds, err := pdbIndexer.ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return nil, err
	}

	pdbs := []*policyv1.PodDisruptionBudget{}
	for _, pdb := range pbds {
		p := v1.GetControllerOf(pdb.(*policyv1.PodDisruptionBudget))
		if p != nil && p.Kind == virtv1.VirtualMachineInstanceGroupVersionKind.Kind &&
			p.Name == vmi.Name {
			pdbs = append(pdbs, pdb.(*policyv1.PodDisruptionBudget))
		}
	}
	return pdbs, nil
}

func IsPDBFromOldMigrationController(pdb *policyv1.PodDisruptionBudget) bool {
	// The pdb might be from an old migration-controller that used to create 2-pdbs per migration
	_, migrationLabelExists := pdb.ObjectMeta.Labels[virtv1.MigrationNameLabel]
	if migrationLabelExists && strings.HasPrefix(pdb.Name, "kubevirt-migration-pdb-") {
		return true
	}

	owner := v1.GetControllerOf(pdb)
	ownedByVMI := owner != nil && owner.Kind == virtv1.VirtualMachineInstanceGroupVersionKind.Kind
	if ownedByVMI && !migrationLabelExists && pdb.Spec.MinAvailable != nil && pdb.Spec.MinAvailable.IntValue() == 2 {
		return true
	}
	return false
}
