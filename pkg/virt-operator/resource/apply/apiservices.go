/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package apply

import (
	"context"
	"fmt"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (r *Reconciler) createOrUpdateAPIServices(caBundle []byte) error {
	for _, apiService := range r.targetStrategy.APIServices() {
		err := r.createOrUpdateAPIService(apiService.DeepCopy(), caBundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateAPIService(apiService *apiregv1.APIService, caBundle []byte) error {
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &apiService.ObjectMeta, version, imageRegistry, id, true)
	apiService.Spec.CABundle = caBundle

	var cachedAPIService *apiregv1.APIService
	var err error
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
	} else {
		cachedAPIService = obj.(*apiregv1.APIService)
	}

	if !exists {
		r.expectations.APIService.RaiseExpectations(r.kvKey, 1, 0)
		_, err := r.aggregatorclient.Create(context.Background(), apiService, metav1.CreateOptions{})
		if err != nil {
			r.expectations.APIService.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create apiservice %+v: %v", apiService, err)
		}

		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedAPIService.ObjectMeta, apiService.ObjectMeta)
	serviceSame := equality.Semantic.DeepEqual(cachedAPIService.Spec.Service, apiService.Spec.Service)
	certsSame := equality.Semantic.DeepEqual(apiService.Spec.CABundle, cachedAPIService.Spec.CABundle)
	prioritySame := cachedAPIService.Spec.VersionPriority == apiService.Spec.VersionPriority && cachedAPIService.Spec.GroupPriorityMinimum == apiService.Spec.GroupPriorityMinimum
	insecureSame := cachedAPIService.Spec.InsecureSkipTLSVerify == apiService.Spec.InsecureSkipTLSVerify
	// there was no change to metadata, the service and priorities were right
	if !*modified && serviceSame && prioritySame && insecureSame && certsSame {
		log.Log.V(4).Infof("apiservice %v is up-to-date", apiService.GetName())

		return nil
	}

	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &apiService.ObjectMeta, apiService.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = r.aggregatorclient.Patch(context.Background(), apiService.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch apiservice %+v: %v", apiService, err)
	}
	log.Log.V(4).Infof("apiservice %v updated", apiService.GetName())

	return nil
}
