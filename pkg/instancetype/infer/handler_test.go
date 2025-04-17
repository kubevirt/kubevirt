package infer_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	fakeclientset "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/infer"
	"kubevirt.io/kubevirt/pkg/pointer"
)

type inferHandler interface {
	Infer(vm *v1.VirtualMachine) error
}

var _ = Describe("InferFromVolume", func() {
	var (
		vm         *v1.VirtualMachine
		virtClient *kubecli.MockKubevirtClient
		handler    inferHandler

		pvc             *k8sv1.PersistentVolumeClaim
		dvWithSourcePVC *cdiv1.DataVolume
		dvWithLabels    *cdiv1.DataVolume
		dsWithSourcePVC *cdiv1.DataSource
		dsWithLabels    *cdiv1.DataSource
	)

	const (
		inferVolumeName           = "inferVolumeName"
		defaultInferedNameFromPVC = "defaultInferedNameFromPVC"
		defaultInferedKindFromPVC = "defaultInferedKindFromPVC"
		defaultInferedNameFromDV  = "defaultInferedNameFromDV"
		defaultInferedKindFromDV  = "defaultInferedKindFromDV"
		defaultInferedNameFromDS  = "defaultInferedNameFromDS"
		defaultInferedKindFromDS  = "defaultInferedKindFromDS"
		pvcName                   = "pvcName"
		dvWithSourcePVCName       = "dvWithSourcePVCName"
		dsWithSourcePVCName       = "dsWithSourcePVCName"
		dsWithLabelsName          = "dsWithLabelsName"
		unknownPVCName            = "unknownPVCName"
		unknownDVName             = "unknownDVName"
	)

	BeforeEach(func() {
		vm = &v1.VirtualMachine{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().CoreV1().Return(k8sfake.NewSimpleClientset().CoreV1()).AnyTimes()
		virtClient.EXPECT().CdiClient().Return(cdifake.NewSimpleClientset()).AnyTimes()

		virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachinePreferences(vm.Namespace)).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		pvc = &k8sv1.PersistentVolumeClaim{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      pvcName,
				Namespace: vm.Namespace,
				Labels: map[string]string{
					apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromPVC,
					apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromPVC,
					apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromPVC,
					apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromPVC,
				},
			},
		}
		var err error
		pvc, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvc, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		dvWithSourcePVC = &cdiv1.DataVolume{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      dvWithSourcePVCName,
				Namespace: vm.Namespace,
			},
			Spec: cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Name:      pvc.Name,
						Namespace: pvc.Namespace,
					},
				},
			},
		}
		dvWithSourcePVC, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(
			context.Background(), dvWithSourcePVC, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		dsWithSourcePVC = &cdiv1.DataSource{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      dsWithSourcePVCName,
				Namespace: vm.Namespace,
			},
			Spec: cdiv1.DataSourceSpec{
				Source: cdiv1.DataSourceSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Name:      pvc.Name,
						Namespace: pvc.Namespace,
					},
				},
			},
		}
		dsWithSourcePVC, err = virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(
			context.Background(), dsWithSourcePVC, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		dsWithLabels = &cdiv1.DataSource{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      dsWithLabelsName,
				Namespace: vm.Namespace,
				Labels: map[string]string{
					apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromDS,
					apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromDS,
					apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromDS,
					apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromDS,
				},
			},
			Spec: cdiv1.DataSourceSpec{
				Source: cdiv1.DataSourceSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Name:      pvc.Name,
						Namespace: pvc.Namespace,
					},
				},
			},
		}
		_, err = virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(
			context.Background(), dsWithLabels, k8smetav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		handler = infer.New(virtClient)
	})

	DescribeTable("should infer defaults from VolumeSource and PersistentVolumeClaim",
		func(
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
				},
			}}
			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry("for PreferenceMatcher",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
	)

	DescribeTable("should infer defaults from DataVolumeSource and PersistentVolumeClaim",
		func(
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithSourcePVCName,
					},
				},
			}}

			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry("for PreferenceMatcher",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
	)

	DescribeTable("should infer defaults from DataVolumeTemplate, DataVolumeSourcePVC and PersistentVolumeClaim",
		func(
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}}

			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry("for PreferenceMatcher",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
	)
	DescribeTable("should infer defaults from DataVolume with labels",
		func(
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithLabels = &cdiv1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithLabels",
					Namespace: vm.Namespace,
					Labels: map[string]string{
						apiinstancetype.DefaultInstancetypeLabel:     defaultInferedNameFromDV,
						apiinstancetype.DefaultInstancetypeKindLabel: defaultInferedKindFromDV,
						apiinstancetype.DefaultPreferenceLabel:       defaultInferedNameFromDV,
						apiinstancetype.DefaultPreferenceKindLabel:   defaultInferedKindFromDV,
					},
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Name:      pvc.Name,
							Namespace: pvc.Namespace,
						},
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(
				context.Background(), dvWithLabels, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithLabels.Name,
					},
				},
			})

			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromDV,
				Kind: defaultInferedKindFromDV,
			}, nil, nil,
		),
		Entry("for PreferenceMatcher",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromDV,
				Kind: defaultInferedKindFromDV,
			},
		),
	)

	DescribeTable("should infer defaults from DataVolume, DataVolumeSourceRef",
		func(
			sourceRefName, sourceRefKind, sourceRefNamespace string,
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			var sourceRefNamespacePointer *string
			if sourceRefNamespace != "" {
				sourceRefNamespacePointer = &sourceRefNamespace
			}
			dvWithSourceRef := &cdiv1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Name:      sourceRefName,
						Kind:      sourceRefKind,
						Namespace: sourceRefNamespacePointer,
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(
				context.Background(), dvWithSourceRef, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithSourceRef.Name,
					},
				},
			}}

			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry(",DataSource and PersistentVolumeClaim for InstancetypeMatcher",
			dsWithSourcePVCName, "DataSource", k8sv1.NamespaceDefault,
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry(",DataSource and PersistentVolumeClaim for PreferenceMatcher",
			dsWithSourcePVCName, "DataSource", k8sv1.NamespaceDefault,
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
		Entry("and DataSource with annotations for InstancetypeMatcher",
			dsWithLabelsName, "DataSource", k8sv1.NamespaceDefault,
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			}, nil, nil,
		),
		Entry("and DataSource with annotations for PreferenceMatcher",
			dsWithLabelsName, "DataSource", k8sv1.NamespaceDefault,
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			},
		),
		Entry(",DataSource without namespace and PersistentVolumeClaim for InstancetypeMatcher",
			dsWithSourcePVCName, "DataSource", "",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry(",DataSource without namespace and PersistentVolumeClaim for PreferenceMatcher",
			dsWithSourcePVCName, "DataSource", "",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
		Entry("and DataSource without namespace with annotations for InstancetypeMatcher",
			dsWithLabelsName, "DataSource", "",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			}, nil, nil,
		),
		Entry("and DataSource without namespace with annotations for PreferenceMatcher",
			dsWithLabelsName, "DataSource", "",
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			},
		),
	)

	DescribeTable("should infer defaults from DataVolumeTemplate, DataVolumeSourceRef, DataSource and PersistentVolumeClaim",
		func(
			sourceRefName, sourceRefNamespace string,
			instancetypeMatcher, expectedInstancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher, expectedPreferenceMatcher *v1.PreferenceMatcher,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Name:      sourceRefName,
						Kind:      "DataSource",
						Namespace: &sourceRefNamespace,
					},
				},
			}}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}

			Expect(handler.Infer(vm)).To(Succeed())
			Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))
			Expect(vm.Spec.Preference).To(Equal(expectedPreferenceMatcher))
		},
		Entry("for InstancetypeMatcher",
			dsWithSourcePVCName, k8sv1.NamespaceDefault,
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			}, nil, nil,
		),
		Entry("for PreferenceMatcher",
			dsWithSourcePVCName, k8sv1.NamespaceDefault,
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromPVC,
				Kind: defaultInferedKindFromPVC,
			},
		),
		Entry("and DataSource with annotations for InstancetypeMatcher",
			dsWithLabelsName, k8sv1.NamespaceDefault,
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.InstancetypeMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			}, nil, nil,
		),
		Entry("and DataSource with annotations for PreferenceMatcher",
			dsWithLabelsName, k8sv1.NamespaceDefault,
			nil, nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
			&v1.PreferenceMatcher{
				Name: defaultInferedNameFromDS,
				Kind: defaultInferedKindFromDS,
			},
		),
	)

	DescribeTable("should fail to infer defaults from unknown Volume ",
		func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher

			// Remove all volumes to cause the failure
			vm.Spec.Template.Spec.Volumes = nil

			Expect(handler.Infer(vm)).To(MatchError(ContainSubstring("unable to find volume %s to infer defaults", inferVolumeName)))
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil,
		),
		Entry("for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil,
		),
		Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil,
		),
		Entry("for PreferenceMatcher",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			},
		),
		Entry("for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			},
		),
		Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			},
		),
	)

	DescribeTable("should fail to infer defaults from Volume ",
		func(
			volumeSource v1.VolumeSource,
			messageSubstring string,
			instancetypeMatcher *v1.InstancetypeMatcher,
			preferenceMatcher *v1.PreferenceMatcher,
			allowed bool,
		) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name:         inferVolumeName,
				VolumeSource: volumeSource,
			}}

			if allowed {
				// Expect matchers to be cleared on failure during inference
				Expect(handler.Infer(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			} else {
				Expect(handler.Infer(vm)).To(MatchError(ContainSubstring(messageSubstring)))
			}
		},
		Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("with unknown PersistentVolumeClaim for InstancetypeMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("with unknown PersistentVolumeClaim for PreferenceMatcher",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("with unknown PersistentVolumeClaim for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, false,
		),
		Entry("with unknown PersistentVolumeClaim for PreferenceMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: unknownPVCName,
					},
				},
			},
			fmt.Sprintf("persistentvolumeclaims %q not found", unknownPVCName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for InstancetypeMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, false,
		),
		Entry("with unknown DataVolume and PersistentVolumeClaim for PreferenceMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: unknownDVName,
				},
			}, fmt.Sprintf("persistentvolumeclaims %q not found", unknownDVName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
		Entry("with unsupported VolumeSource type for InstancetypeMatcher",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			},
			fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("but still admit with unsupported VolumeSource type for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			}, "",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, true,
		),
		Entry("with unsupported VolumeSource type for InstancetypeMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			},
			fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("with unsupported VolumeSource type for PreferenceMatcher",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			},
			fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("but still admit with unsupported VolumeSource type for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			}, "", nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, true,
		),
		Entry("with unsupported VolumeSource type for PreferenceMatcher with RejectInferFromVolumeFailure",
			v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{},
			},
			fmt.Sprintf("unable to infer defaults from volume %s as type is not supported", inferVolumeName),
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
	)

	DescribeTable("should fail to infer defaults from DataVolume with an unsupported DataVolumeSource",
		func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithUnsupportedSource := &cdiv1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						VDDK: &cdiv1.DataVolumeSourceVDDK{},
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(
				context.Background(), dvWithUnsupportedSource, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithUnsupportedSource.Name,
					},
				},
			}}

			if allowed {
				// Expect matchers to be cleared on failure during inference
				Expect(handler.Infer(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			} else {
				Expect(handler.Infer(vm)).To(
					MatchError(ContainSubstring("unable to infer defaults from DataVolumeSpec as DataVolumeSource is not supported")))
			}
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, true,
		),
		Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("for PreferenceMatcher",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, true,
		),
		Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
	)

	DescribeTable("should fail to infer defaults from DataVolume with an unknown DataVolumeSourceRef Kind",
		func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dvWithUnknownSourceRefKind := &cdiv1.DataVolume{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dvWithSourceRef",
					Namespace: vm.Namespace,
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Kind: "foo",
					},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(
				context.Background(), dvWithUnknownSourceRefKind, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvWithUnknownSourceRefKind.Name,
					},
				},
			}}

			if allowed {
				// Expect matchers to be cleared on failure during inference
				Expect(handler.Infer(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			} else {
				Expect(handler.Infer(vm)).To(
					MatchError(ContainSubstring("unable to infer defaults from DataVolumeSourceRef as Kind foo is not supported")))
			}
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, true,
		),
		Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("for PreferenceMatcher",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, true,
		),
		Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
	)

	DescribeTable("should fail to infer defaults from DataSource missing DataVolumeSourcePVC",
		func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			dsWithoutSourcePVC := &cdiv1.DataSource{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "dsWithoutSourcePVC",
					Namespace: vm.Namespace,
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{},
				},
			}
			_, err := virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(
				context.Background(), dsWithoutSourcePVC, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "dataVolume",
				},
				Spec: cdiv1.DataVolumeSpec{
					SourceRef: &cdiv1.DataVolumeSourceRef{
						Kind:      "DataSource",
						Name:      dsWithoutSourcePVC.Name,
						Namespace: &dsWithoutSourcePVC.Namespace,
					},
				},
			}}
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: "dataVolume",
					},
				},
			}}

			if allowed {
				// Expect matchers to be cleared on failure during inference
				Expect(handler.Infer(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			} else {
				Expect(handler.Infer(vm)).To(
					MatchError(ContainSubstring("unable to infer defaults from DataSource that doesn't provide DataVolumeSourcePVC")))
			}
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, false,
		),
		Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, true,
		),
		Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, false,
		),
		Entry("for PreferenceMatcher",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, false,
		),
		Entry("but still admit for PreferenceMatcher with IgnoreInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, true,
		),
		Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, false,
		),
	)

	DescribeTable("should fail to infer defaults from PersistentVolumeClaim without default instance type label",
		func(instancetypeMatcher *v1.InstancetypeMatcher, preferenceMatcher *v1.PreferenceMatcher, requiredLabel string, allowed bool) {
			vm.Spec.Instancetype = instancetypeMatcher
			vm.Spec.Preference = preferenceMatcher
			pvcWithoutLabels := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "pvcWithoutLabels",
					Namespace: vm.Namespace,
				},
			}
			_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(
				context.Background(), pvcWithoutLabels, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: inferVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcWithoutLabels.Name,
						},
					},
				},
			}}

			if allowed {
				// Expect matchers to be cleared on failure during inference
				Expect(handler.Infer(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
			} else {
				Expect(handler.Infer(vm)).To(MatchError(ContainSubstring("unable to find required %s label on the volume", requiredLabel)))
			}
		},
		Entry("for InstancetypeMatcher",
			&v1.InstancetypeMatcher{
				InferFromVolume: inferVolumeName,
			}, nil, apiinstancetype.DefaultInstancetypeLabel, false,
		),
		Entry("but still admit for InstancetypeMatcher with IgnoreInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, nil, apiinstancetype.DefaultInstancetypeLabel, true,
		),
		Entry("for InstancetypeMatcher with RejectInferFromVolumeFailure",
			&v1.InstancetypeMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, nil, apiinstancetype.DefaultInstancetypeLabel, false,
		),
		Entry("for PreferenceMatcher",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume: inferVolumeName,
			}, apiinstancetype.DefaultPreferenceLabel, false,
		),
		Entry("but still admit for PreferenceMatcher with with IgnoreInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.IgnoreInferFromVolumeFailure),
			}, apiinstancetype.DefaultPreferenceLabel, true,
		),
		Entry("for PreferenceMatcher with RejectInferFromVolumeFailure",
			nil,
			&v1.PreferenceMatcher{
				InferFromVolume:              inferVolumeName,
				InferFromVolumeFailurePolicy: pointer.P(v1.RejectInferFromVolumeFailure),
			}, apiinstancetype.DefaultPreferenceLabel, false,
		),
	)

	It("should infer defaults from garbage collected DataVolume using PVC with the same name", func() {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			InferFromVolume: inferVolumeName,
		}
		vm.Spec.Preference = &v1.PreferenceMatcher{
			InferFromVolume: inferVolumeName,
		}
		// No DataVolume with the name of pvcName exists but a PVC does
		vm.Spec.Template.Spec.Volumes = []v1.Volume{{
			Name: inferVolumeName,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: pvcName,
				},
			},
		}}
		Expect(handler.Infer(vm)).To(Succeed())
		Expect(vm.Spec.Instancetype).To(Equal(&v1.InstancetypeMatcher{
			Name: defaultInferedNameFromPVC,
			Kind: defaultInferedKindFromPVC,
		}))
		Expect(vm.Spec.Preference).To(Equal(&v1.PreferenceMatcher{
			Name: defaultInferedNameFromPVC,
			Kind: defaultInferedKindFromPVC,
		}))
	})

	DescribeTable("When inference was successful", func(failurePolicy v1.InferFromVolumeFailurePolicy, expectMemoryCleared bool) {
		By("Setting guest memory")
		guestMemory := resource.MustParse("512Mi")
		vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
			Guest: &guestMemory,
		}

		By("Creating a VM using a PVC as boot and inference Volume")
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			InferFromVolume:              inferVolumeName,
			InferFromVolumeFailurePolicy: &failurePolicy,
		}
		vm.Spec.Template.Spec.Volumes = []v1.Volume{{
			Name: inferVolumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc.Name,
					},
				},
			},
		}}
		Expect(handler.Infer(vm)).To(Succeed())

		expectedInstancetypeMatcher := &v1.InstancetypeMatcher{
			Name: defaultInferedNameFromPVC,
			Kind: defaultInferedKindFromPVC,
		}
		Expect(vm.Spec.Instancetype).To(Equal(expectedInstancetypeMatcher))

		if expectMemoryCleared {
			Expect(vm.Spec.Template.Spec.Domain.Memory).To(BeNil())
		} else {
			Expect(vm.Spec.Template.Spec.Domain.Memory).ToNot(BeNil())
			Expect(*vm.Spec.Template.Spec.Domain.Memory.Guest).To(HaveValue(Equal(guestMemory)))
		}
	},
		Entry("it should clear guest memory when ignoring inference failures", v1.IgnoreInferFromVolumeFailure, true),
		Entry("it should not clear guest memory when rejecting inference failures", v1.RejectInferFromVolumeFailure, false),
	)
})
