package pdbs

import (
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
