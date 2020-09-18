package checks

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/util"
)

var DiscoveredClusterProfile = ClusterProfile{}

// ClusterProfile holds cluster-configuration expectations. If tests are skipped although the expectations should be met, the test suite is
// supposed to fail the test to highlight the mismatch.
type ClusterProfile struct {
	// openShiftVersion specifies the openshift version of the cluster.
	// All tests which which require <= this version are supposed to run.
	OpenShiftMajorVersion *int `json:"openShiftMajorVersion,omitempty"`
	// kubernetesVersion specifies the kubernetes version of the test cluster
	// All tests which which require <= this version are supposed to run.
	KubernetesVersion *string `json:"kubernetesVersion,omitempty"`
	// dualNetworkStack indicates if the cluster provides IPv4 and IPv6 addresses
	// All tests which require a dual-stack setup are supposed to run.
	DualNetworkStack *bool `json:"dualNetworkStack,omitempty"`
	// minimumSchedulableNodes is the minimum of nodes which are capable of scheduling and running VMs
	// All tests which require <= this amount of nodes are supposed to be run
	MinimumSchedulableNodes *int `json:"minimumSchedulableNodes,omitempty"`
	// nodesWithCPUManager contains the number of nodes with the CPU manager preset
	MinimumNodesWithCPUManager *int `json:"minimumNodesWithCPUManager,omitempty"`
	// isKind tells if the tests are running on kind
	IsKind *bool `json:"isKind,omitempty"`
}

func IntrospectCluster() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtClient).Items
	nodeCount := len(nodes)
	DiscoveredClusterProfile.MinimumSchedulableNodes = &nodeCount
	if IsOpenShift() {
		majorVersion := cluster.GetOpenShiftMajorVersion(virtClient)
		DiscoveredClusterProfile.OpenShiftMajorVersion = &majorVersion
	}
	kubernetesVersion, err := cluster.GetKubernetesVersion(virtClient)
	if err != nil {
		util.PanicOnError(err)
	}
	DiscoveredClusterProfile.KubernetesVersion = &kubernetesVersion

	dualStack, err := libnet.IsClusterDualStack(virtClient)
	if err != nil {
		util.PanicOnError(err)
	}
	DiscoveredClusterProfile.DualNetworkStack = &dualStack

	withCPUManager := 0
	for _, node := range nodes {
		if IsCPUManagerPresent(&node) {
			withCPUManager++
		}
	}
	DiscoveredClusterProfile.MinimumNodesWithCPUManager = &withCPUManager

	isKind := IsRunningOnKindInfra() || IsRunningOnKindInfraIPv6()
	DiscoveredClusterProfile.IsKind = &isKind

}

type DiscoveryError struct {
	errors []error
}

func (d *DiscoveryError) Error() string {
	return fmt.Sprintf("%v", d.errors)
}

func VerifyClusterExpectations() error {
	expected := &ClusterProfile{}

	// XXX this flags needs to go
	// it makes dual stack the default assumption if no profile file is provided
	if len(flags.ClusterProfilePath) == 0 {
		deprecatedIsDualStack := !flags.SkipDualStackTests
		expected.DualNetworkStack = &deprecatedIsDualStack
	}

	if len(flags.ClusterProfilePath) > 0 {
		reader, err := os.Open(flags.ClusterProfilePath)
		if err != nil {
			log.DefaultLogger().Reason(err).Critical("Could not find the provided cluster profile file.")
		}
		if err := yaml.NewYAMLOrJSONDecoder(reader, 1024).Decode(expected); err != nil {
			log.DefaultLogger().Reason(err).Critical("Could not decode the provided cluster profile file.")
		}
	}

	printProfile("Discovered cluster:", &DiscoveredClusterProfile)
	printProfile("Expected cluster:", expected)

	errors := &DiscoveryError{}
	if expected.OpenShiftMajorVersion != nil {
		if *expected.OpenShiftMajorVersion != *DiscoveredClusterProfile.OpenShiftMajorVersion {
			errors.errors = append(errors.errors, fmt.Errorf("OpenShiftMajorVersion was answered with %v, expected %v", *DiscoveredClusterProfile.OpenShiftMajorVersion, *expected.OpenShiftMajorVersion))
		}
	}
	if expected.KubernetesVersion != nil {
		if *expected.KubernetesVersion != *DiscoveredClusterProfile.KubernetesVersion {
			errors.errors = append(errors.errors, fmt.Errorf("KubernetesVersion was answered with %v, expected %v", *DiscoveredClusterProfile.KubernetesVersion, *expected.KubernetesVersion))
		}
	}
	if expected.DualNetworkStack != nil {
		if *expected.DualNetworkStack != *DiscoveredClusterProfile.DualNetworkStack {
			errors.errors = append(errors.errors, fmt.Errorf("IsDualStack was answered with %v, expected %v", *DiscoveredClusterProfile.DualNetworkStack, *expected.DualNetworkStack))
		}
	}
	if expected.MinimumSchedulableNodes != nil {
		if *expected.MinimumSchedulableNodes >= *DiscoveredClusterProfile.MinimumSchedulableNodes {
			errors.errors = append(errors.errors, fmt.Errorf("Got a schedulable node count of %v, expected at least %v", *DiscoveredClusterProfile.MinimumSchedulableNodes, *expected.MinimumSchedulableNodes))
		}
	}
	if expected.MinimumNodesWithCPUManager != nil {
		if *expected.MinimumNodesWithCPUManager >= *DiscoveredClusterProfile.MinimumNodesWithCPUManager {
			errors.errors = append(errors.errors, fmt.Errorf("Got a schedulable node count with CPU manager of %v, expected at least %v", *DiscoveredClusterProfile.MinimumNodesWithCPUManager, *expected.MinimumNodesWithCPUManager))
		}
	}
	if expected.IsKind != nil {
		if *expected.IsKind != *DiscoveredClusterProfile.IsKind {
			errors.errors = append(errors.errors, fmt.Errorf("IsKind was answered with %v, expected %v", *DiscoveredClusterProfile.IsKind, *expected.IsKind))
		}
	}

	if len(errors.errors) > 0 {
		return errors
	}
	return nil
}

func printProfile(title string, profile *ClusterProfile) {
	fmt.Println(title)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(profile)
	util.PanicOnError(err)
}
