package operands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/reformatobj"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	// This is initially set to 2 replicas, to maintain the behavior of the previous SSP operator.
	// After SSP implements its defaulting webhook, we can change this to 0 replicas,
	// and let the webhook set the default.
	defaultTemplateValidatorReplicas = int32(2)

	defaultCommonTemplatesNamespace = util.OpenshiftNamespace

	dataImportCronTemplatesFileLocation = "./dataImportCronTemplates"

	CDIImmediateBindAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
)

var (
	// dataImportCronTemplateHardCodedMap are set of data import cron template configurations. The handler reads a list
	// of data import cron templates from a local file and updates SSP with the up-to-date list
	dataImportCronTemplateHardCodedMap map[string]hcov1beta1.DataImportCronTemplate
)

func init() {
	if err := readDataImportCronTemplatesFromFile(); err != nil {
		panic(fmt.Errorf("can't process the data import cron template file; %s; %w", err.Error(), err))
	}
}

type sspHandler genericOperand

func newSspHandler(Client client.Client, Scheme *runtime.Scheme) *sspHandler {
	return &sspHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "SSP",
		setControllerReference: false,
		hooks:                  &sspHooks{},
	}
}

type sspHooks struct {
	sync.Mutex
	cache        *sspv1beta2.SSP
	dictStatuses []hcov1beta1.DataImportCronTemplateStatus
}

func (h *sspHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	h.Lock()
	defer h.Unlock()

	if h.cache == nil {
		ssp, dictStatus, err := NewSSP(hc)
		if err != nil {
			return nil, err
		}
		h.cache = ssp
		h.dictStatuses = dictStatus
	}
	return h.cache, nil
}

func (*sspHooks) getEmptyCr() client.Object { return &sspv1beta2.SSP{} }
func (*sspHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*sspv1beta2.SSP).Status.Conditions)
}
func (*sspHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1beta2.SSP)
	return checkComponentVersion(util.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h *sspHooks) reset() {
	h.Lock()
	defer h.Unlock()

	h.cache = nil
	h.dictStatuses = nil
}

func (*sspHooks) updateCr(req *common.HcoRequest, client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ssp, ok1 := required.(*sspv1beta2.SSP)
	found, ok2 := exists.(*sspv1beta2.SSP)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to SSP")
	}
	if !reflect.DeepEqual(ssp.Spec, found.Spec) ||
		!util.CompareLabels(ssp, found) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing SSP's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated SSP's Spec to its opinionated values")
		}
		util.MergeLabels(&ssp.ObjectMeta, &found.ObjectMeta)
		ssp.Spec.DeepCopyInto(&found.Spec)
		err := client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h *sspHooks) justBeforeComplete(req *common.HcoRequest) {
	if !reflect.DeepEqual(h.dictStatuses, req.Instance.Status.DataImportCronTemplates) {
		req.Instance.Status.DataImportCronTemplates = h.dictStatuses
		req.StatusDirty = true
	}
}

func NewSSP(hc *hcov1beta1.HyperConverged, opts ...string) (*sspv1beta2.SSP, []hcov1beta1.DataImportCronTemplateStatus, error) {
	templatesNamespace := defaultCommonTemplatesNamespace

	if hc.Spec.CommonTemplatesNamespace != nil {
		templatesNamespace = *hc.Spec.CommonTemplatesNamespace
	}

	applyDataImportSchedule(hc)

	dataImportCronStatuses, err := getDataImportCronTemplates(hc)
	if err != nil {
		return nil, nil, err
	}

	var dataImportCronTemplates []hcov1beta1.DataImportCronTemplate
	for _, dictStatus := range dataImportCronStatuses {
		dataImportCronTemplates = append(dataImportCronTemplates, dictStatus.DataImportCronTemplate)
	}

	spec := sspv1beta2.SSPSpec{
		TemplateValidator: &sspv1beta2.TemplateValidator{
			Replicas: ptr.To(defaultTemplateValidatorReplicas),
		},
		CommonTemplates: sspv1beta2.CommonTemplates{
			Namespace:               templatesNamespace,
			DataImportCronTemplates: hcoDictSliceToSSP(dataImportCronTemplates),
		},
		// NodeLabeller field is explicitly initialized to its zero-value,
		// in order to future-proof from bugs if SSP changes it to pointer-type,
		// causing nil pointers dereferences at the DeepCopyInto() below.
		TLSSecurityProfile: util.GetClusterInfo().GetTLSSecurityProfile(hc.Spec.TLSSecurityProfile),
		FeatureGates:       &sspv1beta2.FeatureGates{},
	}

	if hc.Spec.DeployVMConsoleProxy != nil {
		spec.TokenGenerationService = &sspv1beta2.TokenGenerationService{
			Enabled: *hc.Spec.DeployVMConsoleProxy,
		}
	}

	// Disable common-instancetypes deployment by SSP from 4.16, now handled by virt-operator
	spec.FeatureGates.DeployCommonInstancetypes = ptr.To(false)

	if hc.Spec.Infra.NodePlacement != nil {
		spec.TemplateValidator.Placement = hc.Spec.Infra.NodePlacement.DeepCopy()
	}

	ssp := NewSSPWithNameOnly(hc)
	ssp.Spec = spec

	if err = applyPatchToSpec(hc, common.JSONPatchSSPAnnotationName, ssp); err != nil {
		return nil, nil, err
	}

	ssp, err = reformatobj.ReformatObj(ssp)
	if err != nil {
		return nil, nil, err
	}

	return ssp, dataImportCronStatuses, nil
}

func NewSSPWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *sspv1beta2.SSP {
	return &sspv1beta2.SSP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssp-" + hc.Name,
			Labels:    getLabels(hc, util.AppComponentSchedule),
			Namespace: getNamespace(hc.Namespace, opts),
		},
	}
}

var getDataImportCronTemplatesFileLocation = func() string {
	return dataImportCronTemplatesFileLocation
}

func readDataImportCronTemplatesFromFile() error {
	dataImportCronTemplateHardCodedMap = make(map[string]hcov1beta1.DataImportCronTemplate)

	fileLocation := getDataImportCronTemplatesFileLocation()

	err := util.ValidateManifestDir(fileLocation)
	if err != nil {
		return errors.Unwrap(err) // if not wrapped, then it's not an error that stops processing, and it returns nil
	}

	return filepath.Walk(fileLocation, func(filePath string, info fs.FileInfo, internalErr error) error {
		if internalErr != nil {
			return internalErr
		}

		if !info.IsDir() && path.Ext(info.Name()) == ".yaml" {
			file, internalErr := os.Open(filePath)
			if internalErr != nil {
				logger.Error(internalErr, "Can't open the dataImportCronTemplate yaml file", "file name", filePath)
				return internalErr
			}

			dataImportCronTemplateFromFile := make([]hcov1beta1.DataImportCronTemplate, 0)
			internalErr = util.UnmarshalYamlFileToObject(file, &dataImportCronTemplateFromFile)
			if internalErr != nil {
				return internalErr
			}

			for _, dict := range dataImportCronTemplateFromFile {
				dataImportCronTemplateHardCodedMap[dict.Name] = dict
			}
		}

		return nil
	})
}

func getDataImportCronTemplates(hc *hcov1beta1.HyperConverged) ([]hcov1beta1.DataImportCronTemplateStatus, error) {
	crDicts, err := getDicMapFromCr(hc)
	if err != nil {
		return nil, err
	}

	var dictList []hcov1beta1.DataImportCronTemplateStatus
	if hc.Spec.EnableCommonBootImageImport != nil && *hc.Spec.EnableCommonBootImageImport {
		dictList = getCommonDicts(dictList, crDicts, hc)
	}
	dictList = getCustomDicts(dictList, crDicts)

	sort.Sort(dataImportTemplateSlice(dictList))

	return dictList, nil
}

func getCommonDicts(list []hcov1beta1.DataImportCronTemplateStatus, crDicts map[string]hcov1beta1.DataImportCronTemplate, hc *hcov1beta1.HyperConverged) []hcov1beta1.DataImportCronTemplateStatus {
	for dictName, commonDict := range dataImportCronTemplateHardCodedMap {
		targetDict := hcov1beta1.DataImportCronTemplateStatus{
			DataImportCronTemplate: *commonDict.DeepCopy(),
			Status: hcov1beta1.DataImportCronStatus{
				CommonTemplate: true,
			},
		}

		if crDict, found := crDicts[dictName]; found {
			if !isDataImportCronTemplateEnabled(crDict) {
				continue
			}

			// if the schedule is missing, copy from the common dict:
			if len(crDict.Spec.Schedule) == 0 {
				crDict.Spec.Schedule = targetDict.Spec.Schedule
			}
			targetDict.Spec = crDict.Spec.DeepCopy()
			targetDict.Namespace = crDict.Namespace
			targetDict.Status.Modified = true
		} else if ns := hc.Spec.CommonBootImageNamespace; ns != nil && len(*ns) > 0 {
			targetDict.Namespace = *ns
		}

		list = append(list, targetDict)
	}

	return list
}

func isDataImportCronTemplateEnabled(dict hcov1beta1.DataImportCronTemplate) bool {
	annotationVal, found := dict.Annotations[util.DataImportCronEnabledAnnotation]
	return !found || strings.ToLower(annotationVal) == "true"
}

func getCustomDicts(list []hcov1beta1.DataImportCronTemplateStatus, crDicts map[string]hcov1beta1.DataImportCronTemplate) []hcov1beta1.DataImportCronTemplateStatus {
	for dictName, crDict := range crDicts {
		if !isDataImportCronTemplateEnabled(crDict) {
			continue
		}

		if _, isCommon := dataImportCronTemplateHardCodedMap[dictName]; !isCommon {
			list = append(list, hcov1beta1.DataImportCronTemplateStatus{
				DataImportCronTemplate: *crDict.DeepCopy(),
				Status: hcov1beta1.DataImportCronStatus{
					CommonTemplate: false,
				},
			})
		}
	}

	return list
}

func getDicMapFromCr(hc *hcov1beta1.HyperConverged) (map[string]hcov1beta1.DataImportCronTemplate, error) {
	dictMap := make(map[string]hcov1beta1.DataImportCronTemplate)
	for _, dict := range hc.Spec.DataImportCronTemplates {
		_, foundCustom := dictMap[dict.Name]
		if foundCustom {
			return nil, fmt.Errorf("%s DataImportCronTable is already defined", dict.Name)
		}
		dictMap[dict.Name] = dict
	}
	return dictMap, nil
}

func applyDataImportSchedule(hc *hcov1beta1.HyperConverged) {
	if hc.Status.DataImportSchedule != "" {
		overrideDataImportSchedule(hc.Status.DataImportSchedule)
	}
}

func overrideDataImportSchedule(schedule string) {
	for dictName := range dataImportCronTemplateHardCodedMap {
		dict := dataImportCronTemplateHardCodedMap[dictName]
		dict.Spec.Schedule = schedule
		dataImportCronTemplateHardCodedMap[dictName] = dict
	}
}

// implement sort.Interface
type dataImportTemplateSlice []hcov1beta1.DataImportCronTemplateStatus

func (d dataImportTemplateSlice) Len() int           { return len(d) }
func (d dataImportTemplateSlice) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d dataImportTemplateSlice) Less(i, j int) bool { return d[i].Name < d[j].Name }

func hcoDictToSSP(hcoDict hcov1beta1.DataImportCronTemplate) sspv1beta2.DataImportCronTemplate {
	spec := cdiv1beta1.DataImportCronSpec{}
	if hcoDict.Spec != nil {
		hcoDict.Spec.DeepCopyInto(&spec)
	}

	dict := sspv1beta2.DataImportCronTemplate{
		ObjectMeta: *hcoDict.ObjectMeta.DeepCopy(),
		Spec:       spec,
	}

	if dict.Annotations == nil {
		dict.Annotations = make(map[string]string)
	}

	if _, foundAnnotation := dict.Annotations[CDIImmediateBindAnnotation]; !foundAnnotation {
		dict.Annotations[CDIImmediateBindAnnotation] = "true"
	}

	return dict
}

func hcoDictSliceToSSP(hcoDicts []hcov1beta1.DataImportCronTemplate) []sspv1beta2.DataImportCronTemplate {
	if len(hcoDicts) == 0 {
		return nil
	}

	res := make([]sspv1beta2.DataImportCronTemplate, len(hcoDicts))

	for i, hcoDict := range hcoDicts {
		res[i] = hcoDictToSSP(hcoDict)
	}

	return res
}
