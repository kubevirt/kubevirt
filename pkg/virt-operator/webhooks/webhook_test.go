package webhooks

import (
	"encoding/json"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Delete Webhook", func() {

	var ctrl *gomock.Controller
	var admitter *KubeVirtDeletionAdmitter

	var kubeCli *kubecli.MockKubevirtClient
	var vmirsInterface *kubecli.MockReplicaSetInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var kvInterface *kubecli.MockKubeVirtInterface
	var kv *k6tv1.KubeVirt

	BeforeEach(func() {
		kv = &k6tv1.KubeVirt{}
		kv.Status.Phase = k6tv1.KubeVirtPhaseDeployed
		ctrl = gomock.NewController(GinkgoT())
		kubeCli = kubecli.NewMockKubevirtClient(ctrl)
		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)
		kvInterface.EXPECT().Get("kubevirt", gomock.Any()).Return(kv, nil).AnyTimes()
		kvInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.KubeVirtList{}, nil).AnyTimes()
		admitter = &KubeVirtDeletionAdmitter{kubeCli}
		kubeCli.EXPECT().KubeVirt("test").Return(kvInterface)

		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmirsInterface = kubecli.NewMockReplicaSetInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("if uninstall strategy is BlockUninstallIfWorkloadExists", func() {
		BeforeEach(func() {
			kubeCli.EXPECT().VirtualMachineInstance(v1.NamespaceAll).Return(vmiInterface).AnyTimes()
			kubeCli.EXPECT().ReplicaSet(v1.NamespaceAll).Return(vmirsInterface).AnyTimes()
			kubeCli.EXPECT().VirtualMachine(v1.NamespaceAll).Return(vmInterface).AnyTimes()

			kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
		})

		It("should allow the deletion if no workload exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{}, nil)

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeTrue())
		})

		It("should deny the deletion if a VMI exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{Items: []k6tv1.VirtualMachineInstance{{}}}, nil)

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VM exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{Items: []k6tv1.VirtualMachine{{}}}, nil)

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if a VMIRS exists", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{Items: []k6tv1.VirtualMachineInstanceReplicaSet{{}}}, nil)

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIs fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMs fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})

		It("should deny the deletion if checking VMIRS fails", func() {
			vmiInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceList{}, nil)
			vmInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineList{}, nil)
			vmirsInterface.EXPECT().List(gomock.Any()).Return(&k6tv1.VirtualMachineInstanceReplicaSetList{}, fmt.Errorf("whatever"))

			response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
			Expect(response.Allowed).To(BeFalse())
		})
	})

	It("should allow the deletion if the strategy is empty", func() {
		kv.Spec.UninstallStrategy = ""
		response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion if the strategy is set to RemoveWorkloads", func() {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	})

	It("should allow the deletion of namespaces, where it gets an admission request without a resource name", func() {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyRemoveWorkloads
		response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: ""}})
		Expect(response.Allowed).To(BeTrue())
	})

	table.DescribeTable("should not check for workloads if kubevirt phase is", func(phase k6tv1.KubeVirtPhase) {
		kv.Spec.UninstallStrategy = k6tv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
		kv.Status.Phase = phase
		response := admitter.Admit(&v1beta1.AdmissionReview{Request: &v1beta1.AdmissionRequest{Namespace: "test", Name: "kubevirt"}})
		Expect(response.Allowed).To(BeTrue())
	},
		table.Entry("unset", k6tv1.KubeVirtPhase("")),
		table.Entry("deploying", k6tv1.KubeVirtPhaseDeploying),
		table.Entry("deleting", k6tv1.KubeVirtPhaseDeleting),
		table.Entry("deleted", k6tv1.KubeVirtPhaseDeleted),
	)
})

var _ = Describe("KubeVirt Mutating Webhook", func() {
	var ctrl *gomock.Controller
	var admitter *KubeVirtMutationAdmitter

	var kubeCli *kubecli.MockKubevirtClient
	var kv *k6tv1.KubeVirt

	BeforeEach(func() {
		kv = &k6tv1.KubeVirt{Spec: k6tv1.KubeVirtSpec{MetricsConfig: &k6tv1.MetricsConfig{}}}
		ctrl = gomock.NewController(GinkgoT())
		kubeCli = kubecli.NewMockKubevirtClient(ctrl)
		admitter = NewKubeVirtMutationAdmitter(kubeCli)

	})
	Context("by setting invalid metrics configuration", func() {
		It("it should deny metrics with missing BucketValues field", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: nil,
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Message).To(BeEquivalentTo(fmt.Errorf(missingFieldErrorMsg, "migration").Error()))
		})

		It("it should deny metrics with unordered buckets", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: []float64{60, 30, 150},
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Message).To(BeEquivalentTo(unorderedBucketsErrorMsg))
		})

		It("it should deny metrics with invalid initial bucket", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: []float64{-1, 30, 150},
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Message).To(BeEquivalentTo(invalidInitialBucketErrorMsg))
		})

		It("it should deny metrics with repeating buckets", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: []float64{30, 30, 150},
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Message).To(BeEquivalentTo(repeatingBucketsErrorMsg))
		})

		It("it should deny metrics with insufficient buckets", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: []float64{30},
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Message).To(BeEquivalentTo(insufficientBucketsErrorMsg))
		})
	})

	Context("by setting valid metrics configuration", func() {
		It("it should allow valid metrics config", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = &k6tv1.HistogramsConfig{
				DurationHistogram: &k6tv1.HistogramMetric{
					BucketValues: []float64{60, 180, 300, 1800, 3600, 36000},
				},
			}
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeTrue())
		})

		It("it should allow missing migration metrics config", func() {
			kv.Spec.MetricsConfig.MigrationMetrics = nil
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeTrue())
		})

		It("it should allow missing metrics config", func() {
			kv.Spec.MetricsConfig = nil
			kvBytes, _ := json.Marshal(&kv)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: kvBytes,
					},
				},
			}

			response := admitter.Admit(ar)
			Expect(response.Allowed).To(BeTrue())
		})
	})
})
