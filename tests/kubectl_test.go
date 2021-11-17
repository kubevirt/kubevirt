package tests_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[sig-compute]oc/kubectl integration", func() {
	var (
		k8sClient, result string
		err               error
	)
	BeforeEach(func() {

		k8sClient = tests.GetK8sCmdClient()
		tests.SkipIfNoCmd(k8sClient)
		tests.BeforeTestCleanup()
	})

	table.DescribeTable("[test_id:3812]explain vm/vmi", func(resource string) {
		output, stderr, err := tests.RunCommand(k8sClient, "explain", resource)
		// kubectl will not find resource for the first time this command is issued
		if err != nil {
			output, _, err = tests.RunCommand(k8sClient, "explain", resource)
		}
		Expect(err).NotTo(HaveOccurred(), stderr)
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(ContainSubstring("metadata	<Object>"))
		Expect(output).To(ContainSubstring("spec	<Object>"))
		Expect(output).To(ContainSubstring("status	<Object>"))
	},
		table.Entry("[test_id:3810]explain vm", "vm"),
		table.Entry("[test_id:3811]explain vmi", "vmi"),
		table.Entry("[test_id:5178]explain vmim", "vmim"),
		table.Entry("[test_id:5179]explain kv", "kv"),
		table.Entry("[test_id:5180]explain vmsnapshot", "vmsnapshot"),
		table.Entry("[test_id:5181]explain vmsnapshotcontent", "vmsnapshotcontent"),
	)

	It("[test_id:5182]vmipreset have validation", func() {
		output, _, err := tests.RunCommand(k8sClient, "explain", "vmipreset")
		if err != nil {
			output, _, err = tests.RunCommand(k8sClient, "explain", "vmipreset")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(ContainSubstring("metadata	<Object>"))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

	It("[test_id:5183]vmirs have validation", func() {
		output, _, err := tests.RunCommand(k8sClient, "explain", "vmirs")
		if err != nil {
			output, _, err = tests.RunCommand(k8sClient, "explain", "vmirs")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(ContainSubstring("metadata	<Object>"))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

	Describe("[rfe_id:3423][vendor:cnv-qe@redhat.com][level:component]oc/kubectl get vm/vmi tests", func() {
		var (
			virtCli kubecli.KubevirtClient
			vm      *v1.VirtualMachine
		)

		BeforeEach(func() {
			virtCli, err = kubecli.GetKubevirtClient()
			util.PanicOnError(err)

			vm = tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
			vm, err = virtCli.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).NotTo(HaveOccurred())
			tests.StartVirtualMachine(vm)
		})

		AfterEach(func() {
			virtCli.VirtualMachine(util.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
		})

		table.DescribeTable("should verify set of columns for", func(verb, resource string, expectedHeader []string) {
			result, _, err = tests.RunCommand(k8sClient, verb, resource, vm.Name)
			// due to issue of kubectl that sometimes doesn't show CRDs on the first try, retry the same command
			if err != nil {
				result, _, err = tests.RunCommand(k8sClient, verb, resource, vm.Name)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).ToNot(Equal(0))
			resultFields := strings.Fields(result)
			// Verify that only Header is not present
			Expect(len(resultFields)).Should(BeNumerically(">", len(expectedHeader)))
			columnHeaders := resultFields[:len(expectedHeader)]
			// Verify the generated header is same as expected
			Expect(columnHeaders).To(Equal(expectedHeader))
			// Name will be there in all the cases, so verify name
			Expect(resultFields[len(expectedHeader)]).To(Equal(vm.Name))
		},
			table.Entry("[test_id:3464]virtualmachine", "get", "vm", []string{"NAME", "AGE", "STATUS", "READY"}),
			table.Entry("[test_id:3465]virtualmachineinstance", "get", "vmi", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "READY"}),
		)

		table.DescribeTable("should verify set of wide columns for", func(verb, resource, option string, expectedHeader []string, verifyPos int, expectedData string) {

			result, _, err := tests.RunCommand(k8sClient, verb, resource, vm.Name, "-o", option)
			// due to issue of kubectl that sometimes doesn't show CRDs on the first try, retry the same command
			if err != nil {
				result, _, err = tests.RunCommand(k8sClient, verb, resource, vm.Name, "-o", option)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(len(result)).ToNot(Equal(0))
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
			table.Entry("[test_id:3468]virtualmachine", "get", "vm", "wide", []string{"NAME", "AGE", "STATUS", "READY"}, 1, "True"),
			table.Entry("[test_id:3466]virtualmachineinstance", "get", "vmi", "wide", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "READY", "LIVE-MIGRATABLE", "PAUSED"}, 1, "True"),
		)

	})

})
