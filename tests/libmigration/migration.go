package libmigration

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/onsi/gomega/gstruct"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libinfra"

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

const MigrationWaitTime = 240

func ExpectMigrationToSucceed(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	return ExpectMigrationToSucceedWithOffset(2, virtClient, migration, timeout)
}

func ExpectMigrationToSucceedWithDefaultTimeout(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	return ExpectMigrationToSucceed(virtClient, migration, MigrationWaitTime)
}

func ExpectMigrationToSucceedWithOffset(offset int, virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
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

func RunMigrationAndExpectToComplete(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) *v1.VirtualMachineInstanceMigration {
	migration = RunMigration(virtClient, migration)

	return ExpectMigrationToSucceed(virtClient, migration, timeout)
}

func RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	return RunMigrationAndExpectToComplete(virtClient, migration, MigrationWaitTime)
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

func setOrClearDedicatedMigrationNetwork(nad string, set bool) *v1.KubeVirt {
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
	return setOrClearDedicatedMigrationNetwork(nad, true)
}

func ClearDedicatedMigrationNetwork() *v1.KubeVirt {
	return setOrClearDedicatedMigrationNetwork("", false)
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
func ConfirmVMIPostMigrationFailed(vmi *v1.VirtualMachineInstance, migrationUID string) {
	virtClient := kubevirt.Client()
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Verifying the VMI's migration state")
	Expect(vmi.Status.MigrationState).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
	Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
	Expect(vmi.Status.MigrationState.Completed).To(BeTrue())
	Expect(vmi.Status.MigrationState.Failed).To(BeTrue())
	Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
	Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))

	By("Verifying the VMI's is in the running state")
	Expect(vmi.Status.Phase).To(Equal(v1.Running))
}

func ConfirmVMIPostMigrationAborted(vmi *v1.VirtualMachineInstance, migrationUID string, timeout int) *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()
	By("Waiting until the migration is completed")
	EventuallyWithOffset(1, func() v1.VirtualMachineInstanceMigrationState {
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		if vmi.Status.MigrationState != nil {
			return *vmi.Status.MigrationState
		}
		return v1.VirtualMachineInstanceMigrationState{}

	}, timeout, 1*time.Second).Should(
		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Completed":   BeTrue(),
			"AbortStatus": Equal(v1.MigrationAbortSucceeded),
		}),
	)

	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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

func CancelMigration(migration *v1.VirtualMachineInstanceMigration, vminame string, with_virtctl bool) {
	virtClient := kubevirt.Client()
	if !with_virtctl {
		By("Cancelling a Migration")
		Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &metav1.DeleteOptions{})).To(Succeed(), "Migration should be deleted successfully")
	} else {
		By("Cancelling a Migration with virtctl")
		command := clientcmd.NewRepeatableVirtctlCommand("migrate-cancel", "--namespace", migration.Namespace, vminame)
		Expect(command()).To(Succeed(), "should successfully migrate-cancel a migration")
	}
}

func RunAndCancelMigration(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, with_virtctl bool, timeout int) *v1.VirtualMachineInstanceMigration {
	var err error
	virtClient := kubevirt.Client()
	By("Starting a Migration")
	Eventually(func() error {
		migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

	By("Waiting until the Migration is Running")

	Eventually(func() bool {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
		if migration.Status.Phase == v1.MigrationRunning {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if vmi.Status.MigrationState.Completed != true {
				return true
			}
		}
		return false

	}, timeout, 1*time.Second).Should(BeTrue())

	CancelMigration(migration, vmi.Name, with_virtctl)

	return migration
}

func RunAndImmediatelyCancelMigration(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, with_virtctl bool, timeout int) *v1.VirtualMachineInstanceMigration {
	var err error
	virtClient := kubevirt.Client()
	By("Starting a Migration")
	Eventually(func() error {
		migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

	By("Waiting until the Migration is Running")
	Eventually(func() bool {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return migration.Status.Phase == v1.MigrationRunning
	}, timeout, 1*time.Second).Should(BeTrue())

	CancelMigration(migration, vmi.Name, with_virtctl)

	return migration
}

func RunMigrationAndExpectFailure(migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	var err error
	virtClient := kubevirt.Client()
	By("Starting a Migration")
	Eventually(func() error {
		migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	By("Waiting until the Migration Completes")

	uid := ""
	Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		phase := migration.Status.Phase
		Expect(phase).NotTo(Equal(v1.MigrationSucceeded))

		uid = string(migration.UID)
		return phase

	}, timeout, 1*time.Second).Should(Equal(v1.MigrationFailed))
	return uid
}

func RunMigrationAndCollectMigrationMetrics(vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration) {
	var err error
	virtClient := kubevirt.Client()
	var pod *k8sv1.Pod
	var metricsIPs []string
	var migrationMetrics = []string{
		"kubevirt_vmi_migration_data_remaining_bytes",
		"kubevirt_vmi_migration_data_processed_bytes",
		"kubevirt_vmi_migration_dirty_memory_rate_bytes",
		"kubevirt_vmi_migration_disk_transfer_rate_bytes",
		"kubevirt_vmi_migration_memory_transfer_rate_bytes",
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
	ip := libinfra.GetSupportedIP(metricsIPs, family)

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

	getKubevirtVMMetricsFunc := tests.GetKubevirtVMMetricsFunc(&virtClient, pod)
	Eventually(func() error {
		out := getKubevirtVMMetricsFunc(ip)
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
