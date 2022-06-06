package alerts

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	operatorName        = "hyperconverged-cluster-operator"
	roleName            = operatorName + "-metrics"
	monitoringNamespace = "openshift-monitoring"
)

// RoleReconciler maintains an RBAC Role to allow Prometheus operator to read from HCO metric
type RoleReconciler struct {
	theRole *rbacv1.Role
}

func newRoleReconciler(namespace string, owner metav1.OwnerReference) *RoleReconciler {
	return &RoleReconciler{
		theRole: newRole(owner, namespace),
	}
}

func (r *RoleReconciler) Kind() string {
	return "Role"
}

func (r *RoleReconciler) ResourceName() string {
	return r.theRole.Name
}

func (r *RoleReconciler) GetFullResource() client.Object {
	return r.theRole.DeepCopy()
}

func (r *RoleReconciler) EmptyObject() client.Object {
	return &rbacv1.Role{}
}

func (r *RoleReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, resource client.Object, logger logr.Logger) (client.Object, bool, error) {
	needUpdate := false
	role := resource.(*rbacv1.Role)

	if !reflect.DeepEqual(r.theRole.Rules, role.Rules) {
		if len(r.theRole.Rules) > 0 {
			needUpdate = true
			role.Rules = make([]rbacv1.PolicyRule, len(r.theRole.Rules))
			for i, rule := range r.theRole.Rules {
				rule.DeepCopyInto(&role.Rules[i])
			}
		} else {
			role.Rules = nil
		}
	}

	needUpdate = updateCommonDetails(&r.theRole.ObjectMeta, &role.ObjectMeta) || needUpdate

	if needUpdate {
		logger.Info("updating the Role")
		err := cl.Update(ctx, role)
		if err != nil {
			logger.Error(err, "failed to update the Role")
			return nil, false, err
		}
		logger.Info("successfully updated the Role")
	}

	return role, needUpdate, nil

}

func newRole(owner metav1.OwnerReference, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       namespace,
			Labels:          hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

// RoleBindingReconciler maintains an RBAC RoleBinding to allow Prometheus operator to read from HCO metric
type RoleBindingReconciler struct {
	theRoleBinding *rbacv1.RoleBinding
}

func newRoleBindingReconciler(namespace string, owner metav1.OwnerReference) *RoleBindingReconciler {
	return &RoleBindingReconciler{
		theRoleBinding: newRoleBinding(owner, namespace),
	}
}

func (r *RoleBindingReconciler) Kind() string {
	return "RoleBinding"
}

func (r *RoleBindingReconciler) ResourceName() string {
	return r.theRoleBinding.Name
}

func (r *RoleBindingReconciler) GetFullResource() client.Object {
	return r.theRoleBinding.DeepCopy()
}

func (r *RoleBindingReconciler) EmptyObject() client.Object {
	return &rbacv1.RoleBinding{}
}

func (r *RoleBindingReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, resource client.Object, logger logr.Logger) (client.Object, bool, error) {
	needUpdate := false
	existing := resource.(*rbacv1.RoleBinding)

	if !reflect.DeepEqual(r.theRoleBinding.RoleRef, existing.RoleRef) {
		r.theRoleBinding.RoleRef.DeepCopyInto(&existing.RoleRef)
		needUpdate = true
	}

	if !reflect.DeepEqual(r.theRoleBinding.Subjects, existing.Subjects) {
		if len(r.theRoleBinding.Subjects) > 0 {
			existing.Subjects = make([]rbacv1.Subject, len(r.theRoleBinding.Subjects))
			for i, sub := range r.theRoleBinding.Subjects {
				sub.DeepCopyInto(&existing.Subjects[i])
			}
		} else {
			existing.Subjects = nil
		}

		needUpdate = true
	}

	needUpdate = updateCommonDetails(&r.theRoleBinding.ObjectMeta, &existing.ObjectMeta) || needUpdate

	if needUpdate {
		err := cl.Update(ctx, existing)
		if err != nil {
			logger.Error(err, "failed to update the RoleBinding")
			return nil, false, err
		}
		logger.Info("successfully updated the RoleBinding")
	}

	return existing, needUpdate, nil
}

func newRoleBinding(owner metav1.OwnerReference, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       namespace,
			Labels:          hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "prometheus-k8s",
				Namespace: monitoringNamespace,
			},
		},
	}
}
