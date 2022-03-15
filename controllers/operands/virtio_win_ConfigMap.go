package operands

import (
	"errors"
	"os"

	log "github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const virtioWinCmName = "virtio-win"

// **** Virtio-Win ConfigMap Handler ****

func newVirtioWinCmHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	virtioWincm, err := NewVirtioWinCm(hc)
	if err != nil {
		return nil, err
	}
	return []Operand{newCmHandler(Client, Scheme, virtioWincm)}, nil
}

// **** Virtio-Win ConfigMap Role Handler ****

func newVirtioWinCmReaderRoleHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	virtioWinRole := NewVirtioWinCmReaderRole(hc)

	return []Operand{newRoleHandler(Client, Scheme, virtioWinRole)}, nil
}

// **** Virtio-Win ConfigMap RoleBinding Handler ****
func newVirtioWinCmReaderRoleBindingHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	virtioWinRoleBinding := NewVirtioWinCmReaderRoleBinding(hc)

	return []Operand{newRoleBindingHandler(Client, Scheme, virtioWinRoleBinding)}, nil
}

func NewVirtioWinCm(hc *hcov1beta1.HyperConverged) (*corev1.ConfigMap, error) {
	virtiowinContainer := os.Getenv("VIRTIOWIN_CONTAINER")
	if virtiowinContainer == "" {
		return nil, errors.New("kv-virtiowin-image-name was not specified")
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtioWinCmName,
			Labels:    getLabels(hc, hcoutil.AppComponentDeployment),
			Namespace: hc.Namespace,
		},
		Data: map[string]string{
			"virtio-win-image": virtiowinContainer,
		},
	}, nil
}

func NewVirtioWinCmReaderRole(hc *hcov1beta1.HyperConverged) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtioWinCmName,
			Labels:    getLabels(hc, hcoutil.AppComponentDeployment),
			Namespace: hc.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{virtioWinCmName},
				Verbs:         []string{"get"},
			},
		},
	}
}

func NewVirtioWinCmReaderRoleBinding(hc *hcov1beta1.HyperConverged) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      virtioWinCmName,
			Labels:    getLabels(hc, hcoutil.AppComponentStorage),
			Namespace: hc.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     virtioWinCmName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			},
		},
	}
}
