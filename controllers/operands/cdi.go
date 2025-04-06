package operands

import (
	"errors"
	"reflect"
	"sync"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/reformatobj"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	honorWaitForFirstConsumerGate = "HonorWaitForFirstConsumer"
	dataVolumeClaimAdoptionGate   = "DataVolumeClaimAdoption"
	cdiConfigAuthorityAnnotation  = "cdi.kubevirt.io/configAuthority"
)

type cdiHandler genericOperand

func newCdiHandler(Client client.Client, Scheme *runtime.Scheme) *cdiHandler {
	return &cdiHandler{
		Client: Client,
		Scheme: Scheme,
		crType: "CDI",
		hooks:  &cdiHooks{Client: Client, Scheme: Scheme},
	}
}

type cdiHooks struct {
	sync.Mutex
	Client client.Client
	Scheme *runtime.Scheme
	cache  *cdiv1beta1.CDI
}

func (h *cdiHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	h.Lock()
	defer h.Unlock()

	if h.cache == nil {
		cdi, err := NewCDI(hc)
		if err != nil {
			return nil, err
		}
		h.cache = cdi
	}
	return h.cache, nil
}
func (*cdiHooks) getEmptyCr() client.Object { return &cdiv1beta1.CDI{} }
func (*cdiHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*cdiv1beta1.CDI).Status.Conditions)
}
func (*cdiHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*cdiv1beta1.CDI)
	return checkComponentVersion(util.CdiVersionEnvV, found.Status.ObservedVersion)
}
func (h *cdiHooks) reset() {
	h.Lock()
	defer h.Unlock()

	h.cache = nil
}

func (*cdiHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	cdi, ok1 := required.(*cdiv1beta1.CDI)
	found, ok2 := exists.(*cdiv1beta1.CDI)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to CDI")
	}

	if !reflect.DeepEqual(found.Spec, cdi.Spec) ||
		!util.CompareLabels(cdi, found) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing CDI's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated CDI's Spec to its opinionated values")
			overwritten = true
		}
		util.MergeLabels(&cdi.ObjectMeta, &found.ObjectMeta)
		cdi.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, overwritten, nil
	}
	return false, false, nil
}

func (*cdiHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func getDefaultFeatureGates() []string {
	return []string{honorWaitForFirstConsumerGate, dataVolumeClaimAdoptionGate}
}

func NewCDI(hc *hcov1beta1.HyperConverged, opts ...string) (*cdiv1beta1.CDI, error) {
	uninstallStrategy := cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist
	if hc.Spec.UninstallStrategy == hcov1beta1.HyperConvergedUninstallStrategyRemoveWorkloads {
		uninstallStrategy = cdiv1beta1.CDIUninstallStrategyRemoveWorkloads
	}

	spec := cdiv1beta1.CDISpec{
		UninstallStrategy: &uninstallStrategy,
		Config: &cdiv1beta1.CDIConfigSpec{
			FeatureGates:       getDefaultFeatureGates(),
			TLSSecurityProfile: openshift2CdiSecProfile(util.GetClusterInfo().GetTLSSecurityProfile(hc.Spec.TLSSecurityProfile)),
		},
		CertConfig: &cdiv1beta1.CDICertConfig{
			CA: &cdiv1beta1.CertConfig{
				Duration:    hc.Spec.CertConfig.CA.Duration,
				RenewBefore: hc.Spec.CertConfig.CA.RenewBefore,
			},
			Server: &cdiv1beta1.CertConfig{
				Duration:    hc.Spec.CertConfig.Server.Duration,
				RenewBefore: hc.Spec.CertConfig.Server.RenewBefore,
			},
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
		hc.Spec.Infra.NodePlacement.DeepCopyInto(&spec.Infra.NodePlacement)
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		hc.Spec.Workloads.NodePlacement.DeepCopyInto(&spec.Workloads)
	}

	if lv := hc.Spec.LogVerbosityConfig; lv != nil {
		spec.Config.LogVerbosity = lv.CDI
	}

	cdi := NewCDIWithNameOnly(hc, opts...)
	cdi.Spec = spec

	if err := applyPatchToSpec(hc, common.JSONPatchCDIAnnotationName, cdi); err != nil {
		return nil, err
	}

	return reformatobj.ReformatObj(cdi)
}

func NewCDIWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *cdiv1beta1.CDI {
	return &cdiv1beta1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "cdi-" + hc.Name,
			Labels:      getLabels(hc, util.AppComponentStorage),
			Namespace:   getNamespace(util.UndefinedNamespace, opts),
			Annotations: map[string]string{cdiConfigAuthorityAnnotation: ""},
		},
	}
}

func openshift2CdiSecProfile(hcProfile *openshiftconfigv1.TLSSecurityProfile) *cdiv1beta1.TLSSecurityProfile {
	var custom *cdiv1beta1.CustomTLSProfile
	if hcProfile.Custom != nil {
		custom = &cdiv1beta1.CustomTLSProfile{
			TLSProfileSpec: cdiv1beta1.TLSProfileSpec{
				Ciphers:       hcProfile.Custom.Ciphers,
				MinTLSVersion: cdiv1beta1.TLSProtocolVersion(hcProfile.Custom.MinTLSVersion),
			},
		}
	}

	return &cdiv1beta1.TLSSecurityProfile{
		Type:         cdiv1beta1.TLSProfileType(hcProfile.Type),
		Old:          (*cdiv1beta1.OldTLSProfile)(hcProfile.Old),
		Intermediate: (*cdiv1beta1.IntermediateTLSProfile)(hcProfile.Intermediate),
		Modern:       (*cdiv1beta1.ModernTLSProfile)(hcProfile.Modern),
		Custom:       custom,
	}
}
