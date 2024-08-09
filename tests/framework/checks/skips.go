package checks

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/cluster"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"
)

const diskRhel = "disk-rhel"

// Deprecated: SkipTestIfNoCPUManager should be converted to check & fail
func SkipTestIfNoCPUManager() {
	if !HasFeature(virtconfig.CPUManager) {
		ginkgo.Skip("the CPUManager feature gate is not enabled.")
	}

	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsCPUManagerPresent(&node) {
			return
		}
	}
	ginkgo.Skip("no node with CPUManager detected", 1)
}

// Deprecated: SkipTestIfNoFeatureGate should be converted to check & fail
func SkipTestIfNoFeatureGate(featureGate string) {
	if !HasFeature(featureGate) {
		ginkgo.Skip(fmt.Sprintf("the %v feature gate is not enabled.", featureGate))
	}
}

// Deprecated: SkipTestIfNotEnoughNodesWithCPUManager should be converted to check & fail
func SkipTestIfNotEnoughNodesWithCPUManager(nodeCount int) {
	if !HasFeature(virtconfig.CPUManager) {
		ginkgo.Skip("the CPUManager feature gate is not enabled.")
	}

	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	found := 0
	for _, node := range nodes.Items {
		if IsCPUManagerPresent(&node) {
			found++
		}
	}

	if found < nodeCount {
		msg := fmt.Sprintf(
			"not enough node with CPUManager detected: expected %v nodes, but got %v",
			nodeCount,
			found,
		)
		ginkgo.Skip(msg, 1)
	}
}

// Deprecated: SkipTestIfNotEnoughNodesWith2MiHugepages should be converted to check & fail
func SkipTestIfNotEnoughNodesWith2MiHugepages(nodeCount int) {
	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	found := 0
	for _, node := range nodes.Items {
		if Has2MiHugepages(&node) {
			found++
		}
	}

	if found < nodeCount {
		msg := fmt.Sprintf(
			"not enough node with 2Mi hugepages detected: expected %v nodes, but got %v",
			nodeCount,
			found,
		)
		ginkgo.Skip(msg, 1)
	}
}

// Deprecated: SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages should be converted to check & fail
func SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(nodeCount int) {
	SkipTestIfNotEnoughNodesWithCPUManager(nodeCount)
	SkipTestIfNotEnoughNodesWith2MiHugepages(nodeCount)
}

// Deprecated: SkipTestIfNotRealtimeCapable should be converted to check & fail
func SkipTestIfNotRealtimeCapable() {

	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsRealtimeCapable(&node) && IsCPUManagerPresent(&node) && Has2MiHugepages(&node) {
			return
		}
	}
	ginkgo.Skip("no node capable of running realtime workloads detected", 1)

}

// Deprecated: SkipTestIfNotSEVCapable should be converted to check & fail
func SkipTestIfNotSEVCapable() {
	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsSEVCapable(&node, v1.SEVLabel) {
			return
		}
	}
	ginkgo.Skip("no node capable of running SEV workloads detected", 1)
}

// Deprecated: SkipTestIfNotSEVESCapable should be converted to check & fail
func SkipTestIfNotSEVESCapable() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsSEVCapable(&node, v1.SEVESLabel) {
			return
		}
	}
	ginkgo.Skip("no node capable of running SEV-ES workloads detected", 1)
}

// Deprecated: SkipIfMissingRequiredImage should be converted to check & fail
func SkipIfMissingRequiredImage(virtClient kubecli.KubevirtClient, imageName string) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), imageName, metav1.GetOptions{})
	if err != nil || windowsPv.Status.Phase == k8sv1.VolumePending || windowsPv.Status.Phase == k8sv1.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip tests that requires PV %s", imageName))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), windowsPv, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

// Deprecated: SkipIfNoRhelImage should be converted to check & fail
func SkipIfNoRhelImage(virtClient kubecli.KubevirtClient) {
	rhelPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), diskRhel, metav1.GetOptions{})
	if err != nil || rhelPv.Status.Phase == k8sv1.VolumePending || rhelPv.Status.Phase == k8sv1.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip RHEL tests that requires PVC %s", diskRhel))
	} else if rhelPv.Status.Phase == k8sv1.VolumeReleased {
		rhelPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), rhelPv, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

// Deprecated: SkipIfUseFlannel should be converted to check & fail
func SkipIfUseFlannel(virtClient kubecli.KubevirtClient) {
	labelSelector := "app=flannel"
	flannelpod, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if len(flannelpod.Items) > 0 {
		ginkgo.Skip("Skip networkpolicy test for flannel network")
	}
}

// Deprecated: SkipIfPrometheusRuleIsNotEnabled should be converted to check & fail
func SkipIfPrometheusRuleIsNotEnabled(virtClient kubecli.KubevirtClient) {
	ext, err := clientset.NewForConfig(virtClient.Config())
	util.PanicOnError(err)

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "prometheusrules.monitoring.coreos.com", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ginkgo.Skip("Skip monitoring tests when PrometheusRule CRD is not available in the cluster")
	} else if err != nil {
		util.PanicOnError(err)
	}
}

// Deprecated: SkipIfSingleReplica should be converted to check & fail
func SkipIfSingleReplica(virtClient kubecli.KubevirtClient) {
	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv.Spec.Infra != nil && kv.Spec.Infra.Replicas != nil && *(kv.Spec.Infra.Replicas) == 1 {
		ginkgo.Skip("Skip multi-replica test on single-replica deployments")
	}
}

// Deprecated: SkipIfMultiReplica should be converted to check & fail
func SkipIfMultiReplica(virtClient kubecli.KubevirtClient) {
	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv.Spec.Infra == nil || kv.Spec.Infra.Replicas == nil || *(kv.Spec.Infra.Replicas) > 1 {
		ginkgo.Skip("Skip single-replica test on multi-replica deployments")
	}
}

// Deprecated: SkipIfOpenShift should be converted to check & fail
func SkipIfOpenShift(message string) {
	if IsOpenShift() {
		ginkgo.Skip("Openshift detected: " + message)
	}
}

// Deprecated: SkipIfOpenShift4 should be converted to check & fail
func SkipIfOpenShift4(message string) {
	virtClient := kubevirt.Client()

	if t, err := cluster.IsOnOpenShift(virtClient); err != nil {
		util.PanicOnError(err)
	} else if t && cluster.GetOpenShiftMajorVersion(virtClient) == cluster.OpenShift4Major {
		ginkgo.Skip(message)
	}
}

// Deprecated: SkipIfMigrationIsNotPossible should be converted to check & fail
func SkipIfMigrationIsNotPossible() {
	if !HasAtLeastTwoNodes() {
		ginkgo.Skip("Migration tests require at least 2 nodes")
	}
}

// Deprecated: SkipIfARM64 should be converted to check & fail
func SkipIfARM64(arch string, message string) {
	if IsARM64(arch) {
		ginkgo.Skip("Skip test on arm64: " + message)
	}
}

// Deprecated: SkipIfS390X should be converted to check & fail
func SkipIfS390X(arch string, message string) {
	if IsS390X(arch) {
		ginkgo.Skip("Skip test on s390x: " + message)
	}
}

// Deprecated: SkipIfRunningOnKindInfra should be converted to check & fail
func SkipIfRunningOnKindInfra(message string) {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip test on kind infra: " + message)
	}
}
