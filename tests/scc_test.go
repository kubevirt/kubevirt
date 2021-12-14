package tests_test

import (
	"context"
	"fmt"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"
	util2 "kubevirt.io/kubevirt/tests/util"

	"time"

	"kubevirt.io/kubevirt/tests/libvmi"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial][sig-operator] SCC", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		if !tests.IsOpenShift() {
			Skip("OpenShift operator tests should not be started on k8s")
		}
	})

	checkHandlerSCC := func() {
		By("Checking if virt-handler is assigned to kubevirt-handler SCC")
		l, err := labels.Parse("kubevirt.io=virt-handler")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: l.String()})
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should get virt-handler")
		ExpectWithOffset(1, pods.Items).ToNot(BeEmpty())
		ExpectWithOffset(1, pods.Items[0].Annotations["openshift.io/scc"]).To(
			Equal("kubevirt-handler"), "Should virt-handler be assigned to kubevirt-handler SCC",
		)
	}

	Context("[rfe_id:2897][crit:medium][vendor:cnv-qe@redhat.com][level:component]With OpenShift cluster", func() {

		const OpenShiftSCCLabel = "openshift.io/scc"
		BeforeEach(func() {
			if !tests.IsOpenShift() {
				Skip("OpenShift operator tests should not be started on k8s")
			}
		})

		checkSCCs := func(managed bool) {
			var expectedSCCs, sccs []string

			By("Checking if kubevirt SCCs have been created")
			secClient := virtClient.SecClient()
			operatorSCCs := components.GetAllSCC(flags.KubeVirtInstallNamespace, managed)
			for _, scc := range operatorSCCs {
				expectedSCCs = append(expectedSCCs, scc.GetName())
			}

			createdSCCs, err := secClient.SecurityContextConstraints().List(context.Background(), metav1.ListOptions{LabelSelector: controller.OperatorLabel})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			for _, scc := range createdSCCs.Items {
				sccs = append(sccs, scc.GetName())
			}
			ExpectWithOffset(1, sccs).To(ConsistOf(expectedSCCs))
		}

		createLauncherToCheckSCC := func() k8sv1.Pod {
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			tests.RunVMI(vmi, 180)
			tests.WaitForSuccessfulVMIStart(vmi)

			uid := vmi.GetObjectMeta().GetUID()
			labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
			pods, err := virtClient.CoreV1().Pods(util2.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should get virt-launcher")
			ExpectWithOffset(1, len(pods.Items)).To(Equal(1))
			return pods.Items[0]
		}

		It("[test_id:2910]Should have kubevirt SCCs created", func() {
			checkSCCs(!checks.HasFeature(virtconfig.NoManagedSCC))

			checkHandlerSCC()

			By("Checking if virt-launcher is assigned to kubevirt-controller SCC")
			pod := createLauncherToCheckSCC()
			Expect(pod.Annotations[OpenShiftSCCLabel]).To(
				Equal("kubevirt-controller"), "Should virt-launcher be assigned to kubevirt-controller SCC",
			)
		})
	})

	Context("NoManagedSCC feature gate enabled", func() {
		var dissableFeature func()

		BeforeEach(func() {
			if !checks.HasFeature(virtconfig.NoManagedSCC) {
				tests.EnableFeatureGate(virtconfig.NoManagedSCC)

				dissableFeature = func() {
					tests.DisableFeatureGate(virtconfig.NoManagedSCC)
					Eventually(func() error {
						_, err := virtClient.SecClient().SecurityContextConstraints().Get(context.Background(), "kubevirt-controller", metav1.GetOptions{})
						return err
					}, 2*time.Minute, time.Second).ShouldNot(HaveOccurred())
				}
			}
			Eventually(func() error {
				ssc, err := virtClient.SecClient().SecurityContextConstraints().Get(context.Background(), "kubevirt-controller", metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return nil
				}
				if err == nil {
					return fmt.Errorf("The SSC exists %v", ssc)
				}

				return err
			}, 5*time.Minute, time.Minute).ShouldNot(HaveOccurred())

		})

		AfterEach(func() {
			if dissableFeature != nil {
				dissableFeature()
			}
		})

		shouldFailToCreate := func(vmi *v1.VirtualMachineInstance) {
			vmi = tests.RunVMI(vmi, 60)
			var err error

			refreshVMI := ThisVMI(vmi)
			EventuallyWithOffset(1, func() bool {
				vmi, err = refreshVMI()
				ExpectWithOffset(2, err).NotTo(HaveOccurred())

				for _, condition := range vmi.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceSynchronized {
						Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
						ExpectWithOffset(2, condition.Reason).To(Equal("FailedCreate"))
						ExpectWithOffset(2, condition.Message).To(ContainSubstring("security context constraint"))
						return true
					}
				}
				return false
			}, 30*time.Second, time.Second).Should(BeTrue())
		}

		type saCreator func() (string, func())

		createSaAndSCC := func(sscName string) func() (string, func()) {

			return func() (string, func()) {
				By("Creating testing SA")
				sa := &k8sv1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-correct-scc",
					},
				}
				sa, err := virtClient.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Create(context.TODO(), sa, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				saName := fmt.Sprintf("system:serviceaccount:%s:%s", sa.Namespace, sa.Name)
				scc, err := virtClient.SecClient().SecurityContextConstraints().Get(context.TODO(), sscName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				scc.Users = append(scc.Users, saName)
				scc, err = virtClient.SecClient().SecurityContextConstraints().Update(context.TODO(), scc, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				cleanup := func() {
					err := virtClient.CoreV1().ServiceAccounts(util.NamespaceTestDefault).Delete(context.TODO(), sa.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())

					scc, err := virtClient.SecClient().SecurityContextConstraints().Get(context.TODO(), sscName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					users := []string{}
					for _, user := range scc.Users {
						if user != saName {
							users = append(users, user)
						}
					}
					scc.Users = users
					scc, err = virtClient.SecClient().SecurityContextConstraints().Update(context.TODO(), scc, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
				return sa.Name, cleanup
			}
		}

		checkIfSCChasSA := func(sccName, saName string) {
			By("Checking if SA is in SCC")
			scc, err := virtClient.SecClient().SecurityContextConstraints().Get(context.Background(), sccName, metav1.GetOptions{})
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			saName = fmt.Sprintf("system:serviceaccount:%s:%s", util.NamespaceTestDefault, saName)
			ExpectWithOffset(1, scc.Users).To(ContainElement(saName))
		}

		It("multiple times should not remove users from SCCs", func() {
			By("Adding SA to SCC")
			sccName := "kubevirt-base"
			saName, removeSA := createSaAndSCC(sccName)()
			defer removeSA()

			checkIfSCChasSA(sccName, saName)

			By("Disable NoManagedSCC feature gate")
			tests.DisableFeatureGate(virtconfig.NoManagedSCC)

			checkIfSCChasSA(sccName, saName)

			By("Enable NoManagedSCC feature gate")
			tests.EnableFeatureGate(virtconfig.NoManagedSCC)

			checkIfSCChasSA(sccName, saName)
		})

		table.DescribeTable("testing CPU pinning VM", func(createSA saCreator, allowed bool) {
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				DedicatedCPUPlacement: true,
			}
			vmi.Spec.Hostname = "vmi"

			sa, removeSA := createSA()
			defer removeSA()
			vmi.Spec.ServiceAccountName = sa

			if allowed {
				tests.RunVMIAndExpectLaunch(vmi, 180)

				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
				return
			}
			shouldFailToCreate(vmi)
		},
			table.Entry("with base SCC should fail", createSaAndSCC("kubevirt-base"), false),
			table.FEntry("with cpu-pinning SCC should success", createSaAndSCC("kubevirt-cpu-pinning"), true),
			table.Entry("with cpu-pinning & host-disk SCC should success", createSaAndSCC("kubevirt-cpu-pinning-and-host-disk"), true),
			table.Entry("with host-disk SCC should fail", createSaAndSCC("kubevirt-host-disk"), false),
		)

		table.DescribeTable("Should succesfully create VMI", func(createSA saCreator) {
			vmi := libvmi.NewCirros()
			sa, removeSA := createSA()
			defer removeSA()
			vmi.Spec.ServiceAccountName = sa

			tests.RunVMIAndExpectLaunch(vmi, 180)

			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).NotTo(HaveOccurred())
			tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
		},
			table.Entry("with kubevirt-base SCC", createSaAndSCC("kubevirt-base")),
			table.Entry("with privileged SCC", createSaAndSCC("privileged")),
		)

		It("Should fail to create VMI without SA", func() {
			By("Checking operator deleted the SCC")
			checkHandlerSCC()

			By("Checking if virt-launcher is assigned to kubevirt-controller SCC")
			_, err := virtClient.SecClient().SecurityContextConstraints().Get(context.Background(), "kubevirt-controller", metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			shouldFailToCreate(vmi)
		})
	})

})
