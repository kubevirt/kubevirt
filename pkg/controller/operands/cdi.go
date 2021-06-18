package operands

import (
	"errors"
	"reflect"

	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	cdiRoleName                   = "hco.kubevirt.io:config-reader"
	HonorWaitForFirstConsumerGate = "HonorWaitForFirstConsumer"
	cdiConfigAuthorityAnnotation  = "cdi.kubevirt.io/configAuthority"
)

type cdiHandler genericOperand

func newCdiHandler(Client client.Client, Scheme *runtime.Scheme) *cdiHandler {
	return &cdiHandler{
		Client: Client,
		Scheme: Scheme,
		crType: "CDI",
		// Previous versions used to have HCO-operator (scope namespace)
		// as the owner of CDI (scope cluster).
		// It's not legal, so remove that.
		removeExistingOwner: true,
		hooks:               &cdiHooks{Client: Client, Scheme: Scheme},
	}
}

type cdiHooks struct {
	Client client.Client
	Scheme *runtime.Scheme
	cache  *cdiv1beta1.CDI
}

func (h *cdiHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		cdi, err := NewCDI(hc)
		if err != nil {
			return nil, err
		}
		h.cache = cdi
	}
	return h.cache, nil
}
func (h cdiHooks) getEmptyCr() client.Object { return &cdiv1beta1.CDI{} }
func (h cdiHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*cdiv1beta1.CDI).Status.Conditions)
}
func (h cdiHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*cdiv1beta1.CDI)
	return checkComponentVersion(hcoutil.CdiVersionEnvV, found.Status.ObservedVersion)
}
func (h cdiHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*cdiv1beta1.CDI).ObjectMeta
}
func (h *cdiHooks) reset() {
	h.cache = nil
}

func (h *cdiHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	cdi, ok1 := required.(*cdiv1beta1.CDI)
	found, ok2 := exists.(*cdiv1beta1.CDI)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to CDI")
	}

	if !reflect.DeepEqual(found.Spec, cdi.Spec) ||
		!reflect.DeepEqual(found.Labels, cdi.Labels) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing CDI's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated CDI's Spec to its opinionated values")
			overwritten = true
		}
		util.DeepCopyLabels(&cdi.ObjectMeta, &found.ObjectMeta)
		cdi.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, overwritten, nil
	}
	return false, false, nil
}

func getDefaultFeatureGates() []string {
	return []string{HonorWaitForFirstConsumerGate}
}

func (h *cdiHooks) postFound(req *common.HcoRequest, exists runtime.Object) error {
	err := h.ensureKubeVirtStorageRole(req)
	if err != nil {
		return err
	}

	err = h.ensureKubeVirtStorageRoleBinding(req)
	if err != nil {
		return err
	}

	return nil
}

func NewCDI(hc *hcov1beta1.HyperConverged, opts ...string) (*cdiv1beta1.CDI, error) {
	uninstallStrategy := cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist

	spec := cdiv1beta1.CDISpec{
		UninstallStrategy: &uninstallStrategy,
		Config: &cdiv1beta1.CDIConfigSpec{
			FeatureGates: getDefaultFeatureGates(),
		},
	}

	if hc.Spec.ResourceRequirements != nil && hc.Spec.ResourceRequirements.StorageWorkloads != nil {
		spec.Config.PodResourceRequirements = hc.Spec.ResourceRequirements.StorageWorkloads.DeepCopy()
	}

	if hc.Spec.ScratchSpaceStorageClass != nil {
		spec.Config.ScratchSpaceStorageClass = hc.Spec.ScratchSpaceStorageClass
	}

	if hc.Spec.StorageImport != nil {
		if length := len(hc.Spec.StorageImport.InsecureRegistries); length > 0 {
			spec.Config.InsecureRegistries = make([]string, length)
			copy(spec.Config.InsecureRegistries, hc.Spec.StorageImport.InsecureRegistries)
		}
	}

	if hc.Spec.Infra.NodePlacement != nil {
		hc.Spec.Infra.NodePlacement.DeepCopyInto(&spec.Infra)
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		hc.Spec.Workloads.NodePlacement.DeepCopyInto(&spec.Workloads)
	}

	certConfig := hc.Spec.CertConfig

	spec.CertConfig = &cdiv1beta1.CDICertConfig{}

	spec.CertConfig.CA = &cdiv1beta1.CertConfig{
		Duration:    certConfig.CA.Duration.DeepCopy(),
		RenewBefore: certConfig.CA.RenewBefore.DeepCopy(),
	}

	spec.CertConfig.Server = &cdiv1beta1.CertConfig{
		Duration:    certConfig.Server.Duration.DeepCopy(),
		RenewBefore: certConfig.Server.RenewBefore.DeepCopy(),
	}

	cdi := NewCDIWithNameOnly(hc, opts...)
	cdi.Spec = spec

	if err := applyPatchToSpec(hc, common.JSONPatchCDIAnnotationName, cdi); err != nil {
		return nil, err
	}

	return cdi, nil
}

func NewCDIWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *cdiv1beta1.CDI {
	return &cdiv1beta1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "cdi-" + hc.Name,
			Labels:      getLabels(hc, hcoutil.AppComponentStorage),
			Namespace:   getNamespace(hcoutil.UndefinedNamespace, opts),
			Annotations: map[string]string{cdiConfigAuthorityAnnotation: ""},
		},
	}
}

func (h *cdiHooks) ensureKubeVirtStorageRole(req *common.HcoRequest) error {
	kubevirtStorageRole := NewKubeVirtStorageRoleForCR(req.Instance, req.Namespace, h.Scheme)

	found := &rbacv1.Role{}
	err := h.Client.Get(req.Ctx, client.ObjectKeyFromObject(kubevirtStorageRole), found)
	if apierrors.IsNotFound(err) {
		req.Logger.Info("Creating kubevirt storage role")
		return h.Client.Create(req.Ctx, kubevirtStorageRole)
	}

	if err != nil {
		return err
	}

	if !reflect.DeepEqual(found.Labels, kubevirtStorageRole.Labels) {
		req.Logger.Info("Updating KubeVirt storage role for labels")
		util.DeepCopyLabels(&kubevirtStorageRole.ObjectMeta, &found.ObjectMeta)
		return h.Client.Update(req.Ctx, found)
	}

	req.Logger.Info("KubeVirt storage role already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return err
	}
	if err = objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef); err != nil {
		return err
	}

	return nil
}

func (h *cdiHooks) ensureKubeVirtStorageRoleBinding(req *common.HcoRequest) error {
	kubevirtStorageRoleBinding := NewKubeVirtStorageRoleBindingForCR(req.Instance, req.Namespace, h.Scheme)

	found := &rbacv1.RoleBinding{}
	err := h.Client.Get(req.Ctx, client.ObjectKeyFromObject(kubevirtStorageRoleBinding), found)
	if err != nil && apierrors.IsNotFound(err) {
		req.Logger.Info("Creating kubevirt storage rolebinding")
		return h.Client.Create(req.Ctx, kubevirtStorageRoleBinding)
	}

	if err != nil {
		return err
	}

	if !reflect.DeepEqual(found.Labels, kubevirtStorageRoleBinding.Labels) {
		req.Logger.Info("Updating KubeVirt storage rolebinding for labels")
		util.DeepCopyLabels(&kubevirtStorageRoleBinding.ObjectMeta, &found.ObjectMeta)
		return h.Client.Update(req.Ctx, found)
	}

	req.Logger.Info("KubeVirt storage rolebinding already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(h.Scheme, found)
	if err != nil {
		return err
	}
	err = objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)
	if err != nil {
		return err
	}

	return nil
}

func NewKubeVirtStorageRoleForCR(cr *hcov1beta1.HyperConverged, namespace string, scheme *runtime.Scheme) *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cdiRoleName,
			Labels:    getLabels(cr, hcoutil.AppComponentStorage),
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"kubevirt-storage-class-defaults"},
				Verbs:         []string{"get", "watch", "list"},
			},
		},
	}

	_ = controllerutil.SetControllerReference(cr, role, scheme)
	return role
}

func NewKubeVirtStorageRoleBindingForCR(cr *hcov1beta1.HyperConverged, namespace string, scheme *runtime.Scheme) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cdiRoleName,
			Labels:    getLabels(cr, hcoutil.AppComponentStorage),
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     cdiRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			},
		},
	}

	_ = controllerutil.SetControllerReference(cr, roleBinding, scheme)

	return roleBinding
}

// ************** CDI Storage Config Handler **************
type storageConfigHandler genericOperand

func newStorageConfigHandler(Client client.Client, Scheme *runtime.Scheme) *storageConfigHandler {
	return &storageConfigHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "StorageConfigmap",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &storageConfigHooks{},
	}
}

type storageConfigHooks struct{}

func (h storageConfigHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewKubeVirtStorageConfigForCR(hc, hc.Namespace), nil
}
func (h storageConfigHooks) getEmptyCr() client.Object                              { return &corev1.ConfigMap{} }
func (h storageConfigHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h storageConfigHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.ConfigMap).ObjectMeta
}
func (h *storageConfigHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	storageConfig, ok1 := required.(*corev1.ConfigMap)
	found, ok2 := exists.(*corev1.ConfigMap)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to a ConfigMap")
	}

	// Merge old & new values. This is necessary in case the user has defined
	// their own chosen values in the configmap.
	needsUpdate := false
	for key, value := range storageConfig.Data {
		if found.Data[key] != value {
			found.Data[key] = value
			needsUpdate = true
		}
	}

	if !reflect.DeepEqual(found.Labels, storageConfig.Labels) {
		util.DeepCopyLabels(&storageConfig.ObjectMeta, &found.ObjectMeta)
		needsUpdate = true
	}

	if needsUpdate {
		req.Logger.Info("Updating existing KubeVirt Storage Configmap to its default values")
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func NewKubeVirtStorageConfigForCR(cr *hcov1beta1.HyperConverged, namespace string) *corev1.ConfigMap {
	localSC := "local-sc"
	if cr.Spec.LocalStorageClassName != "" {
		localSC = cr.Spec.LocalStorageClassName
	}

	ocsRBD := "ocs-storagecluster-ceph-rbd"

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-storage-class-defaults",
			Labels:    getLabels(cr, hcoutil.AppComponentStorage),
			Namespace: namespace,
		},
		Data: map[string]string{
			"accessMode":            "ReadWriteOnce",
			"volumeMode":            "Filesystem",
			localSC + ".accessMode": "ReadWriteOnce",
			localSC + ".volumeMode": "Filesystem",
			ocsRBD + ".accessMode":  "ReadWriteMany",
			ocsRBD + ".volumeMode":  "Block",
		},
	}
}
