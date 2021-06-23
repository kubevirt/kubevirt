package tests

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/tests/console"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

func ExpectMigrationSuccess(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	By("Waiting until the Migration Completes")
	uid := ""
	Eventually(func() error {
		migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed), "migration should not fail")

		uid = string(migration.UID)
		if migration.Status.Phase == v1.MigrationSucceeded {
			return nil
		}
		return fmt.Errorf("migration is in the phase: %s", migration.Status.Phase)

	}, timeout, 1*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("migration should succeed after %d s", timeout))
	return uid
}

func RunMigrationAndExpectCompletion(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	By("Starting a Migration")
	var err error
	var migrationCreated *v1.VirtualMachineInstanceMigration
	Eventually(func() error {
		migrationCreated, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
		return err
	}, timeout, 1*time.Second).Should(Succeed(), "migration creation should succeed")
	migration = migrationCreated

	return ExpectMigrationSuccess(virtClient, migration, timeout)
}

func ConfirmVMIPostMigration(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, migrationUID string) {
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

func RunMigrationAndExpectFailure(virtClient kubecli.KubevirtClient, migration *v1.VirtualMachineInstanceMigration, timeout int) string {
	var err error
	var createdMigration *v1.VirtualMachineInstanceMigration

	By("Starting a migration")
	Eventually(func() error {
		createdMigration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

	By("Waiting until the migration completes with Failure phase")
	uid := ""
	Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
		createdMigration, err = virtClient.VirtualMachineInstanceMigration(createdMigration.Namespace).Get(createdMigration.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		phase := createdMigration.Status.Phase
		Expect(phase).NotTo(Equal(v1.MigrationSucceeded))

		uid = string(createdMigration.UID)
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

func RunStressTest(vmi *v1.VirtualMachineInstance, vmsize, stressTimeoutSeconds int) {
	By("Run a stress test to dirty some pages and slow down the migration")
	stressCmd := fmt.Sprintf("stress-ng --vm 1 --vm-bytes %sM --vm-keep --timeout %ds&\n", vmsize, stressTimeoutSeconds)
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

func WaitForMigrationRunning(virtClient kubecli.KubevirtClient, vmim *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, timeout time.Duration) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		migration, err := virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(vmim.Name, &metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get VirtualMachineInstanceMigration %s/%s: %v", migration.Namespace, migration.Name, err)
		}

		migrationPhase := migration.Status.Phase
		switch migrationPhase {
		case v1.MigrationFailed, v1.MigrationSucceeded:
			return false, fmt.Errorf("migration is final with phase: %s instead of %s", migrationPhase, v1.MigrationRunning)
		case v1.MigrationRunning:
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf("failed to get VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
			}

			if vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.Completed {
				return true, nil
			}
		}

		return false, nil
	})
}
