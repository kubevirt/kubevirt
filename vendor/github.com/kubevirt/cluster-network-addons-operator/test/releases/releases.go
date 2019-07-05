package releases

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"

	"github.com/blang/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	. "github.com/kubevirt/cluster-network-addons-operator/test/operations"
)

type Release struct {
	// Release version
	Version string
	// Containers and their images for given release
	Containers []opv1alpha1.Container
	// SupportedSpec for given release should be upgradable
	SupportedSpec opv1alpha1.NetworkAddonsConfigSpec
	// Manifest that can be used to install the operator in given release
	Manifests []string
}

// Releases are populated by respective release modules using init()
var releases = []Release{}
var releasesSorted = false

// Returns list of releases sorted from oldest to newest
func Releases() []Release {
	if releasesSorted {
		return releases
	}

	sort.Slice(releases, func(a, b int) bool {
		releaseAVersion, err := semver.Make(releases[a].Version)
		if err != nil {
			panic(err)
		}
		releaseBVersion, err := semver.Make(releases[b].Version)
		if err != nil {
			panic(err)
		}
		return releaseAVersion.LT(releaseBVersion)
	})
	releasesSorted = true

	return releases
}

// Iterates registered releases and returns the latest (master) based on semver
func LatestRelease() Release {
	r := Releases()
	return r[len(r)-1]
}

// Installs given release (RBAC and Deployment)
func InstallRelease(release Release) {
	By(fmt.Sprintf("Installing release %s", release.Version))
	for _, manifestName := range release.Manifests {
		out, err := kubectl("apply", "-f", "_out/cluster-network-addons/"+release.Version+"/"+manifestName)
		Expect(err).NotTo(HaveOccurred(), out)
	}
}

// Removes given release from cluster
func UninstallRelease(release Release) {
	By(fmt.Sprintf("Uninstalling release %s", release.Version))
	for _, manifestName := range release.Manifests {
		out, err := kubectl("delete", "--ignore-not-found", "-f", "_out/cluster-network-addons/"+release.Version+"/"+manifestName)
		Expect(err).NotTo(HaveOccurred(), out)
	}
}

// Make sure that container images currently used (reported in NetworkAddonsConfig)
// are matching images expected for given release
func CheckReleaseUsesExpectedContainerImages(release Release) {
	By(fmt.Sprintf("Checking that all deployed images match release %s", release.Version))

	expectedContainers := sortContainers(release.Containers)

	config := GetConfig()
	deployedContainers := sortContainers(config.Status.Containers)

	Expect(deployedContainers).To(Equal(expectedContainers))
}

func sortContainers(containers []opv1alpha1.Container) []opv1alpha1.Container {
	sort.Slice(containers, func(a, b int) bool {
		return (sort.StringsAreSorted([]string{containers[a].Namespace, containers[b].Namespace}) &&
			sort.StringsAreSorted([]string{containers[a].ParentKind, containers[b].ParentKind}) &&
			sort.StringsAreSorted([]string{containers[a].ParentName, containers[b].ParentName}) &&
			sort.StringsAreSorted([]string{containers[a].Name, containers[b].Name}))
	})
	return containers
}

func kubectl(command ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("./cluster/kubectl.sh", command...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.String() + stderr.String(), err
}
