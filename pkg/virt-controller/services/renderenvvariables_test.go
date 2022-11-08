package services

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Environment Variable renderer", func() {
	const beLessVerbose = 1

	DescribeTable(
		"computes the environment variables for the launcher container",
		func(networkResources map[string]string, vmiLabels map[string]string, loggingLevel uint, matcher gomegatypes.GomegaMatcher) {
			Expect(NewEnvVariablesRenderer(networkResources, vmiLabels, loggingLevel).Render()).To(matcher)
		},
		Entry(
			"without network resources it only defines the pod name",
			map[string]string{},
			map[string]string{},
			uint(virtconfig.DefaultVirtLauncherLogVerbosity),
			ConsistOf(podNameEnvVar()),
		),
		Entry(
			"defines the network resources when provided",
			map[string]string{"net1": "bag-o-beans"},
			map[string]string{},
			uint(virtconfig.DefaultVirtLauncherLogVerbosity),
			ConsistOf(
				k8sv1.EnvVar{Name: "KUBEVIRT_RESOURCE_NAME_net1", Value: "bag-o-beans"},
				podNameEnvVar(),
			),
		),
		Entry(
			"override default cluster wide log verbosity via VMI labels",
			map[string]string{},
			map[string]string{logVerbosity: envVarLogVerbosity(beLessVerbose)},
			uint(virtconfig.DefaultVirtLauncherLogVerbosity),
			ConsistOf(
				k8sv1.EnvVar{Name: "VIRT_LAUNCHER_LOG_VERBOSITY", Value: envVarLogVerbosity(beLessVerbose)},
				podNameEnvVar(),
			),
		),
		Entry(
			"override default cluster wide log verbosity via cluster-wide config",
			map[string]string{},
			map[string]string{},
			uint(beLessVerbose),
			ConsistOf(
				k8sv1.EnvVar{Name: "VIRT_LAUNCHER_LOG_VERBOSITY", Value: envVarLogVerbosity(beLessVerbose)},
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

func envVarLogVerbosity(verbosity int) string {
	return fmt.Sprintf("%d", verbosity)
}
