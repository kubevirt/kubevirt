package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// Install registers the API group and adds all of its types to the given scheme.
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(operators.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1.SchemeGroupVersion, v1alpha1.SchemeGroupVersion))
}