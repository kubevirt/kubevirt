package checks

import (
	"fmt"
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/util"
)

func SkipIfNoWindowsImage(virtClient kubecli.KubevirtClient) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(flags.DiskWindows, v1.GetOptions{})
	if err != nil || windowsPv.Status.Phase == v12.VolumePending || windowsPv.Status.Phase == v12.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip Windows tests that requires PVC %s", flags.DiskWindows))
	} else if windowsPv.Status.Phase == v12.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(windowsPv)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func SkipIfNoRhelImage(virtClient kubecli.KubevirtClient) {
	rhelPv, err := virtClient.CoreV1().PersistentVolumes().Get(flags.DiskRhel, v1.GetOptions{})
	if err != nil || rhelPv.Status.Phase == v12.VolumePending || rhelPv.Status.Phase == v12.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip RHEL tests that requires PVC %s", flags.DiskRhel))
	} else if rhelPv.Status.Phase == v12.VolumeReleased {
		rhelPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(rhelPv)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func SkipIfUseFlannel(virtClient kubecli.KubevirtClient) {
	labelSelector := "app=flannel"
	flannelpod, err := virtClient.CoreV1().Pods(v1.NamespaceSystem).List(v1.ListOptions{LabelSelector: labelSelector})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if len(flannelpod.Items) > 0 {
		ginkgo.Skip("Skip networkpolicy test for flannel network")
	}
}

func SkipIfNoCmd(cmdName string) {
	var cmdPath string
	switch strings.ToLower(cmdName) {
	case "oc":
		cmdPath = flags.KubeVirtOcPath
	case "kubectl":
		cmdPath = flags.KubeVirtKubectlPath
	case "virtctl":
		cmdPath = flags.KubeVirtVirtctlPath
	case "gocli":
		cmdPath = flags.KubeVirtGoCliPath
	}
	if cmdPath == "" {
		ginkgo.Skip(fmt.Sprintf("Skip test that requires %s binary", cmdName))
	}
}

// SkipIfVersionBelow will skip tests if it runs on an environment with k8s version below specified
func SkipIfVersionBelow(message string, expectedVersion string) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if curVersion < expectedVersion {
		ginkgo.Skip(message)
	}
}

func SkipIfVersionAboveOrEqual(message string, expectedVersion string) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if curVersion >= expectedVersion {
		ginkgo.Skip(message)
	}
}

func SkipIfOpenShift(message string) {
	if IsOpenShift() {
		ginkgo.Skip("Openshift detected: " + message)
	}
}

func SkipIfOpenShiftAndBelowOrEqualVersion(message string, version string) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// version is above
	if curVersion > version {
		return
	}

	if IsOpenShift() {
		ginkgo.Skip(message)
	}
}

func SkipIfOpenShift4(message string) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	if t, err := cluster.IsOnOpenShift(virtClient); err != nil {
		util.PanicOnError(err)
	} else if t && cluster.GetOpenShiftMajorVersion(virtClient) == cluster.OpenShift4Major {
		ginkgo.Skip(message)
	}
}

func SkipTestIfNoCPUManager() {
	if !HasFeature(virtconfig.CPUManager) {
		ginkgo.Skip("the CPUManager feature gate is not enabled.")
	}

	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsCPUManagerPresent(&node) {
			return
		}
	}
	if *DiscoveredClusterProfile.MinimumNodesWithCPUManager > 0 {
		ginkgo.Fail("No nodes with CPU manager present, although initially discovered", 1)
	}
	ginkgo.Skip("no node with CPUManager detected", 1)
}

func SkipPVCTestIfRunnigOnKindInfra() {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip PVC tests till PR https://github.com/kubevirt/kubevirt/pull/3171 is merged")
	}
}

func SkipNFSTestIfRunnigOnKindInfra() {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip NFS tests till issue https://github.com/kubevirt/kubevirt/issues/3322 is fixed")
	}
}

func SkipSELinuxTestIfRunnigOnKindInfra() {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip SELinux tests till issue https://github.com/kubevirt/kubevirt/issues/3780 is fixed")
	}
}

func SkipMigrationFailTestIfRunningOnKindInfraIPv6() {
	if IsRunningOnKindInfraIPv6() {
		ginkgo.Skip("Skip Migration fail test till issue https://github.com/kubevirt/kubevirt/issues/4086 is fixed")
	}
}

func SkipDmidecodeTestIfRunningOnKindInfraIPv6() {
	if IsRunningOnKindInfraIPv6() {
		ginkgo.Skip("Skip dmidecode tests till issue https://github.com/kubevirt/kubevirt/issues/3901 is fixed")
	}
}
