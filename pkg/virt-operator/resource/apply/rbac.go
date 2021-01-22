package apply

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/creation/rbac"
)

func (r *Reconciler) createOrUpdateClusterRole(cr *rbacv1.ClusterRole, imageTag string, imageRegistry string, id string) error {

	var err error
	rbac := r.clientset.RbacV1()

	var cachedCr *rbacv1.ClusterRole

	cr = cr.DeepCopy()
	obj, exists, _ := r.stores.ClusterRoleCache.Get(cr)

	if exists {
		cachedCr = obj.(*rbacv1.ClusterRole)
	}

	injectOperatorMetadata(r.kv, &cr.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.ClusterRole.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.ClusterRoles().Create(context.Background(), cr, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ClusterRole.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("clusterrole %v created", cr.GetName())
	} else if !objectMatchesVersion(&cachedCr.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.ClusterRoles().Update(context.Background(), cr, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update clusterrole %+v: %v", cr, err)
		}
		log.Log.V(2).Infof("clusterrole %v updated", cr.GetName())

	} else {
		log.Log.V(4).Infof("clusterrole %v already exists", cr.GetName())
	}

	return nil
}

func (r *Reconciler) createOrUpdateRoleBinding(rb *rbacv1.RoleBinding,
	imageTag string,
	imageRegistry string,
	id string,
	namespace string) error {

	if !r.stores.ServiceMonitorEnabled && (rb.Name == rbac.MONITOR_SERVICEACCOUNT_NAME) {
		return nil
	}

	var err error
	rbac := r.clientset.RbacV1()

	var cachedRb *rbacv1.RoleBinding

	rb = rb.DeepCopy()
	obj, exists, _ := r.stores.RoleBindingCache.Get(rb)

	if exists {
		cachedRb = obj.(*rbacv1.RoleBinding)
	}

	injectOperatorMetadata(r.kv, &rb.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.RoleBinding.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.RoleBindings(namespace).Create(context.Background(), rb, metav1.CreateOptions{})
		if err != nil {
			r.expectations.RoleBinding.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create rolebinding %+v: %v", rb, err)
		}

		log.Log.V(2).Infof("rolebinding %v created", rb.GetName())
	} else if !objectMatchesVersion(&cachedRb.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.RoleBindings(namespace).Update(context.Background(), rb, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update rolebinding %+v: %v", rb, err)
		}

		log.Log.V(2).Infof("rolebinding %v updated", rb.GetName())
	} else {
		log.Log.V(4).Infof("rolebinding %v already exists", rb.GetName())
	}

	return nil
}

func (r *Reconciler) createOrUpdateRole(role *rbacv1.Role,
	imageTag string,
	imageRegistry string,
	id string,
	namespace string) error {

	if !r.stores.ServiceMonitorEnabled && (role.Name == rbac.MONITOR_SERVICEACCOUNT_NAME) {
		return nil
	}

	var err error
	rbac := r.clientset.RbacV1()

	var cachedR *rbacv1.Role

	role = role.DeepCopy()
	obj, exists, _ := r.stores.RoleCache.Get(role)
	if exists {
		cachedR = obj.(*rbacv1.Role)
	}

	injectOperatorMetadata(r.kv, &role.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.Role.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{})
		if err != nil {
			r.expectations.Role.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create role %+v: %v", r, err)
		}

		log.Log.V(2).Infof("role %v created", role.GetName())
	} else if !objectMatchesVersion(&cachedR.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.Roles(namespace).Update(context.Background(), role, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update role %+v: %v", r, err)
		}
		log.Log.V(2).Infof("role %v updated", role.GetName())

	} else {
		log.Log.V(4).Infof("role %v already exists", role.GetName())
	}
	return nil
}

func (r *Reconciler) createOrUpdateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding,
	imageTag string,
	imageRegistry string,
	id string) error {

	var err error
	rbac := r.clientset.RbacV1()

	var cachedCrb *rbacv1.ClusterRoleBinding

	crb = crb.DeepCopy()
	obj, exists, _ := r.stores.ClusterRoleBindingCache.Get(crb)
	if exists {
		cachedCrb = obj.(*rbacv1.ClusterRoleBinding)
	}

	injectOperatorMetadata(r.kv, &crb.ObjectMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		r.expectations.ClusterRoleBinding.RaiseExpectations(r.kvKey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
		if err != nil {
			r.expectations.ClusterRoleBinding.LowerExpectations(r.kvKey, 1, 0)
			return fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("clusterrolebinding %v created", crb.GetName())
	} else if !objectMatchesVersion(&cachedCrb.ObjectMeta, imageTag, imageRegistry, id, r.kv.GetGeneration()) {
		// Update existing, we don't need to patch for rbac rules.
		_, err = rbac.ClusterRoleBindings().Update(context.Background(), crb, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update clusterrolebinding %+v: %v", crb, err)
		}
		log.Log.V(2).Infof("clusterrolebinding %v updated", crb.GetName())

	} else {
		log.Log.V(4).Infof("clusterrolebinding %v already exists", crb.GetName())
	}

	return nil
}
