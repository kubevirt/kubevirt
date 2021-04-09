package tests_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	ioerrorPV  = "ioerror-pv"
	ioerrorPVC = "ioerror-pvc"
	deviceName = "errdev0"
	diskName   = "disk0"
)

var _ = Describe("[Serial] K8s IO events", func() {
	var (
		ns         string
		node       string
		virtClient kubecli.KubevirtClient
		vmi        *v1.VirtualMachineInstance
		sc         = "test-ioerror"
	)
	executeCommandInVirtHandlerPod := func(args []string) error {
		var stdout, stderr string
		pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
		if err != nil {
			return err
		}
		stdout, stderr, err = tests.ExecuteCommandOnPodV2(virtClient, pod, "virt-handler", args)
		if err != nil {
			return fmt.Errorf("Failed excuting command=%v, error=%v, stdout=%s, stderr=%s", args, err, stdout, stderr)
		}
		return nil
	}

	createFaultyDisk := func() {
		var n *corev1.Node
		var err error
		listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
		Expect(err).ToNot(HaveOccurred())
		n, err = virtClient.CoreV1().Nodes().Get(context.Background(), virtHandlerPods.Items[0].Spec.NodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		node = n.ObjectMeta.Name
		args := []string{"dmsetup", "create", deviceName, "--table", "0 204791 error"}
		err = executeCommandInVirtHandlerPod(args)
		Expect(err).ToNot(HaveOccurred())
	}

	removeFaultyDisk := func() error {
		args := []string{"dmsetup", "remove", deviceName}
		return executeCommandInVirtHandlerPod(args)
	}

	createPVCwithFaultyDisk := func(ns string) {
		size := resource.MustParse("1Gi")
		vMode := corev1.PersistentVolumeBlock
		affinity := corev1.VolumeNodeAffinity{
			Required: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{node},
							},
						},
					},
				},
			},
		}
		pv := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: ioerrorPV,
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity:         map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: size},
				StorageClassName: sc,
				VolumeMode:       &vMode,
				NodeAffinity:     &affinity,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					Local: &corev1.LocalVolumeSource{
						Path: "/dev/mapper/" + deviceName,
					},
				},
			},
		}
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: ioerrorPVC,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeMode:       &vMode,
				StorageClassName: &sc,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: size},
				},
			},
		}
		virtCli, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		_, err = virtCli.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		_, err = virtCli.CoreV1().PersistentVolumeClaims(ns).Create(context.Background(), pvc, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}

	createVMIwithFaultyPVC := func(ns string) {
		vmi = tests.NewRandomVMIWithNS(ns)
		bus := "virtio"
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: diskName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: bus,
				},
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: diskName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ioerrorPVC,
				},
			},
		})

		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
		_, err := virtClient.VirtualMachineInstance(ns).Create(vmi)
		Expect(err).To(BeNil(), "Create VMI successfully")
	}

	removeVMI := func(ns string) {
		err := virtClient.VirtualMachineInstance(ns).Delete(vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
		Expect(err).To(BeNil(), "Delete VMI successfully")
	}

	removePVwithFaultyDisk := func() {
		err := virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), ioerrorPV, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	isExpectedIOEvent := func(e corev1.Event) bool {
		if e.Type == "Warning" &&
			e.Reason == "IOerror" &&
			e.Message == "VM Paused due to IO error at the volume: "+diskName &&
			e.InvolvedObject.Kind == "VirtualMachineInstance" &&
			e.InvolvedObject.Name == vmi.ObjectMeta.Name {
			return true
		}
		return false
	}

	BeforeEach(func() {
		ns = tests.NamespaceTestDefault
		createFaultyDisk()
		createPVCwithFaultyDisk(ns)
	})
	AfterEach(func() {
		// Try a couple of times to remove the disk in case the device is busy
		Eventually(func() error {
			return removeFaultyDisk()
		}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())
		removePVwithFaultyDisk()
	})
	It("Should catch the IO error event", func() {
		createVMIwithFaultyPVC(ns)
		tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi, 120)
		Eventually(func() bool {
			events, err := virtClient.CoreV1().Events(ns).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, e := range events.Items {
				if isExpectedIOEvent(e) {
					return true
				}
			}

			return false
		}, 30*time.Second, 5*time.Second).Should(BeTrue())
		removeVMI(ns)
		tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
	})
})
