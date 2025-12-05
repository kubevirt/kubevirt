package apply

import (
	"context"
	"encoding/json"
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func (r *Reconciler) createOrUpdateRoutes(caBundle []byte) error {
	if !r.config.IsOnOpenshift {
		return nil
	}

	for _, route := range r.targetStrategy.Routes() {
		switch route.Name {
		case components.VirtExportProxyName:
			return r.syncExportProxyRoute(route.DeepCopy(), caBundle)
		default:
			return fmt.Errorf("unknown route %s", route.Name)
		}
	}

	return nil
}

func (r *Reconciler) syncExportProxyRoute(route *routev1.Route, caBundle []byte) error {
	if !r.exportProxyEnabled() {
		return r.deleteRoute(route)
	}

	return r.syncRoute(route, caBundle)
}

func (r *Reconciler) syncRoute(route *routev1.Route, caBundle []byte) error {
	version, imageRegistry, id := getTargetVersionRegistryID(r.kv)
	injectOperatorMetadata(r.kv, &route.ObjectMeta, version, imageRegistry, id, true)
	route.Spec.TLS.DestinationCACertificate = string(caBundle)

	var oldResourceVersion string
	obj, exists, err := r.stores.RouteCache.Get(route)
	if err != nil {
		return err
	}

	if exists {
		cached := obj.(*routev1.Route)
		oldResourceVersion = cached.ResourceVersion
	}

	routeJSON, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("unable to marshal route: %w", err)
	}

	patchOptions := metav1.PatchOptions{
		FieldManager: v1.ManagedByLabelOperatorValue,
		Force:        pointer.P(true),
	}

	result, err := r.clientset.RouteClient().Routes(route.Namespace).Patch(context.Background(), route.Name, types.ApplyPatchType, routeJSON, patchOptions)
	if err != nil {
		return fmt.Errorf("unable to perform server-side apply on route: %w", err)
	}

	if result != nil && result.ResourceVersion != oldResourceVersion {
		log.Log.V(4).Infof("route %v updated", route.GetName())
	}

	return nil
}

func (r *Reconciler) deleteRoute(route *routev1.Route) error {
	obj, exists, err := r.stores.RouteCache.Get(route)
	if err != nil {
		return err
	}

	if !exists || obj.(*routev1.Route).DeletionTimestamp != nil {
		return nil
	}

	err = r.clientset.RouteClient().Routes(route.Namespace).Delete(context.Background(), route.Name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	return nil
}
