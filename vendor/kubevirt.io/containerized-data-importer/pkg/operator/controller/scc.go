/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"github.com/go-logr/logr"
	secv1 "github.com/openshift/api/security/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
)

func (r *ReconcileCDI) watchSecurityContextConstraints(c controller.Controller) error {
	err := c.Watch(
		&source.Kind{Type: &secv1.SecurityContextConstraints{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				var rrs []reconcile.Request
				cdiList := &cdiv1alpha1.CDIList{}

				if err := r.client.List(context.TODO(), &client.ListOptions{}, cdiList); err != nil {
					log.Error(err, "Error listing all CDI objects")
					return nil
				}

				for _, cdi := range cdiList.Items {
					rr := reconcile.Request{
						NamespacedName: types.NamespacedName{Namespace: cdi.Namespace, Name: cdi.Name},
					}
					rrs = append(rrs, rr)
				}

				return rrs
			}),
		})
	if err != nil {
		if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
			log.Info("Not watching SecurityContextConstraints")
			return nil
		}

		return err
	}

	return nil
}

func (r *ReconcileCDI) syncPrivilegedAccounts(logger logr.Logger, cr *cdiv1alpha1.CDI, add bool) error {
	constraints := &secv1.SecurityContextConstraints{}
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: "privileged"}, constraints); err != nil {
		if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}

		return err
	}

	accounts := cdinamespaced.GetPrivilegedAccounts(r.getNamespacedArgs(cr))

	update := false
	for _, account := range accounts {
		i := -1
		for j, u := range constraints.Users {
			if u == account {
				i = j
				break
			}
		}

		if i == -1 && add {
			constraints.Users = append(constraints.Users, account)
			update = true
		} else if i >= 0 && !add {
			constraints.Users = append(constraints.Users[:i], constraints.Users[i+1:]...)
			update = true
		}
	}

	if update {
		logger.Info("Updating SecurityContextConstraints")

		if err := r.client.Update(context.TODO(), constraints); err != nil {
			return err
		}
	}

	return nil
}
