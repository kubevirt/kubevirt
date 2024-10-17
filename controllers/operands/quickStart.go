package operands

import (
	"errors"
	"os"
	filepath "path/filepath"
	"reflect"
	"strings"

	log "github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// ConsoleQuickStart resources are a short user guids
const (
	quickStartManifestLocationVarName = "QUICK_START_FILES_LOCATION"
	quickStartDefaultManifestLocation = "./quickStart"
)

var quickstartNames []string

func newQuickStartHandler(Client client.Client, Scheme *runtime.Scheme, required *consolev1.ConsoleQuickStart) Operand {
	h := &genericOperand{
		Client: Client,
		Scheme: Scheme,
		crType: "ConsoleQuickStart",
		hooks:  &qsHooks{required: required},
	}

	return h
}

type qsHooks struct {
	required *consolev1.ConsoleQuickStart
}

func (h qsHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h qsHooks) getEmptyCr() client.Object {
	return &consolev1.ConsoleQuickStart{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}

func (h qsHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*consolev1.ConsoleQuickStart)

	if !ok {
		return false, false, errors.New("can't convert to ConsoleQuickStart")
	}

	if !reflect.DeepEqual(h.required.Spec, found.Spec) ||
		!util.CompareLabels(h.required, found) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsoleQuickStart's Spec to new opinionated values", "name", h.required.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated ConsoleQuickStart's Spec to its opinionated values", "name", h.required.Name)
		}
		util.MergeLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		h.required.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func (qsHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func getQuickStartHandlers(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	filesLocation := util.GetManifestDirPath(quickStartManifestLocationVarName, quickStartDefaultManifestLocation)

	err := util.ValidateManifestDir(filesLocation)
	if err != nil {
		return nil, errors.Unwrap(err) // if not wrapped, then it's not an error that stops processing, and it return nil
	}

	return createQuickstartHandlersFromFiles(logger, Client, Scheme, hc, filesLocation)
}

func createQuickstartHandlersFromFiles(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged, filesLocation string) ([]Operand, error) {
	var handlers []Operand
	quickstartNames = []string{}

	err := filepath.Walk(filesLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		qs, err := processQuickstartFile(path, info, logger, hc, Client, Scheme)
		if err != nil {
			return err
		}

		if qs != nil {
			handlers = append(handlers, qs)
		}

		return nil
	})

	return handlers, err
}

func processQuickstartFile(path string, info os.FileInfo, logger log.Logger, hc *hcov1beta1.HyperConverged, Client client.Client, Scheme *runtime.Scheme) (Operand, error) {
	if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
		file, err := os.Open(path)
		if err != nil {
			logger.Error(err, "Can't open the quickStart yaml file", "file name", path)
			return nil, err
		}

		qs := &consolev1.ConsoleQuickStart{}
		err = util.UnmarshalYamlFileToObject(file, qs)
		if err != nil {
			logger.Error(err, "Can't generate a ConsoleQuickStart object from yaml file", "file name", path)
		} else {
			qs.Labels = getLabels(hc, util.AppComponentCompute)
			quickstartNames = append(quickstartNames, qs.Name)
			return newQuickStartHandler(Client, Scheme, qs), nil
		}
	}
	return nil, nil
}
