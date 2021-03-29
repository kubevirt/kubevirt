package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	"kubevirt.io/client-go/log"
)

func (r *Reconciler) createOrUpdateAPIServices(caBundle []byte) error {

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	for _, apiService := range r.targetStrategy.APIServices() {
		var cachedAPIService *v1beta1.APIService
		var err error

		apiService = apiService.DeepCopy()

		apiService.Spec.CABundle = caBundle

		obj, exists, _ := r.stores.APIServiceCache.Get(apiService)
		// since these objects was in the past unmanaged, reconcile and pick it up if it exists
		if !exists {
			cachedAPIService, err = r.aggregatorclient.Get(context.Background(), apiService.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				exists = false
			} else if err != nil {
				return err
			} else {
				exists = true
			}
		} else if exists {
			cachedAPIService = obj.(*v1beta1.APIService)
		}

		certsMatch := true
		if exists {
			if !reflect.DeepEqual(apiService.Spec.CABundle, cachedAPIService.Spec.CABundle) {
				certsMatch = false
			}
		}

		injectOperatorMetadata(r.kv, &apiService.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			r.expectations.APIService.RaiseExpectations(r.kvKey, 1, 0)
			_, err := r.aggregatorclient.Create(context.Background(), apiService, metav1.CreateOptions{})
			if err != nil {
				r.expectations.APIService.LowerExpectations(r.kvKey, 1, 0)
				return fmt.Errorf("unable to create apiservice %+v: %v", apiService, err)
			}
		} else {
			if !objectMatchesVersion(&cachedAPIService.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) || !certsMatch {
				spec, err := json.Marshal(apiService.Spec)
				if err != nil {
					return err
				}

				ops, err := getPatchWithObjectMetaAndSpec([]string{}, &apiService.ObjectMeta, spec)
				if err != nil {
					return err
				}
				_, err = r.aggregatorclient.Patch(context.Background(), apiService.Name, types.JSONPatchType, generatePatchBytes(ops), metav1.PatchOptions{})
				if err != nil {
					return fmt.Errorf("unable to patch apiservice %+v: %v", apiService, err)
				}
				log.Log.V(2).Infof("apiservice %v updated", apiService.GetName())

			} else {
				log.Log.V(4).Infof("apiservice %v is up-to-date", apiService.GetName())
			}
		}
	}
	return nil
}
