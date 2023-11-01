package checks

import (
	"context"
	"fmt"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/pkg/util/cluster"

	"github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/ginkgo/v2"

	kubev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"
)

const diskRhel = "disk-rhel"

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

func SkipTestIfNoFeatureGate(featureGate string) {
	if !HasFeature(featureGate) {
		ginkgo.Skip(fmt.Sprintf("the %v feature gate is not enabled.", featureGate))
	}
}

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

func SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(nodeCount int) {
	SkipTestIfNotEnoughNodesWithCPUManager(nodeCount)
	SkipTestIfNotEnoughNodesWith2MiHugepages(nodeCount)
}

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

func SkipTestIfNotSEVCapable() {
	virtClient := kubevirt.Client()
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsSEVCapable(&node, kubev1.SEVLabel) {
			return
		}
	}
	ginkgo.Skip("no node capable of running SEV workloads detected", 1)
}

func SkipTestIfNotSEVESCapable() {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := libnode.GetAllSchedulableNodes(virtClient)

	for _, node := range nodes.Items {
		if IsSEVCapable(&node, kubev1.SEVESLabel) {
			return
		}
	}
	ginkgo.Skip("no node capable of running SEV-ES workloads detected", 1)
}

func SkipIfMissingRequiredImage(virtClient kubecli.KubevirtClient, imageName string) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), imageName, v1.GetOptions{})
	if err != nil || windowsPv.Status.Phase == v12.VolumePending || windowsPv.Status.Phase == v12.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip tests that requires PV %s", imageName))
	} else if windowsPv.Status.Phase == v12.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), windowsPv, v1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func SkipIfNoRhelImage(virtClient kubecli.KubevirtClient) {
	rhelPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), diskRhel, v1.GetOptions{})
	if err != nil || rhelPv.Status.Phase == v12.VolumePending || rhelPv.Status.Phase == v12.VolumeFailed {
		ginkgo.Skip(fmt.Sprintf("Skip RHEL tests that requires PVC %s", diskRhel))
	} else if rhelPv.Status.Phase == v12.VolumeReleased {
		rhelPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), rhelPv, v1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func SkipIfUseFlannel(virtClient kubecli.KubevirtClient) {
	labelSelector := "app=flannel"
	flannelpod, err := virtClient.CoreV1().Pods(v1.NamespaceSystem).List(context.Background(), v1.ListOptions{LabelSelector: labelSelector})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if len(flannelpod.Items) > 0 {
		ginkgo.Skip("Skip networkpolicy test for flannel network")
	}
}

func SkipIfPrometheusRuleIsNotEnabled(virtClient kubecli.KubevirtClient) {
	ext, err := clientset.NewForConfig(virtClient.Config())
	util.PanicOnError(err)

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "prometheusrules.monitoring.coreos.com", v1.GetOptions{})
	if errors.IsNotFound(err) {
		ginkgo.Skip("Skip monitoring tests when PrometheusRule CRD is not available in the cluster")
	} else if err != nil {
		util.PanicOnError(err)
	}
}

func SkipIfSingleReplica(virtClient kubecli.KubevirtClient) {
	kv := util.GetCurrentKv(virtClient)
	if kv.Spec.Infra != nil && kv.Spec.Infra.Replicas != nil && *(kv.Spec.Infra.Replicas) == 1 {
		ginkgo.Skip("Skip multi-replica test on single-replica deployments")
	}
}

func SkipIfMultiReplica(virtClient kubecli.KubevirtClient) {
	kv := util.GetCurrentKv(virtClient)
	if kv.Spec.Infra == nil || kv.Spec.Infra.Replicas == nil || *(kv.Spec.Infra.Replicas) > 1 {
		ginkgo.Skip("Skip single-replica test on multi-replica deployments")
	}
}

func SkipIfOpenShift(message string) {
	if IsOpenShift() {
		ginkgo.Skip("Openshift detected: " + message)
	}
}

func SkipIfOpenShift4(message string) {
	virtClient := kubevirt.Client()

	if t, err := cluster.IsOnOpenShift(virtClient); err != nil {
		util.PanicOnError(err)
	} else if t && cluster.GetOpenShiftMajorVersion(virtClient) == cluster.OpenShift4Major {
		ginkgo.Skip(message)
	}
}

func SkipIfMigrationIsNotPossible() {
	if !HasAtLeastTwoNodes() {
		ginkgo.Skip("Migration tests require at least 2 nodes")
	}
}

func SkipIfARM64(arch string, message string) {
	if IsARM64(arch) {
		ginkgo.Skip("Skip test on arm64: " + message)
	}
}

func SkipIfRunningOnKindInfra(message string) {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip test on kind infra: " + message)
	}
}
