package operatorrules

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildRoleAndRoleBinding(namePrefix, namespace, promSAName, promSANamespace string, labels map[string]string) (*rbacv1.Role, *rbacv1.RoleBinding) {
	r := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namePrefix + "-role",
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namePrefix + "-rolebinding",
			Namespace: namespace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     namePrefix + "-role",
			APIGroup: rbacv1.GroupName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      promSAName,
				Namespace: promSANamespace,
			},
		},
	}

	return r, rb
}
