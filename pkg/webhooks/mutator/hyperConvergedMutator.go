package mutator

import (
	"context"
	"fmt"
	"net/http"

	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
)

var (
	hcMutatorLogger = logf.Log.WithName("hyperConverged mutator")

	_ admission.Handler = &NsMutator{}
)

// HyperConvergedMutator mutates HyperConverged requests
type HyperConvergedMutator struct {
	decoder admission.Decoder
	cli     client.Client
}

func NewHyperConvergedMutator(cli client.Client, decoder admission.Decoder) *HyperConvergedMutator {
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

	patches = mutateEvictionStrategy(hc, patches)

	if hc.Spec.MediatedDevicesConfiguration != nil {
		if len(hc.Spec.MediatedDevicesConfiguration.MediatedDevicesTypes) > 0 && len(hc.Spec.MediatedDevicesConfiguration.MediatedDeviceTypes) == 0 { //nolint SA1019
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/mediatedDevicesConfiguration/mediatedDeviceTypes",
				Value:     hc.Spec.MediatedDevicesConfiguration.MediatedDevicesTypes, //nolint SA1019
			})
		}
		for i, hcoNodeMdevTypeConf := range hc.Spec.MediatedDevicesConfiguration.NodeMediatedDeviceTypes {
			if len(hcoNodeMdevTypeConf.MediatedDevicesTypes) > 0 && len(hcoNodeMdevTypeConf.MediatedDeviceTypes) == 0 { //nolint SA1019
				patches = append(patches, jsonpatch.JsonPatchOperation{
					Operation: "add",
					Path:      fmt.Sprintf("/spec/mediatedDevicesConfiguration/nodeMediatedDeviceTypes/%d/mediatedDeviceTypes", i),
					Value:     hcoNodeMdevTypeConf.MediatedDevicesTypes, //nolint SA1019
				})
			}
		}
	}

	if len(patches) > 0 {
		return admission.Patched("mutated", patches...)
	}

	return admission.Allowed("")
}

func mutateEvictionStrategy(hc *hcov1beta1.HyperConverged, patches []jsonpatch.JsonPatchOperation) []jsonpatch.JsonPatchOperation {
	if hc.Status.InfrastructureHighlyAvailable == nil || hc.Spec.EvictionStrategy != nil { // New HyperConverged CR
		return patches
	}

	var value = kubevirtcorev1.EvictionStrategyNone
	if *hc.Status.InfrastructureHighlyAvailable {
		value = kubevirtcorev1.EvictionStrategyLiveMigrate
	}

	patches = append(patches, jsonpatch.JsonPatchOperation{
		Operation: "replace",
		Path:      "/spec/evictionStrategy",
		Value:     value,
	})

	return patches
}
