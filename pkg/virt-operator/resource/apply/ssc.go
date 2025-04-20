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

	secv1 "github.com/openshift/api/security/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
)

func (r *Reconciler) createOrUpdateSCC() error {
	sec := r.clientset.SecClient()

	if !r.config.IsOnOpenshift {
		return nil
	}

	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)

	for _, scc := range r.targetStrategy.SCCs() {
		var cachedSCC *secv1.SecurityContextConstraints
		scc := scc.DeepCopy()
		obj, exists, _ := r.stores.SCCCache.GetByKey(scc.Name)
		if exists {
			cachedSCC = obj.(*secv1.SecurityContextConstraints)
		}

		injectOperatorMetadata(r.kv, &scc.ObjectMeta, version, imageRegistry, id, true)
		if !exists {
			r.expectations.SCC.RaiseExpectations(r.kvKey, 1, 0)
			_, err := sec.SecurityContextConstraints().Create(context.Background(), scc, metav1.CreateOptions{})
			if err != nil {
				r.expectations.SCC.LowerExpectations(r.kvKey, 1, 0)
				return fmt.Errorf("unable to create SCC %+v: %v", scc, err)
			}

			log.Log.V(2).Infof("SCC %v created", scc.Name)
		} else if !objectMatchesVersion(&cachedSCC.ObjectMeta, version, imageRegistry, id, r.kv.GetGeneration()) {
			scc.ObjectMeta = *cachedSCC.ObjectMeta.DeepCopy()
			injectOperatorMetadata(r.kv, &scc.ObjectMeta, version, imageRegistry, id, true)
			_, err := sec.SecurityContextConstraints().Update(context.Background(), scc, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("Unable to update %s SecurityContextConstraints", scc.Name)
			}

			log.Log.V(2).Infof("SecurityContextConstraints %s updated", scc.Name)
		} else {
			log.Log.V(4).Infof("SCC %s is up to date", scc.Name)
		}

	}

	return nil
}

func (r *Reconciler) removeKvServiceAccountsFromDefaultSCC(targetNamespace string) error {
	var remainedUsersList []string

	SCCObj, exists, err := r.stores.SCCCache.GetByKey("privileged")
	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	SCC, ok := SCCObj.(*secv1.SecurityContextConstraints)
	if !ok {
		return fmt.Errorf("couldn't cast object to SecurityContextConstraints: %+v", SCCObj)
	}

	modified := false
	kvServiceAccounts := rbac.GetKubevirtComponentsServiceAccounts(targetNamespace)
	for _, acc := range SCC.Users {
		if _, ok := kvServiceAccounts[acc]; !ok {
			remainedUsersList = append(remainedUsersList, acc)
		} else {
			modified = true
		}
	}

	if modified {
		patchBytes, err := patch.New(
			patch.WithTest("/users", SCC.Users),
			patch.WithReplace("/users", remainedUsersList),
		).GeneratePayload()
		if err != nil {
			return err
		}

		_, err = r.clientset.SecClient().SecurityContextConstraints().Patch(context.Background(), "privileged", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("unable to patch scc: %v", err)
		}
	}

	return nil
}
