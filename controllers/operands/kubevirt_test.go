package operands

import (
	"context"
	"fmt"
	"maps"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("KubeVirt Operand", func() {

	var (
		basicNumFgOnOpenshift = len(hardCodeKvFgs) + len(sspConditionKvFgs)
	)

	Context("KubeVirt Priority Classes", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewKubeVirtPriorityClass(hco)
			cl := commontestutils.InitClient([]client.Object{})
			handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &schedulingv1.PriorityClass{}
			Expect(cl.Get(context.TODO(), key, foundResource)).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Value).To(Equal(expectedResource.Value))
			Expect(foundResource.GlobalDefault).To(Equal(expectedResource.GlobalDefault))
		})

		It("should do nothing if already exists", func() {
			expectedResource := NewKubeVirtPriorityClass(hco)
			cl := commontestutils.InitClient([]client.Object{expectedResource})
			handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modifiedPC *schedulingv1.PriorityClass) {
			expectedPC := NewKubeVirtPriorityClass(hco)
			key := client.ObjectKeyFromObject(expectedPC)

			cl := commontestutils.InitClient([]client.Object{modifiedPC})

			origPC := &schedulingv1.PriorityClass{}
			Expect(cl.Get(context.TODO(), key, origPC)).To(Succeed())

			handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundPC := &schedulingv1.PriorityClass{}
			Expect(cl.Get(context.TODO(), key, foundPC)).To(Succeed())
			Expect(foundPC.Name).To(Equal(expectedPC.Name))
			Expect(foundPC.Value).To(Equal(expectedPC.Value))
			Expect(foundPC.GlobalDefault).To(Equal(expectedPC.GlobalDefault))
			Expect(foundPC.UID).ToNot(Equal(origPC.UID))

			newReference, err := reference.GetReference(cl.Scheme(), foundPC)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(ContainElement(*newReference))
		},
			Entry("with modified value",
				&schedulingv1.PriorityClass{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "scheduling.k8s.io/v1",
						Kind:       "PriorityClass",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: kvPriorityClass,
						UID:  "origPC",
					},
					Value:         1,
					GlobalDefault: false,
					Description:   "",
				}),
			Entry("with modified global default",
				&schedulingv1.PriorityClass{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "scheduling.k8s.io/v1",
						Kind:       "PriorityClass",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: kvPriorityClass,
						UID:  "origPC",
					},
					Value:         1000000000,
					GlobalDefault: true,
					Description:   "",
				}),
		)

		DescribeTable("should return error when there is something wrong", func(initiateErrors func(testClient *commontestutils.HcoTestClient) error) {
			modifiedResource := NewKubeVirtPriorityClass(hco)
			modifiedResource.Value = 1

			cl := commontestutils.InitClient([]client.Object{modifiedResource})
			expectedError := initiateErrors(cl)

			handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(Equal(expectedError))
		},
			Entry("creation error", func(testClient *commontestutils.HcoTestClient) error {
				expectedError := fmt.Errorf("fake PriorityClass creation error")
				testClient.InitiateCreateErrors(func(obj client.Object) error {
					if _, ok := obj.(*schedulingv1.PriorityClass); ok {
						return expectedError
					}
					return nil
				})
				return expectedError
			}),
			Entry("deletion error", func(testClient *commontestutils.HcoTestClient) error {
				expectedError := fmt.Errorf("fake PriorityClass deletion error")
				testClient.InitiateDeleteErrors(func(obj client.Object) error {
					if _, ok := obj.(*schedulingv1.PriorityClass); ok {
						return expectedError
					}
					return nil
				})

				return expectedError
			}),
			Entry("get error", func(testClient *commontestutils.HcoTestClient) error {
				expectedError := fmt.Errorf("fake PriorityClass get error")
				testClient.InitiateGetErrors(func(key client.ObjectKey) error {
					if key.Name == kvPriorityClass {
						return expectedError
					}
					return nil
				})

				return expectedError
			}),
		)

		Context("check labels", func() {
			const origUID = types.UID("origPC")
			It("should add missing labels", func(ctx context.Context) {
				expectedResource := NewKubeVirtPriorityClass(hco)
				expectedResource.UID = origUID
				delete(expectedResource.Labels, hcoutil.AppLabelComponent)

				cl := commontestutils.InitClient([]client.Object{expectedResource})
				handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundPC := schedulingv1.PriorityClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: kvPriorityClass,
					},
				}

				Expect(cl.Get(ctx, client.ObjectKeyFromObject(&foundPC), &foundPC)).To(Succeed())
				Expect(foundPC.Labels).To(HaveKeyWithValue(hcoutil.AppLabelComponent, string(hcoutil.AppComponentCompute)))
				Expect(foundPC.UID).To(Equal(origUID))
			})

			It("should fix wrong labels", func(ctx context.Context) {
				expectedResource := NewKubeVirtPriorityClass(hco)
				expectedResource.UID = "origPC"
				expectedResource.Labels[hcoutil.AppLabelComponent] = string(hcoutil.AppComponentStorage)

				cl := commontestutils.InitClient([]client.Object{expectedResource})
				handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundPC := schedulingv1.PriorityClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: kvPriorityClass,
					},
				}

				Expect(cl.Get(ctx, client.ObjectKeyFromObject(&foundPC), &foundPC)).To(Succeed())
				Expect(foundPC.Labels).To(HaveKeyWithValue(hcoutil.AppLabelComponent, string(hcoutil.AppComponentCompute)))
				Expect(foundPC.UID).To(Equal(origUID))
			})

			It("should keep user-defined labels", func(ctx context.Context) {
				const customLabel = "custom-label"
				expectedResource := NewKubeVirtPriorityClass(hco)
				expectedResource.Labels[customLabel] = "test"
				expectedResource.Labels[hcoutil.AppLabelComponent] = string(hcoutil.AppComponentStorage)
				expectedResource.UID = "origPC"

				cl := commontestutils.InitClient([]client.Object{expectedResource})
				handler := (*genericOperand)(newKvPriorityClassHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundPC := schedulingv1.PriorityClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: kvPriorityClass,
					},
				}

				Expect(cl.Get(ctx, client.ObjectKeyFromObject(&foundPC), &foundPC)).To(Succeed())
				Expect(foundPC.Labels).To(HaveKeyWithValue(customLabel, "test"))
				Expect(foundPC.Labels).To(HaveKeyWithValue(hcoutil.AppLabelComponent, string(hcoutil.AppComponentCompute)))
				Expect(foundPC.UID).To(Equal(origUID))
			})
		})

	})

	Context("KubeVirt", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
			os.Setenv(smbiosEnvName,
				`Family: smbios family
Product: smbios product
Manufacturer: smbios manufacturer
Sku: 1.2.3
Version: 1.2.3`)

			os.Setenv(amd64MachineTypeEnvName, "q35")
			os.Setenv(arm64MachineTypeEnvName, "virt")
			os.Setenv(kvmEmulationEnvName, "false")

			DeferCleanup(func() {
				os.Unsetenv(smbiosEnvName)
				os.Unsetenv(machineTypeEnvName)
				os.Unsetenv(amd64MachineTypeEnvName)
				os.Unsetenv(arm64MachineTypeEnvName)
				os.Unsetenv(kvmEmulationEnvName)
			})
		})

		It("should create if not present", func() {
			mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
			hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
				DownwardMetrics: ptr.To(true),
			}
			bindingPlugins := map[string]kubevirtcorev1.InterfaceBindingPlugin{
				"binding1": {SidecarImage: "image1", NetworkAttachmentDefinition: "nad1"},
				"l2bridge": {Migration: &kubevirtcorev1.InterfaceBindingMigration{}, DomainAttachmentType: "managedTap"},
			}
			hco.Spec.NetworkBinding = bindingPlugins

			expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).To(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))

			Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(basicNumFgOnOpenshift + 1))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(
				hardCodeKvFgs,
			))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(
				sspConditionKvFgs,
			))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement(
				kvDownwardMetrics,
			))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.DiskVerification).ToNot(BeNil())
			Expect(*foundResource.Spec.Configuration.DeveloperConfiguration.DiskVerification.MemoryLimit).To(Equal(kvDiskVerificationMemoryLimit))

			Expect(foundResource.Spec.Configuration.MachineType).To(BeEmpty())
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Amd64.MachineType).To(Equal("q35"))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Amd64.OVMFPath).To(Equal(DefaultAMD64OVMFPath))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Arm64.MachineType).To(Equal("virt"))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Arm64.OVMFPath).To(Equal(DefaultARM64OVMFPath))

			Expect(foundResource.Spec.Configuration.SMBIOSConfig).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Family).To(Equal("smbios family"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Product).To(Equal("smbios product"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Manufacturer).To(Equal("smbios manufacturer"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Sku).To(Equal("1.2.3"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Version).To(Equal("1.2.3"))

			Expect(foundResource.Spec.Configuration.NetworkConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.NetworkConfiguration.NetworkInterface).To(Equal(string(kubevirtcorev1.MasqueradeInterface)))
			Expect(foundResource.Spec.Configuration.NetworkConfiguration.Binding).To(Equal(bindingPlugins))

			// LiveMigration Configurations
			mc := foundResource.Spec.Configuration.MigrationConfiguration
			Expect(mc).ToNot(BeNil())
			Expect(mc.BandwidthPerMigration).To(BeNil())
			Expect(*mc.CompletionTimeoutPerGiB).To(Equal(int64(150)))
			Expect(*mc.ParallelMigrationsPerCluster).To(Equal(uint32(5)))
			Expect(*mc.ParallelOutboundMigrationsPerNode).To(Equal(uint32(2)))
			Expect(*mc.ProgressTimeout).To(Equal(int64(150)))
			Expect(mc.Network).To(BeNil())
			Expect(*mc.AllowAutoConverge).To(BeFalse())
			Expect(*mc.AllowPostCopy).To(BeFalse())
		})

		It("should find if present", func() {
			expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "KubeVirtConditions",
				Message: "KubeVirt resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "KubeVirtConditions",
				Message: "KubeVirt resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "KubeVirtConditions",
				Message: "KubeVirt resource has no conditions",
			}))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewKubeVirt(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, "app.kubernetes.io/version")

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default on label deletion without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewKubeVirt(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should force mandatory configurations", func() {
			mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
			hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
				DownwardMetrics: ptr.To(true),
			}

			os.Setenv(smbiosEnvName,
				`Family: smbios family
Product: smbios product
Manufacturer: smbios manufacturer
Sku: 1.2.3
Version: 1.2.3`)
			os.Setenv(amd64MachineTypeEnvName, "q35")
			os.Setenv(arm64MachineTypeEnvName, "virt")

			existKv, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			existKv.Spec.Configuration.DeveloperConfiguration = &kubevirtcorev1.DeveloperConfiguration{
				FeatureGates: []string{"wrongFG1", "wrongFG2", "wrongFG3"},
			}
			existKv.Spec.Configuration.ArchitectureConfiguration = &kubevirtcorev1.ArchConfiguration{
				Amd64: &kubevirtcorev1.ArchSpecificConfiguration{
					MachineType: "wrong amd64 machine type",
				},
				Arm64: &kubevirtcorev1.ArchSpecificConfiguration{
					MachineType: "wrong arm64 machine type",
				},
			}
			existKv.Spec.Configuration.SMBIOSConfig = &kubevirtcorev1.SMBiosConfiguration{
				Family:       "wrong family",
				Product:      "wrong product",
				Manufacturer: "wrong manifaturer",
				Sku:          "0.0.0",
				Version:      "1.1.1",
			}
			existKv.Spec.Configuration.SELinuxLauncherType = "wrongSELinuxLauncherType"
			existKv.Spec.Configuration.NetworkConfiguration = &kubevirtcorev1.NetworkConfiguration{
				NetworkInterface: "wrong network interface",
			}
			existKv.Spec.Configuration.EmulatedMachines = []string{"wrong"}

			// LiveMigration Configurations
			existKv.Spec.Configuration.MigrationConfiguration = &kubevirtcorev1.MigrationConfiguration{
				BandwidthPerMigration:             ptr.To(resource.MustParse("16Mi")),
				CompletionTimeoutPerGiB:           ptr.To[int64](0),
				ParallelMigrationsPerCluster:      ptr.To[uint32](0),
				ParallelOutboundMigrationsPerNode: ptr.To[uint32](0),
				ProgressTimeout:                   ptr.To[int64](0),
				Network:                           ptr.To("testNetwork"),
				AllowAutoConverge:                 ptr.To(false),
				AllowPostCopy:                     ptr.To(false),
			}

			cl := commontestutils.InitClient([]client.Object{hco, existKv})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)

			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(basicNumFgOnOpenshift + 1))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(
				hardCodeKvFgs,
			))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(
				sspConditionKvFgs,
			))
			Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement(
				kvDownwardMetrics,
			))

			Expect(foundResource.Spec.Configuration.MachineType).To(BeEmpty())
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Amd64.MachineType).To(Equal("q35"))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Amd64.OVMFPath).To(Equal(DefaultAMD64OVMFPath))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Arm64.MachineType).To(Equal("virt"))
			Expect(foundResource.Spec.Configuration.ArchitectureConfiguration.Arm64.OVMFPath).To(Equal(DefaultARM64OVMFPath))

			Expect(foundResource.Spec.Configuration.SMBIOSConfig).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Family).To(Equal("smbios family"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Product).To(Equal("smbios product"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Manufacturer).To(Equal("smbios manufacturer"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Sku).To(Equal("1.2.3"))
			Expect(foundResource.Spec.Configuration.SMBIOSConfig.Version).To(Equal("1.2.3"))

			Expect(foundResource.Spec.Configuration.NetworkConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.Configuration.NetworkConfiguration.NetworkInterface).To(Equal(string(kubevirtcorev1.MasqueradeInterface)))

			Expect(foundResource.Spec.Configuration.EmulatedMachines).To(BeEmpty())

			// LiveMigration Configurations
			mc := foundResource.Spec.Configuration.MigrationConfiguration
			Expect(mc).ToNot(BeNil())
			Expect(mc.BandwidthPerMigration).To(BeNil())
			Expect(*mc.CompletionTimeoutPerGiB).To(Equal(int64(150)))
			Expect(*mc.ParallelMigrationsPerCluster).To(Equal(uint32(5)))
			Expect(*mc.ParallelOutboundMigrationsPerNode).To(Equal(uint32(2)))
			Expect(*mc.ProgressTimeout).To(Equal(int64(150)))
			Expect(mc.Network).To(BeNil())
			Expect(*mc.AllowAutoConverge).To(BeFalse())
			Expect(*mc.AllowPostCopy).To(BeFalse())
		})

		It("should use legacy MACHINETYPE env if provided", func() {
			os.Setenv(machineTypeEnvName, "legacy")
			os.Setenv(amd64MachineTypeEnvName, "q35")
			os.Unsetenv(arm64MachineTypeEnvName)

			kv, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())

			Expect(kv.Spec.Configuration.MachineType).To(BeEmpty())
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Amd64.MachineType).To(Equal("legacy"))
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Amd64.OVMFPath).To(Equal(DefaultAMD64OVMFPath))
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Arm64).To(BeNil())
		})

		It("should not use legacy MACHINETYPE env if empty", func() {
			os.Setenv(machineTypeEnvName, "")
			os.Setenv(amd64MachineTypeEnvName, "q35")
			os.Unsetenv(arm64MachineTypeEnvName)

			kv, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())

			Expect(kv.Spec.Configuration.MachineType).To(BeEmpty())
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Amd64.MachineType).To(Equal("q35"))
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Amd64.OVMFPath).To(Equal(DefaultAMD64OVMFPath))
			Expect(kv.Spec.Configuration.ArchitectureConfiguration.Arm64).To(BeNil())
		})

		It("should fail if the SMBIOS is wrongly formatted mandatory configurations", func() {
			hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
				WithHostPassthroughCPU: ptr.To(true),
			}

			_ = os.Setenv(smbiosEnvName, "WRONG YAML")

			_, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the Spec.LiveMigrationConfig.BandwidthPerMigration is wrongly formatted", func() {
			hco.Spec.LiveMigrationConfig.BandwidthPerMigration = ptr.To("Wrong Format")

			_, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).To(HaveOccurred())
		})

		It("should set default UninstallStrategy if missing", func() {
			expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			missingUSResource, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			missingUSResource.Spec.UninstallStrategy = ""

			cl := commontestutils.InitClient([]client.Object{hco, missingUSResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Spec.UninstallStrategy).To(Equal(expectedResource.Spec.UninstallStrategy))
		})

		Context("Test UninstallStrategy", func() {

			It("should set BlockUninstallIfWorkloadsExist if missing HCO CR", func() {
				expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())
				hco.Spec.UninstallStrategy = ""

				cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.UninstallStrategy).ToNot(BeNil())
				Expect(foundResource.Spec.UninstallStrategy).To(Equal(kubevirtcorev1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist))
			})

			It("should set BlockUninstallIfWorkloadsExist if set on HCO CR", func() {
				expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())
				uninstallStrategy := hcov1beta1.HyperConvergedUninstallStrategyBlockUninstallIfWorkloadsExist
				hco.Spec.UninstallStrategy = uninstallStrategy

				cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.UninstallStrategy).ToNot(BeNil())
				Expect(foundResource.Spec.UninstallStrategy).To(Equal(kubevirtcorev1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist))
			})

			It("should set BlockUninstallIfRemoveWorkloads if set on HCO CR", func() {
				expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
				Expect(err).ToNot(HaveOccurred())
				uninstallStrategy := hcov1beta1.HyperConvergedUninstallStrategyRemoveWorkloads
				hco.Spec.UninstallStrategy = uninstallStrategy

				cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.UninstallStrategy).ToNot(BeNil())
				Expect(foundResource.Spec.UninstallStrategy).To(Equal(kubevirtcorev1.KubeVirtUninstallStrategyRemoveWorkloads))
			})

		})

		It("should propagate the live migration configuration from the HC", func() {
			existKv, err := NewKubeVirt(hco)
			Expect(err).ToNot(HaveOccurred())

			const (
				bandwidthPerMigration             = "16Mi"
				completionTimeoutPerGiB           = int64(100)
				parallelOutboundMigrationsPerNode = uint32(7)
				parallelMigrationsPerCluster      = uint32(18)
				progressTimeout                   = int64(5000)
				network                           = "testNetwork"
			)

			hco.Spec.LiveMigrationConfig.BandwidthPerMigration = ptr.To(bandwidthPerMigration)
			hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = ptr.To(completionTimeoutPerGiB)
			hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = ptr.To(parallelOutboundMigrationsPerNode)
			hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = ptr.To(parallelMigrationsPerCluster)
			hco.Spec.LiveMigrationConfig.ProgressTimeout = ptr.To(progressTimeout)
			hco.Spec.LiveMigrationConfig.Network = ptr.To(network)
			hco.Spec.LiveMigrationConfig.AllowAutoConverge = ptr.To(true)
			hco.Spec.LiveMigrationConfig.AllowPostCopy = ptr.To(true)

			cl := commontestutils.InitClient([]client.Object{hco, existKv})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)

			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			mc := foundResource.Spec.Configuration.MigrationConfiguration
			Expect(mc).ToNot(BeNil())
			Expect(mc.BandwidthPerMigration).To(HaveValue(Equal(resource.MustParse(bandwidthPerMigration))))
			Expect(mc.CompletionTimeoutPerGiB).To(HaveValue(Equal(completionTimeoutPerGiB)))
			Expect(mc.ParallelOutboundMigrationsPerNode).To(HaveValue(Equal(parallelOutboundMigrationsPerNode)))
			Expect(mc.ParallelMigrationsPerCluster).To(HaveValue(Equal(parallelMigrationsPerCluster)))
			Expect(mc.ProgressTimeout).To(HaveValue(Equal(progressTimeout)))
			Expect(mc.Network).To(HaveValue(Equal(network)))
			Expect(mc.AllowAutoConverge).To(HaveValue(BeTrue()))
			Expect(mc.AllowPostCopy).To(HaveValue(BeTrue()))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existKv)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

		})

		Context("test mediated device configuration", func() {
			It("should propagate the mediated devices configuration from the HC with deprecated APIs", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				mdevConf := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdevConf).ToNot(BeNil())
				Expect(mdevConf.MediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.MediatedDeviceTypes).To(ContainElements("nvidia-222", "nvidia-230"))
				Expect(mdevConf.MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019

			})

			It("should propagate the mediated devices configuration from the HC - mediatedDeviceTypes", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				mdevConf := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdevConf).ToNot(BeNil())
				Expect(mdevConf.MediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.MediatedDeviceTypes).To(ContainElements("nvidia-222", "nvidia-230"))
				Expect(mdevConf.MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019

			})

			It("should propagate the mediated devices configuration from the HC with node selectors with deprecated APIs", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
					NodeMediatedDeviceTypes: []hcov1beta1.NodeMediatedDeviceTypesConfig{
						{
							NodeSelector: map[string]string{
								"testLabel1": "true",
							},
							MediatedDevicesTypes: []string{ //nolint SA1019
								"nvidia-223",
							},
						},
						{
							NodeSelector: map[string]string{
								"testLabel2": "true",
							},
							MediatedDevicesTypes: []string{ //nolint SA1019
								"nvidia-229",
							},
						},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				mdevConf := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdevConf).ToNot(BeNil())
				Expect(mdevConf.MediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.MediatedDeviceTypes).To(ContainElements("nvidia-222", "nvidia-230"))
				Expect(mdevConf.MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019
				Expect(mdevConf.NodeMediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].MediatedDeviceTypes).To(ContainElements("nvidia-223"))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].NodeSelector).To(HaveKeyWithValue("testLabel1", "true"))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019
				Expect(mdevConf.NodeMediatedDeviceTypes[1].MediatedDeviceTypes).To(ContainElements("nvidia-229"))
				Expect(mdevConf.NodeMediatedDeviceTypes[1].NodeSelector).To(HaveKeyWithValue("testLabel2", "true"))
				Expect(mdevConf.NodeMediatedDeviceTypes[1].MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019

			})
			It("should propagate the mediated devices configuration from the HC with node selectors - mediatedDeviceTypes", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
					NodeMediatedDeviceTypes: []hcov1beta1.NodeMediatedDeviceTypesConfig{
						{
							NodeSelector: map[string]string{
								"testLabel1": "true",
							},
							MediatedDeviceTypes: []string{
								"nvidia-223",
							},
						},
						{
							NodeSelector: map[string]string{
								"testLabel2": "true",
							},
							MediatedDeviceTypes: []string{
								"nvidia-229",
							},
						},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				mdevConf := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdevConf).ToNot(BeNil())
				Expect(mdevConf.MediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.MediatedDeviceTypes).To(ContainElements("nvidia-222", "nvidia-230"))
				Expect(mdevConf.MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019
				Expect(mdevConf.NodeMediatedDeviceTypes).To(HaveLen(2))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].MediatedDeviceTypes).To(ContainElements("nvidia-223"))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].NodeSelector).To(HaveKeyWithValue("testLabel1", "true"))
				Expect(mdevConf.NodeMediatedDeviceTypes[0].MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019
				Expect(mdevConf.NodeMediatedDeviceTypes[1].MediatedDeviceTypes).To(ContainElements("nvidia-229"))
				Expect(mdevConf.NodeMediatedDeviceTypes[1].NodeSelector).To(HaveKeyWithValue("testLabel2", "true"))
				Expect(mdevConf.NodeMediatedDeviceTypes[1].MediatedDevicesTypes).To(BeEmpty()) //nolint SA1019

			})
			It("should update the permitted host devices configuration from the HC - mediatedDeviceTypes", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				existKv.Spec.Configuration.MediatedDevicesConfiguration = &kubevirtcorev1.MediatedDevicesConfiguration{
					MediatedDeviceTypes: []string{"nvidia-222", "nvidia-230"},
				}

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDeviceTypes: []string{"nvidia-181", "nvidia-191", "nvidia-224"},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})

				By("Check before reconciling", func() {
					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
							foundResource),
					).To(Succeed())

					mdc := foundResource.Spec.Configuration.MediatedDevicesConfiguration
					Expect(mdc).ToNot(BeNil())
					Expect(mdc.MediatedDeviceTypes).To(HaveLen(2))
					Expect(mdc.MediatedDeviceTypes).To(ContainElements("nvidia-222", "nvidia-230"))

				})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				By("Check after reconcile")
				mdc := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdc).ToNot(BeNil())
				Expect(mdc.MediatedDeviceTypes).To(HaveLen(3))
				Expect(mdc.MediatedDeviceTypes).To(ContainElements("nvidia-181", "nvidia-191", "nvidia-224"))
			})

			It("should update the permitted host devices configuration from the HC migrating to mediatedDeviceTypes", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				existKv.Spec.Configuration.MediatedDevicesConfiguration = &kubevirtcorev1.MediatedDevicesConfiguration{
					MediatedDevicesTypes: []string{"nvidia-222", "nvidia-230"}, //nolint SA1019
				}

				hco.Spec.MediatedDevicesConfiguration = &hcov1beta1.MediatedDevicesConfiguration{
					MediatedDevicesTypes: []string{"nvidia-181", "nvidia-191", "nvidia-224"}, //nolint SA1019
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})

				By("Check before reconciling", func() {
					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
							foundResource),
					).To(Succeed())

					mdc := foundResource.Spec.Configuration.MediatedDevicesConfiguration
					Expect(mdc).ToNot(BeNil())
					Expect(mdc.MediatedDevicesTypes).To(HaveLen(2))                                  //nolint SA1019
					Expect(mdc.MediatedDevicesTypes).To(ContainElements("nvidia-222", "nvidia-230")) //nolint SA1019

				})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				By("Check after reconcile")
				mdc := foundResource.Spec.Configuration.MediatedDevicesConfiguration
				Expect(mdc).ToNot(BeNil())
				Expect(mdc.MediatedDeviceTypes).To(HaveLen(3))
				Expect(mdc.MediatedDeviceTypes).To(ContainElements("nvidia-181", "nvidia-191", "nvidia-224"))
			})
		})

		Context("test permitted host devices", func() {
			It("should propagate the permitted host devices configuration from the HC", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.PermittedHostDevices = &hcov1beta1.PermittedHostDevices{
					PciHostDevices: []hcov1beta1.PciHostDevice{
						{
							PCIDeviceSelector:        "vendor1",
							ResourceName:             "resourceName1",
							ExternalResourceProvider: true,
						},
						{
							PCIDeviceSelector:        "vendor2",
							ResourceName:             "resourceName2",
							ExternalResourceProvider: false,
						},
						{
							PCIDeviceSelector:        "vendor3",
							ResourceName:             "resourceName3",
							ExternalResourceProvider: true,
							Disabled:                 false,
						},
						{
							PCIDeviceSelector:        "disabled4",
							ResourceName:             "disabled4",
							ExternalResourceProvider: true,
							Disabled:                 true,
						},
					},
					MediatedDevices: []hcov1beta1.MediatedHostDevice{
						{
							MDEVNameSelector:         "selector1",
							ResourceName:             "resource1",
							ExternalResourceProvider: true,
						},
						{
							MDEVNameSelector:         "selector2",
							ResourceName:             "resource2",
							ExternalResourceProvider: false,
						},
						{
							MDEVNameSelector:         "selector3",
							ResourceName:             "resource3",
							ExternalResourceProvider: true,
						},
						{
							MDEVNameSelector:         "selector4",
							ResourceName:             "resource4",
							ExternalResourceProvider: false,
							Disabled:                 false,
						},
						{
							MDEVNameSelector:         "disabled5",
							ResourceName:             "disabled5",
							ExternalResourceProvider: false,
							Disabled:                 true,
						},
					},
					USBHostDevices: []hcov1beta1.USBHostDevice{
						{
							ResourceName: "kubevirt.io/usbstorage",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0001",
								},
							},
							ExternalResourceProvider: false,
						},
						{
							ResourceName: "kubevirt.io/usbstorageerp",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0002",
								},
							},
							ExternalResourceProvider: true,
						},
						{
							ResourceName: "kubevirt.io/peripherals",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "045e",
									Product: "07a5",
								},
								{
									Vendor:  "062a",
									Product: "4102",
								},
								{
									Vendor:  "072f",
									Product: "07a5",
								},
								{
									Vendor:  "045e",
									Product: "b100",
								},
							},
							ExternalResourceProvider: true,
						},
						{
							ResourceName: "kubevirt.io/usbstoragedisabled",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0003",
								},
							},
							ExternalResourceProvider: false,
							Disabled:                 true,
						},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				phd := foundResource.Spec.Configuration.PermittedHostDevices
				Expect(phd).ToNot(BeNil())
				Expect(phd.PciHostDevices).To(HaveLen(3))
				Expect(phd.PciHostDevices).To(ContainElements(
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor1",
						ResourceName:             "resourceName1",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor2",
						ResourceName:             "resourceName2",
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor3",
						ResourceName:             "resourceName3",
						ExternalResourceProvider: true,
					},
				))

				Expect(phd.MediatedDevices).To(HaveLen(4))
				Expect(phd.MediatedDevices).To(ContainElements(
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector1",
						ResourceName:             "resource1",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector2",
						ResourceName:             "resource2",
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector3",
						ResourceName:             "resource3",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector4",
						ResourceName:             "resource4",
						ExternalResourceProvider: false,
					},
				))

				Expect(phd.USB).To(HaveLen(3))
				Expect(phd.USB).To(ContainElements(
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/usbstorage",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "46f4", Product: "0001"},
						},
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/usbstorageerp",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "46f4", Product: "0002"},
						},
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/peripherals",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "045e", Product: "07a5"},
							{Vendor: "062a", Product: "4102"},
							{Vendor: "072f", Product: "07a5"},
							{Vendor: "045e", Product: "b100"},
						},
						ExternalResourceProvider: true,
					},
				))

			})

			It("should update the permitted host devices configuration from the HC", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				existKv.Spec.Configuration.PermittedHostDevices = &kubevirtcorev1.PermittedHostDevices{
					PciHostDevices: []kubevirtcorev1.PciHostDevice{
						{
							PCIVendorSelector:        "other1",
							ResourceName:             "otherResourceName1",
							ExternalResourceProvider: true,
						},
						{
							PCIVendorSelector:        "other2",
							ResourceName:             "otherResourceName2",
							ExternalResourceProvider: false,
						},
						{
							PCIVendorSelector:        "other3",
							ResourceName:             "otherResourceName3",
							ExternalResourceProvider: true,
						},
						{
							PCIVendorSelector:        "other4",
							ResourceName:             "otherResourceName4",
							ExternalResourceProvider: true,
						},
					},
					MediatedDevices: []kubevirtcorev1.MediatedHostDevice{
						{
							MDEVNameSelector:         "otherSelector1",
							ResourceName:             "otherResource1",
							ExternalResourceProvider: false,
						},
						{
							MDEVNameSelector:         "otherSelector2",
							ResourceName:             "otherResource2",
							ExternalResourceProvider: true,
						},
						{
							MDEVNameSelector:         "otherSelector3",
							ResourceName:             "otherResource3",
							ExternalResourceProvider: true,
						},
					},
					USB: []kubevirtcorev1.USBHostDevice{
						{
							ResourceName: "otherUSBResource1",
							Selectors: []kubevirtcorev1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0001",
								},
							},
							ExternalResourceProvider: false,
						},
						{
							ResourceName: "otherUSBResource2",
							Selectors: []kubevirtcorev1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0002",
								},
							},
							ExternalResourceProvider: true,
						},
						{
							ResourceName: "otherUSBResource3",
							Selectors: []kubevirtcorev1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0003",
								},
								{
									Vendor:  "46f4",
									Product: "0004",
								},
							},
							ExternalResourceProvider: false,
						},
					},
				}

				hco.Spec.PermittedHostDevices = &hcov1beta1.PermittedHostDevices{
					PciHostDevices: []hcov1beta1.PciHostDevice{
						{
							PCIDeviceSelector:        "vendor1",
							ResourceName:             "resourceName1",
							ExternalResourceProvider: true,
						},
						{
							PCIDeviceSelector:        "vendor2",
							ResourceName:             "resourceName2",
							ExternalResourceProvider: false,
						},
						{
							PCIDeviceSelector:        "vendor3",
							ResourceName:             "resourceName3",
							ExternalResourceProvider: true,
							Disabled:                 false,
						},
						{
							PCIDeviceSelector:        "disabled4",
							ResourceName:             "disabled4",
							ExternalResourceProvider: true,
							Disabled:                 true,
						},
					},
					MediatedDevices: []hcov1beta1.MediatedHostDevice{
						{
							MDEVNameSelector:         "selector1",
							ResourceName:             "resource1",
							ExternalResourceProvider: true,
						},
						{
							MDEVNameSelector:         "selector2",
							ResourceName:             "resource2",
							ExternalResourceProvider: false,
						},
						{
							MDEVNameSelector:         "selector3",
							ResourceName:             "resource3",
							ExternalResourceProvider: true,
						},
						{
							MDEVNameSelector:         "selector4",
							ResourceName:             "resource4",
							ExternalResourceProvider: false,
							Disabled:                 false,
						},
						{
							MDEVNameSelector:         "disabled5",
							ResourceName:             "disabled5",
							ExternalResourceProvider: false,
							Disabled:                 true,
						},
					},
					USBHostDevices: []hcov1beta1.USBHostDevice{
						{
							ResourceName: "kubevirt.io/usbstorage",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0001",
								},
							},
							ExternalResourceProvider: false,
						},
						{
							ResourceName: "kubevirt.io/usbstorageerp",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0002",
								},
							},
							ExternalResourceProvider: true,
						},
						{
							ResourceName: "kubevirt.io/peripherals",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "045e",
									Product: "07a5",
								},
								{
									Vendor:  "062a",
									Product: "4102",
								},
								{
									Vendor:  "072f",
									Product: "07a5",
								},
								{
									Vendor:  "045e",
									Product: "b100",
								},
							},
							ExternalResourceProvider: true,
						},
						{
							ResourceName: "kubevirt.io/usbstoragedisabled",
							Selectors: []hcov1beta1.USBSelector{
								{
									Vendor:  "46f4",
									Product: "0003",
								},
							},
							ExternalResourceProvider: false,
							Disabled:                 true,
						},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existKv})

				By("Check before reconciling", func() {
					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
							foundResource),
					).To(Succeed())

					phd := foundResource.Spec.Configuration.PermittedHostDevices
					Expect(phd).ToNot(BeNil())
					Expect(phd.PciHostDevices).To(HaveLen(4))
					Expect(phd.PciHostDevices).To(ContainElements(
						kubevirtcorev1.PciHostDevice{
							PCIVendorSelector:        "other1",
							ResourceName:             "otherResourceName1",
							ExternalResourceProvider: true,
						},
						kubevirtcorev1.PciHostDevice{
							PCIVendorSelector:        "other2",
							ResourceName:             "otherResourceName2",
							ExternalResourceProvider: false,
						},
						kubevirtcorev1.PciHostDevice{
							PCIVendorSelector:        "other3",
							ResourceName:             "otherResourceName3",
							ExternalResourceProvider: true,
						},
						kubevirtcorev1.PciHostDevice{
							PCIVendorSelector:        "other4",
							ResourceName:             "otherResourceName4",
							ExternalResourceProvider: true,
						},
					))

					Expect(phd.MediatedDevices).To(HaveLen(3))
					Expect(phd.MediatedDevices).To(ContainElements(
						kubevirtcorev1.MediatedHostDevice{
							MDEVNameSelector:         "otherSelector1",
							ResourceName:             "otherResource1",
							ExternalResourceProvider: false,
						},
						kubevirtcorev1.MediatedHostDevice{
							MDEVNameSelector:         "otherSelector2",
							ResourceName:             "otherResource2",
							ExternalResourceProvider: true,
						},
						kubevirtcorev1.MediatedHostDevice{
							MDEVNameSelector:         "otherSelector3",
							ResourceName:             "otherResource3",
							ExternalResourceProvider: true,
						},
					))

					Expect(phd.USB).To(HaveLen(3))
					Expect(phd.USB).To(ContainElements(
						kubevirtcorev1.USBHostDevice{
							ResourceName: "otherUSBResource1",
							Selectors: []kubevirtcorev1.USBSelector{
								{Vendor: "46f4", Product: "0001"},
							},
							ExternalResourceProvider: false,
						},
						kubevirtcorev1.USBHostDevice{
							ResourceName: "otherUSBResource2",
							Selectors: []kubevirtcorev1.USBSelector{
								{Vendor: "46f4", Product: "0002"},
							},
							ExternalResourceProvider: true,
						},
						kubevirtcorev1.USBHostDevice{
							ResourceName: "otherUSBResource3",
							Selectors: []kubevirtcorev1.USBSelector{
								{Vendor: "46f4", Product: "0003"},
								{Vendor: "46f4", Product: "0004"},
							},
							ExternalResourceProvider: false,
						},
					))

				})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				By("Check after reconcile")
				phd := foundResource.Spec.Configuration.PermittedHostDevices
				Expect(phd).ToNot(BeNil())
				Expect(phd.PciHostDevices).To(HaveLen(3))
				Expect(phd.PciHostDevices).To(ContainElements(
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor1",
						ResourceName:             "resourceName1",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor2",
						ResourceName:             "resourceName2",
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.PciHostDevice{
						PCIVendorSelector:        "vendor3",
						ResourceName:             "resourceName3",
						ExternalResourceProvider: true,
					},
				))

				Expect(phd.MediatedDevices).To(HaveLen(4))
				Expect(phd.MediatedDevices).To(ContainElements(
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector1",
						ResourceName:             "resource1",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector2",
						ResourceName:             "resource2",
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector3",
						ResourceName:             "resource3",
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.MediatedHostDevice{
						MDEVNameSelector:         "selector4",
						ResourceName:             "resource4",
						ExternalResourceProvider: false,
					},
				))

				Expect(phd.USB).To(HaveLen(3))
				Expect(phd.USB).To(ContainElements(
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/usbstorage",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "46f4", Product: "0001"},
						},
						ExternalResourceProvider: false,
					},
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/usbstorageerp",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "46f4", Product: "0002"},
						},
						ExternalResourceProvider: true,
					},
					kubevirtcorev1.USBHostDevice{
						ResourceName: "kubevirt.io/peripherals",
						Selectors: []kubevirtcorev1.USBSelector{
							{Vendor: "045e", Product: "07a5"},
							{Vendor: "062a", Product: "4102"},
							{Vendor: "072f", Product: "07a5"},
							{Vendor: "045e", Product: "b100"},
						},
						ExternalResourceProvider: true,
					},
				))

			})
		})

		Context("test CPUModel", func() {

			It("should propagate the CPUModel from the HC if set", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				testCPUModel := "testValue"
				hco.Spec.DefaultCPUModel = ptr.To(testCPUModel)

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				kvCPUModel := foundResource.Spec.Configuration.CPUModel
				Expect(kvCPUModel).ToNot(BeEmpty())
				Expect(kvCPUModel).To(Equal(testCPUModel))

			})

			It("should not propagate the CPUModel from the HC if not set", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.DefaultCPUModel = nil

				cl := commontestutils.InitClient([]client.Object{hco, existKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				kvCPUModel := foundResource.Spec.Configuration.CPUModel
				Expect(kvCPUModel).To(BeEmpty())

			})

			It("should update the CPUModel from the HC", func() {
				existKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				const (
					oldKVCPUmodel = "oldKVCPUmodel"
					testCPUModel  = "testValue"
				)

				existKv.Spec.Configuration.CPUModel = oldKVCPUmodel

				hco.Spec.DefaultCPUModel = ptr.To(testCPUModel)

				cl := commontestutils.InitClient([]client.Object{hco, existKv})

				By("Check before reconciling", func() {
					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
							foundResource),
					).To(Succeed())

					kvCPUModel := foundResource.Spec.Configuration.CPUModel
					Expect(kvCPUModel).ToNot(BeNil())
					Expect(kvCPUModel).To(Equal(oldKVCPUmodel))
				})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)

				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				By("Check after reconciling", func() {
					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existKv.Name, Namespace: existKv.Namespace},
							foundResource),
					).To(Succeed())

					Expect(foundResource.Spec.Configuration.CPUModel).To(Equal(testCPUModel))
				})

			})

		})

		Context("Test node placement", func() {

			getClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
					return &commontestutils.ClusterInfoMock{}
				}
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			It("should add node placement if missing in KubeVirt", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Infra).To(BeNil())
				Expect(existingResource.Spec.Workloads).To(BeNil())

				Expect(foundResource.Spec.Infra).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodePlacement).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodePlacement.Affinity).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("value1"))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key2"]).To(Equal("value2"))

				Expect(foundResource.Spec.Workloads).ToNot(BeNil())
				Expect(foundResource.Spec.Workloads.NodePlacement).ToNot(BeNil())
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(Equal(hco.Spec.Workloads.NodePlacement.Tolerations))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should remove node placement if missing in HCO CR", func() {
				hcoNodePlacement := commontestutils.NewHco()
				hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				existingResource, err := NewKubeVirt(hcoNodePlacement)
				Expect(err).ToNot(HaveOccurred())

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Infra).ToNot(BeNil())
				Expect(existingResource.Spec.Workloads).ToNot(BeNil())

				Expect(foundResource.Spec.Infra).To(BeNil())
				Expect(foundResource.Spec.Workloads).To(BeNil())

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify node placement according to HCO CR", func() {
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				// now, modify HCO's node placement
				hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})

				hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Infra).ToNot(BeNil())
				Expect(existingResource.Spec.Infra.NodePlacement).ToNot(BeNil())
				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.Workloads).ToNot(BeNil())

				Expect(existingResource.Spec.Workloads.NodePlacement).ToNot(BeNil())
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key1"]).To(Equal("value1"))

				Expect(foundResource.Spec.Infra).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodePlacement).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))

				Expect(foundResource.Spec.Workloads).ToNot(BeNil())
				Expect(foundResource.Spec.Workloads.NodePlacement).ToNot(BeNil())
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key1"]).To(Equal("something else"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite node placement if directly set on KV CR", func() {
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in KV CR
				req.HCOTriggered = false

				// now, modify KV's node placement
				existingResource.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})
				existingResource.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})

				existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"] = "BADvalue2"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("value2"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Feature Gates", func() {

			getClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
					return &commontestutils.ClusterInfoMock{}
				}
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			Context("test feature gates in NewKubeVirt", func() {
				It("should add the PersistentReservation feature gate if PersistentReservation is true in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						PersistentReservation: ptr.To(true),
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should contain the PersistentReservation feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement(kvPersistentReservation))
					})
				})

				It("should not add the PersistentReservation feature gate if PersistentReservation is not set in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						PersistentReservation: nil,
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should not contain the PersistentReservation feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvPersistentReservation))
					})
				})

				It("should not add the PersistentReservation feature gate if PersistentReservation is false in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						PersistentReservation: ptr.To(false),
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should not contain the PersistentReservation feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvPersistentReservation))
					})
				})

				It("should not add the AlignCPUs feature gate if AlignCPUs is false in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						AlignCPUs: ptr.To(false),
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should not contain the AlignCPUs feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvAlignCPUs))
					})

					Expect(existingResource.Annotations).ToNot(HaveKey(kubevirtcorev1.EmulatorThreadCompleteToEvenParity))
				})

				It("should not add the AlignCPUs feature gate if AlignCPUs is not set in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						AlignCPUs: nil,
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should not contain the AlignCPUs feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvAlignCPUs))
					})

					Expect(existingResource.Annotations).ToNot(HaveKey(kubevirtcorev1.EmulatorThreadCompleteToEvenParity))
				})

				It("should not add the feature gates if FeatureGates field is empty", func() {
					mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(existingResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
					fgList := getKvFeatureGateList(&hco.Spec.FeatureGates)
					Expect(fgList).To(HaveLen(basicNumFgOnOpenshift))
					Expect(fgList).To(ContainElements(hardCodeKvFgs))
					Expect(fgList).To(ContainElements(sspConditionKvFgs))
				})

				It("should add the DownwardMetrics if feature gate DownwardMetrics is true in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						DownwardMetrics: ptr.To(true),
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should contain the DownwardMetrics feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement(kvDownwardMetrics))
					})
				})

				It("should no add the DownwardMetrics if feature gate DownwardMetrics is not in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						DownwardMetrics: nil,
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should contain the DownwardMetrics feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvDownwardMetrics))
					})
				})

				It("should not add the DownwardMetrics if feature gate DownwardMetrics is set to false in HyperConverged CR", func() {
					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						DownwardMetrics: ptr.To(false),
					}

					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					By("KV CR should not contain the DownwardMetrics feature gate", func() {
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvDownwardMetrics))
					})
				})
			})

			Context("test feature gates in KV handler", func() {

				getClusterInfo := hcoutil.GetClusterInfo

				BeforeEach(func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
				})

				AfterEach(func() {
					hcoutil.GetClusterInfo = getClusterInfo
				})

				It("should add feature gates if they are set to true", func() {
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						DownwardMetrics:       ptr.To(true),
						PersistentReservation: ptr.To(true),
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					By("KV CR should contain the HC enabled managed feature gates", func() {
						Expect(foundResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
						Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).
							To(ContainElements(kvDownwardMetrics, kvPersistentReservation))
					})
				})

				It("should not add feature gates if they are set to false", func() {
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						WithHostPassthroughCPU:   ptr.To(false),
						DisableMDevConfiguration: ptr.To(false),
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeFalse())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					By("KV CR should contain the HC enabled managed feature gates", func() {
						mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
						Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
						fgList := getKvFeatureGateList(&hco.Spec.FeatureGates)
						Expect(fgList).To(HaveLen(basicNumFgOnOpenshift))
						Expect(fgList).To(ContainElements(hardCodeKvFgs))
						Expect(fgList).To(ContainElements(sspConditionKvFgs))
					})
				})

				It("should not add feature gates if they are not exist", func() {
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeFalse())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					By("KV CR should contain the HC enabled managed feature gates", func() {
						mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
						Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
						fgList := getKvFeatureGateList(&hco.Spec.FeatureGates)
						Expect(fgList).To(HaveLen(len(getKvFeatureGateList(&hco.Spec.FeatureGates))))
						Expect(fgList).To(ContainElements(hardCodeKvFgs))
						Expect(fgList).To(ContainElements(sspConditionKvFgs))
					})
				})

				It("should keep FG if already exist", func() {
					mandatoryKvFeatureGates = getMandatoryKvFeatureGates(true)
					fgs := getKvFeatureGateList(&hco.Spec.FeatureGates)
					fgs = append(fgs, kvPersistentReservation)
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates = fgs

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						PersistentReservation: ptr.To(true),
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeFalse())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					Expect(foundResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).
						To(ContainElements(kvPersistentReservation))

				})

				It("should remove FG if it disabled in HC CR", func() {
					mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingResource.Spec.Configuration.DeveloperConfiguration = &kubevirtcorev1.DeveloperConfiguration{
						FeatureGates: []string{kvPersistentReservation},
					}

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
						PersistentReservation: ptr.To(false),
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(len(getKvFeatureGateList(&hco.Spec.FeatureGates))))
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(hardCodeKvFgs))
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(sspConditionKvFgs))
				})

				It("should remove FG if it missing from the HC CR", func() {
					mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingResource.Spec.Configuration.DeveloperConfiguration = &kubevirtcorev1.DeveloperConfiguration{
						FeatureGates: []string{kvPersistentReservation},
					}

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(len(getKvFeatureGateList(&hco.Spec.FeatureGates))))
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(hardCodeKvFgs))
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(sspConditionKvFgs))
				})

				It("should remove FG if it the HC CR does not contain the featureGates field", func() {
					mandatoryKvFeatureGates = getMandatoryKvFeatureGates(true)
					existingResource, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingResource.Spec.Configuration.DeveloperConfiguration = &kubevirtcorev1.DeveloperConfiguration{
						FeatureGates: []string{kvPersistentReservation},
					}

					hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{}

					cl := commontestutils.InitClient([]client.Object{hco, existingResource})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundResource := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
							foundResource),
					).ToNot(HaveOccurred())

					Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(len(hardCodeKvFgs)))
					Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(hardCodeKvFgs))
				})
			})

			Context("Test getKvFeatureGateList", func() {

				getClusterInfo := hcoutil.GetClusterInfo

				BeforeEach(func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
				})

				AfterEach(func() {
					hcoutil.GetClusterInfo = getClusterInfo
				})

				DescribeTable("Should return featureGate slice",
					func(isKVMEmulation bool, fgs *hcov1beta1.HyperConvergedFeatureGates, expectedLength int, expectedFgs [][]string) {
						mandatoryKvFeatureGates = getMandatoryKvFeatureGates(isKVMEmulation)
						fgList := getKvFeatureGateList(fgs)
						Expect(getKvFeatureGateList(fgs)).To(HaveLen(expectedLength))
						for _, expected := range expectedFgs {
							Expect(fgList).To(ContainElements(expected))
						}
					},
					Entry("When not using kvm-emulation and FG is empty",
						false,
						&hcov1beta1.HyperConvergedFeatureGates{},
						basicNumFgOnOpenshift,
						[][]string{hardCodeKvFgs, sspConditionKvFgs},
					),
					Entry("When using kvm-emulation and FG is empty",
						true,
						&hcov1beta1.HyperConvergedFeatureGates{},
						len(hardCodeKvFgs),
						[][]string{hardCodeKvFgs},
					),
					Entry("When not using kvm-emulation and all FGs are disabled",
						false,
						&hcov1beta1.HyperConvergedFeatureGates{WithHostPassthroughCPU: ptr.To(false)},
						basicNumFgOnOpenshift,
						[][]string{hardCodeKvFgs, sspConditionKvFgs},
					),
					Entry("When using kvm-emulation all FGs are disabled",
						true,
						&hcov1beta1.HyperConvergedFeatureGates{WithHostPassthroughCPU: ptr.To(false)},
						len(hardCodeKvFgs),
						[][]string{hardCodeKvFgs},
					),
					Entry("When not using kvm-emulation and all FGs are enabled",
						false,
						&hcov1beta1.HyperConvergedFeatureGates{DownwardMetrics: ptr.To(true)},
						basicNumFgOnOpenshift+1,
						[][]string{hardCodeKvFgs, sspConditionKvFgs, {kvDownwardMetrics}},
					),
					Entry("When using kvm-emulation all FGs are enabled",
						true,
						&hcov1beta1.HyperConvergedFeatureGates{DownwardMetrics: ptr.To(true)},
						len(hardCodeKvFgs)+1,
						[][]string{hardCodeKvFgs, {kvDownwardMetrics}},
					))
			})

			Context("Test getMandatoryKvFeatureGates", func() {
				It("Should include the sspConditionKvFgs if running in openshift", func() {
					fgs := getMandatoryKvFeatureGates(false)
					Expect(fgs).To(HaveLen(basicNumFgOnOpenshift))
					Expect(fgs).To(ContainElements(hardCodeKvFgs))
					Expect(fgs).To(ContainElements(sspConditionKvFgs))
				})

				It("Should not include the sspConditionKvFgs if not running in openshift", func() {
					fgs := getMandatoryKvFeatureGates(true)
					Expect(fgs).To(HaveLen(len(hardCodeKvFgs)))
					Expect(fgs).To(ContainElements(hardCodeKvFgs))
				})
			})
		})

		Context("Obsolete CPU Models", func() {
			Context("test Obsolete CPU Models in NewKubeVirt", func() {
				It("should add obsolete CPU Models if exists in HC CR", func() {
					hco.Spec.ObsoleteCPUs = &hcov1beta1.HyperConvergedObsoleteCPUs{
						CPUModels: []string{"aaa", "bbb", "ccc"},
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveLen(3 + len(hardcodedObsoleteCPUModels)))
					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("aaa", true))
					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("bbb", true))
					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("ccc", true))
					for _, cpu := range hardcodedObsoleteCPUModels {
						Expect(kv.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
					}

					Expect(kv.Spec.Configuration.MinCPUModel).To(BeEmpty())
				})

				It("should add min CPU Model if exists in HC CR", func() {
					hco.Spec.ObsoleteCPUs = &hcov1beta1.HyperConvergedObsoleteCPUs{
						MinCPUModel: "Penryn",
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(kv.Spec.Configuration.ObsoleteCPUModels).ToNot(BeEmpty())
					for _, cpu := range hardcodedObsoleteCPUModels {
						Expect(kv.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
					}
					Expect(kv.Spec.Configuration.MinCPUModel).To(Equal("Penryn"))
				})

				It("should not add min CPU Model and obsolete CPU Models if HC does not contain ObsoleteCPUs", func() {
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveLen(len(hardcodedObsoleteCPUModels)))
					for _, cpu := range hardcodedObsoleteCPUModels {
						Expect(kv.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
					}

					Expect(kv.Spec.Configuration.MinCPUModel).To(BeEmpty())
				})

				It("should not add min CPU Model and add only the hard coded obsolete CPU Models if ObsoleteCPUs is empty", func() {
					hco.Spec.ObsoleteCPUs = &hcov1beta1.HyperConvergedObsoleteCPUs{}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(kv.Spec.Configuration.ObsoleteCPUModels).To(HaveLen(len(hardcodedObsoleteCPUModels)))
					for _, cpu := range hardcodedObsoleteCPUModels {
						Expect(kv.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
					}

					Expect(kv.Spec.Configuration.MinCPUModel).To(BeEmpty())
				})
			})

			Context("test Obsolete CPU Models in KV handler", func() {
				It("Should add obsolete CPU model if they are set in HC CR", func() {
					existingKV, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())

					hco.Spec.ObsoleteCPUs = &hcov1beta1.HyperConvergedObsoleteCPUs{
						CPUModels:   []string{"aaa", "bbb", "ccc"},
						MinCPUModel: "Penryn",
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingKV})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundKV := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingKV.Name, Namespace: existingKV.Namespace},
							foundKV),
					).ToNot(HaveOccurred())

					By("KV CR should contain the HC obsolete CPU models and minCPUModel", func() {
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveLen(3 + len(hardcodedObsoleteCPUModels)))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("aaa", true))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("bbb", true))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("ccc", true))
						for _, cpu := range hardcodedObsoleteCPUModels {
							Expect(foundKV.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
						}

						Expect(foundKV.Spec.Configuration.MinCPUModel).To(Equal("Penryn"))
					})

				})

				It("Should modify obsolete CPU model if they are not the same as in HC CR", func() {
					existingKV, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingKV.Spec.Configuration.MinCPUModel = "Haswell"
					existingKV.Spec.Configuration.ObsoleteCPUModels = map[string]bool{
						"shouldStay":      true,
						"shouldBeTrue":    false,
						"shouldBeRemoved": true,
					}

					hco.Spec.ObsoleteCPUs = &hcov1beta1.HyperConvergedObsoleteCPUs{
						CPUModels:   []string{"shouldStay", "shouldBeTrue", "newOne"},
						MinCPUModel: "Penryn",
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingKV})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundKV := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingKV.Name, Namespace: existingKV.Namespace},
							foundKV),
					).ToNot(HaveOccurred())

					By("KV CR should contain the HC obsolete CPU models and minCPUModel", func() {
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveLen(3 + len(hardcodedObsoleteCPUModels)))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("shouldStay", true))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("shouldBeTrue", true))
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).To(HaveKeyWithValue("newOne", true))
						for _, cpu := range hardcodedObsoleteCPUModels {
							Expect(foundKV.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
						}

						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).ToNot(HaveKey("shouldBeRemoved"))

						Expect(foundKV.Spec.Configuration.MinCPUModel).To(Equal("Penryn"))
					})
				})

				It("Should remove obsolete CPU model if they are not set in HC CR", func() {
					existingKV, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					existingKV.Spec.Configuration.MinCPUModel = "Penryn"
					existingKV.Spec.Configuration.ObsoleteCPUModels = map[string]bool{
						"aaa": true,
						"bbb": true,
						"ccc": true,
					}

					cl := commontestutils.InitClient([]client.Object{hco, existingKV})
					handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
					res := handler.ensure(req)
					Expect(res.UpgradeDone).To(BeFalse())
					Expect(res.Updated).To(BeTrue())
					Expect(res.Overwritten).To(BeFalse())
					Expect(res.Err).ToNot(HaveOccurred())

					foundKV := &kubevirtcorev1.KubeVirt{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: existingKV.Name, Namespace: existingKV.Namespace},
							foundKV),
					).ToNot(HaveOccurred())

					By("KV CR ObsoleteCPUModels field should contain only the hard-coded values", func() {
						Expect(foundKV.Spec.Configuration.ObsoleteCPUModels).ToNot(BeEmpty())
						for _, cpu := range hardcodedObsoleteCPUModels {
							Expect(foundKV.Spec.Configuration.ObsoleteCPUModels[cpu]).To(BeTrue())
						}
					})

					By("KV CR minCPUModel field should be empty", func() {
						Expect(foundKV.Spec.Configuration.MinCPUModel).To(BeEmpty())
					})
				})
			})
		})

		Context("Certificate rotation strategy", func() {
			It("should add certificate rotation strategy if missing in KV", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
					CA: hcov1beta1.CertRotateConfigCA{
						Duration:    &metav1.Duration{Duration: 24 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
					},
					Server: hcov1beta1.CertRotateConfigServer{
						Duration:    &metav1.Duration{Duration: 12 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				certificateRotationStrategy := foundResource.Spec.CertificateRotationStrategy
				Expect(certificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("1h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("12h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("30m0s"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should set certificate rotation strategy to defaults if missing in HCO CR", func() {
				existingResource := NewKubeVirtWithNameOnly(hco)

				cl := commontestutils.InitClient([]client.Object{hco})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.CertificateRotationStrategy.SelfSigned).To(BeNil())

				Expect(foundResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				certificateRotationStrategy := foundResource.Spec.CertificateRotationStrategy
				Expect(certificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("24h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(certificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("12h0m0s"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify certificate rotation strategy according to HCO CR", func() {

				hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
					CA: hcov1beta1.CertRotateConfigCA{
						Duration:    &metav1.Duration{Duration: 24 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
					},
					Server: hcov1beta1.CertRotateConfigServer{
						Duration:    &metav1.Duration{Duration: 12 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
					},
				}
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Modify HCO's cert configuration")
				hco.Spec.CertConfig.CA.Duration.Duration *= 2
				hco.Spec.CertConfig.CA.RenewBefore.Duration *= 2
				hco.Spec.CertConfig.Server.Duration.Duration *= 2
				hco.Spec.CertConfig.Server.RenewBefore.Duration *= 2

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				existingCertificateRotationStrategy := existingResource.Spec.CertificateRotationStrategy
				Expect(existingCertificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("1h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("12h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("30m0s"))

				Expect(foundResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				foundCertificateRotationStrategy := foundResource.Spec.CertificateRotationStrategy
				Expect(foundCertificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("2h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("1h0m0s"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite certificate rotation strategy if directly set on KV CR", func() {

				hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
					CA: hcov1beta1.CertRotateConfigCA{
						Duration:    &metav1.Duration{Duration: 24 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
					},
					Server: hcov1beta1.CertRotateConfigServer{
						Duration:    &metav1.Duration{Duration: 12 * time.Hour},
						RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
					},
				}
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Mock a reconciliation triggered by a change in KV CR")
				req.HCOTriggered = false

				By("Modify KV's cert configuration")
				existingResource.Spec.CertificateRotationStrategy.SelfSigned.CA.Duration = &metav1.Duration{Duration: 48 * time.Hour}
				existingResource.Spec.CertificateRotationStrategy.SelfSigned.CA.RenewBefore = &metav1.Duration{Duration: 2 * time.Hour}
				existingResource.Spec.CertificateRotationStrategy.SelfSigned.Server.Duration = &metav1.Duration{Duration: 24 * time.Hour}
				existingResource.Spec.CertificateRotationStrategy.SelfSigned.Server.RenewBefore = &metav1.Duration{Duration: 1 * time.Hour}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				existingCertificateRotationStrategy := existingResource.Spec.CertificateRotationStrategy
				Expect(existingCertificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("48h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("2h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(existingCertificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("1h0m0s"))

				Expect(foundResource.Spec.CertificateRotationStrategy).ToNot(BeNil())
				foundCertificateRotationStrategy := foundResource.Spec.CertificateRotationStrategy
				Expect(foundCertificateRotationStrategy.SelfSigned.CA.Duration.Duration.String()).To(Equal("24h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.CA.RenewBefore.Duration.String()).To(Equal("1h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.Server.Duration.Duration.String()).To(Equal("12h0m0s"))
				Expect(foundCertificateRotationStrategy.SelfSigned.Server.RenewBefore.Duration.String()).To(Equal("30m0s"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Workload Update Strategy", func() {
			const defaultBatchEvictionSize = 10

			getClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
					return &commontestutils.ClusterInfoMock{}
				}
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			It("should add Workload Update Strategy if missing in KV", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.WorkloadUpdateStrategy = hcov1beta1.HyperConvergedWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []string{"aaa", "bbb"},
					BatchEvictionInterval: &metav1.Duration{Duration: time.Minute * 1},
					BatchEvictionSize:     ptr.To(defaultBatchEvictionSize),
				}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.WorkloadUpdateStrategy).ToNot(BeNil())
				kvUpdateStrategy := foundResource.Spec.WorkloadUpdateStrategy
				Expect(kvUpdateStrategy.BatchEvictionInterval.Duration.String()).To(Equal("1m0s"))
				Expect(kvUpdateStrategy.BatchEvictionSize).To(HaveValue(Equal(defaultBatchEvictionSize)))
				Expect(kvUpdateStrategy.WorkloadUpdateMethods).To(HaveLen(2))
				Expect(kvUpdateStrategy.WorkloadUpdateMethods).To(ContainElements(kubevirtcorev1.WorkloadUpdateMethod("aaa"), kubevirtcorev1.WorkloadUpdateMethod("bbb")))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should set Workload Update Strategy to defaults if missing in HCO CR", func() {
				existingResource := NewKubeVirtWithNameOnly(hco)

				cl := commontestutils.InitClient([]client.Object{hco})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.WorkloadUpdateStrategy).ToNot(BeNil())
				kvUpdateStrategy := foundResource.Spec.WorkloadUpdateStrategy
				Expect(kvUpdateStrategy.BatchEvictionInterval.Duration.String()).To(Equal("1m0s"))
				Expect(kvUpdateStrategy.BatchEvictionSize).To(HaveValue(Equal(defaultBatchEvictionSize)))
				Expect(kvUpdateStrategy.WorkloadUpdateMethods).To(HaveLen(1))
				Expect(kvUpdateStrategy.WorkloadUpdateMethods).To(
					ContainElements(
						kubevirtcorev1.WorkloadUpdateMethodLiveMigrate,
					),
				)

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify Workload Update Strategy according to HCO CR", func() {

				existingKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				const modifiedBatchEvictionSize = 5
				hco.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []string{"aaa", "bbb", "ccc"}
				hco.Spec.WorkloadUpdateStrategy.BatchEvictionInterval = &metav1.Duration{Duration: time.Minute * 3}
				hco.Spec.WorkloadUpdateStrategy.BatchEvictionSize = ptr.To(modifiedBatchEvictionSize)

				cl := commontestutils.InitClient([]client.Object{hco, existingKv})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundKv := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingKv.Name, Namespace: existingKv.Namespace},
						foundKv),
				).ToNot(HaveOccurred())

				Expect(foundKv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods).To(HaveLen(3))
				Expect(foundKv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods).To(
					ContainElements(
						kubevirtcorev1.WorkloadUpdateMethod("aaa"),
						kubevirtcorev1.WorkloadUpdateMethod("bbb"),
						kubevirtcorev1.WorkloadUpdateMethod("ccc"),
					),
				)

				Expect(foundKv.Spec.WorkloadUpdateStrategy.BatchEvictionInterval).To(HaveValue(Equal(metav1.Duration{Duration: time.Minute * 3})))
				Expect(foundKv.Spec.WorkloadUpdateStrategy.BatchEvictionSize).To(HaveValue(Equal(modifiedBatchEvictionSize)))
			})

			It("should overwrite Workload Update Strategy if directly set on KV CR", func() {
				const (
					hcoModifiedBatchEvictionSize = 5
					kvModifiedBatchEvictionSize  = 7
				)
				hco.Spec.WorkloadUpdateStrategy = hcov1beta1.HyperConvergedWorkloadUpdateStrategy{
					WorkloadUpdateMethods: []string{"LiveMigrate"},
					BatchEvictionInterval: &metav1.Duration{Duration: time.Minute * 5},
					BatchEvictionSize:     ptr.To(hcoModifiedBatchEvictionSize),
				}

				existingKV, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Mock a reconciliation triggered by a change in KV CR")
				req.HCOTriggered = false

				By("Modify KV's Workload Update Strategy configuration")
				existingKV.Spec.WorkloadUpdateStrategy.BatchEvictionInterval = &metav1.Duration{Duration: 3 * time.Minute}
				existingKV.Spec.WorkloadUpdateStrategy.BatchEvictionSize = ptr.To(kvModifiedBatchEvictionSize)
				existingKV.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []kubevirtcorev1.WorkloadUpdateMethod{kubevirtcorev1.WorkloadUpdateMethodEvict}

				cl := commontestutils.InitClient([]client.Object{hco, existingKV})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundKV := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingKV.Name, Namespace: existingKV.Namespace},
						foundKV),
				).ToNot(HaveOccurred())

				Expect(existingKV.Spec.CertificateRotationStrategy).ToNot(BeNil())
				existingUpdateStrategy := existingKV.Spec.WorkloadUpdateStrategy
				Expect(existingUpdateStrategy.WorkloadUpdateMethods).To(HaveLen(1))
				Expect(existingUpdateStrategy.WorkloadUpdateMethods).To(ContainElements(
					kubevirtcorev1.WorkloadUpdateMethodEvict,
				))
				Expect(*existingUpdateStrategy.BatchEvictionSize).To(Equal(kvModifiedBatchEvictionSize))
				Expect(existingUpdateStrategy.BatchEvictionInterval.Duration.String()).To(Equal("3m0s"))

				Expect(foundKV.Spec.CertificateRotationStrategy).ToNot(BeNil())
				foundUpdateStrategy := foundKV.Spec.WorkloadUpdateStrategy
				Expect(foundUpdateStrategy.WorkloadUpdateMethods).To(HaveLen(1))
				Expect(foundUpdateStrategy.WorkloadUpdateMethods).To(ContainElements(
					kubevirtcorev1.WorkloadUpdateMethodLiveMigrate,
				))
				Expect(foundUpdateStrategy.BatchEvictionSize).To(HaveValue(Equal(hcoModifiedBatchEvictionSize)))
				Expect(foundUpdateStrategy.BatchEvictionInterval.Duration.String()).To(Equal("5m0s"))
			})

		})

		Context("SNO replicas", func() {

			getClusterInfo := hcoutil.GetClusterInfo

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			Context("Custom Infra placement, default Workloads placement", func() {

				BeforeEach(func() {
					hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				})

				It("should set replica=1 on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSNOMock{}
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(Not(BeNil()))
					Expect(*kv.Spec.Infra.Replicas).To(Equal(uint8(1)))
					Expect(kv.Spec.Workloads).To(BeNil())
				})

				It("should not set replica when not on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(BeNil())
					Expect(kv.Spec.Workloads).To(BeNil())
				})

				It("should set replica=1 with SingleReplica ControlPlane and HighAvailable Infrastructure ", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSRCPHAIMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(HaveValue(Equal(uint8(1))))
					Expect(kv.Spec.Workloads).To(BeNil())
				})

			})

			Context("Custom Workloads placement, default Infra placement", func() {

				BeforeEach(func() {
					hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				})

				It("should set replica=1 on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSNOMock{}
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(Not(BeNil()))
					Expect(*kv.Spec.Infra.Replicas).To(Equal(uint8(1)))
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

				It("should not set replica when not on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(BeNil())
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

				It("should set replica=1 with SingleReplica ControlPlane but HighAvailable Infrastructure ", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSRCPHAIMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(HaveValue(Equal(uint8(1))))
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

			})

			Context("Default Infra and Workload placement", func() {

				It("should set replica=1 on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSNOMock{}
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(Not(BeNil()))
					Expect(*kv.Spec.Infra.Replicas).To(Equal(uint8(1)))
					Expect(kv.Spec.Infra.NodePlacement).To(BeNil())
					Expect(kv.Spec.Workloads).To(BeNil())
				})

				It("should not set replica when not on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(BeNil())
					Expect(kv.Spec.Workloads).To(BeNil())
				})

				It("should set replica=1 with SingleReplica ControlPlane but HighAvailable Infrastructure ", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSRCPHAIMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(HaveValue(Equal(uint8(1))))
					Expect(kv.Spec.Workloads).To(BeNil())
				})

			})

			Context("Custom Infra and Workloads placement", func() {

				BeforeEach(func() {
					hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
					hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				})

				It("should set replica=1 on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSNOMock{}
					}

					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(HaveValue(Equal(uint8(1))))
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

				It("should not set replica when not on SNO", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(kv.Spec.Infra.Replicas).To(BeNil())
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

				It("should set replica=1 with SingleReplica ControlPlane and HighAvailable Infrastructure ", func() {
					hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
						return &commontestutils.ClusterInfoSRCPHAIMock{}
					}
					kv, err := NewKubeVirt(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Infra).To(Not(BeNil()))
					Expect(*kv.Spec.Infra.Replicas).To(Equal(uint8(1)))
					Expect(kv.Spec.Workloads).To(Not(BeNil()))
					Expect(kv.Spec.Workloads.Replicas).To(BeNil())
				})

			})

		})

		Context("Cluster level EvictionStrategy", func() {
			It("should add eviction strategy if missing in KV", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.EvictionStrategy = ptr.To(kubevirtcorev1.EvictionStrategyLiveMigrate)

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.Configuration.EvictionStrategy).ToNot(BeNil())
				evictionStrategy := foundResource.Spec.Configuration.EvictionStrategy
				Expect(*evictionStrategy).To(Equal(kubevirtcorev1.EvictionStrategyLiveMigrate))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify eviction strategy according to HCO CR", func() {

				hco.Spec.EvictionStrategy = ptr.To(kubevirtcorev1.EvictionStrategyNone)
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Modify HCO's eviction strategy configuration")
				hco.Spec.EvictionStrategy = ptr.To(kubevirtcorev1.EvictionStrategyLiveMigrateIfPossible)

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Configuration.EvictionStrategy).ToNot(BeNil())
				existingEvictionStrategy := *existingResource.Spec.Configuration.EvictionStrategy
				Expect(existingEvictionStrategy).To(Equal(kubevirtcorev1.EvictionStrategyNone))

				Expect(foundResource.Spec.Configuration.EvictionStrategy).ToNot(BeNil())
				foundEvictionStrategy := *foundResource.Spec.Configuration.EvictionStrategy
				Expect(foundEvictionStrategy).To(Equal(kubevirtcorev1.EvictionStrategyLiveMigrateIfPossible))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("VM state storage class", func() {
			It("should modify storage class according to HCO CR", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Modify HCO's VM state storage class configuration")
				hco.Spec.VMStateStorageClass = ptr.To("rook-cephfs")

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Configuration.VMStateStorageClass).To(BeEmpty())

				Expect(foundResource.Spec.Configuration.VMStateStorageClass).To(Equal("rook-cephfs"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Auto CPU limit", func() {
			It("should set the namespace label selector according to HCO CR", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					AutoCPULimitNamespaceLabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"someLabel": "true"},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Configuration.AutoCPULimitNamespaceLabelSelector).To(BeNil())

				Expect(foundResource.Spec.Configuration.AutoCPULimitNamespaceLabelSelector).NotTo(BeNil())
				Expect(foundResource.Spec.Configuration.AutoCPULimitNamespaceLabelSelector.MatchLabels).To(HaveLen(1))
				Expect(foundResource.Spec.Configuration.AutoCPULimitNamespaceLabelSelector.MatchLabels).To(HaveKeyWithValue("someLabel", "true"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Virtual machine options", func() {
			It("should set VirtualMachineOptions by default", func() {
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Spec.Configuration).To(Not(BeNil()))
				Expect(kv.Spec.Configuration.VirtualMachineOptions).To(BeNil())
			})

			DescribeTable("Should set VirtualMachineOptions according to HCO CR options", func(hcoVMOptions *hcov1beta1.VirtualMachineOptions, kvVMOptions *kubevirtcorev1.VirtualMachineOptions) {
				hco.Spec.VirtualMachineOptions = hcoVMOptions
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Spec.Configuration).To(Not(BeNil()))
				Expect(kv.Spec.Configuration.VirtualMachineOptions).To(BeEquivalentTo(kvVMOptions))
			},
				Entry("nil VirtualMachineOptions",
					nil,
					nil,
				),
				Entry("disableFreePageReporting only, false",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(false)},
					nil,
				),
				Entry("disableFreePageReporting only, true",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(true)},
					&kubevirtcorev1.VirtualMachineOptions{DisableFreePageReporting: &kubevirtcorev1.DisableFreePageReporting{}},
				),
				Entry("disableSerialConsoleLog only, false",
					&hcov1beta1.VirtualMachineOptions{DisableSerialConsoleLog: ptr.To(false)},
					nil,
				),
				Entry("disableSerialConsoleLog only, true",
					&hcov1beta1.VirtualMachineOptions{DisableSerialConsoleLog: ptr.To(true)},
					&kubevirtcorev1.VirtualMachineOptions{DisableSerialConsoleLog: &kubevirtcorev1.DisableSerialConsoleLog{}},
				),
				Entry("disableFreePageReporting false, disableSerialConsoleLog false",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(false), DisableSerialConsoleLog: ptr.To(false)},
					nil,
				),
				Entry("disableFreePageReporting true, disableSerialConsoleLog false",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(true), DisableSerialConsoleLog: ptr.To(false)},
					&kubevirtcorev1.VirtualMachineOptions{DisableFreePageReporting: &kubevirtcorev1.DisableFreePageReporting{}},
				),
				Entry("disableFreePageReporting false, disableSerialConsoleLog true",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(false), DisableSerialConsoleLog: ptr.To(true)},
					&kubevirtcorev1.VirtualMachineOptions{DisableSerialConsoleLog: &kubevirtcorev1.DisableSerialConsoleLog{}},
				),
				Entry("disableFreePageReporting true, disableSerialConsoleLog true",
					&hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(true), DisableSerialConsoleLog: ptr.To(true)},
					&kubevirtcorev1.VirtualMachineOptions{DisableFreePageReporting: &kubevirtcorev1.DisableFreePageReporting{}, DisableSerialConsoleLog: &kubevirtcorev1.DisableSerialConsoleLog{}},
				),
			)

			DescribeTable("should modify disableFreePageReporting according to HCO CR", func(virtualMachineOptions *hcov1beta1.VirtualMachineOptions, updated, expectDisableFreePageReporting bool) {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Modify HCO's virtual machine options configuration")
				hco.Spec.VirtualMachineOptions = virtualMachineOptions

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				if updated {
					Expect(res.Updated).To(BeTrue())
				} else {
					Expect(res.Updated).To(BeFalse())
				}
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				if expectDisableFreePageReporting {
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions).ToNot(BeNil())
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions.DisableFreePageReporting).ToNot(BeNil())
				} else {
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions).To(BeNil())
				}

			},
				Entry("with virtualMachineOptions containing disableFreePageReporting false", &hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(false)}, false, false),
				Entry("with virtualMachineOptions containing disableFreePageReporting true", &hcov1beta1.VirtualMachineOptions{DisableFreePageReporting: ptr.To(true)}, true, true),
				Entry("with empty virtualMachineOptions", &hcov1beta1.VirtualMachineOptions{}, false, false),
			)

			DescribeTable("should modify disableSerialConsoleLog according to HCO CR", func(virtualMachineOptions *hcov1beta1.VirtualMachineOptions, updated, expectDisableSerialConsoleLog bool) {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				By("Modify HCO's virtual machine options configuration")
				hco.Spec.VirtualMachineOptions = virtualMachineOptions

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				if updated {
					Expect(res.Updated).To(BeTrue())
				} else {
					Expect(res.Updated).To(BeFalse())
				}
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				if expectDisableSerialConsoleLog {
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions).ToNot(BeNil())
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions.DisableSerialConsoleLog).ToNot(BeNil())
				} else {
					Expect(foundResource.Spec.Configuration.VirtualMachineOptions).To(BeNil())
				}
			},
				Entry("with virtualMachineOptions containing disableSerialConsoleLog false", &hcov1beta1.VirtualMachineOptions{DisableSerialConsoleLog: ptr.To(false)}, false, false),
				Entry("with virtualMachineOptions containing disableSerialConsoleLog true", &hcov1beta1.VirtualMachineOptions{DisableSerialConsoleLog: ptr.To(true)}, true, true),
				Entry("with empty virtualMachineOptions", &hcov1beta1.VirtualMachineOptions{}, false, false),
			)

		})

		Context("VmiCPUAllocationRatio", func() {
			It("should add CPUAllocationRatio if missing in KV CR", func() {
				const expectedCPUAllocationRatio = 16

				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					VmiCPUAllocationRatio: ptr.To(expectedCPUAllocationRatio),
				}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio).To(Equal(expectedCPUAllocationRatio))
			})

			It("should remove CPUAllocationRatio if missing in HCO CR", func() {
				const initialCPUAllocationRatio = 16

				hcoResourceRequirements := commontestutils.NewHco()
				hcoResourceRequirements.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					VmiCPUAllocationRatio: ptr.To(initialCPUAllocationRatio),
				}

				existingResource, err := NewKubeVirt(hcoResourceRequirements)
				Expect(err).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(existingResource.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio).To(Equal(initialCPUAllocationRatio))

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio).To(Equal(0))
			})

			It("should modify CPUAllocationRatio according to HCO CR", func() {
				const (
					initialCPUAllocationRatio  = 16
					expectedCPUAllocationRatio = 25
				)
				hcoResourceRequirements := commontestutils.NewHco()

				hcoResourceRequirements.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					VmiCPUAllocationRatio: ptr.To(initialCPUAllocationRatio),
				}

				existingResource, err := NewKubeVirt(hcoResourceRequirements)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					VmiCPUAllocationRatio: ptr.To(expectedCPUAllocationRatio),
				}

				Expect(existingResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(existingResource.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio).To(Equal(initialCPUAllocationRatio))

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio).To(Equal(expectedCPUAllocationRatio))
			})

		})

		Context("KSM Configuration", func() {
			It("should set the namespace label selector according to HCO CR", func() {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.KSMConfiguration = &kubevirtcorev1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"someLabel": "true"},
					},
				}

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Configuration.KSMConfiguration).To(BeNil())

				Expect(foundResource.Spec.Configuration.KSMConfiguration).NotTo(BeNil())
				Expect(foundResource.Spec.Configuration.KSMConfiguration.NodeLabelSelector).NotTo(BeNil())
				Expect(foundResource.Spec.Configuration.KSMConfiguration.NodeLabelSelector.MatchLabels).To(HaveLen(1))
				Expect(foundResource.Spec.Configuration.KSMConfiguration.NodeLabelSelector.MatchLabels).To(HaveKeyWithValue("someLabel", "true"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Rollout Strategy", func() {
			It("should be set to live update", func() {
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				Expect(kv.Spec.Configuration.VMRolloutStrategy).To(HaveValue(Equal(kubevirtcorev1.VMRolloutStrategyLiveUpdate)))
			})
		})

		It("should handle conditions", func() {
			expectedResource, err := NewKubeVirt(hco, commontestutils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.Status.Conditions = []kubevirtcorev1.KubeVirtCondition{
				{
					Type:    kubevirtcorev1.KubeVirtConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "Foo",
					Message: "Bar",
				},
				{
					Type:    kubevirtcorev1.KubeVirtConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "Foo",
					Message: "Bar",
				},
				{
					Type:    kubevirtcorev1.KubeVirtConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "Foo",
					Message: "Bar",
				},
			}
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "KubeVirtNotAvailable",
				Message: "KubeVirt is not available: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "KubeVirtProgressing",
				Message: "KubeVirt is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "KubeVirtProgressing",
				Message: "KubeVirt is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionDegraded]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "KubeVirtDegraded",
				Message: "KubeVirt is degraded: Bar",
			}))
		})

		Context("Tune KubeVirt rateLimiters", func() {

			var hco *hcov1beta1.HyperConverged

			BeforeEach(func() {
				hco = commontestutils.NewHco()
			})

			It("Should be empty by default", func() {
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Spec.TuningPolicy).To(BeEmpty())
				Expect(kv.Spec.Configuration.APIConfiguration).To(BeNil())
				Expect(kv.Spec.Configuration.ControllerConfiguration).To(BeNil())
				Expect(kv.Spec.Configuration.WebhookConfiguration).To(BeNil())
				Expect(kv.Spec.Configuration.HandlerConfiguration).To(BeNil())

			})

			Context("with annotations", func() {
				It("Should return error if annotation is not present", func() {
					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedAnnotationTuningPolicy

					kv, err := NewKubeVirt(hco)
					Expect(err).To(MatchError("tuning policy set but annotation not present or wrong"))

					Expect(kv).To(BeNil())

				})
				It("Should return error if the annotation is present but the parameters are wrong", func() {

					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedAnnotationTuningPolicy
					hco.Annotations = make(map[string]string, 1)
					//burst is missing
					hco.Annotations["hco.kubevirt.io/tuningPolicy"] = `{"qps": 100}`

					kv, err := NewKubeVirt(hco)
					Expect(err).To(MatchError("burst parameter not found in annotation"))
					Expect(kv).To(BeNil())

				})

				It("Should return error if the json annotation is corrupted", func() {
					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedAnnotationTuningPolicy
					hco.Annotations = make(map[string]string, 1)
					// qps field is missing a "
					hco.Annotations["hco.kubevirt.io/tuningPolicy"] = `{"qps: 100, "burst": 200}`

					kv, err := NewKubeVirt(hco)

					Expect(err).To(HaveOccurred())
					Expect(kv).To(BeNil())
				})

				It("Should create the fields and populate them when requested", func() {
					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedAnnotationTuningPolicy
					hco.Annotations = make(map[string]string, 1)
					hco.Annotations["hco.kubevirt.io/tuningPolicy"] = `{"qps": 100, "burst": 200}`

					kv, err := NewKubeVirt(hco)

					Expect(kv).ToNot(BeNil())
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Configuration.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(float32(100)))
					Expect(kv.Spec.Configuration.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(200))
					Expect(kv.Spec.Configuration.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(float32(100)))
					Expect(kv.Spec.Configuration.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(200))
					Expect(kv.Spec.Configuration.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(float32(100)))
					Expect(kv.Spec.Configuration.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(200))
					Expect(kv.Spec.Configuration.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(float32(100)))
					Expect(kv.Spec.Configuration.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(200))
				})

			})

			Context("with highBurst profile", func() {

				It("Should return error if the json annotation tuningPolicy is present", func() {
					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedHighBurstProfile
					hco.Annotations = make(map[string]string, 1)
					hco.Annotations["hco.kubevirt.io/tuningPolicy"] = `{"qps": 100, "burst": 200}`

					kv, err := NewKubeVirt(hco)

					Expect(err).To(HaveOccurred())
					Expect(kv).To(BeNil())
				})

				It("Should create the fields and populate them using the highBurst profile values", func() {
					hco.Spec.TuningPolicy = hcov1beta1.HyperConvergedHighBurstProfile
					kv, err := NewKubeVirt(hco)
					const expectedQPS = float32(200)
					const expectedBurst = 400

					Expect(kv).ToNot(BeNil())
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Configuration.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(expectedQPS))
					Expect(kv.Spec.Configuration.APIConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(expectedBurst))
					Expect(kv.Spec.Configuration.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(expectedQPS))
					Expect(kv.Spec.Configuration.ControllerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(expectedBurst))
					Expect(kv.Spec.Configuration.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(expectedQPS))
					Expect(kv.Spec.Configuration.WebhookConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(expectedBurst))
					Expect(kv.Spec.Configuration.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.QPS).To(Equal(expectedQPS))
					Expect(kv.Spec.Configuration.HandlerConfiguration.RestClient.RateLimiter.TokenBucketRateLimiter.Burst).To(Equal(expectedBurst))

				})
			})

		})

		Context("jsonpath Annotation", func() {

			getClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
					return &commontestutils.ClusterInfoMock{}
				}
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			mandatoryKvFeatureGates = getMandatoryKvFeatureGates(false)
			It("Should create KV object with changes from the annotation", func() {

				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/configuration/cpuRequest",
						"value": "12m"
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration",
						"value": {"featureGates": ["fg1"]}
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration/featureGates/-",
						"value": "fg2"
					}
				]`}

				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(2))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg1"))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg2"))
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())

				quantity, err := resource.ParseQuantity("12m")
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())
				Expect(*kv.Spec.Configuration.CPURequest).To(Equal(quantity))
			})

			It("Should fail to create KV object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/config/featureGates/-",
						"value": "fg1"
					}
				]`}

				_, err := NewKubeVirt(hco)
				Expect(err).To(HaveOccurred())
			})

			It("Ensure func should create KV object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/configuration/cpuRequest",
						"value": "12m"
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration",
						"value": {"featureGates": ["fg1"]}
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration/featureGates/-",
						"value": "fg2"
					}
				]`}

				expectedResource := NewKubeVirtWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{hco})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Created).To(BeTrue())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				kv := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						kv),
				).ToNot(HaveOccurred())

				Expect(kv).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(2))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg1"))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg2"))
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())

				quantity, err := resource.ParseQuantity("12m")
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())
				Expect(*kv.Spec.Configuration.CPURequest).To(Equal(quantity))
			})

			It("Ensure func should fail to create KV object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/configuration/developerConfiguration",
						"value": {"featureGates": ["fg1"]}
					}
				]`}

				expectedResource := NewKubeVirtWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{hco})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				kv := &kubevirtcorev1.KubeVirt{}

				Expect(cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					kv,
				)).To(MatchError(errors.IsNotFound, "not found error"))
			})

			It("Ensure func should update KV object with changes from the annotation", func() {
				existsCdi, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/configuration/cpuRequest",
						"value": "12m"
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration",
						"value": {"featureGates": ["fg1"]}
					},
					{
						"op": "add",
						"path": "/spec/configuration/developerConfiguration/featureGates/-",
						"value": "fg2"
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsCdi})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())
				Expect(res.UpgradeDone).To(BeFalse())

				kv := &kubevirtcorev1.KubeVirt{}

				expectedResource := NewKubeVirtWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						kv),
				).ToNot(HaveOccurred())

				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(2))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg1"))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("fg2"))
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())

				quantity, err := resource.ParseQuantity("12m")
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Spec.Configuration.CPURequest).ToNot(BeNil())
				Expect(*kv.Spec.Configuration.CPURequest).To(Equal(quantity))
			})

			It("Ensure func should fail to update KV object with wrong jsonPatch", func() {
				existsKv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchKVAnnotationName: `[
					{
						"op": "notExistsOp",
						"path": "/spec/configuration/cpuRequest",
						"value": "12m"
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsKv})

				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				kv := &kubevirtcorev1.KubeVirt{}

				expectedResource := NewKubeVirtWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						kv),
				).ToNot(HaveOccurred())

				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(len(getKvFeatureGateList(&hco.Spec.FeatureGates))))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(hardCodeKvFgs))
				Expect(kv.Spec.Configuration.CPURequest).To(BeNil())

			})
		})

		Context("Cache", func() {
			cl := commontestutils.InitClient([]client.Object{})
			handler := newKubevirtHandler(cl, commontestutils.GetScheme())

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*kubevirtHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*kubevirtHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*kubevirtHooks).cache).To(BeIdenticalTo(cr))

					crII, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(crII).ToNot(BeNil())
					Expect(cr).To(BeIdenticalTo(crII))
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*kubevirtHooks).reset()
				Expect(handler.hooks.(*kubevirtHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*kubevirtHooks).cache).ToNot(BeNil())

				handler.hooks.(*kubevirtHooks).reset()
				Expect(handler.hooks.(*kubevirtHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*kubevirtHooks).cache).ToNot(BeNil())

				Expect(crI).ToNot(BeIdenticalTo(crII))
				Expect(handler.hooks.(*kubevirtHooks).cache).ToNot(BeIdenticalTo(crI))
				Expect(handler.hooks.(*kubevirtHooks).cache).To(BeIdenticalTo(crII))
			})
		})

		Context("Log verbosity", func() {

			It("Should be defined for KubevirtCR if defined in HCO CR", func() {
				logVerbosity := kubevirtcorev1.LogVerbosity{
					VirtLauncher:   123,
					VirtAPI:        456,
					VirtController: 789,
				}
				hco.Spec.LogVerbosityConfig = &hcov1beta1.LogVerbosityConfiguration{Kubevirt: &logVerbosity}
				devConfig := getKVDevConfig(hco)

				Expect(devConfig).ToNot(BeNil())
				Expect(*devConfig.LogVerbosity).To(Equal(logVerbosity))
			})

			DescribeTable("Should not be defined for KubevirtCR if not defined in HCO CR", func(logConfig *hcov1beta1.LogVerbosityConfiguration) {
				hco.Spec.LogVerbosityConfig = logConfig
				devConfig := getKVDevConfig(hco)

				Expect(devConfig).ToNot(BeNil())
				Expect(devConfig.LogVerbosity).To(BeNil())
			},
				Entry("nil LogVerbosityConfiguration", nil),
				Entry("nil Kubevirt logs", &hcov1beta1.LogVerbosityConfiguration{Kubevirt: nil}),
			)

		})

		Context("DefaultRuntimeClass", func() {

			It("Should be defined for KubevirtCR if defined in HCO CR", func() {
				const runtimeClass = "myCustomRuntimeClass"
				hco.Spec.DefaultRuntimeClass = ptr.To(runtimeClass)
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				Expect(kv.Spec.Configuration.DefaultRuntimeClass).To(Equal(runtimeClass))
			})

			DescribeTable("Should be empty on KubevirtCR if not defined in HCO CR", func(runtimeClass *string) {
				hco.Spec.DefaultRuntimeClass = runtimeClass
				kv, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				Expect(kv.Spec.Configuration.DefaultRuntimeClass).To(BeEmpty())
			},
				Entry("nil defaultRuntimeClass", nil),
				Entry("empty defaultRuntimeClass", ptr.To("")),
			)

		})

		Context("AlignCPUs", func() {
			DescribeTable("AlignCPUs is enabled in HCO", func(isAlignCPUsFGEnabledOnKV, isAnnotationPresentOnKV bool) {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				if isAlignCPUsFGEnabledOnKV {
					existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
						existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates,
						kvAlignCPUs,
					)
				}

				if isAnnotationPresentOnKV {
					existingResource.Annotations[kubevirtcorev1.EmulatorThreadCompleteToEvenParity] = ""
				}

				hco.Spec.FeatureGates.AlignCPUs = ptr.To(true)

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Annotations).To(HaveKeyWithValue(kubevirtcorev1.EmulatorThreadCompleteToEvenParity, ""))
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement(kvAlignCPUs))
			},
				Entry("FG and annotation are missing in KubeVirt", false, false),
				Entry("FG and annotation are present in KubeVirt", true, true),
				Entry("FG missing, annotation is present in KubeVirt", false, true),
				Entry("FG present, annotation is missing in KubeVirt", true, false),
			)

			DescribeTable("AlignCPUs is disabled in HCO", func(alignCPUsValue *bool, isAlignCPUsFGEnabledOnKV, isAnnotationPresentOnKV bool) {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())

				if isAlignCPUsFGEnabledOnKV {
					existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
						existingResource.Spec.Configuration.DeveloperConfiguration.FeatureGates,
						kvAlignCPUs,
					)
				}

				if isAnnotationPresentOnKV {
					existingResource.Annotations[kubevirtcorev1.EmulatorThreadCompleteToEvenParity] = ""
				}

				hco.Spec.FeatureGates.AlignCPUs = alignCPUsValue

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Annotations).ToNot(HaveKey(kubevirtcorev1.EmulatorThreadCompleteToEvenParity))
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration).NotTo(BeNil())
				Expect(foundResource.Spec.Configuration.DeveloperConfiguration.FeatureGates).ToNot(ContainElement(kvAlignCPUs))
			},
				Entry("implicitly disabled, FG and annotation are missing in KubeVirt", nil, false, false),
				Entry("implicitly disabled, FG and annotation are present in KubeVirt", nil, true, true),
				Entry("implicitly disabled, FG missing, annotation is present in KubeVirt", nil, false, true),
				Entry("implicitly disabled, FG present, annotation is missing in KubeVirt", nil, true, false),
				Entry("explicitly disabled, FG and annotation are missing in KubeVirt", ptr.To(false), false, false),
				Entry("explicitly disabled, FG and annotation are present in KubeVirt", ptr.To(false), true, true),
				Entry("explicitly disabled, FG missing, annotation is present in KubeVirt", ptr.To(false), false, true),
				Entry("explicitly disabled, FG present, annotation is missing in KubeVirt", ptr.To(false), true, false),
			)
		})

		Context("Higher workload density", func() {
			It("should convert ratio to corresponding percentage when overcommit ratio is set", func() {
				const expectedPercentage int = 125

				hco.Spec.HigherWorkloadDensity = &hcov1beta1.HigherWorkloadDensityConfiguration{
					MemoryOvercommitPercentage: expectedPercentage,
				}
				devConfig := getKVDevConfig(hco)
				Expect(devConfig.MemoryOvercommit).To(Equal(expectedPercentage))
			})
		})

		Context("InstancetypeConfig", func() {
			DescribeTable("should", func(spec hcov1beta1.HyperConvergedSpec, expectedConfig *kubevirtcorev1.InstancetypeConfiguration) {
				hco.Spec = spec
				config, err := getKVConfig(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(config.Instancetype).To(Equal(expectedConfig))
			},
				Entry("pass to KubeVirt when provided",
					hcov1beta1.HyperConvergedSpec{
						InstancetypeConfig: &kubevirtcorev1.InstancetypeConfiguration{
							ReferencePolicy: ptr.To(kubevirtcorev1.Reference),
						},
					},
					&kubevirtcorev1.InstancetypeConfiguration{
						ReferencePolicy: ptr.To(kubevirtcorev1.Reference),
					},
				),
				Entry("not pass to KubeVirt when nil", hcov1beta1.HyperConvergedSpec{}, nil),
			)
		})
	})

	Context("Test hcLiveMigrationToKv", func() {

		const (
			bandwidthPerMigration             = "64Mi"
			completionTimeoutPerGiB           = int64(100)
			parallelMigrationsPerCluster      = uint32(100)
			parallelOutboundMigrationsPerNode = uint32(100)
			progressTimeout                   = int64(100)
			network                           = "testNetwork"
		)
		It("should create valid KV LM config from a valid HC LM config", func() {
			lmc := hcov1beta1.LiveMigrationConfigurations{
				BandwidthPerMigration:             ptr.To(bandwidthPerMigration),
				CompletionTimeoutPerGiB:           ptr.To(completionTimeoutPerGiB),
				ParallelMigrationsPerCluster:      ptr.To(parallelMigrationsPerCluster),
				ParallelOutboundMigrationsPerNode: ptr.To(parallelOutboundMigrationsPerNode),
				ProgressTimeout:                   ptr.To(progressTimeout),
				Network:                           ptr.To(network),
				AllowAutoConverge:                 ptr.To(true),
				AllowPostCopy:                     ptr.To(true),
			}
			mc, err := hcLiveMigrationToKv(lmc)
			Expect(err).ToNot(HaveOccurred())

			Expect(mc.BandwidthPerMigration).To(HaveValue(Equal(resource.MustParse(bandwidthPerMigration))))
			Expect(mc.CompletionTimeoutPerGiB).To(HaveValue(Equal(completionTimeoutPerGiB)))
			Expect(mc.ParallelMigrationsPerCluster).To(HaveValue(Equal(parallelMigrationsPerCluster)))
			Expect(mc.ParallelOutboundMigrationsPerNode).To(HaveValue(Equal(parallelOutboundMigrationsPerNode)))
			Expect(mc.ProgressTimeout).To(HaveValue(Equal(progressTimeout)))
			Expect(mc.Network).To(HaveValue(Equal(network)))
			Expect(mc.AllowAutoConverge).To(HaveValue(BeTrue()))
			Expect(mc.AllowPostCopy).To(HaveValue(BeTrue()))
		})

		It("should create valid empty KV LM config from a valid empty HC LM config", func() {
			lmc := hcov1beta1.LiveMigrationConfigurations{}
			mc, err := hcLiveMigrationToKv(lmc)
			Expect(err).ToNot(HaveOccurred())

			Expect(mc.BandwidthPerMigration).To(BeNil())
			Expect(mc.CompletionTimeoutPerGiB).To(BeNil())
			Expect(mc.ParallelMigrationsPerCluster).To(BeNil())
			Expect(mc.ParallelOutboundMigrationsPerNode).To(BeNil())
			Expect(mc.ProgressTimeout).To(BeNil())
			Expect(mc.Network).To(BeNil())
			Expect(mc.AllowAutoConverge).To(BeNil())
			Expect(mc.AllowPostCopy).To(BeNil())
		})

		It("should return error if the value of the BandwidthPerMigration field is not valid", func() {
			lmc := hcov1beta1.LiveMigrationConfigurations{
				BandwidthPerMigration:             ptr.To("Wrong BandwidthPerMigration"),
				CompletionTimeoutPerGiB:           ptr.To(completionTimeoutPerGiB),
				ParallelMigrationsPerCluster:      ptr.To(parallelMigrationsPerCluster),
				ParallelOutboundMigrationsPerNode: ptr.To(parallelOutboundMigrationsPerNode),
				ProgressTimeout:                   ptr.To(progressTimeout),
				Network:                           ptr.To(network),
				AllowAutoConverge:                 ptr.To(true),
				AllowPostCopy:                     ptr.To(true),
			}
			mc, err := hcLiveMigrationToKv(lmc)
			Expect(err).To(HaveOccurred())
			Expect(mc).To(BeNil())
		})
	})

	Context("Test toKvPermittedHostDevices", func() {
		It("should return nil if the input is nil", func() {
			Expect(toKvPermittedHostDevices(nil)).To(BeNil())
		})

		It("should return an empty lists if the input is empty", func() {
			kvCopy := toKvPermittedHostDevices(&hcov1beta1.PermittedHostDevices{})
			Expect(kvCopy).ToNot(BeNil())
			Expect(kvCopy.PciHostDevices).To(BeEmpty())
			Expect(kvCopy.MediatedDevices).To(BeEmpty())
		})

		It("should copy all the values", func() {
			hcoCopy := &hcov1beta1.PermittedHostDevices{
				PciHostDevices: []hcov1beta1.PciHostDevice{
					{
						PCIDeviceSelector:        "vendor1",
						ResourceName:             "resourceName1",
						ExternalResourceProvider: true,
					},
					{
						PCIDeviceSelector:        "vendor2",
						ResourceName:             "resourceName2",
						ExternalResourceProvider: false,
					},
					{
						PCIDeviceSelector:        "vendor3",
						ResourceName:             "resourceName3",
						ExternalResourceProvider: true,
						Disabled:                 false,
					},
					{
						PCIDeviceSelector:        "disabledSelector",
						ResourceName:             "disabledName",
						ExternalResourceProvider: true,
						Disabled:                 true,
					},
				},
				MediatedDevices: []hcov1beta1.MediatedHostDevice{
					{
						MDEVNameSelector:         "selector1",
						ResourceName:             "resource1",
						ExternalResourceProvider: true,
					},
					{
						MDEVNameSelector:         "selector2",
						ResourceName:             "resource2",
						ExternalResourceProvider: false,
					},
					{
						MDEVNameSelector:         "selector3",
						ResourceName:             "resource3",
						ExternalResourceProvider: true,
					},
					{
						MDEVNameSelector:         "selector4",
						ResourceName:             "resource4",
						ExternalResourceProvider: false,
						Disabled:                 false,
					},
					{
						MDEVNameSelector:         "disabledSelector",
						ResourceName:             "disabledName",
						ExternalResourceProvider: false,
						Disabled:                 true,
					},
				},
			}

			kvCopy := toKvPermittedHostDevices(hcoCopy)
			Expect(kvCopy).ToNot(BeNil())

			Expect(kvCopy.PciHostDevices).To(HaveLen(3))
			Expect(kvCopy.PciHostDevices).To(ContainElements(
				kubevirtcorev1.PciHostDevice{
					PCIVendorSelector:        "vendor1",
					ResourceName:             "resourceName1",
					ExternalResourceProvider: true,
				},
				kubevirtcorev1.PciHostDevice{
					PCIVendorSelector:        "vendor2",
					ResourceName:             "resourceName2",
					ExternalResourceProvider: false,
				},
				kubevirtcorev1.PciHostDevice{
					PCIVendorSelector:        "vendor3",
					ResourceName:             "resourceName3",
					ExternalResourceProvider: true,
				},
			))

			Expect(kvCopy.MediatedDevices).To(HaveLen(4))
			Expect(kvCopy.MediatedDevices).To(ContainElements(
				kubevirtcorev1.MediatedHostDevice{
					MDEVNameSelector:         "selector1",
					ResourceName:             "resource1",
					ExternalResourceProvider: true,
				},
				kubevirtcorev1.MediatedHostDevice{
					MDEVNameSelector:         "selector2",
					ResourceName:             "resource2",
					ExternalResourceProvider: false,
				},
				kubevirtcorev1.MediatedHostDevice{
					MDEVNameSelector:         "selector3",
					ResourceName:             "resource3",
					ExternalResourceProvider: true,
				},
				kubevirtcorev1.MediatedHostDevice{
					MDEVNameSelector:         "selector4",
					ResourceName:             "resource4",
					ExternalResourceProvider: false,
				},
			))
		})
	})

	Context("TLSSecurityProfile", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		oldTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileOldType,
			Old:  &openshiftconfigv1.OldTLSProfile{},
		}
		intermediateTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
			Type:         openshiftconfigv1.TLSProfileIntermediateType,
			Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
		}
		modernTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
			Type:   openshiftconfigv1.TLSProfileModernType,
			Modern: &openshiftconfigv1.ModernTLSProfile{},
		}

		kvOldCiphers := []string{
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
			"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
			"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
			"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
			"TLS_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_RSA_WITH_AES_128_CBC_SHA256",
			"TLS_RSA_WITH_AES_128_CBC_SHA",
			"TLS_RSA_WITH_AES_256_CBC_SHA",
			"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		}

		kvIntermediateCiphers := []string{
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		}

		// it's not possible to specify ciphers when minTLSVersion is 1.3
		var kvModernCiphers []string = nil

		DescribeTable("should modify TLSSecurityProfile on Kubevirt CR according to ApiServer or HCO CR",
			func(hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile, expectedKubevirtTLSVersion kubevirtcorev1.TLSProtocolVersion, expectedKubevirtCiphers []string) {
				existingResource, err := NewKubeVirt(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(existingResource.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(Equal(kubevirtcorev1.VersionTLS12))
				Expect(existingResource.Spec.Configuration.TLSConfiguration.Ciphers).To(Equal(kvIntermediateCiphers))

				// now, modify HCO's TLSSecurityProfile
				hco.Spec.TLSSecurityProfile = hcoTLSSecurityProfile

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(Equal(expectedKubevirtTLSVersion))
				Expect(foundResource.Spec.Configuration.TLSConfiguration.Ciphers).To(Equal(expectedKubevirtCiphers))

				Expect(req.Conditions).To(BeEmpty())
			},
			Entry("Setting Old TLSSecurityProfile on HCO",
				oldTLSSecurityProfile,
				kubevirtcorev1.VersionTLS10,
				kvOldCiphers,
			),
			Entry("Setting Modern TLSSecurityProfile on HCO",
				modernTLSSecurityProfile,
				kubevirtcorev1.VersionTLS13,
				kvModernCiphers,
			),
			Entry("Setting Custom TLSSecurityProfile on HCO",
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							Ciphers: []string{
								"ECDHE-ECDSA-AES256-GCM-SHA384",
								"ECDHE-RSA-CHACHA20-POLY1305",
								"ECDHE-RSA-AES128-SHA256",
							},
							MinTLSVersion: openshiftconfigv1.VersionTLS11,
						},
					},
				},
				kubevirtcorev1.VersionTLS11,
				[]string{
					"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
					"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
				},
			),
		)

		It("should overwrite TLSSecurityProfile if directly set on Kubevirt CR", func() {
			hco.Spec.TLSSecurityProfile = intermediateTLSSecurityProfile
			existingResource, err := NewKubeVirt(hco)
			Expect(err).ToNot(HaveOccurred())

			// mock a reconciliation triggered by a change in Kubevirt CR
			req.HCOTriggered = false

			// now, modify Kubevirt TLSConfiguration
			existingResource.Spec.Configuration.TLSConfiguration.MinTLSVersion = kubevirtcorev1.VersionTLS13
			existingResource.Spec.Configuration.TLSConfiguration.Ciphers = kvModernCiphers

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newKubevirtHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &kubevirtcorev1.KubeVirt{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(Equal(kubevirtcorev1.VersionTLS12))
			Expect(foundResource.Spec.Configuration.TLSConfiguration.Ciphers).To(Equal(kvIntermediateCiphers))

			Expect(req.Conditions).To(BeEmpty())
		})

	})

})
