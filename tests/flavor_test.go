package tests_test

import (
	"context"
	goerrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Flavor", func() {
	const (
		namespacedFlavorKind = "VirtualMachineFlavor"
	)

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.BeforeTestCleanup()
	})

	Context("Flavor validation", func() {
		It("[test_id:TODO] should allow valid flavor", func() {
			flavor := newVirtualMachineFlavor()
			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VM with invalid FlavorMatcher", func() {
		It("[test_id:TODO] should fail to create VM with non-existing cluster flavor", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "non-existing-cluster-flavor",
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing namespaced flavor", func() {
			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: "non-existing-flavor",
				Kind: namespacedFlavorKind,
			}

			_, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})
	})

	Context("Flavor application", func() {
		Context("CPU", func() {
			It("[test_id:TODO] should apply flavor to CPU", func() {
				cpu := &v1.CPU{Sockets: 2, Cores: 1, Threads: 1, Model: v1.DefaultCPUModel}

				flavor := newVirtualMachineFlavor()
				flavor.Spec.CPU = cpu

				flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
					Create(context.Background(), flavor, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := tests.NewRandomVMIWithEphemeralDisk(
					cd.ContainerDiskFor(cd.ContainerDiskCirros),
				)
				vmi.Spec.Domain.CPU = nil

				vm := tests.NewRandomVirtualMachine(vmi, false)
				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
					Kind: namespacedFlavorKind,
				}

				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				tests.StartVMAndExpectRunning(virtClient, vm)

				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.CPU).To(Equal(cpu))
				Expect(vmi.Annotations[v1.FlavorAnnotation]).To(Equal(flavor.Name))
				Expect(vmi.Annotations[v1.ClusterFlavorAnnotation]).To(Equal(""))
			})

			It("[test_id:TODO] should fail if flavor and VMI define CPU", func() {
				flavor := newVirtualMachineFlavor()
				flavor.Spec.CPU = &v1.CPU{Sockets: 2, Cores: 1, Threads: 1}

				flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
					Create(context.Background(), flavor, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := tests.NewRandomVMI()
				vmi.Spec.Domain.CPU = &v1.CPU{Sockets: 1, Cores: 1, Threads: 1}

				vm := tests.NewRandomVirtualMachine(vmi, false)
				vm.Spec.Flavor = &v1.FlavorMatcher{
					Name: flavor.Name,
					Kind: namespacedFlavorKind,
				}

				_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).To(HaveOccurred())
				var apiStatus errors.APIStatus
				Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

				Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
				cause := apiStatus.Status().Details.Causes[0]

				Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
				Expect(cause.Message).To(Equal("VMI field conflicts with selected Flavor"))
				Expect(cause.Field).To(Equal("spec.template.spec.domain.cpu"))
			})
		})
	})
})

func newVirtualMachineFlavor() *flavorv1alpha1.VirtualMachineFlavor {
	return &flavorv1alpha1.VirtualMachineFlavor{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vm-flavor-",
			Namespace:    util.NamespaceTestDefault,
		},
		Spec: flavorv1alpha1.VirtualMachineFlavorSpec{},
	}
}
