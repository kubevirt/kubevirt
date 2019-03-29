package network

import (
	"log"
	"reflect"

	osnetv1 "github.com/openshift/cluster-network-operator/pkg/apis/networkoperator/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

// Canonicalize converts configuration to a canonical form.
func Canonicalize(conf *opv1alpha1.NetworkAddonsConfigSpec) {
	// TODO
}

// Validate checks that the supplied configuration is reasonable.
// This should be called after Canonicalize
func Validate(conf *opv1alpha1.NetworkAddonsConfigSpec, openshiftNetworkConfig *osnetv1.NetworkConfig) error {
	errs := []error{}

	errs = append(errs, validateMultus(conf, openshiftNetworkConfig)...)
	errs = append(errs, validateImagePullPolicy(conf)...)

	if len(errs) > 0 {
		return errors.Errorf("invalid configuration: %v", errs)
	}
	return nil
}

// FillDefaults computes any default values and applies them to the configuration
// This is a mutating operation. It should be called after Validate.
//
// Defaults are carried forward from previous if it is provided. This is so we
// can change defaults as we move forward, but won't disrupt existing clusters.
func FillDefaults(conf, previous *opv1alpha1.NetworkAddonsConfigSpec) {
	fillDefaultsImagePullPolicy(conf, previous)
}

// IsChangeSafe checks to see if the change between prev and next are allowed
// FillDefaults and Validate should have been called.
func IsChangeSafe(prev, next *opv1alpha1.NetworkAddonsConfigSpec) error {
	if prev == nil {
		return nil
	}

	// Easy way out: nothing changed.
	if reflect.DeepEqual(prev, next) {
		return nil
	}

	errs := []error{}

	errs = append(errs, changeSafeMultus(prev, next)...)
	errs = append(errs, changeSafeLinuxBridge(prev, next)...)
	errs = append(errs, changeSafeImagePullPolicy(prev, next)...)

	if len(errs) > 0 {
		return errors.Errorf("invalid configuration: %v", errs)
	}
	return nil
}

func Render(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, openshiftNetworkConfig *osnetv1.NetworkConfig, enableSCC bool) ([]*unstructured.Unstructured, error) {
	log.Print("starting render phase")
	objs := []*unstructured.Unstructured{}

	// render Multus
	o, err := renderMultus(conf, manifestDir, openshiftNetworkConfig, enableSCC)
	if err != nil {
		return nil, err
	}
	objs = append(objs, o...)

	// render Linux Bridge
	o, err = renderLinuxBridge(conf, manifestDir, enableSCC)
	if err != nil {
		return nil, err
	}
	objs = append(objs, o...)

	log.Printf("render phase done, rendered %d objects", len(objs))
	return objs, nil
}
