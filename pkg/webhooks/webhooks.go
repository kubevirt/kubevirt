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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

const (
	updateDryRunTimeOut = time.Second * 3
)

type WebhookHandler struct {
	logger    logr.Logger
	cli       client.Client
	namespace string
}

func (wh *WebhookHandler) Init(logger logr.Logger, cli client.Client, namespace string) {
	wh.logger = logger
	wh.cli = cli
	wh.namespace = namespace
}

func (wh WebhookHandler) ValidateCreate(hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating create", "name", hc.Name, "namespace:", hc.Namespace)

	if hc.Namespace != wh.namespace {
		return fmt.Errorf("invalid namespace for v1beta1.HyperConverged - please use the %s namespace", wh.namespace)
	}

	return nil
}

// ValidateUpdate is the ValidateUpdate webhook implementation. It calls all the resources in parallel, to dry-run the
// upgrade.
func (wh WebhookHandler) ValidateUpdate(requested *v1beta1.HyperConverged, exists *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating update", "name", requested.Name)
	ctx, cancel := context.WithTimeout(context.Background(), updateDryRunTimeOut)
	defer cancel()

	if !reflect.DeepEqual(
		exists.Spec,
		requested.Spec) {

		wg := sync.WaitGroup{}
		errorCh := make(chan error)
		done := make(chan bool)

		opts := &client.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
		resources := []runtime.Object{
			operands.NewKubeVirt(requested),
			operands.NewCDI(requested),
			operands.NewNetworkAddons(requested),
			operands.NewKubeVirtCommonTemplateBundle(requested),
			operands.NewKubeVirtNodeLabellerBundleForCR(requested, requested.Namespace),
			operands.NewKubeVirtTemplateValidatorForCR(requested, requested.Namespace),
			operands.NewKubeVirtMetricsAggregationForCR(requested, requested.Namespace),
			operands.NewVMImportForCR(requested),
		}

		wg.Add(len(resources))

		go func() {
			wg.Wait()
			close(done)
		}()

		for _, obj := range resources {
			go func(o runtime.Object, wgr *sync.WaitGroup) {
				defer wgr.Done()
				if err := wh.updateOperatorCr(ctx, requested, o, opts); err != nil {
					errorCh <- err
				}
			}(obj, &wg)
		}

		select {
		case err := <-errorCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			// just in case close(done) was selected while there is an error,
			// check the error channel again.
			if len(errorCh) != 0 {
				err := <-errorCh
				return err
			}
			return nil
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

	switch existing := exists.(type) {
	case *kubevirtv1.KubeVirt:
		required := operands.NewKubeVirt(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *cdiv1beta1.CDI:
		required := operands.NewCDI(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *networkaddonsv1.NetworkAddonsConfig:
		required := operands.NewNetworkAddons(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtCommonTemplatesBundle:
		required := operands.NewKubeVirtCommonTemplateBundle(hc)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtNodeLabellerBundle:
		required := operands.NewKubeVirtNodeLabellerBundleForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtTemplateValidator:
		required := operands.NewKubeVirtTemplateValidatorForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1.KubevirtMetricsAggregation:
		required := operands.NewKubeVirtMetricsAggregationForCR(hc, hc.Namespace)
		required.Spec.DeepCopyInto(&existing.Spec)

	case *vmimportv1beta1.VMImportConfig:
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

func (wh WebhookHandler) HandleMutatingNsDelete(ns *corev1.Namespace, dryRun bool) (bool, error) {
	wh.logger.Info("validating namespace deletion", "name", ns.Name)

	if ns.Name != wh.namespace {
		wh.logger.Info("ignoring request for a different namespace")
		return true, nil
	}

	ctx := context.TODO()
	hco := &v1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoutil.HyperConvergedName,
			Namespace: wh.namespace,
		},
	}

	// TODO: once the deletion of HCO CR is really safe during namespace deletion
	// (foreground deletion, context timeouts...) try to automatically
	// delete HCO CR if there.
	// For now let's simply block the deletion if the namespace with a clear error message
	// if HCO CR is still there

	key, err := client.ObjectKeyFromObject(hco)
	if err != nil {
		wh.logger.Error(err, "failed to get object key for HyperConverged CR")
		return false, err
	}
	found := &v1beta1.HyperConverged{}
	err = wh.cli.Get(ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			wh.logger.Info("HCO CR doesn't not exist, allow namespace deletion")
			return true, nil
		}
		wh.logger.Error(err, "failed getting HyperConverged CR")
		return false, err
	}
	wh.logger.Info("HCO CR still exists, forbid namespace deletion")
	return false, nil
}
