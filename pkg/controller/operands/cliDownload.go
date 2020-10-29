package operands

import (
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/reference"
	"reflect"
)

type CLIDownloadHandler genericOperand

func (h CLIDownloadHandler) Ensure(req *common.HcoRequest) error {
	ccd := req.Instance.NewConsoleCLIDownload()

	found := req.Instance.NewConsoleCLIDownload()
	err := hcoutil.EnsureCreated(req.Ctx, h.Client, found, req.Logger)
	if err != nil {
		if meta.IsNoMatchError(err) {
			req.Logger.Info("ConsoleCLIDownload was not found, skipping")
		}
		return err
	}

	// Make sure we hold the right link spec
	if reflect.DeepEqual(found.Spec, ccd.Spec) {
		objectRef, err := reference.GetReference(h.Scheme, found)
		if err != nil {
			req.Logger.Error(err, "failed getting object reference for ConsoleCLIDownload")
			return err
		}
		objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)
		return nil
	}

	ccd.Spec.DeepCopyInto(&found.Spec)

	err = h.Client.Update(req.Ctx, found)
	if err != nil {
		return err
	}

	return nil
}
