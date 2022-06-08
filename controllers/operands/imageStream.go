package operands

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	"k8s.io/client-go/tools/reference"

	log "github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
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

type imageStreamOperand struct {
	operand *genericOperand
}

func (iso imageStreamOperand) ensure(req *common.HcoRequest) *EnsureResult {
	if req.Instance.Spec.FeatureGates.EnableCommonBootImageImport {
		// if the FG is set, make sure the imageStream is in place and up-to-date
		return iso.operand.ensure(req)
	}

	// if the FG is not set, make sure the imageStream is not exist
	cr := iso.operand.hooks.getEmptyCr()
	res := NewEnsureResult(cr)
	res.SetName(cr.GetName())
	deleted, err := util.EnsureDeleted(req.Ctx, iso.operand.Client, cr, req.Instance.Name, req.Logger, false, false, true)
	if err != nil {
		return res.Error(err)
	}

	if deleted {
		res.SetDeleted()
		objectRef, err := reference.GetReference(iso.operand.Scheme, cr)
		if err != nil {
			return res.Error(err)
		}

		if err = objectreferencesv1.RemoveObjectReference(&req.Instance.Status.RelatedObjects, *objectRef); err != nil {
			return res.Error(err)
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

func (iso imageStreamOperand) reset() {
	iso.operand.reset()
}

func newImageStreamHandler(Client client.Client, Scheme *runtime.Scheme, required *imagev1.ImageStream) Operand {
	return &imageStreamOperand{
		operand: &genericOperand{
			Client: Client,
			Scheme: Scheme,
			crType: "ImageStream",
			hooks:  newIsHook(required),
		},
	}
}

type isHooks struct {
	required *imagev1.ImageStream
	tags     map[string]imagev1.TagReference
}

func newIsHook(required *imagev1.ImageStream) *isHooks {
	tags := make(map[string]imagev1.TagReference)
	for _, tag := range required.Spec.Tags {
		tags[tag.Name] = tag
	}
	return &isHooks{required: required, tags: tags}
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

func (h isHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (h *isHooks) compareAndUpgradeImageStream(found *imagev1.ImageStream) bool {
	modified := false
	if !reflect.DeepEqual(h.required.Labels, found.Labels) {
		util.DeepCopyLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		modified = true
	}

	newTags := make([]imagev1.TagReference, 0)

	for _, foundTag := range found.Spec.Tags {
		reqTag, ok := h.tags[foundTag.Name]
		if !ok {
			modified = true
			continue
		}

		if h.compareOneTag(&foundTag, &reqTag) {
			modified = true
		}

		newTags = append(newTags, foundTag)
	}

	// find and add missing tags
	newTags, modified = h.addMissingTags(found, newTags, modified)

	if modified {
		found.Spec.Tags = newTags
	}

	return modified
}

func (h *isHooks) addMissingTags(found *imagev1.ImageStream, newTags []imagev1.TagReference, modified bool) ([]imagev1.TagReference, bool) {
	for reqTagName, reqTag := range h.tags {
		tagExist := false
		for _, foundTag := range found.Spec.Tags {
			if reqTagName == foundTag.Name {
				tagExist = true
			}
		}

		if !tagExist {
			newTags = append(newTags, reqTag)
			modified = true
		}
	}
	return newTags, modified
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

func (h *isHooks) compareOneTag(foundTag, reqTag *imagev1.TagReference) bool {
	modified := false
	if reqTag.From.Name != foundTag.From.Name || reqTag.From.Kind != foundTag.From.Kind {
		foundTag.From = reqTag.From.DeepCopy()
		modified = true
	}

	if !reflect.DeepEqual(reqTag.ImportPolicy, foundTag.ImportPolicy) {
		foundTag.ImportPolicy = *reqTag.ImportPolicy.DeepCopy()
		modified = true
	}

	return modified
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
		}

		is.Labels = getLabels(hc, util.AppComponentCompute)
		imageStreamNames = append(imageStreamNames, is.Name)
		return newImageStreamHandler(Client, Scheme, is), nil
	}

	return nil, nil
}
