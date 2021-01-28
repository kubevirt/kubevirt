package operands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	filepath "path/filepath"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	log "github.com/go-logr/logr"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	consolev1 "github.com/openshift/api/console/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConsoleQuickStart resources are a short user guids
const (
	consoleQuickStartCrdName     = "consolequickstarts.console.openshift.io"
	customResourceDefinitionName = "CustomResourceDefinition"
	manifestLocationVarName      = "QUICK_START_FILES_LOCATION"
	defaultManifestLocation      = "./quickStart"
)

func newQuickStartHandler(Client client.Client, Scheme *runtime.Scheme, required *consolev1.ConsoleQuickStart) Operand {
	h := &genericOperand{
		Client: Client,
		Scheme: Scheme,
		crType: "ConsoleQuickStart",
		isCr:   false,
		// Previous versions used to have HCO-operator (scope namespace)
		// as the owner of NetworkAddons (scope cluster).
		// It's not legal, so remove that.
		removeExistingOwner: false,
		hooks:               &qsHooks{required: required},
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

func (h qsHooks) validate() error                                        { return nil }
func (h qsHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h qsHooks) getConditions(_ runtime.Object) []conditionsv1.Condition {
	return nil
}

func (h qsHooks) checkComponentVersion(_ runtime.Object) bool {
	return true
}

func (h qsHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*consolev1.ConsoleQuickStart).ObjectMeta
}

func (h qsHooks) reset() { /* no implementation */ }

func (h qsHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*consolev1.ConsoleQuickStart)

	if !ok {
		return false, false, errors.New("can't convert to ConsoleQuickStart")
	}

	if !reflect.DeepEqual(found.Spec, h.required.Spec) ||
		!reflect.DeepEqual(found.Labels, h.required.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsoleQuickStart's Spec to new opinionated values", "name", h.required.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated ConsoleQuickStart's Spec to its opinionated values", "name", h.required.Name)
		}
		util.DeepCopyLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		h.required.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

// This function returns 3-state error:
//   err := checkCrdExists(...)
//   err == nil - OK, CRD exists
//   err != nil && errors.Unwrap(err) == nil - CRD does not exist, but that ok
//   err != nil && errors.Unwrap(err) != nil - actual error
func checkCrdExists(ctx context.Context, Client client.Client, logger log.Logger) error {
	qsCrd := &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind: customResourceDefinitionName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: consoleQuickStartCrdName,
		},
	}

	logger.Info("Read the ConsoleQuickStart CRD")
	if err := Client.Get(ctx, client.ObjectKeyFromObject(qsCrd), qsCrd); err != nil {
		if apierrors.IsNotFound(err) {
			return newProcessingError(nil)
		}
		return newProcessingError(err)
	}

	return nil
}

func getQuickStartHandlers(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	err := checkCrdExists(context.TODO(), Client, logger)
	if err != nil {
		return nil, errors.Unwrap(err)
	}

	filesLocation := getQuickstartDirPath()

	err = validateQuickstartDir(filesLocation)
	if err != nil {
		return nil, errors.Unwrap(err) // if not wrapped, then it's not an error that stops processing, and it return nil
	}

	return createQuickstartHandlersFromFiles(logger, Client, Scheme, hc, filesLocation)
}

func getQuickstartDirPath() string {
	filesLocation := os.Getenv(manifestLocationVarName)
	if filesLocation == "" {
		return defaultManifestLocation
	}

	return filesLocation
}

// This function returns 3-state error:
//   err := validateQuickstartDir(...)
//   err == nil - OK: quickstart directory exists
//   err != nil && errors.Unwrap(err) == nil - quickstart directory does not exist, but that ok
//   err != nil && errors.Unwrap(err) != nil - actual error
func validateQuickstartDir(filesLocation string) error {
	info, err := os.Stat(filesLocation)
	if err != nil {
		if os.IsNotExist(err) { // don't return error if there is no quickstart dir, just ignore it
			return newProcessingError(nil) // return error, but don't stop processing
		}
		return newProcessingError(err)
	}

	if !info.IsDir() {
		err := fmt.Errorf("%s is not a directory", filesLocation)
		return newProcessingError(err) // return error
	}
	return nil
}

func createQuickstartHandlersFromFiles(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged, filesLocation string) ([]Operand, error) {
	var handlers []Operand
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

		qs, err := yamlToQuickStart(file)
		if err != nil {
			logger.Error(err, "Can't generate a ConsoleQuickStart object from yaml file", "file name", path)
		} else {
			qs.Labels = getLabels(hc, util.AppComponentCompute)
			return newQuickStartHandler(Client, Scheme, qs), nil
		}
	}
	return nil, nil
}

func yamlToQuickStart(file io.Reader) (*consolev1.ConsoleQuickStart, error) {
	qs := &consolev1.ConsoleQuickStart{}

	yamlBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(yamlBytes, qs); err != nil {
		return nil, err
	}

	return qs, nil
}

// wrap an error if we want to actually stop processing by the error.
// unwrapping it will return the original error.
// unwrapping a regular error will return nil
func newProcessingError(err error) error {
	return fmt.Errorf("%w", err)
}
