package libmigration

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	k8snetworkplumbingwgv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
)

const MigrationWaitTime = 240

func New(vmiName string, namespace string) *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-migration-",
			Namespace:    namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
}

func NewSource(vmiName, namespace, migrationID, connectURL string) *v1.VirtualMachineInstanceMigration {
	migration := New(vmiName, namespace)
	migration.Spec.SendTo = &v1.VirtualMachineInstanceMigrationSource{
		MigrationID: migrationID,
		ConnectURL:  connectURL,
	}
	return migration
}

func NewTarget(vmiName, namespace, migrationID string) *v1.VirtualMachineInstanceMigration {
	migration := New(vmiName, namespace)
	migration.Spec.Receive = &v1.VirtualMachineInstanceMigrationTarget{
		MigrationID: migrationID,
	}
	return migration
}

func ExpectMigrationToSucceed(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	return ExpectMigrationToSucceedWithOffset(2, virtClient, migration, timeout)
}

func ExpectMigrationToSucceedWithDefaultTimeout(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	return ExpectMigrationToSucceed(virtClient, migration, MigrationWaitTime)
}

func ExpectMigrationToSucceedWithOffset(offset int, virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	By("Waiting until the Migration Completes")
	EventuallyWithOffset(offset, func() error {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
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

func RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient kubecli.KubevirtClient, sourceMigration, targetMigration *v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, *v1.VirtualMachineInstanceMigration) {
	// increase timeout on decentralized migration.
	return RunDecentralizedMigrationAndExpectToComplete(virtClient, sourceMigration, targetMigration, MigrationWaitTime*2)
}

func CheckSynchronizationAddressPopulated(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) {
	kv := libkubevirt.GetCurrentKv(virtClient)
	Expect(kv.Status.SynchronizationAddresses).ToNot(BeNil())
	synchronizationAddress := kv.Status.SynchronizationAddresses[0]

	Eventually(func() string {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if migration.Status.SynchronizationAddresses == nil {
			return ""
		}
		return migration.Status.SynchronizationAddresses[0]
	}).WithTimeout(time.Second * 20).WithPolling(500 * time.Millisecond).Should(Equal(synchronizationAddress))
}

func RunDecentralizedMigrationAndExpectToComplete(virtClient kubecli.KubevirtClient, sourceMigration, targetMigration *v1.VirtualMachineInstanceMigration, timeout int) (*v1.VirtualMachineInstanceMigration, *v1.VirtualMachineInstanceMigration) {
	sourceMigration = RunMigration(virtClient, sourceMigration)
	CheckSynchronizationAddressPopulated(virtClient, sourceMigration)

	targetMigration = RunMigration(virtClient, targetMigration)
	CheckSynchronizationAddressPopulated(virtClient, targetMigration)
	sourceMigration = ExpectMigrationToSucceed(virtClient, sourceMigration, timeout)
	targetMigration = ExpectMigrationToSucceed(virtClient, targetMigration, timeout)
	return sourceMigration, targetMigration
}

func RunMigrationAndExpectToComplete(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	migration = RunMigration(virtClient, migration)

	return ExpectMigrationToSucceed(virtClient, migration, timeout)
}

func RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	return RunMigrationAndExpectToComplete(virtClient, migration, MigrationWaitTime)
}

func RunMigration(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	By("Starting a Migration")

	migrationCreated, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return migrationCreated
}

func ConfirmMigrationDataIsStored(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance) {
	By("Retrieving the VMI and the migration object")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should have been able to retrieve the VMI instance")
	migration, migerr := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
	Expect(migerr).ToNot(HaveOccurred(), "should have been able to retrieve the migration")

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
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should have been able to retrieve the VMI instance")

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
	Eventually(func() v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "should have been able to retrieve the VMI instance")
		return vmi.Status.Phase
	}, 60*time.Second, 1*time.Second).Should(Equal(v1.Running), fmt.Sprintf("the VMI %s/%s must be in `Running` state after the migration", vmi.Namespace, vmi.Name))

	return vmi
}

func setOrClearDedicatedMigrationNetwork(nad string, set bool) *v1.KubeVirt {
	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)

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

	res := config.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

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
	return setOrClearDedicatedMigrationNetwork(nad, true)
}

func ClearDedicatedMigrationNetwork() *v1.KubeVirt {
	return setOrClearDedicatedMigrationNetwork("", false)
}

func GenerateMigrationCNINetworkAttachmentDefinition() *k8snetworkplumbingwgv1.NetworkAttachmentDefinition {
	config := map[string]interface{}{
		"cniVersion": "0.3.1",
		"name":       "migration-bridge",
		"type":       "macvlan",
		"master":     flags.MigrationNetworkNIC,
		"mode":       "bridge",
		"ipam": map[string]string{
			"type":  "whereabouts",
			"range": "172.21.42.0/24",
		},
	}

	configJSON, err := json.Marshal(config)
	Expect(err).ToNot(HaveOccurred())

	return &k8snetworkplumbingwgv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "migration-cni",
			Namespace: flags.KubeVirtInstallNamespace,
		},
		Spec: k8snetworkplumbingwgv1.NetworkAttachmentDefinitionSpec{
			Config: string(configJSON),
		},
	}
}

func EnsureNoMigrationMetadataInPersistentXML(vmi *v1.VirtualMachineInstance) {
	domXML := libpod.RunCommandOnVmiPod(vmi, []string{"virsh", "dumpxml", "1"})
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

func GetValidSourceNodeAndTargetNodeForHostModelMigration(virtCli kubecli.KubevirtClient) (sourceNode *k8sv1.Node, targetNode *k8sv1.Node, err error) {
	getNodeHostRequiredFeatures := func(node *k8sv1.Node) (features []string) {
		for key := range node.Labels {
			if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
				features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
			}
		}
		return features
	}
	areFeaturesSupportedOnNode := func(node *k8sv1.Node, features []string) bool {
		isFeatureSupported := func(feature string) bool {
			for key := range node.Labels {
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

			sourceHostCpuModel = libnode.GetNodeHostModel(&potentialSourceNode)
			if sourceHostCpuModel == "" {
				continue
			}
			supportedInTarget := false
			for key := range potentialTargetNode.Labels {
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

func CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode *k8sv1.Node, targetNode *k8sv1.Node) (nodefiinity *k8sv1.NodeAffinity, err error) {
	if sourceNode == nil || targetNode == nil {
		return nil, fmt.Errorf("couldn't find valid nodes for host-model migration")
	}
	return &k8sv1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
			NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
				{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{
						{
							Key:      k8sv1.LabelHostname,
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
							Key:      k8sv1.LabelHostname,
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
func ConfirmVMIPostMigrationFailed(vmi *v1.VirtualMachineInstance, migrationUID string) *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Verifying the VMI's migration state")
	Expect(vmi.Status.MigrationState).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
	Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
	Expect(vmi.Status.MigrationState.Completed).To(BeFalse())
	Expect(vmi.Status.MigrationState.Failed).To(BeTrue())
	Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
	Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

	By("Verifying the VMI's is in the running state")
	Expect(vmi.Status.Phase).To(Equal(v1.Running))

	return vmi
}

func ConfirmVMIPostMigrationAborted(vmi *v1.VirtualMachineInstance, migrationUID string, timeout int) *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()
	By("Waiting until the migration is completed")
	EventuallyWithOffset(1, func() v1.VirtualMachineInstanceMigrationState {
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		if vmi.Status.MigrationState != nil {
			return *vmi.Status.MigrationState
		}
		return v1.VirtualMachineInstanceMigrationState{}

	}, timeout, 1*time.Second).Should(
		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Completed":   BeFalse(),
			"AbortStatus": Equal(v1.MigrationAbortSucceeded),
		}),
	)

	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By("Verifying the VMI's migration state")
	ExpectWithOffset(1, vmi.Status.MigrationState).ToNot(BeNil())
	ExpectWithOffset(1, vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
	ExpectWithOffset(1, vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
	ExpectWithOffset(1, vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
	ExpectWithOffset(1, vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
	ExpectWithOffset(1, vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
	ExpectWithOffset(1, string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))
	ExpectWithOffset(1, vmi.Status.MigrationState.Failed).To(BeTrue())
	ExpectWithOffset(1, vmi.Status.MigrationState.AbortRequested).To(BeTrue())

	By("Verifying the VMI's is in the running state")
	ExpectWithOffset(1, vmi).To(matcher.BeInPhase(v1.Running))
	return vmi
}

func RunMigrationAndExpectFailure(migration *v1.VirtualMachineInstanceMigration, timeout int) string {

	virtClient := kubevirt.Client()
	By("Starting a Migration")
	createdMigration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting until the Migration Completes")
	Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
		migration, err := virtClient.VirtualMachineInstanceMigration(createdMigration.Namespace).Get(context.Background(), createdMigration.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		phase := migration.Status.Phase
		Expect(phase).NotTo(Equal(v1.MigrationSucceeded))

		return phase

	}, timeout, 1*time.Second).Should(Equal(v1.MigrationFailed))
	return string(createdMigration.UID)
}

func RunMigrationAndCollectMigrationMetrics(vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration) {
	var err error
	virtClient := kubevirt.Client()
	var pod *k8sv1.Pod
	var metricsIPs []string
	var migrationMetrics = []string{
		"kubevirt_vmi_migration_data_total_bytes",
		"kubevirt_vmi_migration_data_remaining_bytes",
		"kubevirt_vmi_migration_data_processed_bytes",
		"kubevirt_vmi_migration_dirty_memory_rate_bytes",
		"kubevirt_vmi_migration_disk_transfer_rate_bytes",
	}
	const family = k8sv1.IPv4Protocol

	By("Finding the prometheus endpoint")
	pod, err = libnode.GetVirtHandlerPod(virtClient, vmi.Status.NodeName)
	Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
	Expect(pod.Status.PodIPs).ToNot(BeEmpty(), "pod IPs must not be empty")
	for _, ip := range pod.Status.PodIPs {
		metricsIPs = append(metricsIPs, ip.IP)
	}

	By("Waiting until the Migration Completes")
	ip := libnet.GetIP(metricsIPs, family)

	_ = RunMigration(virtClient, migration)

	By("Scraping the Prometheus endpoint")
	validateNoZeroMetrics := func(metrics map[string]float64) error {
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		for _, key := range keys {
			value := metrics[key]

			if value == 0 {
				return fmt.Errorf("metric value for %s is not expected to be zero", key)
			}
		}
		return nil
	}

	Eventually(func() error {
		out := libmonitoring.GetKubevirtVMMetrics(pod, ip)
		for _, metricName := range migrationMetrics {
			lines := libinfra.TakeMetricsWithPrefix(out, metricName)
			metrics, err := libinfra.ParseMetricsToMap(lines)
			Expect(err).ToNot(HaveOccurred())

			if err := validateNoZeroMetrics(metrics); err != nil {
				return err
			}
		}

		return nil
	}, 100*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func ConfirmMigrationMode(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode) {
	var err error
	By("Retrieving the VMI post migration")
	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("couldn't find vmi err: %v \n", err))

	By("Verifying the VMI's migration mode")
	ExpectWithOffset(1, vmi.Status.MigrationState.Mode).To(Equal(expectedMode), fmt.Sprintf("expected migration state: %v got :%v \n", vmi.Status.MigrationState.Mode, expectedMode))
}

func WaitUntilMigrationMode(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode, timeout time.Duration) {
	By("Waiting until migration status")
	EventuallyWithOffset(1, func() v1.MigrationMode {
		By("Retrieving the VMI to check its migration mode")
		newVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if newVMI.Status.MigrationState != nil {
			return newVMI.Status.MigrationState.Mode
		}
		return ""
	}, timeout, 1*time.Second).Should(Equal(expectedMode))
}
