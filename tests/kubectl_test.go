package tests_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:3423][vendor:cnv-qe@redhat.com][level:component]oc/kubectl get vm/vmi tests", func() {
	tests.FlagParse()

	var k8sClient, result string
	var vm *v1.VirtualMachine
	var virtCli kubecli.KubevirtClient
	var err error

	BeforeEach(func() {

		virtCli, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		k8sClient = tests.GetK8sCmdClient()
		tests.SkipIfNoCmd(k8sClient)
		tests.BeforeTestCleanup()

		vm = tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)
		vm, err = virtCli.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
		Expect(err).NotTo(HaveOccurred())
		tests.StartVirtualMachine(vm)
	})

	AfterEach(func() {
		virtCli.VirtualMachine(tests.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
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
		table.Entry("[test_id:3464]virtualmachine", "get", "vm", []string{"NAME", "AGE", "VOLUME"}),
		table.Entry("[test_id:3465]virtualmachineinstance", "get", "vmi", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME"}),
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
		table.Entry("[test_id:3468]virtualmachine", "get", "vm", "wide", []string{"NAME", "AGE", "VOLUME", "CREATED"}, 1, "true"),
		table.Entry("[test_id:3466]virtualmachineinstance", "get", "vmi", "wide", []string{"NAME", "AGE", "PHASE", "IP", "NODENAME", "LIVE-MIGRATABLE", "PAUSED"}, 1, "True"),
	)
})
