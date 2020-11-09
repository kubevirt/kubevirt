package operands

import (
	"fmt"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	consolev1 "github.com/openshift/api/console/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	"os"
	"reflect"
)

type CLIDownloadHandler genericOperand

func (h CLIDownloadHandler) Ensure(req *common.HcoRequest) error {
	ccd := NewConsoleCLIDownload(req.Instance)

	found := NewConsoleCLIDownload(req.Instance)
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

func NewConsoleCLIDownload(hc *hcov1beta1.HyperConverged) *consolev1.ConsoleCLIDownload {
	kv := os.Getenv(hcoutil.KubevirtVersionEnvV)
	url := fmt.Sprintf("https://github.com/kubevirt/kubevirt/releases/%s", kv)
	text := fmt.Sprintf("KubeVirt %s release downloads", kv)

	if val, ok := os.LookupEnv("VIRTCTL_DOWNLOAD_URL"); ok && val != "" {
		url = val
	}

	if val, ok := os.LookupEnv("VIRTCTL_DOWNLOAD_TEXT"); ok && val != "" {
		text = val
	}

	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "virtctl-clidownloads-" + hc.Name,
			Labels: getLabels(hc),
		},

		Spec: consolev1.ConsoleCLIDownloadSpec{
			Description: "The virtctl client is a supplemental command-line utility for managing virtualization resources from the command line.",
			DisplayName: "virtctl - KubeVirt command line interface",
			Links: []consolev1.CLIDownloadLink{
				{
					Href: url,
					Text: text,
				},
			},
		},
	}
}
