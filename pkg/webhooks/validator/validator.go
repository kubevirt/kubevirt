package validator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	updateDryRunTimeOut = time.Second * 3
)

var _ v1beta1.ValidatorWebhookHandler = &WebhookHandler{}

type WebhookHandler struct {
	logger      logr.Logger
	cli         client.Client
	namespace   string
	isOpenshift bool
}

func NewWebhookHandler(logger logr.Logger, cli client.Client, namespace string, isOpenshift bool) *WebhookHandler {
	return &WebhookHandler{
		logger:      logger,
		cli:         cli,
		namespace:   namespace,
		isOpenshift: isOpenshift,
	}
}

func (wh WebhookHandler) ValidateCreate(hc *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating create", "name", hc.Name, "namespace:", hc.Namespace)

	if err := wh.validateCertConfig(hc); err != nil {
		return err
	}

	if hc.Namespace != wh.namespace {
		return fmt.Errorf("invalid namespace for v1beta1.HyperConverged - please use the %s namespace", wh.namespace)
	}

	if _, err := operands.NewKubeVirt(hc); err != nil {
		return err
	}

	if _, err := operands.NewCDI(hc); err != nil {
		return err
	}

	if _, err := operands.NewNetworkAddons(hc); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate is the ValidateUpdate webhook implementation. It calls all the resources in parallel, to dry-run the
// upgrade.
func (wh WebhookHandler) ValidateUpdate(requested *v1beta1.HyperConverged, exists *v1beta1.HyperConverged) error {
	wh.logger.Info("Validating update", "name", requested.Name)
	ctx, cancel := context.WithTimeout(context.Background(), updateDryRunTimeOut)
	defer cancel()

	// If no change is detected in the spec nor the annotations - nothing to validate
	if reflect.DeepEqual(exists.Spec, requested.Spec) &&
		reflect.DeepEqual(exists.Annotations, requested.Annotations) {
		return nil
	}

	if err := wh.validateCertConfig(requested); err != nil {
		return err
	}

	kv, err := operands.NewKubeVirt(requested)
	if err != nil {
		return err
	}

	cdi, err := operands.NewCDI(requested)
	if err != nil {
		return err
	}

	cna, err := operands.NewNetworkAddons(requested)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errorCh := make(chan error)
	done := make(chan bool)

	opts := &client.UpdateOptions{DryRun: []string{metav1.DryRunAll}}

	resources := []client.Object{
		kv,
		cdi,
		cna,
		operands.NewVMImportForCR(requested),
	}

	if wh.isOpenshift {
		resources = append(resources,
			operands.NewSSP(requested),
		)
	}

	wg.Add(len(resources))

	go func() {
		wg.Wait()
		close(done)
	}()

	for _, obj := range resources {
		go func(o client.Object, wgr *sync.WaitGroup) {
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

func (wh WebhookHandler) updateOperatorCr(ctx context.Context, hc *v1beta1.HyperConverged, exists client.Object, opts *client.UpdateOptions) error {
	err := hcoutil.GetRuntimeObject(ctx, wh.cli, exists, wh.logger)
	if err != nil {
		wh.logger.Error(err, "failed to get object from kubernetes", "kind", exists.GetObjectKind())
		return err
	}

	switch existing := exists.(type) {
	case *kubevirtv1.KubeVirt:
		required, err := operands.NewKubeVirt(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *cdiv1beta1.CDI:
		required, err := operands.NewCDI(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *networkaddonsv1.NetworkAddonsConfig:
		required, err := operands.NewNetworkAddons(hc)
		if err != nil {
			return err
		}
		required.Spec.DeepCopyInto(&existing.Spec)

	case *sspv1beta1.SSP:
		required := operands.NewSSP(hc)
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

	kv := operands.NewKubeVirtWithNameOnly(hc)
	cdi := operands.NewCDIWithNameOnly(hc)

	for _, obj := range []client.Object{
		kv,
		cdi,
	} {
		err := hcoutil.EnsureDeleted(ctx, wh.cli, obj, hc.Name, wh.logger, true, false)
		if err != nil {
			wh.logger.Error(err, "Delete validation failed", "GVK", obj.GetObjectKind().GroupVersionKind())
			return err
		}
	}

	return nil
}

func (wh WebhookHandler) validateCertConfig(hc *v1beta1.HyperConverged) error {
	minimalDuration := metav1.Duration{Duration: 10 * time.Minute}

	ccValues := make(map[string]time.Duration)
	ccValues["spec.certConfig.ca.duration"] = hc.Spec.CertConfig.CA.Duration.Duration
	ccValues["spec.certConfig.ca.renewBefore"] = hc.Spec.CertConfig.CA.RenewBefore.Duration
	ccValues["spec.certConfig.server.duration"] = hc.Spec.CertConfig.Server.Duration.Duration
	ccValues["spec.certConfig.server.renewBefore"] = hc.Spec.CertConfig.Server.RenewBefore.Duration

	for key, value := range ccValues {
		if value < minimalDuration.Duration {
			return fmt.Errorf("%v: value is too small", key)
		}
	}

	if hc.Spec.CertConfig.CA.Duration.Duration < hc.Spec.CertConfig.CA.RenewBefore.Duration {
		return errors.New("spec.certConfig.ca: duration is smaller than renewBefore")
	}

	if hc.Spec.CertConfig.Server.Duration.Duration < hc.Spec.CertConfig.Server.RenewBefore.Duration {
		return errors.New("spec.certConfig.server: duration is smaller than renewBefore")
	}

	if hc.Spec.CertConfig.CA.Duration.Duration < hc.Spec.CertConfig.Server.Duration.Duration {
		return errors.New("spec.certConfig: ca.duration is smaller than server.duration")
	}

	return nil
}
