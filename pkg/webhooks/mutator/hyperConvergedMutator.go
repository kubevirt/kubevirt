package mutator

import (
	"context"
	"fmt"
	"net/http"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

var (
	hcMutatorLogger = logf.Log.WithName("hyperConverged mutator")

	_ admission.Handler = &NsMutator{}
)

// HyperConvergedMutator mutates HyperConverged requests
type HyperConvergedMutator struct {
	decoder *admission.Decoder
	cli     client.Client
}

func NewHyperConvergedMutator(cli client.Client, decoder *admission.Decoder) *HyperConvergedMutator {
	return &HyperConvergedMutator{
		cli:     cli,
		decoder: decoder,
	}
}

func (hcm *HyperConvergedMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	hcMutatorLogger.Info("reaching HyperConvergedMutator.Handle")

	if req.Operation == admissionv1.Update || req.Operation == admissionv1.Create {
		return hcm.mutateHyperConverged(ctx, req)
	}

	// ignoring other operations
	return admission.Allowed(ignoreOperationMessage)

}

const (
	annotationPathTemplate     = "/spec/dataImportCronTemplates/%d/metadata/annotations"
	dictAnnotationPathTemplate = annotationPathTemplate + "/cdi.kubevirt.io~1storage.bind.immediate.requested"
)

func (hcm *HyperConvergedMutator) mutateHyperConverged(_ context.Context, req admission.Request) admission.Response {
	hc := &hcov1beta1.HyperConverged{}
	err := hcm.decoder.Decode(req, hc)
	if err != nil {
		hcMutatorLogger.Error(err, "failed to read the HyperConverged custom resource")
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("failed to parse the HyperConverged"))
	}

	var patches []jsonpatch.JsonPatchOperation
	for index, dict := range hc.Spec.DataImportCronTemplates {
		if dict.Annotations == nil {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(annotationPathTemplate, index),
				Value:     map[string]string{operands.CDIImmediateBindAnnotation: "true"},
			})
		} else if _, annotationFound := dict.Annotations[operands.CDIImmediateBindAnnotation]; !annotationFound {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf(dictAnnotationPathTemplate, index),
				Value:     "true",
			})
		}
	}

	if hc.Spec.FeatureGates.Root == nil {
		value := false
		//nolint SA1019
		if hc.Spec.FeatureGates.NonRoot != nil {
			value = !*hc.Spec.FeatureGates.NonRoot
		}
		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/featureGates/root",
			Value:     value,
		})
	}

	if hc.Spec.EvictionStrategy == nil {
		ci := hcoutil.GetClusterInfo()
		if ci.IsInfrastructureHighlyAvailable() {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/evictionStrategy",
				Value:     kubevirtcorev1.EvictionStrategyLiveMigrate,
			})
		} else {
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/evictionStrategy",
				Value:     kubevirtcorev1.EvictionStrategyNone,
			})
		}

	}

	if len(patches) > 0 {
		return admission.Patched("mutated", patches...)
	}

	return admission.Allowed("")
}
