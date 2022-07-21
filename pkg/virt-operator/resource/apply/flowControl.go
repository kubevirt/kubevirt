package apply

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/imdario/mergo"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	flowcontrol "k8s.io/api/flowcontrol/v1beta2"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateFlowControls() error {
	for _, flowSchema := range r.targetStrategy.FlowSchemas() {
		log.Log.V(2).Infof("Create or Update flowschema %v", flowSchema)
		if err := r.createOrUpdateFlowSchema(flowSchema.DeepCopy()); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateFlowSchema(flowSchema *flowcontrol.FlowSchema) error {
	flowControlClient := r.clientset.FlowcontrolV1beta2()

	obj, exists, _ := r.stores.FlowControlCache.Get(flowSchema)

	// Remove
	log.Log.V(2).Infof("flowschema %v already exsists???? %v", exists, flowSchema)
	// Remove

	if !exists {
		// Create non existent
		r.expectations.FlowControl.RaiseExpectations(r.kvKey, 1, 0)
		_, err := flowControlClient.FlowSchemas().Create(context.Background(), flowSchema, metav1.CreateOptions{})
		if err != nil {
			r.expectations.FlowControl.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create flowschema %+v: %v", flowSchema, err)
		}

		log.Log.V(2).Infof("flowschema %v created", flowSchema.GetName())
		return nil
	}

	cachedFlowSchema := obj.(*flowcontrol.FlowSchema)
	flowSchemaModified, err := ensureFlowSchemaSpec(flowSchema, cachedFlowSchema)
	if err != nil {
		return err
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedFlowSchema.ObjectMeta, flowSchema.ObjectMeta)

	// there was no change to metadata and the spec fields are equal
	if !*modified && !flowSchemaModified {
		log.Log.V(4).Infof("flowSchema %v is up-to-date", flowSchema.GetName())
		return nil
	}

	// Add Spec Patch
	newSpec, err := json.Marshal(flowSchema.Spec)
	if err != nil {
		return err
	}

	ops, err := getPatchWithObjectMetaAndSpec([]string{}, &flowSchema.ObjectMeta, newSpec)
	if err != nil {
		return err
	}

	_, err = flowControlClient.FlowSchemas().Patch(context.Background(), flowSchema.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch flowSchema %+v: %v", flowSchema, err)
	}

	log.Log.V(2).Infof("flowSchema %v updated", flowSchema.GetName())

	return nil
}

func ensureFlowSchemaSpec(required, existing *flowcontrol.FlowSchema) (bool, error) {
	if err := mergo.Merge(&existing.Spec, &required.Spec); err != nil {
		return false, err
	}

	if equality.Semantic.DeepEqual(existing.Spec, required.Spec) {
		return false, nil
	}

	return true, nil
}
