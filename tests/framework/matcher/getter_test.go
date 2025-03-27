package matcher

import (
	"context"
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	kubevirtv1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

var (
	ctrl       *gomock.Controller
	virtClient *kubecli.MockKubevirtClient
)

var _ = Describe("Getter Matchers", func() {
	var (
		virtClientset *kubevirtfake.Clientset
		kubeClient    *fake.Clientset
		cdiClient     *cdifake.Clientset
	)

	BeforeEach(func() {
		// Initialize the controller and mock client for each test to ensure isolation
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClientset = kubevirtfake.NewSimpleClientset()
		kubeClient = fake.NewSimpleClientset()
		cdiConfig := &cdiv1.CDIConfig{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: storagetypes.ConfigName,
			},
			Spec: cdiv1.CDIConfigSpec{
				UploadProxyURLOverride: nil,
			},
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: cdiv1.Percent(storagetypes.DefaultFSOverhead),
				},
			},
		}
		cdiClient = cdifake.NewSimpleClientset(cdiConfig)

		// Set up the mock client expectations for each test
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8sv1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachines(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().AppsV1().Return(kubeClient.AppsV1()).AnyTimes()
		virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		// Override getClient to return the mock client
		getClient = func() kubecli.KubevirtClient {
			return virtClient
		}
	})

	Context("ThisPodWith", func() {
		const (
			name = "test-pod"
		)

		var (
			pod *v1.Pod
			err error
		)

		DescribeTable("should handle pod retrieval correctly", func(setup func(), expectedPodExists bool, expectedErr error) {
			setup()
			podFunc := ThisPodWith(k8sv1.NamespaceDefault, name)
			pod, err = podFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(pod).To(BeNil())
			} else if expectedPodExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(pod).ToNot(BeNil())
				Expect(pod.Name).To(Equal(name))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(pod).To(BeNil())
			}
		},
			Entry("when the pod exists", func() {
				kubeClient.CoreV1().Pods(k8sv1.NamespaceDefault).Create(context.Background(), &v1.Pod{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name: name,
					},
				}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the pod does not exist", func() {}, false, nil),
			Entry("when different error occurs", func() {
				kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisVMIWith", func() {
		const (
			vmiName = "test-vmi"
		)

		var (
			vmi *kubevirtv1.VirtualMachineInstance
			err error
		)

		DescribeTable("should handle VMI retrieval correctly", func(setup func(), expectedVMIExists bool, expectedErr error) {
			setup()
			vmiFunc := ThisVMIWith(k8sv1.NamespaceDefault, vmiName)
			vmi, err = vmiFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(vmi).To(BeNil())
			} else if expectedVMIExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).ToNot(BeNil())
				Expect(vmi.Name).To(Equal(vmiName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).To(BeNil())
			}
		},
			Entry("when the VMI exists", func() {
				virtClientset.KubevirtV1().VirtualMachineInstances(k8sv1.NamespaceDefault).Create(
					context.Background(), &kubevirtv1.VirtualMachineInstance{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: vmiName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the VMI does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				virtClientset.Fake.PrependReactor("get", "virtualmachineinstances", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisVMWith", func() {
		const (
			vmName = "test-vm"
		)

		var (
			vm  *kubevirtv1.VirtualMachine
			err error
		)

		DescribeTable("should handle VM retrieval correctly", func(setup func(), expectedVMExists bool, expectedErr error) {
			setup()
			vmFunc := ThisVMWith(k8sv1.NamespaceDefault, vmName)
			vm, err = vmFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(vm).To(BeNil())
			} else if expectedVMExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(vm).ToNot(BeNil())
				Expect(vm.Name).To(Equal(vmName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(vm).To(BeNil())
			}
		},
			Entry("when the VM exists", func() {
				virtClientset.KubevirtV1().VirtualMachines(k8sv1.NamespaceDefault).Create(
					context.Background(), &kubevirtv1.VirtualMachine{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: vmName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the VM does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				virtClientset.Fake.PrependReactor("get", "virtualmachines", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisDVWith", func() {
		const (
			dvName = "test-dv"
		)

		var (
			dv  *cdiv1.DataVolume
			err error
		)

		DescribeTable("should handle DV retrieval correctly", func(setup func(), expectedDVExists bool, expectedErr error) {
			setup()
			dvFunc := ThisDVWith(k8sv1.NamespaceDefault, dvName)
			dv, err = dvFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(dv).To(BeNil())
			} else if expectedDVExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(dv).ToNot(BeNil())
				Expect(dv.Name).To(Equal(dvName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(dv).To(BeNil())
			}
		},
			Entry("when the DV exists", func() {
				cdiClient.CdiV1beta1().DataVolumes(k8sv1.NamespaceDefault).Create(
					context.Background(), &cdiv1.DataVolume{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: dvName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the DV does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				cdiClient.Fake.PrependReactor("get", "datavolumes", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisPVCWith", func() {
		const (
			pvcName = "test-pvc"
		)

		var (
			pvc *v1.PersistentVolumeClaim
			err error
		)

		DescribeTable("should handle PVC retrieval correctly", func(setup func(), expectedPVCExists bool, expectedErr error) {
			setup()
			pvcFunc := ThisPVCWith(k8sv1.NamespaceDefault, pvcName)
			pvc, err = pvcFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(pvc).To(BeNil())
			} else if expectedPVCExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc).ToNot(BeNil())
				Expect(pvc.Name).To(Equal(pvcName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc).To(BeNil())
			}
		},
			Entry("when the PVC exists", func() {
				kubeClient.CoreV1().PersistentVolumeClaims(k8sv1.NamespaceDefault).Create(
					context.Background(), &v1.PersistentVolumeClaim{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: pvcName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the PVC does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisMigrationWith", func() {
		const (
			migrationName = "test-migration"
		)

		var (
			migration *kubevirtv1.VirtualMachineInstanceMigration
			err       error
		)

		DescribeTable("should handle Migration retrieval correctly", func(setup func(), expectedMigrationExists bool, expectedErr error) {
			setup()
			migrationFunc := ThisMigrationWith(k8sv1.NamespaceDefault, migrationName)
			migration, err = migrationFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(migration).To(BeNil())
			} else if expectedMigrationExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(migration).ToNot(BeNil())
				Expect(migration.Name).To(Equal(migrationName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(migration).To(BeNil())
			}
		},
			Entry("when the Migration exists", func() {
				virtClientset.KubevirtV1().VirtualMachineInstanceMigrations(k8sv1.NamespaceDefault).Create(
					context.Background(), &kubevirtv1.VirtualMachineInstanceMigration{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: migrationName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the Migration does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				virtClientset.Fake.PrependReactor("get", "virtualmachineinstancemigrations", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})

	Context("ThisDeploymentWith", func() {
		const (
			deploymentName = "test-deployment"
		)

		var (
			deployment *appsv1.Deployment
			err        error
		)

		DescribeTable("should handle Deployment retrieval correctly", func(setup func(), expectedDeploymentExists bool, expectedErr error) {
			setup()
			deploymentFunc := ThisDeploymentWith(k8sv1.NamespaceDefault, deploymentName)
			deployment, err = deploymentFunc()

			if expectedErr != nil {
				Expect(err).To(MatchError(expectedErr))
				Expect(deployment).To(BeNil())
			} else if expectedDeploymentExists {
				Expect(err).ToNot(HaveOccurred())
				Expect(deployment).ToNot(BeNil())
				Expect(deployment.Name).To(Equal(deploymentName))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(deployment).To(BeNil())
			}
		},
			Entry("when the Deployment exists", func() {
				kubeClient.AppsV1().Deployments(k8sv1.NamespaceDefault).Create(
					context.Background(), &appsv1.Deployment{
						ObjectMeta: k8smetav1.ObjectMeta{
							Name: deploymentName,
						},
					}, k8smetav1.CreateOptions{})
			}, true, nil),
			Entry("when the Deployment does not exist", func() {}, false, nil),
			Entry("when an error occurs", func() {
				kubeClient.Fake.PrependReactor("get", "deployments", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("error")
				})
			}, false, errors.New("error")),
		)
	})
})
