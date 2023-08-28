package operands

import (
	"fmt"
	"strconv"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const disableOperandDeletionPatch = `[{"op": "replace", "path": "/metadata/annotations/console.openshift.io~1disable-operand-delete", "value": "%t"}]`

var _ Operand = &csvHandler{}

type csvHandler struct {
	client client.Client
	csvKey client.ObjectKey
}

func newCsvHandler(cli client.Client, ci hcoutil.ClusterInfo) Operand {
	return &csvHandler{
		client: cli,
		csvKey: client.ObjectKeyFromObject(ci.GetCSV()),
	}
}

func (c csvHandler) ensure(req *common.HcoRequest) *EnsureResult {
	csv, err := c.getCsv(req)
	er := NewEnsureResult(csv)
	if err != nil {
		return er.Error(err)
	}

	foundDisableOperandDeletion := csv.Annotations[components.DisableOperandDeletionAnnotation]
	requiredDisableOperandDeletion := req.Instance.Spec.UninstallStrategy == hcov1beta1.HyperConvergedUninstallStrategyBlockUninstallIfWorkloadsExist

	if foundDisableOperandDeletion != strconv.FormatBool(requiredDisableOperandDeletion) {
		updateErr := c.updateCsv(req, csv, requiredDisableOperandDeletion)
		if updateErr != nil {
			return er.Error(updateErr)
		}
		return er.SetUpdated().SetUpgradeDone(true)
	}

	return er.SetUpgradeDone(true)
}

func (c csvHandler) getCsv(req *common.HcoRequest) (*csvv1alpha1.ClusterServiceVersion, error) {
	csv := &csvv1alpha1.ClusterServiceVersion{}
	err := c.client.Get(req.Ctx, c.csvKey, csv)
	if err != nil {
		req.Logger.Error(err, fmt.Sprintf("Could not find resource - APIVersion: %s, Kind: %s, Name: %s",
			csv.APIVersion, csv.Kind, csv.Name))
		return nil, err
	}
	return csv, nil
}

func (c csvHandler) updateCsv(req *common.HcoRequest, csv *csvv1alpha1.ClusterServiceVersion, disableOperandDeletion bool) error {
	if req.HCOTriggered {
		req.Logger.Info("Updating existing CSV disable-operand-delete annotation to new opinionated values")
	} else {
		req.Logger.Info("Reconciling an externally updated CSV disable-operand-delete annotation to its opinionated values")
	}

	patch := fmt.Sprintf(disableOperandDeletionPatch, disableOperandDeletion)
	err := c.client.Patch(req.Ctx, csv, client.RawPatch(types.JSONPatchType, []byte(patch)))
	if err != nil {
		req.Logger.Error(err, "Failed to update CSV disable-operand-delete annotation")
		return err
	}

	req.Logger.Info("Updated CSV disable-operand-delete annotation")
	return nil
}

func (c csvHandler) reset() { /* no implementation */ }
