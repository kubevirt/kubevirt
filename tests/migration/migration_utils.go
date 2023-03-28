package migration

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8snetworkplumbingwgv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func ExpectMigrationSuccess(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	return ExpectMigrationSuccessWithOffset(2, virtClient, migration, timeout)
}

func ExpectMigrationSuccessWithOffset(offset int, virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	By("Waiting until the Migration Completes")
	EventuallyWithOffset(offset, func() error {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		ExpectWithOffset(offset+1, migration.Status.Phase).ToNot(Equal(v1.MigrationFailed), "migration should not fail")

		if migration.Status.Phase == v1.MigrationSucceeded {
			return nil
		}
		return fmt.Errorf("migration is in the phase: %s", migration.Status.Phase)

	}, timeout, 1*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("migration should succeed after %d s", timeout))
	return migration
}

func RunMigrationAndExpectCompletion(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	migration = RunMigration(virtClient, migration)

	return ExpectMigrationSuccess(virtClient, migration, timeout)
}

func RunMigration(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	By("Starting a Migration")

	migrationCreated, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return migrationCreated
}

func ConfirmMigrationDataIsStored(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance) {
	By("Retrieving the VMI and the migration object")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should have been able to retrive the VMI instance")
	migration, migerr := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
	Expect(migerr).ToNot(HaveOccurred(), "should have been able to retrive the migration")

	By("Verifying the stored migration state")
	Expect(migration.Status.MigrationState).ToNot(BeNil(), "should have been able to retrieve the migration `Status::MigrationState`")
	Expect(vmi.Status.MigrationState.StartTimestamp).To(Equal(migration.Status.MigrationState.StartTimestamp), "the VMI and the migration `Status::MigrationState::StartTimestamp` should be equal")
	Expect(vmi.Status.MigrationState.EndTimestamp).To(Equal(migration.Status.MigrationState.EndTimestamp), "the VMI and the migration `Status::MigrationState::EndTimestamp` should be equal")
	Expect(vmi.Status.MigrationState.Completed).To(Equal(migration.Status.MigrationState.Completed), "the VMI and migration completed state should be equal")
	Expect(vmi.Status.MigrationState.Failed).To(Equal(migration.Status.MigrationState.Failed), "the VMI nad migration failed status must be equal")
	Expect(vmi.Status.MigrationState.MigrationUID).To(Equal(migration.Status.MigrationState.MigrationUID), "the VMI migration UID and the migration object UID should match")
	return
}

func ConfirmVMIPostMigration(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstance {
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should have been able to retrive the VMI instance")

	By("Verifying the VMI's migration state")
	Expect(vmi.Status.MigrationState).ToNot(BeNil(), "should have been able to retrieve the VMIs `Status::MigrationState`")
	Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil(), "the VMIs `Status::MigrationState` should have a StartTimestamp")
	Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil(), "the VMIs `Status::MigrationState` should have a EndTimestamp")
	Expect(vmi.Status.MigrationState.TargetNodeDomainReadyTimestamp).ToNot(BeNil(), "the VMIs `Status::MigrationState` should have a TargetNodeDomainReadyTimestamp")
	Expect(vmi.Status.MigrationState.TargetNode).To(Equal(vmi.Status.NodeName), "the VMI should have migrated to the desired node")
	Expect(vmi.Status.MigrationState.TargetNode).NotTo(Equal(vmi.Status.MigrationState.SourceNode), "the VMI must have migrated to a different node from the one it originated from")
	Expect(vmi.Status.MigrationState.Completed).To(BeTrue(), "the VMI migration state must have completed")
	Expect(vmi.Status.MigrationState.Failed).To(BeFalse(), "the VMI migration status must not have failed")
	Expect(vmi.Status.MigrationState.TargetNodeAddress).NotTo(Equal(""), "the VMI `Status::MigrationState::TargetNodeAddress` must not be empty")
	Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(string(migration.UID)), "the VMI migration UID must be the expected one")

	By("Verifying the VMI's is in the running state")
	Expect(vmi.Status.Phase).To(Equal(v1.Running), "the VMI must be in `Running` state after the migration")

	return vmi
}

func SetOrClearDedicatedMigrationNetwork(nad string, set bool) *v1.KubeVirt {
	virtClient := kubevirt.Client()

	kv := util.GetCurrentKv(virtClient)

	// Saving the list of virt-handler pods prior to changing migration settings, see comment below.
	listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
	virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
	Expect(err).ToNot(HaveOccurred(), "Failed to list the virt-handler pods")

	if set {
		if kv.Spec.Configuration.MigrationConfiguration == nil {
			kv.Spec.Configuration.MigrationConfiguration = &v1.MigrationConfiguration{}
		}
		kv.Spec.Configuration.MigrationConfiguration.Network = &nad
	} else {
		if kv.Spec.Configuration.MigrationConfiguration != nil {
			kv.Spec.Configuration.MigrationConfiguration.Network = nil
		}
	}

	res := tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

	// By design, changing migration settings trigger a re-creation of the virt-handler pods, amongst other things.
	//   However, even if SetDedicatedMigrationNetwork() calls UpdateKubeVirtConfigValueAndWait(), VMIs can still get scheduled on outdated virt-handler pods.
	//   Waiting for all "old" virt-handlers to disappear ensures test VMIs will be created on updated virt-handler pods.
	Eventually(func() bool {
		for _, pod := range virtHandlerPods.Items {
			_, err = virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
			if err == nil {
				return false
			}
		}
		return true
	}, 180*time.Second, 10*time.Second).Should(BeTrue(), "Some virt-handler pods survived the migration settings change")

	// Ensure all virt-handlers are ready
	Eventually(func() bool {
		newVirtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
		if len(newVirtHandlerPods.Items) != len(virtHandlerPods.Items) {
			return false
		}
		Expect(err).ToNot(HaveOccurred(), "Failed to list the virt-handler pods")
		for _, pod := range newVirtHandlerPods.Items {
			// TODO implement list option to condition matcher
			if success, err := matcher.HaveConditionTrue(k8sv1.PodReady).Match(pod); !success {
				Expect(err).ToNot(HaveOccurred())
				return false
			}
		}
		return true
	}, 180*time.Second, 10*time.Second).Should(BeTrue(), "Some virt-handler pods never became ready")

	return res
}

func SetDedicatedMigrationNetwork(nad string) *v1.KubeVirt {
	return SetOrClearDedicatedMigrationNetwork(nad, true)
}

func ClearDedicatedMigrationNetwork() *v1.KubeVirt {
	return SetOrClearDedicatedMigrationNetwork("", false)
}

func GenerateMigrationCNINetworkAttachmentDefinition() *k8snetworkplumbingwgv1.NetworkAttachmentDefinition {
	nad := &k8snetworkplumbingwgv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "migration-cni",
			Namespace: flags.KubeVirtInstallNamespace,
		},
		Spec: k8snetworkplumbingwgv1.NetworkAttachmentDefinitionSpec{
			Config: `{
      "cniVersion": "0.3.1",
      "name": "migration-bridge",
      "type": "macvlan",
      "master": "` + flags.MigrationNetworkNIC + `",
      "mode": "bridge",
      "ipam": {
        "type": "whereabouts",
        "range": "172.21.42.0/24"
      }
}`,
		},
	}

	return nad
}

func EnsureNoMigrationMetadataInPersistentXML(vmi *v1.VirtualMachineInstance) {
	domXML := tests.RunCommandOnVmiPod(vmi, []string{"virsh", "dumpxml", "1"})
	decoder := xml.NewDecoder(bytes.NewReader([]byte(domXML)))

	var location = make([]string, 0)
	var found = false
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		Expect(err).ToNot(HaveOccurred(), "error getting token: %v\n", err)

		switch v := token.(type) {
		case xml.StartElement:
			location = append(location, v.Name.Local)

			if len(location) >= 4 &&
				location[0] == "domain" &&
				location[1] == "metadata" &&
				location[2] == "kubevirt" &&
				location[3] == "migration" {
				found = true
			}
			Expect(found).To(BeFalse(), "Unexpected KubeVirt migration metadata found in domain XML")
		case xml.EndElement:
			location = location[:len(location)-1]
		}

	}
}

func StopNodeLabeller(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	var err error
	var node *k8sv1.Node

	Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "stopping / resuming node-labeller is supported for serial tests only")

	By(fmt.Sprintf("Patching node to %s include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	key, value := v1.LabellerSkipNodeAnnotation, "true"
	libnode.AddAnnotationToNode(nodeName, key, value)

	By(fmt.Sprintf("Expecting node %s to include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func() bool {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		value, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
		return exists && value == "true"
	}, 30*time.Second, time.Second).Should(BeTrue(), fmt.Sprintf("node %s is expected to have annotation %s", nodeName, v1.LabellerSkipNodeAnnotation))

	return node
}

func ResumeNodeLabeller(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	var err error
	var node *k8sv1.Node

	node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	if _, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]; !isNodeLabellerStopped {
		// Nothing left to do
		return node
	}

	By(fmt.Sprintf("Patching node to %s not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	libnode.RemoveAnnotationFromNode(nodeName, v1.LabellerSkipNodeAnnotation)

	// In order to make sure node-labeller has updated the node, the host-model label (which node-labeller
	// makes sure always resides on any node) will be removed. After node-labeller is enabled again, the
	// host model label would be expected to show up again on the node.
	By(fmt.Sprintf("Removing host model label %s from node %s (so we can later expect it to return)", v1.HostModelCPULabel, nodeName))
	for _, label := range node.Labels {
		if strings.HasPrefix(label, v1.HostModelCPULabel) {
			libnode.RemoveLabelFromNode(nodeName, label)
		}
	}

	WakeNodeLabellerUp(virtClient)

	By(fmt.Sprintf("Expecting node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func() error {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
		if exists {
			return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
		}

		foundHostModelLabel := false
		for labelKey := range node.Labels {
			if strings.HasPrefix(labelKey, v1.HostModelCPULabel) {
				foundHostModelLabel = true
				break
			}
		}
		if !foundHostModelLabel {
			return fmt.Errorf("node %s is expected to have a label with %s prefix. this means node-labeller is not enabled for the node", nodeName, v1.HostModelCPULabel)
		}

		return nil
	}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

	return node
}

func WakeNodeLabellerUp(virtClient kubecli.KubevirtClient) {
	const fakeModel = "fake-model-1423"

	By("Updating Kubevirt CR to wake node-labeller up")
	kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
	if kvConfig.ObsoleteCPUModels == nil {
		kvConfig.ObsoleteCPUModels = make(map[string]bool)
	}
	kvConfig.ObsoleteCPUModels[fakeModel] = true
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
	delete(kvConfig.ObsoleteCPUModels, fakeModel)
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
}

func GetValidSourceNodeAndTargetNodeForHostModelMigration(virtCli kubecli.KubevirtClient) (sourceNode *k8sv1.Node, targetNode *k8sv1.Node, err error) {
	getNodeHostRequiredFeatures := func(node *k8sv1.Node) (features []string) {
		for key, _ := range node.Labels {
			if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
				features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
			}
		}
		return features
	}
	areFeaturesSupportedOnNode := func(node *k8sv1.Node, features []string) bool {
		isFeatureSupported := func(feature string) bool {
			for key, _ := range node.Labels {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) && strings.Contains(key, feature) {
					return true
				}
			}
			return false
		}
		for _, feature := range features {
			if !isFeatureSupported(feature) {
				return false
			}
		}

		return true
	}

	var sourceHostCpuModel string

	nodes := libnode.GetAllSchedulableNodes(virtCli)
	Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
	for _, potentialSourceNode := range nodes.Items {
		for _, potentialTargetNode := range nodes.Items {
			if potentialSourceNode.Name == potentialTargetNode.Name {
				continue
			}

			sourceHostCpuModel = tests.GetNodeHostModel(&potentialSourceNode)
			if sourceHostCpuModel == "" {
				continue
			}
			supportedInTarget := false
			for key, _ := range potentialTargetNode.Labels {
				if strings.HasPrefix(key, v1.SupportedHostModelMigrationCPU) && strings.Contains(key, sourceHostCpuModel) {
					supportedInTarget = true
					break
				}
			}

			if supportedInTarget == false {
				continue
			}
			sourceNodeHostModelRequiredFeatures := getNodeHostRequiredFeatures(&potentialSourceNode)
			if areFeaturesSupportedOnNode(&potentialTargetNode, sourceNodeHostModelRequiredFeatures) == false {
				continue
			}
			return &potentialSourceNode, &potentialTargetNode, nil
		}
	}
	return nil, nil, fmt.Errorf("couldn't find valid nodes for host-model migration")
}

func AffinityToMigrateFromSourceToTargetAndBack(sourceNode *k8sv1.Node, targetNode *k8sv1.Node) (nodefiinity *k8sv1.NodeAffinity, err error) {
	if sourceNode == nil || targetNode == nil {
		return nil, fmt.Errorf("couldn't find valid nodes for host-model migration")
	}
	return &k8sv1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
			NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
				{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: k8sv1.NodeSelectorOpIn,
							Values:   []string{sourceNode.Name, targetNode.Name},
						},
					},
				},
			},
		},
		PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{
			{
				Preference: k8sv1.NodeSelectorTerm{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: k8sv1.NodeSelectorOpIn,
							Values:   []string{sourceNode.Name},
						},
					},
				},
				Weight: 1,
			},
		},
	}, nil
}
