package webhooks

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WebhookHandler struct {
	logger logr.Logger
	cli    client.Client
}

func (wh *WebhookHandler) Init(logger logr.Logger, cli client.Client) {
	wh.logger = logger
	wh.cli = cli
}

func (wh WebhookHandler) ValidateCreate(hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating create", "name", hc.Name, "namespace:", hc.Namespace)

	operatorNsEnv, err := hcoutil.GetOperatorNamespaceFromEnv()
	if err != nil {
		wh.logger.Error(err, "Failed to get operator namespace from the environment")
		return err
	}

	if hc.Namespace != operatorNsEnv {
		return fmt.Errorf("Invalid namespace for v1beta1.HyperConverged - please use the %s namespace", operatorNsEnv)
	}

	return nil
}

func (wh WebhookHandler) ValidateUpdate(requested *v1beta1.HyperConverged, exists *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating update", "name", requested.Name)
	ctx := context.TODO()

	if !reflect.DeepEqual(
		exists.Spec,
		requested.Spec) {

		opts := &client.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
		for _, obj := range []runtime.Object{
			operands.NewKubeVirt(requested),
			operands.NewCDI(requested),
			operands.NewNetworkAddons(requested),
			operands.NewKubeVirtCommonTemplateBundle(requested),
			operands.NewKubeVirtNodeLabellerBundleForCR(requested, requested.Namespace),
			operands.NewKubeVirtTemplateValidatorForCR(requested, requested.Namespace),
			operands.NewKubeVirtMetricsAggregationForCR(requested, requested.Namespace),
			operands.NewVMImportForCR(requested),
		} {
			if err := wh.updateOperatorCr(ctx, requested, obj, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// currently only supports KV and CDI
func (wh WebhookHandler) updateOperatorCr(ctx context.Context, hc *v1beta1.HyperConverged, exists runtime.Object, opts *client.UpdateOptions) error {
	err := hcoutil.GetRuntimeObject(ctx, wh.cli, exists, wh.logger)
	if err != nil {
		wh.logger.Error(err, "failed to get object from kubernetes", "kind", exists.GetObjectKind())
		return err
	}

	switch obj := exists.(type) {
	case *kubevirtv1.KubeVirt:
		existing := obj
		required := operands.NewKubeVirt(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *cdiv1beta1.CDI:
		existing := obj
		required := operands.NewCDI(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *networkaddonsv1.NetworkAddonsConfig:
		existing := obj
		required := operands.NewNetworkAddons(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtCommonTemplatesBundle:
		existing := obj
		required := operands.NewKubeVirtCommonTemplateBundle(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtNodeLabellerBundle:
		existing := obj
		required := operands.NewKubeVirtNodeLabellerBundleForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtTemplateValidator:
		existing := obj
		required := operands.NewKubeVirtTemplateValidatorForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtMetricsAggregation:
		existing := obj
		required := operands.NewKubeVirtMetricsAggregationForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *vmimportv1beta1.VMImportConfig:
		existing := obj
		required := operands.NewVMImportForCR(hc)
		required.Spec.DeepCopyInto(&existing.Spec)
	}

	if err = wh.cli.Update(ctx, exists, opts); err != nil {
		wh.logger.Error(err, "failed to dry-run update the object", "kind", exists.GetObjectKind())
		return err
	}

	wh.logger.Info("dry-run update the object passed", "kind", exists.GetObjectKind())
	return nil
}

func (wh WebhookHandler) ValidateDelete(hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating delete", "name", hc.Name, "namespace", hc.Namespace)

	ctx := context.TODO()

	for _, obj := range []runtime.Object{
		operands.NewKubeVirt(hc),
		operands.NewCDI(hc),
	} {
		err := hcoutil.EnsureDeleted(ctx, wh.cli, obj, hc.Name, wh.logger, true)
		if err != nil {
			wh.logger.Error(err, "Delete validation failed", "GVK", obj.GetObjectKind().GroupVersionKind())
			return err
		}
	}

	return nil
}
