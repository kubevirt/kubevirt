package apply

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/controller"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
)

func (r *Reconciler) createOrUpdateClusterRole(cr *rbacv1.ClusterRole, imageTag string, imageRegistry string, id string) error {
	return r.createOrUpdate(cr, imageTag, imageRegistry, id, false)
}

func (r *Reconciler) createOrUpdateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding, imageTag string, imageRegistry string, id string) error {
	return r.createOrUpdate(crb, imageTag, imageRegistry, id, false)
}

func (r *Reconciler) createOrUpdateRole(role *rbacv1.Role, imageTag string, imageRegistry string, id string) error {
	return r.createOrUpdate(role, imageTag, imageRegistry, id, true)
}

func (r *Reconciler) createOrUpdateRoleBinding(rb *rbacv1.RoleBinding, imageTag string, imageRegistry string, id string) error {
	return r.createOrUpdate(rb, imageTag, imageRegistry, id, true)
}

func (r *Reconciler) createOrUpdate(role runtime.Object,
	imageTag, imageRegistry, id string,
	avoidIfServiceAccount bool) (err error) {

	roleTypeName := role.GetObjectKind().GroupVersionKind().Kind
	createRole := r.getCreateFunction(role)
	updateRole := r.getUpdateFunction(role)

	cachedRoleInterface, exists, _ := r.getCache(role).Get(role)
	roleMeta := getMetaObject(role)
	if avoidIfServiceAccount && !r.stores.ServiceMonitorEnabled && (roleMeta.Name == rbac.MONITOR_SERVICEACCOUNT_NAME) {
		return nil
	}

	injectOperatorMetadata(r.kv, roleMeta, imageTag, imageRegistry, id, true)
	if !exists {
		// Create non existent
		err = createRole()
		if err != nil {
			return fmt.Errorf("unable to create %v %+v: %v", roleTypeName, role, err)
		}
		log.Log.V(2).Infof("%v %v created", roleTypeName, roleMeta.GetName())
		return nil
	}

	modified := resourcemerge.BoolPtr(false)
	cachedRole := cachedRoleInterface.(runtime.Object)
	cachedRoleMeta := getMetaObject(cachedRole)
	resourcemerge.EnsureObjectMeta(modified, cachedRoleMeta.DeepCopy(), *roleMeta)
	enforceAPIGroup(cachedRole, role)

	// there was no change to metadata, the generation matched
	if !*modified && arePolicyRulesEqual(role, cachedRole) {
		log.Log.V(4).Infof("%v %v already exists", roleTypeName, roleMeta.GetName())
		return nil
	}

	// Update existing, we don't need to patch for rbac rules.
	err = updateRole()
	if err != nil {
		return fmt.Errorf("unable to update %v %+v: %v", roleTypeName, role, err)
	}
	log.Log.V(2).Infof("%v %v updated", roleTypeName, roleMeta.GetName())

	return nil
}

func (r *Reconciler) getCreateFunction(obj runtime.Object) (createFunc func() error) {

	rbacObj := r.clientset.RbacV1()
	namespace := r.kv.Namespace

	raiseExpectation := func(exp *controller.UIDTrackingControllerExpectations) {
		exp.RaiseExpectations(r.kvKey, 1, 0)
	}
	lowerExpectationIfErr := func(exp *controller.UIDTrackingControllerExpectations, err error) {
		if err != nil {
			exp.LowerExpectations(r.kvKey, 1, 0)
		}
	}

	switch obj.(type) {
	case *rbacv1.Role:
		role := obj.(*rbacv1.Role)

		createFunc = func() error {
			raiseExpectation(r.expectations.Role)
			_, err := rbacObj.Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{})
			lowerExpectationIfErr(r.expectations.Role, err)
			return err
		}
	case *rbacv1.ClusterRole:
		role := obj.(*rbacv1.ClusterRole)

		createFunc = func() error {
			raiseExpectation(r.expectations.ClusterRole)
			_, err := rbacObj.ClusterRoles().Create(context.Background(), role, metav1.CreateOptions{})
			lowerExpectationIfErr(r.expectations.ClusterRole, err)
			return err
		}
	case *rbacv1.RoleBinding:
		roleBinding := obj.(*rbacv1.RoleBinding)

		createFunc = func() error {
			raiseExpectation(r.expectations.RoleBinding)
			_, err := rbacObj.RoleBindings(namespace).Create(context.Background(), roleBinding, metav1.CreateOptions{})
			lowerExpectationIfErr(r.expectations.RoleBinding, err)
			return err
		}
	case *rbacv1.ClusterRoleBinding:
		roleBinding := obj.(*rbacv1.ClusterRoleBinding)

		createFunc = func() error {
			raiseExpectation(r.expectations.ClusterRoleBinding)
			_, err := rbacObj.ClusterRoleBindings().Create(context.Background(), roleBinding, metav1.CreateOptions{})
			lowerExpectationIfErr(r.expectations.ClusterRoleBinding, err)
			return err
		}
	}

	return
}

func (r *Reconciler) getUpdateFunction(obj runtime.Object) (updateFunc func() (err error)) {
	rbacObj := r.clientset.RbacV1()
	namespace := r.kv.Namespace

	switch obj.(type) {
	case *rbacv1.Role:
		role := obj.(*rbacv1.Role)

		updateFunc = func() (err error) {
			_, err = rbacObj.Roles(namespace).Update(context.Background(), role, metav1.UpdateOptions{})
			return err
		}
	case *rbacv1.ClusterRole:
		role := obj.(*rbacv1.ClusterRole)

		updateFunc = func() (err error) {
			_, err = rbacObj.ClusterRoles().Update(context.Background(), role, metav1.UpdateOptions{})
			return err
		}
	case *rbacv1.RoleBinding:
		roleBinding := obj.(*rbacv1.RoleBinding)

		updateFunc = func() (err error) {
			_, err = rbacObj.RoleBindings(namespace).Update(context.Background(), roleBinding, metav1.UpdateOptions{})
			return err
		}
	case *rbacv1.ClusterRoleBinding:
		roleBinding := obj.(*rbacv1.ClusterRoleBinding)

		updateFunc = func() (err error) {
			_, err = rbacObj.ClusterRoleBindings().Update(context.Background(), roleBinding, metav1.UpdateOptions{})
			return err
		}
	}

	return
}

func getMetaObject(obj runtime.Object) (meta *metav1.ObjectMeta) {
	switch obj.(type) {
	case *rbacv1.Role:
		role := obj.(*rbacv1.Role)
		meta = &role.ObjectMeta
	case *rbacv1.ClusterRole:
		role := obj.(*rbacv1.ClusterRole)
		meta = &role.ObjectMeta
	case *rbacv1.RoleBinding:
		roleBinding := obj.(*rbacv1.RoleBinding)
		meta = &roleBinding.ObjectMeta
	case *rbacv1.ClusterRoleBinding:
		roleBinding := obj.(*rbacv1.ClusterRoleBinding)
		meta = &roleBinding.ObjectMeta
	}

	return
}

func enforceAPIGroup(existing runtime.Object, required runtime.Object) {
	var existingRoleRef *rbacv1.RoleRef
	var requiredRoleRef *rbacv1.RoleRef
	var existingSubjects []rbacv1.Subject
	var requiredSubjects []rbacv1.Subject

	switch required.(type) {
	case *rbacv1.RoleBinding:
		crExisting := existing.(*rbacv1.RoleBinding)
		crRequired := required.(*rbacv1.RoleBinding)
		existingRoleRef = &crExisting.RoleRef
		requiredRoleRef = &crRequired.RoleRef
		existingSubjects = crExisting.Subjects
		requiredSubjects = crRequired.Subjects
	case *rbacv1.ClusterRoleBinding:
		crbExisting := existing.(*rbacv1.ClusterRoleBinding)
		crbRequired := required.(*rbacv1.ClusterRoleBinding)
		existingRoleRef = &crbExisting.RoleRef
		requiredRoleRef = &crbRequired.RoleRef
		existingSubjects = crbExisting.Subjects
		requiredSubjects = crbRequired.Subjects
	default:
		return
	}

	existingRoleRef.APIGroup = rbacv1.GroupName
	for i := range existingSubjects {
		if existingSubjects[i].Kind == "User" {
			existingSubjects[i].APIGroup = rbacv1.GroupName
		}
	}

	requiredRoleRef.APIGroup = rbacv1.GroupName
	for i := range requiredSubjects {
		if existingSubjects[i].Kind == "User" {
			requiredSubjects[i].APIGroup = rbacv1.GroupName
		}
	}
}

func arePolicyRulesEqual(role1 runtime.Object, role2 runtime.Object) (equal bool) {
	// This is to avoid using reflections for performance reasons
	arePolicyRulesEqualHelper := func(pr1 []rbacv1.PolicyRule, pr2 []rbacv1.PolicyRule) bool {
		if len(pr1) != len(pr2) {
			return false
		}

		areStringListsEqual := func(strList1 []string, strList2 []string) bool {
			if len(strList1) != len(strList2) {
				return false
			}
			for i := range strList1 {
				if strList1[i] != strList2[i] {
					return false
				}
			}
			return true
		}

		for i := range pr1 {
			if !areStringListsEqual(pr1[i].Verbs, pr2[i].Verbs) || !areStringListsEqual(pr1[i].Resources, pr2[i].Resources) ||
				!areStringListsEqual(pr1[i].APIGroups, pr2[i].APIGroups) || !areStringListsEqual(pr1[i].NonResourceURLs, pr2[i].NonResourceURLs) ||
				!areStringListsEqual(pr1[i].ResourceNames, pr2[i].ResourceNames) {
				return false
			}
		}

		return true
	}

	switch role1.(type) {
	case *rbacv1.Role:
		role1Obj := role1.(*rbacv1.Role)
		role2Obj := role2.(*rbacv1.Role)
		equal = arePolicyRulesEqualHelper(role1Obj.Rules, role2Obj.Rules)
	case *rbacv1.ClusterRole:
		role1Obj := role1.(*rbacv1.ClusterRole)
		role2Obj := role2.(*rbacv1.ClusterRole)
		equal = arePolicyRulesEqualHelper(role1Obj.Rules, role2Obj.Rules)
	// Bindings do not have "rules" attribute
	default:
		equal = true
	}

	return
}

func (r *Reconciler) getCache(obj runtime.Object) (cache cache.Store) {
	switch obj.(type) {
	case *rbacv1.Role:
		cache = r.stores.RoleCache
	case *rbacv1.ClusterRole:
		cache = r.stores.ClusterRoleCache
	case *rbacv1.RoleBinding:
		cache = r.stores.RoleBindingCache
	case *rbacv1.ClusterRoleBinding:
		cache = r.stores.ClusterRoleBindingCache
	}

	return cache
}
