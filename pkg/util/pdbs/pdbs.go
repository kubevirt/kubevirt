package pdbs

import (
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/client-go/api/v1"
)

func PDBsForVMI(vmi *virtv1.VirtualMachineInstance, pdbInformer cache.SharedIndexInformer) ([]*v1beta1.PodDisruptionBudget, error) {
	pbds, err := pdbInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return nil, err
	}

	pdbs := []*v1beta1.PodDisruptionBudget{}
	for _, pdb := range pbds {
		p := v1.GetControllerOf(pdb.(*v1beta1.PodDisruptionBudget))
		if p != nil && p.Kind == virtv1.VirtualMachineInstanceGroupVersionKind.Kind &&
			p.Name == vmi.Name {
			pdbs = append(pdbs, pdb.(*v1beta1.PodDisruptionBudget))
		}
	}
	return pdbs, nil
}
