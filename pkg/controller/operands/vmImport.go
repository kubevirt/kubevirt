package operands

import (
	"errors"
	corev1 "k8s.io/api/core/v1"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

type vmImportHandler genericOperand

func (h vmImportHandler) Ensure(req *common.HcoRequest) *EnsureResult {
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

type imsConfigHandler genericOperand

func (h imsConfigHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	imsConfig := NewIMSConfigForCR(req.Instance, req.Namespace)
	res := NewEnsureResult(imsConfig)
	if os.Getenv("CONVERSION_CONTAINER") == "" {
		return res.Error(errors.New("ims-conversion-container not specified"))
	}

	if os.Getenv("VMWARE_CONTAINER") == "" {
		return res.Error(errors.New("ims-vmware-container not specified"))
	}

	err := controllerutil.SetControllerReference(req.Instance, imsConfig, h.Scheme)
	if err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(imsConfig)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for IMS Configmap")
	}

	res.SetName(key.Name)
	found := &corev1.ConfigMap{}

	err = h.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating IMS Configmap")
			err = h.Client.Create(req.Ctx, imsConfig)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("IMS Configmap already exists", "imsConfigMap.Namespace", found.Namespace, "imsConfigMap.Name", found.Name)

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	// in an ideal world HCO should be managing the whole config map,
	// now due to a bad design only a few values of this config map are
	// really managed by HCO while others are managed by other entities
	// TODO: fix this bad design splitting the config map into two distinct objects and reconcile the whole object here
	needsUpdate := false
	for key, value := range imsConfig.Data {
		if found.Data[key] != value {
			found.Data[key] = value
			needsUpdate = true
		}
	}
	if needsUpdate {
		req.Logger.Info("Updating existing IMS Configmap to its default values")
		err = h.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		return res.SetUpdated()
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

func NewIMSConfigForCR(cr *hcov1beta1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v2v-vmware",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"v2v-conversion-image":              os.Getenv("CONVERSION_CONTAINER"),
			"kubevirt-vmware-image":             os.Getenv("VMWARE_CONTAINER"),
			"kubevirt-vmware-image-pull-policy": "IfNotPresent",
		},
	}
}
