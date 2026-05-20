package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8sversion "k8s.io/apimachinery/pkg/version"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
)

func IsCPUManagerPresent(node *k8sv1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	nodeHaveCpuManagerLabel := false

	for label, val := range node.Labels {
		if label == v1.CPUManager && val == "true" {
			nodeHaveCpuManagerLabel = true
			break
		}
	}
	return nodeHaveCpuManagerLabel
}

func IsRealtimeCapable(node *k8sv1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	for label := range node.Labels {
		if label == v1.RealtimeLabel {
			return true
		}
	}
	return false
}

func Has2MiHugepages(node *k8sv1.Node) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	_, exists := node.Status.Capacity[k8sv1.ResourceHugePagesPrefix+"2Mi"]
	return exists
}

func HasFeature(feature string) bool {
	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
	return featuregate.IsEnabled(feature, kv.Spec.Configuration.DeveloperConfiguration)
}

func GetKubernetesVersion() (string, error) {
	var info k8sversion.Info
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return "", err
	}
	response, err := virtClient.RestClient().Get().AbsPath("/version").DoRaw(context.Background())
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(response, &info); err != nil {
		return "", err
	}
	curVersion := strings.Split(info.GitVersion, "+")[0]
	curVersion = strings.Trim(curVersion, "v")
	return curVersion, nil
}

func IsSEVCapable(node *k8sv1.Node, sevLabel string) bool {
	gomega.Expect(node).ToNot(gomega.BeNil())
	for label := range node.Labels {
		if label == sevLabel {
			return true
		}
	}
	return false
}

func IsARM64(arch string) bool {
	return arch == "arm64"
}

func IsS390X(arch string) bool {
	return arch == "s390x"
}

func HasAtLeastTwoNodes() bool {
	var nodes *k8sv1.NodeList
	virtClient := kubevirt.Client()

	gomega.Eventually(func() []k8sv1.Node {
		nodes = libnode.GetAllSchedulableNodes(virtClient)
		return nodes.Items
	}, 60*time.Second, time.Second).ShouldNot(gomega.BeEmpty(), "There should be some compute node")

	return len(nodes.Items) >= 2
}

func IsOpenShift() bool {
	virtClient := kubevirt.Client()

	isOpenShift, err := cluster.IsOnOpenShift(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can not determine cluster type %v\n", err)
		panic(err)
	}

	return isOpenShift
}

func IsRunningOnKindInfra() bool {
	provider := os.Getenv("KUBEVIRT_PROVIDER")
	return strings.HasPrefix(provider, "kind")
}
