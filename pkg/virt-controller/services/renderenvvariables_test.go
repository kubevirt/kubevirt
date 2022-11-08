package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
)

var _ = Describe("Environment Variable renderer", func() {

	DescribeTable(
		"computes the environment variables for the launcher container",
		func(networkResources map[string]string, matcher gomegatypes.GomegaMatcher) {
			Expect(NewEnvVariablesRenderer(networkResources).Render()).To(matcher)
		},
		Entry(
			"without network resources it only defines the pod name",
			map[string]string{},
			ConsistOf(podNameEnvVar()),
		),
		Entry(
			"defines the network resources when provided",
			map[string]string{"net1": "bag-o-beans"},
			ConsistOf(
				k8sv1.EnvVar{Name: "KUBEVIRT_RESOURCE_NAME_net1", Value: "bag-o-beans"},
				podNameEnvVar(),
			),
		),
	)
})

func podNameEnvVar() k8sv1.EnvVar {
	return k8sv1.EnvVar{
		Name:  "POD_NAME",
		Value: "",
		ValueFrom: &k8sv1.EnvVarSource{
			FieldRef: &k8sv1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	}
}
