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
