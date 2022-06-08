package operands

import (
	"errors"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// ********* Role Handler *****************************

func newRoleHandler(Client client.Client, Scheme *runtime.Scheme, required *rbacv1.Role) Operand {
	return &genericOperand{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "Role",
		setControllerReference: true,
		hooks:                  &roleHooks{required: required},
	}
}

type roleHooks struct {
	required *rbacv1.Role
}

func (h roleHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}
func (h roleHooks) getEmptyCr() client.Object {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}
func (h *roleHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	role := h.required
	found, ok := exists.(*rbacv1.Role)
	if !ok {
		return false, false, errors.New("can't convert to a Role")
	}

	if !reflect.DeepEqual(found.Labels, role.Labels) ||
		!reflect.DeepEqual(found.Rules, role.Rules) {

		req.Logger.Info("Updating existing Role to its default values", "name", found.Name)

		found.Rules = make([]rbacv1.PolicyRule, len(role.Rules))
		for i := range role.Rules {
			role.Rules[i].DeepCopyInto(&found.Rules[i])
		}
		util.DeepCopyLabels(&role.ObjectMeta, &found.ObjectMeta)

		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func (h roleHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

// ********* Role Binding Handler *****************************

func newRoleBindingHandler(Client client.Client, Scheme *runtime.Scheme, required *rbacv1.RoleBinding) Operand {
	return &genericOperand{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "RoleBinding",
		setControllerReference: true,
		hooks:                  &roleBindingHooks{required: required},
	}
}

type roleBindingHooks struct {
	required *rbacv1.RoleBinding
}

func (h roleBindingHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}
func (h roleBindingHooks) getEmptyCr() client.Object {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}
func (h *roleBindingHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	configReaderRoleBinding := h.required
	found, ok := exists.(*rbacv1.RoleBinding)
	if !ok {
		return false, false, errors.New("can't convert to a RoleBinding")
	}

	if !reflect.DeepEqual(found.Labels, configReaderRoleBinding.Labels) ||
		!reflect.DeepEqual(found.Subjects, configReaderRoleBinding.Subjects) ||
		!reflect.DeepEqual(found.RoleRef, configReaderRoleBinding.RoleRef) {
		req.Logger.Info("Updating existing RoleBinding to its default values", "name", found.Name)

		found.Subjects = make([]rbacv1.Subject, len(configReaderRoleBinding.Subjects))
		copy(found.Subjects, configReaderRoleBinding.Subjects)
		found.RoleRef = configReaderRoleBinding.RoleRef
		util.DeepCopyLabels(&configReaderRoleBinding.ObjectMeta, &found.ObjectMeta)

		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func (h roleBindingHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }
