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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/operator/resources/utils"
)

const (
	// ReconcileStatePreCreate is the state before a resource is created
	ReconcileStatePreCreate ReconcileState = "PRE_CREATE"
	// ReconcileStatePostCreate is the state sfter a resource is created
	ReconcileStatePostCreate ReconcileState = "POST_CREATE"

	// ReconcileStatePostRead is the state sfter a resource is read
	ReconcileStatePostRead ReconcileState = "POST_READ"

	// ReconcileStatePreUpdate is the state before a resource is updated
	ReconcileStatePreUpdate ReconcileState = "PRE_UPDATE"
	// ReconcileStatePostUpdate is the state after a resource is updated
	ReconcileStatePostUpdate ReconcileState = "POST_UPDATE"

	// ReconcileStatePreDelete is the state before a resource is explicitly deleted (probably during upgrade)
	// don't count on this always being called for your resource
	// ideally we just let garbage collection do it's thing
	ReconcileStatePreDelete ReconcileState = "PRE_DELETE"
	// ReconcileStatePostDelete is the state after a resource is explicitly deleted (probably during upgrade)
	// don't count on this always being called for your resource
	// ideally we just let garbage collection do it's thing
	ReconcileStatePostDelete ReconcileState = "POST_DELETE"

	// ReconcileStateCDIDelete is called during CDI finalizer
	ReconcileStateCDIDelete ReconcileState = "CDI_DELETE"
)

// ReconcileState is the current state of the reconcile for a particuar resource
type ReconcileState string

// ReconcileCallbackArgs contains the data of a ReconcileCallback
type ReconcileCallbackArgs struct {
	Logger logr.Logger
	Client client.Client
	Scheme *runtime.Scheme

	State         ReconcileState
	DesiredObject runtime.Object
	CurrentObject runtime.Object
}

// ReconcileCallback is the callback function
type ReconcileCallback func(args *ReconcileCallbackArgs) error

func getExplicitWatchTypes() []runtime.Object {
	return []runtime.Object{&routev1.Route{}}
}

func addReconcileCallbacks(r *ReconcileCDI) {
	r.addCallback(&appsv1.Deployment{}, reconcileDeleteControllerDeployment)
	r.addCallback(&corev1.ServiceAccount{}, reconcileServiceAccountRead)
	r.addCallback(&corev1.ServiceAccount{}, reconcileServiceAccounts)
	r.addCallback(&appsv1.Deployment{}, reconcileCreateRoute)
}

func isControllerDeployment(d *appsv1.Deployment) bool {
	return d.Name == "cdi-deployment"
}

func reconcileDeleteControllerDeployment(args *ReconcileCallbackArgs) error {
	switch args.State {
	case ReconcileStatePostDelete, ReconcileStateCDIDelete:
	default:
		return nil
	}

	var deployment *appsv1.Deployment
	if args.DesiredObject != nil {
		deployment = args.DesiredObject.(*appsv1.Deployment)
	} else if args.CurrentObject != nil {
		deployment = args.CurrentObject.(*appsv1.Deployment)
	} else {
		args.Logger.Info("Received callback with no desired/current object")
		return nil
	}

	if !isControllerDeployment(deployment) {
		return nil
	}

	args.Logger.Info("Deleting CDI deployment and all import/upload/clone pods/services")

	err := args.Client.Delete(context.TODO(), deployment, func(opts *client.DeleteOptions) {
		p := metav1.DeletePropagationForeground
		opts.PropagationPolicy = &p
	})
	if err != nil && !errors.IsNotFound(err) {
		args.Logger.Error(err, "Error deleting cdi controller deployment")
		return err
	}

	if err = deleteWorkerResources(args.Logger, args.Client); err != nil {
		args.Logger.Error(err, "Error deleting worker resources")
		return err
	}

	return nil
}

func reconcileCreateRoute(args *ReconcileCallbackArgs) error {
	if args.State != ReconcileStatePostRead {
		return nil
	}

	deployment := args.CurrentObject.(*appsv1.Deployment)
	if !isControllerDeployment(deployment) || !checkDeploymentReady(deployment) {
		return nil
	}

	if err := ensureUploadProxyRouteExists(args.Logger, args.Client, args.Scheme, deployment); err != nil {
		return err
	}

	return nil
}

func reconcileServiceAccountRead(args *ReconcileCallbackArgs) error {
	if args.State != ReconcileStatePostRead {
		return nil
	}

	do := args.DesiredObject.(*corev1.ServiceAccount)
	co := args.CurrentObject.(*corev1.ServiceAccount)

	delete(co.Annotations, utils.SCCAnnotation)

	val, exists := do.Annotations[utils.SCCAnnotation]
	if exists {
		if co.Annotations == nil {
			co.Annotations = make(map[string]string)
		}
		co.Annotations[utils.SCCAnnotation] = val
	}

	return nil
}

func reconcileServiceAccounts(args *ReconcileCallbackArgs) error {
	switch args.State {
	case ReconcileStatePreCreate, ReconcileStatePreUpdate, ReconcileStatePostDelete, ReconcileStateCDIDelete:
	default:
		return nil
	}

	var sa *corev1.ServiceAccount
	if args.CurrentObject != nil {
		sa = args.CurrentObject.(*corev1.ServiceAccount)
	} else if args.DesiredObject != nil {
		sa = args.DesiredObject.(*corev1.ServiceAccount)
	} else {
		args.Logger.Info("Received callback with no desired/current object")
		return nil
	}

	desiredSCCs := []string{}
	saName := fmt.Sprintf("system:serviceaccount:%s:%s", sa.Namespace, sa.Name)

	switch args.State {
	case ReconcileStatePreCreate, ReconcileStatePreUpdate:
		val, exists := sa.Annotations[utils.SCCAnnotation]
		if exists {
			if err := json.Unmarshal([]byte(val), &desiredSCCs); err != nil {
				args.Logger.Error(err, "Error unmarshalling data")
				return err
			}
		}
	default:
		// want desiredSCCs empty because deleting resource/CDI
	}

	listObj := &secv1.SecurityContextConstraintsList{}
	if err := args.Client.List(context.TODO(), &client.ListOptions{}, listObj); err != nil {
		if meta.IsNoMatchError(err) {
			// not openshift
			return nil
		}
		args.Logger.Error(err, "Error listing SCCs")
		return err
	}

	for _, scc := range listObj.Items {
		desiredUsers := []string{}
		add := containsValue(desiredSCCs, scc.Name)
		seenUser := false

		for _, u := range scc.Users {
			if u == saName {
				seenUser = true
				if !add {
					continue
				}
			}
			desiredUsers = append(desiredUsers, u)
		}

		if add && !seenUser {
			desiredUsers = append(desiredUsers, saName)
		}

		if !reflect.DeepEqual(desiredUsers, scc.Users) {
			args.Logger.Info("Doing SCC update", "name", scc.Name, "desired", desiredUsers, "current", scc.Users)
			scc.Users = desiredUsers
			if err := args.Client.Update(context.TODO(), &scc); err != nil {
				args.Logger.Error(err, "Error updating SCC")
				return err
			}
		}
	}

	return nil
}

func deleteWorkerResources(l logr.Logger, c client.Client) error {
	listTypes := []runtime.Object{&corev1.PodList{}, &corev1.ServiceList{}}

	for _, lt := range listTypes {
		lo := &client.ListOptions{}
		lo.SetLabelSelector(fmt.Sprintf("cdi.kubevirt.io in (%s, %s, %s)",
			common.ImporterPodName, common.UploadServerCDILabel, common.ClonerSourcePodName))

		if err := c.List(context.TODO(), lo, lt); err != nil {
			l.Error(err, "Error listing resources")
			return err
		}

		sv := reflect.ValueOf(lt).Elem()
		iv := sv.FieldByName("Items")

		for i := 0; i < iv.Len(); i++ {
			obj := iv.Index(i).Addr().Interface().(runtime.Object)
			l.Info("Deleting", "type", reflect.TypeOf(obj), "obj", obj)
			if err := c.Delete(context.TODO(), obj); err != nil {
				l.Error(err, "Error deleting a resource")
				return err
			}
		}
	}

	return nil
}

func containsValue(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
