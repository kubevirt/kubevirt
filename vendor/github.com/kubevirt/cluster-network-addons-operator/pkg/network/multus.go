package network

import (
	"os"
	"path/filepath"
	"reflect"

	osnetv1 "github.com/openshift/cluster-network-operator/pkg/apis/networkoperator/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/render"
)

// ValidateMultus validates the combination of DisableMultiNetwork and AddtionalNetworks
func validateMultus(conf *opv1alpha1.NetworkAddonsConfigSpec, openshiftNetworkConfig *osnetv1.NetworkConfig) []error {
	if conf.Multus == nil {
		return []error{}
	}

	if openshiftNetworkConfig != nil {
		if openshiftNetworkConfig.Spec.DisableMultiNetwork == newTrue() {
			return []error{errors.Errorf("multus has been requested, but is disabled on OpenShift Cluster Network Operator")}
		}
	}

	return []error{}
}

func changeSafeMultus(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.Multus != nil && !reflect.DeepEqual(prev.Multus, next.Multus) {
		return []error{errors.Errorf("cannot modify Multus configuration once it is deployed")}
	}
	return nil
}

// RenderMultus generates the manifests of Multus
func renderMultus(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, openshiftNetworkConfig *osnetv1.NetworkConfig, enableSCC bool) ([]*unstructured.Unstructured, error) {
	if conf.Multus == nil || openshiftNetworkConfig != nil {
		return nil, nil
	}

	// render manifests from disk
	data := render.MakeRenderData()
	data.Data["MultusImage"] = os.Getenv("MULTUS_IMAGE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	data.Data["EnableSCC"] = enableSCC

	objs, err := render.RenderDir(filepath.Join(manifestDir, "multus"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render multus manifests")
	}

	return objs, nil
}

func newTrue() *bool {
	val := true
	return &val
}
