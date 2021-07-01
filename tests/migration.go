package tests

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	k8snetworkplumbingwgv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/console"
)

func ExpectMigrationSuccess(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	return expectMigrationSuccessWithOffset(2, virtClient, migration, timeout)
}

func expectMigrationSuccessWithOffset(offset int, virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	By("Waiting until the Migration Completes")
	uid := ""
	EventuallyWithOffset(offset, func() error {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		ExpectWithOffset(offset+1, migration.Status.Phase).ToNot(Equal(v1.MigrationFailed), "migration should not fail")

		uid = string(migration.UID)
		if migration.Status.Phase == v1.MigrationSucceeded {
			return nil
		}
		return fmt.Errorf("migration is in the phase: %s", migration.Status.Phase)

	}, timeout, 1*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("migration should succeed after %d s", timeout))
	return uid
}

func RunMigrationAndExpectCompletion(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	migration = RunMigration(virtClient, migration)

	return ExpectMigrationSuccess(virtClient, migration, timeout)
}

func RunMigration(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration) *v1.VirtualMachineInstanceMigration {
	By("Starting a Migration")

	migrationCreated, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return migrationCreated
}

func ConfirmVMIPostMigration(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, migrationUID string) *v1.VirtualMachineInstance {
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred(), "should have been able to retrive the VMI instance")

	By("Verifying the VMI's migration state")
	Expect(vmi.Status.MigrationState).ToNot(BeNil(), "should have been able to retrieve the VMIs `Status::MigrationState`")
	Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil(), "the VMIs `Status::MigrationState` should have a StartTimestamp")
	Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil(), "the VMIs `Status::MigrationState` should have a EndTimestamp")
	Expect(vmi.Status.MigrationState.TargetNode).To(Equal(vmi.Status.NodeName), "the VMI should have migrated to the desired node")
	Expect(vmi.Status.MigrationState.TargetNode).NotTo(Equal(vmi.Status.MigrationState.SourceNode), "the VMI must have migrated to a different node from the one it originated from")
	Expect(vmi.Status.MigrationState.Completed).To(BeTrue(), "the VMI migration state must have completed")
	Expect(vmi.Status.MigrationState.Failed).To(BeFalse(), "the VMI migration status must not have failed")
	Expect(vmi.Status.MigrationState.TargetNodeAddress).NotTo(Equal(""), "the VMI `Status::MigrationState::TargetNodeAddress` must not be empty")
	Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID), "the VMI migration UID must be the expected one")

	By("Verifying the VMI's is in the running state")
	Expect(vmi.Status.Phase).To(Equal(v1.Running), "the VMI must be in `Running` state after the migration")

	return vmi
}

func setOrClearDedicatedMigrationNetwork(nad string, set bool) *v1.KubeVirt {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

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

	res := UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

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
		virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
		Expect(err).ToNot(HaveOccurred(), "Failed to list the virt-handler pods")
		for _, pod := range virtHandlerPods.Items {
			if pod.Status.Phase != k8sv1.PodRunning {
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
      "master": "eth1",
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
	domXML := RunCommandOnVmiPod(vmi, []string{"virsh", "dumpxml", "1"})
	decoder := xml.NewDecoder(bytes.NewReader([]byte(domXML)))

	var location = make([]string, 0)
	var found = false
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		Expect(err).To(BeNil(), "error getting token: %v\n", err)

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

func RunStressTest(vmi *v1.VirtualMachineInstance, vmSizeMegabytes int, stressTimeoutSeconds int) {
	By("Run a stress test to dirty some pages and slow down the migration")
	stressCmd := fmt.Sprintf("stress-ng --vm 1 --vm-bytes %dM --vm-keep --timeout %ds&\n", vmSizeMegabytes, stressTimeoutSeconds)
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "which stress-ng\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
		&expect.BSnd{S: stressCmd},
		&expect.BExp{R: console.PromptExpression},
	}, 15)).To(Succeed(), "should run a stress test")

	// give stress tool some time to trash more memory pages before returning control to next steps
	if stressTimeoutSeconds < 15 {
		time.Sleep(time.Duration(stressTimeoutSeconds) * time.Second)
	} else {
		time.Sleep(15 * time.Second)
	}
}

func RunMigrationAndExpectFailure(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	var createdMigration *v1.VirtualMachineInstanceMigration
	By("Starting a Migration")
	Eventually(func() error {
		var err error
		createdMigration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	By("Waiting until the Migration Completes")

	uid := ""
	Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
		migration, err := virtClient.VirtualMachineInstanceMigration(createdMigration.Namespace).Get(createdMigration.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		phase := migration.Status.Phase
		Expect(phase).NotTo(Equal(v1.MigrationSucceeded))

		uid = string(migration.UID)
		return phase

	}, timeout, 1*time.Second).Should(Equal(v1.MigrationFailed))
	return uid
}

func ConfirmVMIPostMigrationFailed(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, migrationUID string) {
	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
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

func ConfirmVMIPostMigrationAborted(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, migrationUID string, timeout int) *v1.VirtualMachineInstance {
	By("Waiting until the migration is completed")
	Eventually(func() bool {
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed &&
			vmi.Status.MigrationState.AbortStatus == v1.MigrationAbortSucceeded {
			return true
		}
		return false

	}, timeout, 1*time.Second).Should(BeTrue())

	By("Retrieving the VMI post migration")
	vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Verifying the VMI's migration state")
	Expect(vmi.Status.MigrationState).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
	Expect(vmi.Status.MigrationState.SourceNode).To(Equal(vmi.Status.NodeName))
	Expect(vmi.Status.MigrationState.TargetNode).ToNot(Equal(vmi.Status.MigrationState.SourceNode))
	Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(""))
	Expect(string(vmi.Status.MigrationState.MigrationUID)).To(Equal(migrationUID))
	Expect(vmi.Status.MigrationState.Failed).To(BeTrue())
	Expect(vmi.Status.MigrationState.AbortRequested).To(BeTrue())

	By("Verifying the VMI's is in the running state")
	Expect(vmi.Status.Phase).To(Equal(v1.Running))
	return vmi
}
