package operands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	// This is initially set to 2 replicas, to maintain the behavior of the previous SSP operator.
	// After SSP implements its defaulting webhook, we can change this to 0 replicas,
	// and let the webhook set the default.
	defaultTemplateValidatorReplicas = 2

	defaultCommonTemplatesNamespace = hcoutil.OpenshiftNamespace

	dataImportCronTemplatesFileLocation = "./dataImportCronTemplates"
)

var (
	// dataImportCronTemplateHardCodedList are set of data import cron template configurations. The handler reads a list
	// of data import cron templates from a local file and updates SSP with the up-to-date list
	dataImportCronTemplateHardCodedList  []sspv1beta1.DataImportCronTemplate
	dataImportCronTemplateHardCodedNames map[string]struct{}
)

func init() {
	if err := readDataImportCronTemplatesFromFile(); err != nil {
		panic(fmt.Errorf("can't process the data import cron template file; %s; %w", err.Error(), err))
	}
}

type sspHandler struct {
	genericOperand
}

func newSspHandler(Client client.Client, Scheme *runtime.Scheme) *sspHandler {
	return &sspHandler{
		genericOperand: genericOperand{
			Client:                 Client,
			Scheme:                 Scheme,
			crType:                 "SSP",
			removeExistingOwner:    false,
			setControllerReference: false,
			hooks:                  &sspHooks{},
		},
	}
}

type sspHooks struct {
	cache *sspv1beta1.SSP
}

func (h *sspHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		ssp, err := NewSSP(hc)
		if err != nil {
			return nil, err
		}
		h.cache = ssp
	}
	return h.cache, nil
}

func (h sspHooks) getEmptyCr() client.Object { return &sspv1beta1.SSP{} }
func (h sspHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return osConditionsToK8s(cr.(*sspv1beta1.SSP).Status.Conditions)
}
func (h sspHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*sspv1beta1.SSP)
	return checkComponentVersion(hcoutil.SspVersionEnvV, found.Status.ObservedVersion)
}
func (h sspHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*sspv1beta1.SSP).ObjectMeta
}
func (h *sspHooks) reset() {
	h.cache = nil
}

func (h *sspHooks) updateCr(req *common.HcoRequest, client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ssp, ok1 := required.(*sspv1beta1.SSP)
	found, ok2 := exists.(*sspv1beta1.SSP)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to SSP")
	}
	if !reflect.DeepEqual(found.Spec, ssp.Spec) ||
		!reflect.DeepEqual(found.Labels, ssp.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing SSP's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated SSP's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&ssp.ObjectMeta, &found.ObjectMeta)
		ssp.Spec.DeepCopyInto(&found.Spec)
		err := client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewSSP(hc *hcov1beta1.HyperConverged, opts ...string) (*sspv1beta1.SSP, error) {
	replicas := int32(defaultTemplateValidatorReplicas)
	templatesNamespace := defaultCommonTemplatesNamespace

	if hc.Spec.CommonTemplatesNamespace != nil {
		templatesNamespace = *hc.Spec.CommonTemplatesNamespace
	}

	applyDataImportSchedule(hc)

	dataImportCronTemplates, err := getDataImportCronTemplates(hc)
	if err != nil {
		return nil, err
	}

	spec := sspv1beta1.SSPSpec{
		TemplateValidator: sspv1beta1.TemplateValidator{
			Replicas: &replicas,
		},
		CommonTemplates: sspv1beta1.CommonTemplates{
			Namespace:               templatesNamespace,
			DataImportCronTemplates: dataImportCronTemplates,
		},
		// NodeLabeller field is explicitly initialized to its zero-value,
		// in order to future-proof from bugs if SSP changes it to pointer-type,
		// causing nil pointers dereferences at the DeepCopyInto() below.
		NodeLabeller: sspv1beta1.NodeLabeller{},
	}

	if hc.Spec.Infra.NodePlacement != nil {
		spec.TemplateValidator.Placement = hc.Spec.Infra.NodePlacement.DeepCopy()
	}

	if hc.Spec.Workloads.NodePlacement != nil {
		spec.NodeLabeller.Placement = hc.Spec.Workloads.NodePlacement.DeepCopy()
	}

	ssp := NewSSPWithNameOnly(hc)
	ssp.Spec = spec

	return ssp, nil
}

func NewSSPWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *sspv1beta1.SSP {
	return &sspv1beta1.SSP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssp-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentSchedule),
			Namespace: getNamespace(hc.Namespace, opts),
		},
	}
}

var getDataImportCronTemplatesFileLocation = func() string {
	return dataImportCronTemplatesFileLocation
}

func readDataImportCronTemplatesFromFile() error {
	dataImportCronTemplateHardCodedList = make([]sspv1beta1.DataImportCronTemplate, 0)
	dataImportCronTemplateHardCodedNames = make(map[string]struct{})

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

			dataImportCronTemplateFromFile := make([]sspv1beta1.DataImportCronTemplate, 0)
			internalErr = util.UnmarshalYamlFileToObject(file, &dataImportCronTemplateFromFile)
			if internalErr != nil {
				return internalErr
			}

			dataImportCronTemplateHardCodedList = append(dataImportCronTemplateHardCodedList, dataImportCronTemplateFromFile...)

			for _, dict := range dataImportCronTemplateFromFile {
				dataImportCronTemplateHardCodedNames[dict.Name] = struct{}{}
			}
		}

		return nil
	})
}

func getDataImportCronTemplates(hc *hcov1beta1.HyperConverged) ([]sspv1beta1.DataImportCronTemplate, error) {
	if err := validateDataImportCronTemplates(hc); err != nil {
		return nil, err
	}

	var dataImportCronTemplateList []sspv1beta1.DataImportCronTemplate = nil

	if hc.Spec.FeatureGates.EnableCommonBootImageImport {
		dataImportCronTemplateList = append(dataImportCronTemplateList, dataImportCronTemplateHardCodedList...)
	}
	dataImportCronTemplateList = append(dataImportCronTemplateList, hc.Spec.DataImportCronTemplates...)

	return dataImportCronTemplateList, nil
}

func validateDataImportCronTemplates(hc *hcov1beta1.HyperConverged) error {
	dictNames := make(map[string]struct{})
	for _, dict := range hc.Spec.DataImportCronTemplates {
		_, foundCommon := dataImportCronTemplateHardCodedNames[dict.Name]
		_, foundCustom := dictNames[dict.Name]
		if foundCustom || foundCommon {
			return fmt.Errorf("%s DataImportCronTable is already defined", dict.Name)
		}
		dictNames[dict.Name] = struct{}{}
	}
	return nil
}

func applyDataImportSchedule(hc *hcov1beta1.HyperConverged) {
	if hc.Status.DataImportSchedule != "" {
		overrideDataImportSchedule(hc.Status.DataImportSchedule)
	}
}

func overrideDataImportSchedule(schedule string) {
	for i := 0; i < len(dataImportCronTemplateHardCodedList); i++ {
		dataImportCronTemplateHardCodedList[i].Spec.Schedule = schedule
	}
}
