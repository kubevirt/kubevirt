package operands

import (
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
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

func (h cdiHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func getDefaultFeatureGates() []string {
	return []string{HonorWaitForFirstConsumerGate}
}

func NewCDI(hc *hcov1beta1.HyperConverged, opts ...string) (*cdiv1beta1.CDI, error) {
	uninstallStrategy := cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist
	if hc.Spec.UninstallStrategy != nil && *hc.Spec.UninstallStrategy == hcov1beta1.HyperConvergedUninstallStrategyRemoveWorkloads {
		uninstallStrategy = cdiv1beta1.CDIUninstallStrategyRemoveWorkloads
	}

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

	if hc.Spec.FilesystemOverhead != nil {
		spec.Config.FilesystemOverhead = hc.Spec.FilesystemOverhead.DeepCopy()
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
