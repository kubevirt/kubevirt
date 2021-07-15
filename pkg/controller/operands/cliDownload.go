package operands

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type cliDownloadHandler genericOperand

func newCLIDownloadHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadHandler {
	return &cliDownloadHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "ConsoleCLIDownload",
		removeExistingOwner:    false,
		setControllerReference: false,
		hooks:                  &cliDownloadHooks{},
	}
}

type cliDownloadHooks struct{}

func (h cliDownloadHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewConsoleCLIDownload(hc), nil
}

func (h cliDownloadHooks) getEmptyCr() client.Object {
	return &consolev1.ConsoleCLIDownload{}
}

func (h cliDownloadHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*consolev1.ConsoleCLIDownload).ObjectMeta
}

func (h *cliDownloadHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ccd, ok1 := required.(*consolev1.ConsoleCLIDownload)
	found, ok2 := exists.(*consolev1.ConsoleCLIDownload)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ConsoleCLIDownload")
	}
	if !reflect.DeepEqual(found.Spec, ccd.Spec) ||
		!reflect.DeepEqual(found.Labels, ccd.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsoleCLIDownload's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated ConsoleCLIDownload's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&ccd.ObjectMeta, &found.ObjectMeta)
		ccd.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
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
			Labels: getLabels(hc, hcoutil.AppComponentCompute),
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
