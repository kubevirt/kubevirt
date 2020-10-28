package operands

import (
	"reflect"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VmImportHandler genericOperand

func (h VmImportHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	vmImport := NewVMImportForCR(req.Instance)
	res := NewEnsureResult(vmImport)

	key, err := client.ObjectKeyFromObject(vmImport)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for vm-import-operator")
	}
	res.SetName(key.Name)

	found := &vmimportv1beta1.VMImportConfig{}
	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating vm import")
			err = h.Client.Create(req.Ctx, vmImport)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	existingOwners := found.GetOwnerReferences()

	// Previous versions used to have HCO-operator (scope namespace)
	// as the owner of VMImportConfig (scope cluster).
	// It's not legal, so remove that.
	if len(existingOwners) > 0 {
		req.Logger.Info("VMImportConfig has owners, removing...")
		found.SetOwnerReferences([]metav1.OwnerReference{})
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			req.Logger.Error(err, "Failed to remove VMImportConfig's previous owners")
		}
	}

	req.Logger.Info("VM import exists", "vmImport.Namespace", found.Namespace, "vmImport.Name", found.Name)
	if !reflect.DeepEqual(vmImport.Spec, found.Spec) {
		req.Logger.Info("Updating existing VM import")
		vmImport.Spec.DeepCopyInto(&found.Spec)
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		return res.SetUpdated()
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	// Handle VMimport resource conditions
	isReady := handleComponentConditions(req, "VMimport", found.Status.Conditions)

	upgradeDone := req.ComponentUpgradeInProgress && isReady && checkComponentVersion(hcoutil.VMImportEnvV, found.Status.ObservedVersion)
	return res.SetUpgradeDone(upgradeDone)
}

//type IMSConfigHandler genericOperand
//func (h IMSConfigHandler) Ensure(req *common.HcoRequest) *EnsureResult {
//
//}

// NewVMImportForCR returns a VM import CR
func NewVMImportForCR(cr *hcov1beta1.HyperConverged) *vmimportv1beta1.VMImportConfig {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}

	spec := vmimportv1beta1.VMImportConfigSpec{}
	if cr.Spec.Infra.NodePlacement != nil {
		cr.Spec.Infra.NodePlacement.DeepCopyInto(&spec.Infra)
	}
	return &vmimportv1beta1.VMImportConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "vmimport-" + cr.Name,
			Labels: labels,
		},
		Spec: spec,
	}
}
