package operands

import (
	"context"
	"errors"
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

func (h qsHooks) reset() {}

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

func checkCrdExists(ctx context.Context, Client client.Client, logger log.Logger) (bool, error) {
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
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func getQuickStartHandlers(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	crdExists, err := checkCrdExists(context.TODO(), Client, logger)
	if err != nil {
		return nil, err
	} else if !crdExists {
		return nil, nil
	}

	filesLocation := os.Getenv(manifestLocationVarName)
	if filesLocation == "" {
		filesLocation = defaultManifestLocation
	}

	info, err := os.Stat(filesLocation)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, nil
	}

	var handlers []Operand
	err = filepath.Walk(filesLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			qs, err := yamlToQuickStart(file)
			if err != nil {
				logger.Info("Can't generate a ConsoleQuickStart object from yaml file", "file name", path)
			} else {
				qs.Labels = getLabels(hc, util.AppComponentCompute)
				handlers = append(handlers, newQuickStartHandler(Client, Scheme, qs))
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return handlers, nil
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
