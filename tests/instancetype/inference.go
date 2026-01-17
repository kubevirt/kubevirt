//nolint:lll
package instancetype

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences inference", Serial, decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var (
		virtClient   kubecli.KubevirtClient
		vm           *virtv1.VirtualMachine
		instancetype *instancetypev1beta1.VirtualMachineInstancetype
		preference   *instancetypev1beta1.VirtualMachinePreference
		sourceDV     *cdiv1beta1.DataVolume
		namespace    string
	)

	const (
		inferFromVolumeName     = "volume"
		dataVolumeTemplateName  = "datatemplate"
		dvSuccessTimeoutSeconds = 180
	)

	createAndValidateVirtualMachine := func() {
		By("Creating the VirtualMachine")
		var err error
		libvmi.WithRunStrategy(virtv1.RunStrategyAlways)(vm)
		vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Validating the VirtualMachine")
		Expect(vm.Spec.Instancetype.Name).To(Equal(instancetype.Name))
		Expect(vm.Spec.Instancetype.Kind).To(Equal(instancetypeapi.SingularResourceName))
		Expect(vm.Spec.Instancetype.InferFromVolume).To(BeEmpty())
		Expect(vm.Spec.Instancetype.InferFromVolumeFailurePolicy).To(BeNil())
		Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
		Expect(vm.Spec.Preference.Kind).To(Equal(instancetypeapi.SingularPreferenceResourceName))
		Expect(vm.Spec.Preference.InferFromVolume).To(BeEmpty())
		Expect(vm.Spec.Preference.InferFromVolumeFailurePolicy).To(BeNil())

		Eventually(matcher.ThisVM(vm)).WithTimeout(timeout).WithPolling(time.Second).Should(matcher.BeReady())

		By("Validating the VirtualMachineInstance")
		var vmi *virtv1.VirtualMachineInstance
		vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetype.Spec.CPU.Guest))
	}

	generateDataVolumeTemplatesFromDataVolume := func(dataVolume *cdiv1beta1.DataVolume) []virtv1.DataVolumeTemplateSpec {
		return []virtv1.DataVolumeTemplateSpec{{
			ObjectMeta: metav1.ObjectMeta{
				Name: dataVolumeTemplateName,
			},
			Spec: dataVolume.Spec,
		}}
	}

	generateVolumesForDataVolumeTemplates := func() []virtv1.Volume {
		return []virtv1.Volume{{
			Name: inferFromVolumeName,
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: dataVolumeTemplateName,
				},
			},
		}}
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if !libstorage.HasCDI() {
			Fail("instance type and preference inferFromVolume tests require CDI to be installed providing the DataVolume and DataSource CRDs")
		}

		namespace = testsuite.GetTestNamespace(nil)

		By("Creating a VirtualMachineInstancetype")
		instancetype = builder.NewInstancetypeFromVMI(nil)
		var err error
		instancetype, err = virtClient.VirtualMachineInstancetype(namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Creating a VirtualMachinePreference")
		preference = builder.NewPreference()
		preference.Spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
			},
		}
		preference, err = virtClient.VirtualMachinePreference(namespace).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		By("Creating source DataVolume and PVC")
		sourceDV = libdv.NewDataVolume(
			libdv.WithNamespace(namespace),
			libdv.WithForceBindAnnotation(),
			libdv.WithBlankImageSource(),
			libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
			libdv.WithDefaultInstancetype(instancetypeapi.SingularResourceName, instancetype.Name),
			libdv.WithDefaultPreference(instancetypeapi.SingularPreferenceResourceName, preference.Name),
		)

		sourceDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), sourceDV, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.EventuallyDV(sourceDV, dvSuccessTimeoutSeconds, matcher.HaveSucceeded())

		// This is the default but it should still be cleared
		failurePolicy := virtv1.RejectInferFromVolumeFailure
		runStrategy := virtv1.RunStrategyHalted

		vm = &virtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "vm-",
				Namespace:    namespace,
			},
			Spec: virtv1.VirtualMachineSpec{
				Instancetype: &virtv1.InstancetypeMatcher{
					InferFromVolume:              inferFromVolumeName,
					InferFromVolumeFailurePolicy: &failurePolicy,
				},
				Preference: &virtv1.PreferenceMatcher{
					InferFromVolume:              inferFromVolumeName,
					InferFromVolumeFailurePolicy: &failurePolicy,
				},
				Template: &virtv1.VirtualMachineInstanceTemplateSpec{
					Spec: virtv1.VirtualMachineInstanceSpec{
						Domain: virtv1.DomainSpec{},
					},
				},
				RunStrategy: &runStrategy,
			},
		}
	})

	It("should infer defaults from PersistentVolumeClaimVolumeSource", func() {
		vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
			Name: inferFromVolumeName,
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: sourceDV.Name,
					},
				},
			},
		}}
		createAndValidateVirtualMachine()
	})

	It("should infer defaults from existing DataVolume with labels", func() {
		vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
			Name: inferFromVolumeName,
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: sourceDV.Name,
				},
			},
		}}
		createAndValidateVirtualMachine()
	})

	DescribeTable("should infer defaults from DataVolumeTemplates", func(generateDataVolumeTemplatesFunc func() []virtv1.DataVolumeTemplateSpec) {
		vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFunc()
		vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()
		createAndValidateVirtualMachine()
	},
		Entry("and DataVolumeSourcePVC",
			func() []virtv1.DataVolumeTemplateSpec {
				dv := libdv.NewDataVolume(
					libdv.WithNamespace(namespace),
					libdv.WithForceBindAnnotation(),
					libdv.WithPVCSource(sourceDV.Namespace, sourceDV.Name),
					libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
				)
				return []virtv1.DataVolumeTemplateSpec{{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeTemplateName,
					},
					Spec: dv.Spec,
				}}
			},
		),
		Entry(", DataVolumeSourceRef and DataSource",
			func() []virtv1.DataVolumeTemplateSpec {
				By("Creating a DataSource")
				// TODO - Replace with libds?
				dataSource := &cdiv1beta1.DataSource{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "datasource-",
						Namespace:    namespace,
					},
					Spec: cdiv1beta1.DataSourceSpec{
						Source: cdiv1beta1.DataSourceSource{
							PVC: &cdiv1beta1.DataVolumeSourcePVC{
								Name:      sourceDV.Name,
								Namespace: namespace,
							},
						},
					},
				}
				dataSource, err := virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				dataVolume := libdv.NewDataVolume(
					libdv.WithNamespace(namespace),
					libdv.WithForceBindAnnotation(),
					libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
					libdv.WithDataVolumeSourceRef("DataSource", namespace, dataSource.Name),
				)

				return generateDataVolumeTemplatesFromDataVolume(dataVolume)
			},
		),
		Entry(", DataVolumeSourceRef and DataSource with labels",
			func() []virtv1.DataVolumeTemplateSpec {
				By("Creating a blank DV and PVC without labels")
				blankDV := libdv.NewDataVolume(
					libdv.WithNamespace(namespace),
					libdv.WithForceBindAnnotation(),
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
				)
				blankDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), blankDV, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(sourceDV, dvSuccessTimeoutSeconds, matcher.HaveSucceeded())

				By("Creating a DataSource")
				// TODO - Replace with libds?
				dataSource := &cdiv1beta1.DataSource{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "datasource-",
						Namespace:    namespace,
						Labels: map[string]string{
							instancetypeapi.DefaultInstancetypeLabel:     instancetype.Name,
							instancetypeapi.DefaultInstancetypeKindLabel: instancetypeapi.SingularResourceName,
							instancetypeapi.DefaultPreferenceLabel:       preference.Name,
							instancetypeapi.DefaultPreferenceKindLabel:   instancetypeapi.SingularPreferenceResourceName,
						},
					},
					Spec: cdiv1beta1.DataSourceSpec{
						Source: cdiv1beta1.DataSourceSource{
							PVC: &cdiv1beta1.DataVolumeSourcePVC{
								Name:      blankDV.Name,
								Namespace: namespace,
							},
						},
					},
				}
				dataSource, err = virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				dataVolume := libdv.NewDataVolume(
					libdv.WithNamespace(namespace),
					libdv.WithForceBindAnnotation(),
					libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
					libdv.WithDataVolumeSourceRef("DataSource", namespace, dataSource.Name),
				)

				return generateDataVolumeTemplatesFromDataVolume(dataVolume)
			},
		),
	)

	It("should ignore failure when trying to infer defaults from DataVolumeSpec with unsupported DataVolumeSource when policy is set", func() {
		guestMemory := resource.MustParse("512Mi")
		vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
			Guest: &guestMemory,
		}

		// Inference from blank image source is not supported
		dv := libdv.NewDataVolume(
			libdv.WithNamespace(namespace),
			libdv.WithForceBindAnnotation(),
			libdv.WithBlankImageSource(),
			libdv.WithStorage(libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce), libdv.StorageWithVolumeSize("1Gi")),
		)
		vm.Spec.DataVolumeTemplates = generateDataVolumeTemplatesFromDataVolume(dv)
		vm.Spec.Template.Spec.Volumes = generateVolumesForDataVolumeTemplates()

		failurePolicy := virtv1.IgnoreInferFromVolumeFailure
		vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
		vm.Spec.Preference.InferFromVolumeFailurePolicy = &failurePolicy

		By("Creating the VirtualMachine")
		var err error
		vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Validating the VirtualMachine")
		Expect(vm.Spec.Instancetype).To(BeNil())
		Expect(vm.Spec.Preference).To(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).ToNot(BeNil())
		Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(guestMemory))
	})

	DescribeTable("should reject VM creation when inference was successful but memory and RejectInferFromVolumeFailure were set", func(explicit bool) {
		guestMemory := resource.MustParse("512Mi")
		vm.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{
			Guest: &guestMemory,
		}

		vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
			Name: inferFromVolumeName,
			VolumeSource: virtv1.VolumeSource{
				PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: sourceDV.Name,
					},
				},
			},
		}}

		if explicit {
			failurePolicy := virtv1.RejectInferFromVolumeFailure
			vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
		}

		By("Creating the VirtualMachine")
		_, err := virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).To(MatchError("admission webhook \"virtualmachine-validator.kubevirt.io\" denied the request: VM field(s) spec.template.spec.domain.memory.guest conflicts with selected instance type"))
	},
		Entry("with explicitly setting RejectInferFromVolumeFailure", true),
		Entry("with implicitly setting RejectInferFromVolumeFailure (default)", false),
	)
})
