package guestfs_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
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

	virtctlcmd "kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	commandName   = "guestfs"
	pvcName       = "test-pvc"
	testNamespace = "default"
)

func fakeAttacherCreator(client *guestfs.K8sClient, p *corev1.Pod, command string) error {
	return nil
}

func fakeSetImage(virtClient kubecli.KubevirtClient) (string, error) {
	return "", nil
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

	Context("attach to PVC", func() {
		BeforeEach(func() {
			guestfs.ImageSetFunc = fakeSetImage
			guestfs.CreateAttacherFunc = fakeAttacherCreator
		})

		AfterEach(func() {
			guestfs.ImageSetFunc = guestfs.SetImage
			guestfs.CreateAttacherFunc = guestfs.CreateAttacher
			guestfs.CreateClientFunc = guestfs.CreateClient
		})

		It("Succesfully attach to PVC", func() {
			guestfs.CreateClientFunc = fakeCreateClientPVC
			cmd := virtctlcmd.NewRepeatableVirtctlCommand(commandName, pvcName)
			Expect(cmd()).To(Succeed())
		})

		It("PVC in use", func() {
			guestfs.CreateClientFunc = fakeCreateClientPVCinUse
			cmd := virtctlcmd.NewRepeatableVirtctlCommand(commandName, pvcName)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("PVC %s is used by another pod", pvcName)))
		})

		It("PVC doesn't exist", func() {
			guestfs.CreateClientFunc = fakeCreateClient
			cmd := virtctlcmd.NewRepeatableVirtctlCommand(commandName, pvcName)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("The PVC %s doesn't exist", pvcName)))
		})

		It("UID cannot be used with root", func() {
			guestfs.CreateClientFunc = fakeCreateClientPVC
			cmd := virtctlcmd.NewRepeatableVirtctlCommand(commandName, pvcName, "--root=true", "--uid=1001")
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("cannot set uid if root is true")))
		})
		It("GID can be use only together with the uid flag", func() {
			guestfs.CreateClientFunc = fakeCreateClientPVC
			cmd := virtctlcmd.NewRepeatableVirtctlCommand(commandName, pvcName, "--gid=1001")
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("gid requires the uid to be set")))
		})
	})

	Context("URL authenticity", func() {
		fakeGetImageInfoNoCustomURL := func(virtClient kubecli.KubevirtClient) (*kubecli.GuestfsInfo, error) {
			info := &kubecli.GuestfsInfo{
				Registry:    "someregistry.io/kubevirt",
				Tag:         "sha256:07c601d33793ee987g5417d755665572dc9a9680cea01dfb9bdbcc3ecf866720",
				Digest:      "89af657d3c226ac3083a0986e19efe70c9ccd7e7278137e9df24b9b430182aa7",
				ImagePrefix: "some-prefix-",
			}

			return info, nil
		}

		fakeGetImageInfoWithCustomURL := func(virtClient kubecli.KubevirtClient) (*kubecli.GuestfsInfo, error) {
			info := &kubecli.GuestfsInfo{
				GsImage: "someregistry.io/kubevirt/libguestfs-tools-centos9@sha256:128736c7736a8791fb8a8de7d92a4f9be886dc6d8e77d01db8bd55253399099b",
			}

			return info, nil
		}

		fakeGetImageInfoWithCustomURLAndRegistrySpecifics := func(virtClient kubecli.KubevirtClient) (*kubecli.GuestfsInfo, error) {
			info := &kubecli.GuestfsInfo{
				Registry:    "someregistry.io/kubevirt",
				Tag:         "sha256:07c601d33793ee987g5417d755665572dc9a9680cea01dfb9bdbcc3ecf866720",
				Digest:      "89af657d3c226ac3083a0986e19efe70c9ccd7e7278137e9df24b9b430182aa7",
				ImagePrefix: "some-prefix-",
				GsImage:     "someregistry.io/kubevirt/libguestfs-tools-centos9@sha256:128736c7736a8791fb8a8de7d92a4f9be886dc6d8e77d01db8bd55253399099b",
			}

			return info, nil
		}

		AfterEach(func() {
			guestfs.ImageInfoGetFunc = guestfs.GetImageInfo
		})

		It("Image prefix from kubevirt config not discarded", func() {
			guestfs.ImageInfoGetFunc = fakeGetImageInfoNoCustomURL
			Expect(guestfs.ImageSetFunc(kubevirtClient)).Should(HaveValue(Equal("someregistry.io/kubevirt/some-prefix-libguestfs-tools@89af657d3c226ac3083a0986e19efe70c9ccd7e7278137e9df24b9b430182aa7")))
		})

		It("Image override alone is enough to assemble url", func() {
			guestfs.ImageInfoGetFunc = fakeGetImageInfoWithCustomURL
			Expect(guestfs.ImageSetFunc(kubevirtClient)).To(HaveValue(Equal("someregistry.io/kubevirt/libguestfs-tools-centos9@sha256:128736c7736a8791fb8a8de7d92a4f9be886dc6d8e77d01db8bd55253399099b")))
		})

		It("Image override takes precedence over registry specifics", func() {
			guestfs.ImageInfoGetFunc = fakeGetImageInfoWithCustomURLAndRegistrySpecifics
			Expect(guestfs.ImageSetFunc(kubevirtClient)).Should(HaveValue(Equal("someregistry.io/kubevirt/libguestfs-tools-centos9@sha256:128736c7736a8791fb8a8de7d92a4f9be886dc6d8e77d01db8bd55253399099b")))
		})
	})
})
