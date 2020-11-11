package operands

import (
	"errors"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type vmImportHandler genericOperand

func newVmImportHandler(Client client.Client, Scheme *runtime.Scheme) *vmImportHandler {
	handler := &vmImportHandler{
		Client: Client,
		Scheme: Scheme,
		crType: "vmImport",
		isCr:   true,
		// Previous versions used to have HCO-operator (scope namespace)
		// as the owner of VMImportConfig (scope cluster).
		// It's not legal, so remove that.
		removeExistingOwner: true,
		getFullCr: func(hc *hcov1beta1.HyperConverged) runtime.Object {
			return NewVMImportForCR(hc)
		},
		getEmptyCr: func() runtime.Object { return &vmimportv1beta1.VMImportConfig{} },
		getConditions: func(cr runtime.Object) []conditionsv1.Condition {
			return cr.(*vmimportv1beta1.VMImportConfig).Status.Conditions
		},
		checkComponentVersion: func(cr runtime.Object) bool {
			found := cr.(*vmimportv1beta1.VMImportConfig)
			return checkComponentVersion(hcoutil.VMImportEnvV, found.Status.ObservedVersion)
		},
		getObjectMeta: func(cr runtime.Object) *metav1.ObjectMeta {
			return &cr.(*vmimportv1beta1.VMImportConfig).ObjectMeta
		},
	}

	handler.updateCr = handler.updateCrImp

	return handler
}

func (h *vmImportHandler) updateCrImp(req *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	vmImport, ok1 := required.(*vmimportv1beta1.VMImportConfig)
	found, ok2 := exists.(*vmimportv1beta1.VMImportConfig)

	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to vmImport")
	}

	if !reflect.DeepEqual(found.Spec, vmImport.Spec) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing vmImport's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated vmImport's Spec to its opinionated values")
		}
		vmImport.Spec.DeepCopyInto(&found.Spec)
		err := h.Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func (h vmImportHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	handler := genericOperand(h)
	return handler.ensure(req)
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

func newImsConfigHandler(Client client.Client, Scheme *runtime.Scheme) *imsConfigHandler {
	handler := &imsConfigHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "IMS Configmap",
		isCr:                   false,
		removeExistingOwner:    false,
		setControllerReference: true,
		getFullCr: func(hc *hcov1beta1.HyperConverged) runtime.Object {
			return NewIMSConfigForCR(hc, hc.Namespace)
		},
		getEmptyCr: func() runtime.Object { return &corev1.ConfigMap{} },
		getObjectMeta: func(cr runtime.Object) *metav1.ObjectMeta {
			return &cr.(*corev1.ConfigMap).ObjectMeta
		},
		validate: validateImsConfig,
	}

	handler.updateCr = handler.updateCrImp

	return handler
}

func (h *imsConfigHandler) updateCrImp(req *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	imsConfig, ok1 := required.(*corev1.ConfigMap)
	found, ok2 := exists.(*corev1.ConfigMap)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to a ConfigMap")
	}

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
		err := h.Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, false, nil
	}

	return false, false, nil
}

func validateImsConfig() error {
	if os.Getenv("CONVERSION_CONTAINER") == "" {
		return errors.New("ims-conversion-container not specified")
	}

	if os.Getenv("VMWARE_CONTAINER") == "" {
		return errors.New("ims-vmware-container not specified")
	}
	return nil
}

func (h imsConfigHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	handler := genericOperand(h)
	return handler.ensure(req)
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
