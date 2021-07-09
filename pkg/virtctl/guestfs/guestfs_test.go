package guestfs_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/tests"
)

const (
	commandName   = "guestfs"
	pvcName       = "test-pvc"
	testNamespace = "default"
)

func fakeAttacherCreator(client *guestfs.K8sClient, p *corev1.Pod, command string) error {
	return nil
}

func fakeSetImage(virtClient kubecli.KubevirtClient) error {
	return nil
}

var _ = Describe("Guestfs shell", func() {
	var (
		kubeClient     *fake.Clientset
		kubevirtClient *kubecli.MockKubevirtClient
	)
	mode := v1.PersistentVolumeFilesystem
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: testNamespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeMode: &mode,
		},
	}
	fakeCreateClientPVC := func(config *rest.Config, virtClientConfig clientcmd.ClientConfig) (*guestfs.K8sClient, error) {
		kubeClient = fake.NewSimpleClientset(pvc)
		kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (bool, runtime.Object, error) {
			podRunning := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "libguestfs-tools",
					Namespace: testNamespace,
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "virt",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{
									StartedAt: metav1.Time{},
								},
							},
						},
					},
				},
			}
			return true, podRunning, nil
		})
		return &guestfs.K8sClient{Client: kubeClient, VirtClient: kubevirtClient}, nil
	}
	fakeCreateClientPVCinUse := func(config *rest.Config, virtClientConfig clientcmd.ClientConfig) (*guestfs.K8sClient, error) {
		otherPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: testNamespace,
			},
			Spec: v1.PodSpec{
				Volumes: []v1.Volume{
					{
						Name: "volume-test",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
			},
		}

		kubeClient = fake.NewSimpleClientset(pvc, otherPod)
		return &guestfs.K8sClient{Client: kubeClient, VirtClient: kubevirtClient}, nil
	}
	fakeCreateClient := func(config *rest.Config, virtClientConfig clientcmd.ClientConfig) (*guestfs.K8sClient, error) {
		kubeClient = fake.NewSimpleClientset()
		return &guestfs.K8sClient{Client: kubeClient, VirtClient: kubevirtClient}, nil
	}
	BeforeEach(func() {
		guestfs.SetImageSetFunc(fakeSetImage)
		guestfs.SetAttacher(fakeAttacherCreator)
	})
	Context("attach to PVC", func() {

		It("Succesfully attach to PVC", func() {
			guestfs.SetClient(fakeCreateClientPVC)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, pvcName)
			Expect(cmd()).To(BeNil())
		})
		It("PVC in use", func() {
			guestfs.SetClient(fakeCreateClientPVCinUse)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, pvcName)
			err := cmd()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("PVC %s is used by another pod", pvcName)))
		})
		It("PVC doesn't exist", func() {
			guestfs.SetClient(fakeCreateClient)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, pvcName)
			err := cmd()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("The PVC %s doesn't exist", pvcName)))
		})
	})

})
