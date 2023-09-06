package tests_test

import (
	"context"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[sig-compute]oc/kubectl integration", decorators.SigCompute, func() {
	var (
		k8sClient, result string
		err               error
	)
	BeforeEach(func() {

		k8sClient = clientcmd.GetK8sCmdClient()
		clientcmd.SkipIfNoCmd(k8sClient)
	})

	DescribeTable("[test_id:3812]explain vm/vmi", func(resource string) {
		output, stderr, err := clientcmd.RunCommand(k8sClient, "explain", resource)
		// kubectl will not find resource for the first time this command is issued
		if err != nil {
			output, _, err = clientcmd.RunCommand(k8sClient, "explain", resource)
		}
		Expect(err).NotTo(HaveOccurred(), stderr)
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
		Expect(output).To(ContainSubstring("status	<Object>"))
	},
		Entry("[test_id:3810]explain vm", "vm"),
		Entry("[test_id:3811]explain vmi", "vmi"),
		Entry("[test_id:5178]explain vmim", "vmim"),
		Entry("[test_id:5179]explain kv", "kv"),
		Entry("[test_id:5180]explain vmsnapshot", "vmsnapshot"),
		Entry("[test_id:5181]explain vmsnapshotcontent", "vmsnapshotcontent"),
	)

	It("[test_id:5182]vmipreset have validation", func() {
		output, _, err := clientcmd.RunCommand(k8sClient, "explain", "vmipreset")
		if err != nil {
			output, _, err = clientcmd.RunCommand(k8sClient, "explain", "vmipreset")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

	It("[test_id:5183]vmirs have validation", func() {
		output, _, err := clientcmd.RunCommand(k8sClient, "explain", "vmirs")
		if err != nil {
			output, _, err = clientcmd.RunCommand(k8sClient, "explain", "vmirs")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

	Describe("[rfe_id:3423][vendor:cnv-qe@redhat.com][level:component]oc/kubectl get vm/vmi tests", func() {
		var (
			virtCli kubecli.KubevirtClient
			vm      *v1.VirtualMachine
		)

		BeforeEach(func() {
			virtCli = kubevirt.Client()

			vm = tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
			vm, err = virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
			Expect(err).NotTo(HaveOccurred())
			tests.StartVirtualMachine(vm)
		})

		AfterEach(func() {
			virtCli.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
		})

		DescribeTable("should verify set of columns for", func(verb, resource string, expectedHeader []string) {
			result, _, err = clientcmd.RunCommand(k8sClient, verb, resource, vm.Name)
			// due to issue of kubectl that sometimes doesn't show CRDs on the first try, retry the same command
			if err != nil {
				result, _, err = clientcmd.RunCommand(k8sClient, verb, resource, vm.Name)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())
			resultFields := strings.Fields(result)
			// Verify that only Header is not present
			Expect(len(resultFields)).Should(BeNumerically(">", len(expectedHeader)))
			columnHeaders := resultFields[:len(expectedHeader)]
			// Verify the generated header is same as expected
			Expect(columnHeaders).To(Equal(expectedHeader))
			// Name will be there in all the cases, so verify name
			Expect(resultFields[len(expectedHeader)]).To(Equal(vm.Name))
		},
			Entry("[test_id:3464]virtualmachine", "get", "vm", []string{"NAME", "AGE", "STATUS", "READY"}),
			Entry("[test_id:3465]virtualmachineinstance", "get", "vmi", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "READY"}),
		)

		DescribeTable("should verify set of wide columns for", func(verb, resource, option string, expectedHeader []string, verifyPos int, expectedData string) {

			result, _, err := clientcmd.RunCommand(k8sClient, verb, resource, vm.Name, "-o", option)
			// due to issue of kubectl that sometimes doesn't show CRDs on the first try, retry the same command
			if err != nil {
				result, _, err = clientcmd.RunCommand(k8sClient, verb, resource, vm.Name, "-o", option)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())
			resultFields := strings.Fields(result)
			// Verify that only Header is not present
			Expect(len(resultFields)).Should(BeNumerically(">", len(expectedHeader)))
			columnHeaders := resultFields[:len(expectedHeader)]
			// Verify the generated header is same as expected
			Expect(columnHeaders).To(Equal(expectedHeader))
			// Name will be there in all the cases, so verify name
			Expect(resultFields[len(expectedHeader)]).To(Equal(vm.Name))
			// Verify one of the wide column output field
			Expect(resultFields[len(resultFields)-verifyPos]).To(Equal(expectedData))

		},
			Entry("[test_id:3468]virtualmachine", "get", "vm", "wide", []string{"NAME", "AGE", "STATUS", "READY"}, 1, "True"),
			Entry("[test_id:3466]virtualmachineinstance", "get", "vmi", "wide", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "READY", "LIVE-MIGRATABLE", "PAUSED"}, 1, "True"),
		)

	})

	Describe("VM instance migration", func() {
		var virtClient kubecli.KubevirtClient

		BeforeEach(func() {
			checks.SkipIfMigrationIsNotPossible()

			virtClient = kubevirt.Client()
		})

		Context("'kubectl get vmim'", func() {
			It("print the expected columns and their corresponding values", func() {
				vmi := libvmi.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, libmigration.MigrationWaitTime)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("creating the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				var migrationCreated *v1.VirtualMachineInstanceMigration
				By("starting migration")
				Eventually(func() error {
					migrationCreated, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
					return err
				}, libmigration.MigrationWaitTime, 1*time.Second).Should(Succeed(), "migration creation should succeed")
				migration = migrationCreated

				libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)

				k8sClient := clientcmd.GetK8sCmdClient()
				result, _, err := clientcmd.RunCommand(k8sClient, "get", "vmim", migration.Name)
				// due to issue of kubectl that sometimes doesn't show CRDs on the first try, retry the same command
				if err != nil {
					result, _, err = clientcmd.RunCommand(k8sClient, "get", "vmim", migration.Name)
				}

				expectedHeader := []string{"NAME", "PHASE", "VMI"}
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeEmpty())
				resultFields := strings.Fields(result)

				By("Verify that only Header is not present")
				Expect(len(resultFields)).Should(BeNumerically(">", len(expectedHeader)))

				columnHeaders := resultFields[:len(expectedHeader)]
				By("Verify the generated header is same as expected")
				Expect(columnHeaders).To(Equal(expectedHeader))

				By("Verify VMIM name")
				Expect(resultFields[len(expectedHeader)]).To(Equal(migration.Name), "should match VMIM object name")
				By("Verify VMIM phase")
				Expect(resultFields[len(expectedHeader)+1]).To(Equal(string(v1.MigrationSucceeded)), "should have successful state")
				By("Verify VMI name related to the VMIM")
				Expect(resultFields[len(expectedHeader)+2]).To(Equal(vmi.Name), "should match the VMI name")
			})
		})
	})

	Describe("oc/kubectl logs", func() {
		var (
			vm *v1.VirtualMachineInstance
		)

		It("oc/kubectl logs <vmi-pod> return default container log", func() {
			vm = libvmi.NewCirros()
			vm = tests.RunVMIAndExpectLaunch(vm, 30)

			k8sClient := clientcmd.GetK8sCmdClient()

			output, _, err := clientcmd.RunCommand(k8sClient, "logs", tests.GetPodByVirtualMachineInstance(vm).Name)
			Expect(err).NotTo(HaveOccurred())

			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("level"))
			Expect(output).To(ContainSubstring("msg"))
			Expect(output).To(ContainSubstring("pos"))
			Expect(output).To(ContainSubstring("timestamp"))
		})
	})
})
