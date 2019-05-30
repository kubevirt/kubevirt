package network

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

const defaultImagePullPolicy = v1.PullIfNotPresent

func validateImagePullPolicy(conf *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if conf.ImagePullPolicy == "" {
		return []error{}
	}

	if valid := verifyPullPolicyType(conf.ImagePullPolicy); !valid {
		return []error{errors.Errorf("requested imagePullPolicy '%s' is not valid", conf.ImagePullPolicy)}
	}

	return []error{}
}

func fillDefaultsImagePullPolicy(conf, previous *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if conf.ImagePullPolicy == "" {
		if previous != nil && previous.ImagePullPolicy != "" {
			conf.ImagePullPolicy = previous.ImagePullPolicy
		} else {
			conf.ImagePullPolicy = defaultImagePullPolicy
		}
	}

	return []error{}
}

func changeSafeImagePullPolicy(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.ImagePullPolicy != "" && prev.ImagePullPolicy != next.ImagePullPolicy {
		return []error{errors.Errorf("cannot modify ImagePullPolicy configuration once components were deployed")}
	}
	return []error{}
}

// Verify if the value is a valid PullPolicy
func verifyPullPolicyType(imagePullPolicy v1.PullPolicy) bool {
	switch imagePullPolicy {
	case v1.PullAlways:
		return true
	case v1.PullNever:
		return true
	case v1.PullIfNotPresent:
		return true
	default:
		return false
	}
}
