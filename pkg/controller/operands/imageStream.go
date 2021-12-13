package operands

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	log "github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// TODO: this is very similar to the quickstart file; on golang 1.18, check if it possible to use type parameters instead.

// ImageStream resources are a short user guids
const (
	imageStreamDefaultManifestLocation = "./imageStreams"
)

var (
	imageStreamNames           []string
	getImageStreamFileLocation = func() string {
		return imageStreamDefaultManifestLocation
	}
)

func newImageStreamHandler(Client client.Client, Scheme *runtime.Scheme, required *imagev1.ImageStream) Operand {
	h := &genericOperand{
		Client: Client,
		Scheme: Scheme,
		crType: "ImageStream",
		// Previous versions used to have HCO-operator (scope namespace)
		// as the owner of NetworkAddons (scope cluster).
		// It's not legal, so remove that.
		removeExistingOwner: false,
		hooks:               &isHooks{required: required},
	}

	return h
}

type isHooks struct {
	required *imagev1.ImageStream
}

func (h isHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h isHooks) getEmptyCr() client.Object {
	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.required.Name,
			Namespace: h.required.Namespace,
		},
	}
}

func (h isHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*imagev1.ImageStream).ObjectMeta
}

func (h isHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*imagev1.ImageStream)

	if !ok {
		return false, false, errors.New("can't convert to ImageStream")
	}

	if label, ok := found.ObjectMeta.Labels[util.AppLabelManagedBy]; !ok || util.OperatorName != label {
		// not our imageStream. we won't reconcile it.
		return false, false, nil
	}

	if !h.compareAndUpgradeImageStream(found) {
		return false, false, nil
	}

	if req.HCOTriggered {
		req.Logger.Info("Updating existing ImageStream's Spec to new opinionated values", "name", h.required.Name)
	} else {
		req.Logger.Info("Reconciling an externally updated ImageStream's Spec to its opinionated values", "name", h.required.Name)
	}

	err := Client.Update(req.Ctx, found)
	if err != nil {
		return false, false, err
	}
	return true, !req.HCOTriggered, nil
}

func (h *isHooks) compareAndUpgradeImageStream(found *imagev1.ImageStream) bool {
	modified := false
	if !reflect.DeepEqual(h.required.Labels, found.Labels) {
		util.DeepCopyLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		modified = true
	}

	if (len(found.Spec.Tags) != 1) || (found.Spec.Tags[0].Name != "latest") || (found.Spec.Tags[0].From.Name != h.required.Spec.Tags[0].From.Name) {
		found.Spec.Tags = h.required.Spec.Tags
		return true
	}

	return modified
}

func getImageStreamHandlers(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	filesLocation := getImageStreamFileLocation()

	err := util.ValidateManifestDir(filesLocation)
	if err != nil {
		logger.Error(err, "can't get manifest directory for imageStreams", "imageStream files location", filesLocation)
		return nil, errors.Unwrap(err) // if not wrapped, then it's not an error that stops processing, and it return nil
	}

	return createImageStreamHandlersFromFiles(logger, Client, Scheme, hc, filesLocation)
}

func createImageStreamHandlersFromFiles(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged, filesLocation string) ([]Operand, error) {
	var handlers []Operand
	imageStreamNames = []string{}

	logger.Info("walking over the files in " + filesLocation + ", to find imageStream files.")

	err := filepath.Walk(filesLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		logger.Info("processing imageStream file", "fileName", path, "fileInfo", info)
		is, err := processImageStreamFile(path, info, logger, hc, Client, Scheme)
		if err != nil {
			return err
		}

		if is != nil {
			handlers = append(handlers, is)
		}

		return nil
	})

	return handlers, err
}

func processImageStreamFile(path string, info os.FileInfo, logger log.Logger, hc *hcov1beta1.HyperConverged, Client client.Client, Scheme *runtime.Scheme) (Operand, error) {
	if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
		file, err := os.Open(path)
		if err != nil {
			logger.Error(err, "Can't open the ImageStream yaml file", "file name", path)
			return nil, err
		}

		is := &imagev1.ImageStream{}
		err = util.UnmarshalYamlFileToObject(file, is)
		if err != nil {
			return nil, err
		} else if err = validateImageStream(is); err != nil {
			return nil, err
		}
		is.Labels = getLabels(hc, util.AppComponentCompute)
		imageStreamNames = append(imageStreamNames, is.Name)
		return newImageStreamHandler(Client, Scheme, is), nil
	}

	return nil, nil
}

func validateImageStream(is *imagev1.ImageStream) error {
	if len(is.Spec.Tags) != 1 && is.Spec.Tags[0].Name != "latest" {
		return errors.New("wrong imageFile format; missing latest tag or more than many tags")
	}
	return nil
}
