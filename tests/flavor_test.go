package tests_test

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	utiltype "kubevirt.io/kubevirt/pkg/util/types"

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

		It("[test_id:TODO] should fail flavor with no profiles", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles = []flavorv1alpha1.VirtualMachineFlavorProfile{}

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})

			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueRequired))
			Expect(cause.Message).To(HavePrefix("A flavor must have at least one profile"))
			Expect(cause.Field).To(Equal("profiles"))
		})

		It("[test_id:TODO] should fail flavor with multiple default profiles", func() {
			flavor := newVirtualMachineFlavor()
			flavor.Profiles = append(flavor.Profiles, flavorv1alpha1.VirtualMachineFlavorProfile{
				Name:    "second-default",
				Default: true,
			})

			_, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})

			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotSupported))
			Expect(cause.Message).To(HavePrefix("Flavor contains more than one default profile"))
			Expect(cause.Field).To(Equal("profiles"))
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
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
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
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing default flavor profile", func() {
			flavor := newVirtualMachineFlavor()
			for i := range flavor.Profiles {
				flavor.Profiles[i].Default = false
			}

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
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
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})

		It("[test_id:TODO] should fail to create VM with non-existing custom flavor profile", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
			vm := tests.NewRandomVirtualMachine(vmi, false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name:    flavor.Name,
				Kind:    namespacedFlavorKind,
				Profile: "nonexisting-profile",
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")

			Expect(apiStatus.Status().Details.Causes).To(HaveLen(1))
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(cause.Message).To(HavePrefix("Could not find flavor profile:"))
			Expect(cause.Field).To(Equal("spec.flavor"))
		})
	})

	Context("Flavor revisions", func() {
		It("[test_id:TODO] should create flavor revision and save it to VM", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			var newVM *v1.VirtualMachine
			Eventually(func() *v1.FlavorRevisionSpec {
				newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return newVM.Status.FlavorRevision
			}, 15*time.Second, 1*time.Second).ShouldNot(BeNil(), "flavor revision can't be nil")

			cr, err := virtClient.AppsV1().ControllerRevisions(newVM.Namespace).Get(context.Background(), newVM.Status.FlavorRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Data.Raw).ToNot(Equal([]byte{}), "data should not be empty")
		})

		It("[test_id:TODO] should delete flavor revision when vm is deleted", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")

			vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred(), "error should not occured")

			var newVM *v1.VirtualMachine
			Eventually(func() *v1.FlavorRevisionSpec {
				newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "error should not occured")
				return newVM.Status.FlavorRevision
			}, 15*time.Second, 1*time.Second).ShouldNot(BeNil(), "flavor revision can't be nil")

			cr, err := virtClient.AppsV1().ControllerRevisions(newVM.Namespace).Get(context.Background(), newVM.Status.FlavorRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")
			Expect(cr.Data.Raw).ToNot(Equal([]byte{}), "data should not be empty")

			err = virtClient.VirtualMachine(util.NamespaceTestDefault).Delete(vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")

			Eventually(func() bool {
				_, err = virtClient.AppsV1().ControllerRevisions(newVM.Namespace).Get(context.Background(), newVM.Status.FlavorRevision.Name, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 15*time.Second, 1*time.Second).Should(BeTrue(), "flavor revision should not exists")

		})

		It("[test_id:TODO] should not create new flavor revision when vm is patched", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			var newVM *v1.VirtualMachine
			Eventually(func() *v1.FlavorRevisionSpec {
				newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "error should not occured")
				return newVM.Status.FlavorRevision
			}, 15*time.Second, 1*time.Second).ShouldNot(BeNil(), "flavor revision can't be nil")

			newVM.Annotations["test"] = "testingAnnotation"

			p := []utiltype.PatchOperation{{
				Op:    "replace",
				Path:  "/metadata/annotations",
				Value: newVM.Annotations,
			}}

			payloadBytes, err := json.Marshal(p)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Patch(newVM.Name, types.JSONPatchType, payloadBytes, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			patchedVM, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")
			Expect(patchedVM.Status.FlavorRevision.Name).To(Equal(newVM.Status.FlavorRevision.Name), "flavor revision's name has to equal")

			rcList, err := virtClient.AppsV1().ControllerRevisions(newVM.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")

			Expect(len(rcList.Items) > 0).To(BeTrue(), "controller revisions should exists")
			occurences := 0
			for _, rc := range rcList.Items {
				if strings.Contains(rc.Name, string(newVM.UID)) {
					occurences++
				}
			}
			Expect(occurences).To(Equal(1), "there should be only 1 revision for vm")
		})

		It("[test_id:TODO] should not create new flavor revision when vm is patched with new flavor name", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			var newVM *v1.VirtualMachine
			Eventually(func() *v1.FlavorRevisionSpec {
				newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "error should not occured")
				return newVM.Status.FlavorRevision
			}, 15*time.Second, 1*time.Second).ShouldNot(BeNil(), "flavor revision can't be nil")

			flavor = newVirtualMachineFlavor()
			flavor.Name = flavor.Name + "updated"
			flavor.Profiles[0].Name = flavor.Profiles[0].Name + "updated"

			flavor, err = virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newVM.Spec.Flavor.Name = flavor.Name

			p := []utiltype.PatchOperation{{
				Op:    "replace",
				Path:  "/spec/flavor",
				Value: newVM.Spec.Flavor,
			}}

			payloadBytes, err := json.Marshal(p)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Patch(newVM.Name, types.JSONPatchType, payloadBytes, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			patchedVM, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(newVM.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")
			Expect(patchedVM.Status.FlavorRevision.Name).To(Equal(newVM.Status.FlavorRevision.Name), "flavor revision names can't be the same")

			rcList, err := virtClient.AppsV1().ControllerRevisions(newVM.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "error should not occured")

			Expect(len(rcList.Items) > 0).To(BeTrue(), "controller revisions should exists")
			occurences := 0
			for _, rc := range rcList.Items {
				if strings.Contains(rc.Name, string(newVM.UID)) {
					occurences++
				}
			}
			Expect(occurences).To(Equal(1), "there should be only 1 revision for vm")
		})

		It("[test_id:TODO] should not update vm, when flavor revision name change", func() {
			flavor := newVirtualMachineFlavor()

			flavor, err := virtClient.VirtualMachineFlavor(util.NamespaceTestDefault).
				Create(context.Background(), flavor, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm := tests.NewRandomVirtualMachine(tests.NewRandomVMI(), false)

			vm.Spec.Flavor = &v1.FlavorMatcher{
				Name: flavor.Name,
				Kind: namespacedFlavorKind,
			}

			_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			var newVM *v1.VirtualMachine
			Eventually(func() *v1.FlavorRevisionSpec {
				newVM, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "error should not occured")
				return newVM.Status.FlavorRevision
			}, 15*time.Second, 1*time.Second).ShouldNot(BeNil(), "flavor revision can't be nil")

			newVM.Status.FlavorRevision.Name = "someWrongValue"
			p := []utiltype.PatchOperation{{
				Op:    "replace",
				Path:  "/status/flavorRevision",
				Value: newVM.Status.FlavorRevision,
			}}

			payloadBytes, err := json.Marshal(p)
			Expect(err).ToNot(HaveOccurred())

			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Patch(newVM.Name, types.JSONPatchType, payloadBytes, &metav1.PatchOptions{})
			Expect(err).To(BeNil())
			Expect(vm.Status.FlavorRevision.Name).ToNot(Equal(newVM.Status.FlavorRevision.Name), "flavor revision names should equal")
		})

	})

	Context("Flavor application", func() {
		startVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			runStrategyAlways := v1.RunStrategyAlways
			By("Starting the VirtualMachine")

			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = nil
				updatedVM.Spec.RunStrategy = &runStrategyAlways
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance created
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &k8smetav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("VMI has the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			return updatedVM
		}

		Context("CPU", func() {
			It("[test_id:TODO] should apply flavor to CPU", func() {
				cpu := &v1.CPU{Sockets: 2, Cores: 1, Threads: 1, Model: v1.DefaultCPUModel}

				flavor := newVirtualMachineFlavor()
				flavor.Profiles[0].CPU = cpu

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

				startVM(vm)

				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.CPU).To(Equal(cpu))
			})

			It("[test_id:TODO] should fail if flavor and VMI define CPU", func() {
				flavor := newVirtualMachineFlavor()
				flavor.Profiles[0].CPU = &v1.CPU{Sockets: 2, Cores: 1, Threads: 1}

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
				Expect(cause.Message).To(Equal("VMI field conflicts with selected Flavor profile"))
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
		Profiles: []flavorv1alpha1.VirtualMachineFlavorProfile{{
			Name:    "default",
			Default: true,
		}},
	}
}
