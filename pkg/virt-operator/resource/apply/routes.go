package apply

import (
	"context"
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
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

	var cachedRoute *routev1.Route
	obj, exists, err := r.stores.RouteCache.Get(route)
	if err != nil {
		return err
	}

	if !exists {
		r.expectations.Route.RaiseExpectations(r.kvKey, 1, 0)
		_, err := r.virtClientset.RouteClient().Routes(route.Namespace).Create(context.Background(), route, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Route.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create route %+v: %v", route, err)
		}

		return nil
	}

	cachedRoute = obj.(*routev1.Route).DeepCopy()
	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &cachedRoute.ObjectMeta, route.ObjectMeta)
	kindSame := equality.Semantic.DeepEqual(cachedRoute.Spec.To.Kind, route.Spec.To.Kind)
	nameSame := equality.Semantic.DeepEqual(cachedRoute.Spec.To.Name, route.Spec.To.Name)
	terminationSame := equality.Semantic.DeepEqual(cachedRoute.Spec.TLS.Termination, route.Spec.TLS.Termination)
	certSame := equality.Semantic.DeepEqual(cachedRoute.Spec.TLS.DestinationCACertificate, route.Spec.TLS.DestinationCACertificate)
	if !*modified && kindSame && nameSame && terminationSame && certSame {
		log.Log.V(4).Infof("route %v is up-to-date", route.GetName())

		return nil
	}

	patchBytes, err := patch.New(getPatchWithObjectMetaAndSpec([]patch.PatchOption{}, &route.ObjectMeta, route.Spec)...).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = r.virtClientset.RouteClient().Routes(route.Namespace).Patch(context.Background(), route.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to patch route %+v: %v", route, err)
	}
	log.Log.V(4).Infof("route %v updated", route.GetName())

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

	key, err := controller.KeyFunc(route)
	if err != nil {
		return err
	}
	r.expectations.Route.AddExpectedDeletion(r.kvKey, key)
	if err := r.virtClientset.RouteClient().Routes(route.Namespace).Delete(context.Background(), route.Name, metav1.DeleteOptions{}); err != nil {
		r.expectations.Route.DeletionObserved(r.kvKey, key)
		return err
	}

	return nil
}
