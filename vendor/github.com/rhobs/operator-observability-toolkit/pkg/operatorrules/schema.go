package operatorrules

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func AddToScheme(scheme *runtime.Scheme) error {
	err := promv1.AddToScheme(scheme)
	if err != nil {
		return err
	}

	err = rbacv1.AddToScheme(scheme)
	if err != nil {
		return err
	}

	return nil
}
