/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package storage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const nfsPort = 2049

var _ = Describe(SIG("NFS failover", func() {
	It("VM with NFS-backed storage should survive temporary NFS unavailability", func() {
		virtClient := kubevirt.Client
		namespace := testsuite.GetTestNamespace(nil)
		privilegedNs := testsuite.NamespacePrivileged
		suffix := rand.String(5)
		nfsServerName := "nfs-server-" + suffix
		nfsServiceName := "nfs-svc-" + suffix
		pvName := "nfs-pv-" + suffix
		pvcName := "nfs-pvc-" + suffix

		By("Creating an NFS server pod")
		nfsPod := renderNFSServerPod(nfsServerName, privilegedNs)
		nfsPod, err := virtClient().CoreV1().Pods(privilegedNs).Create(context.Background(), nfsPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisPod(nfsPod)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))
		nfsPod, err = matcher.ThisPod(nfsPod)()
		Expect(err).ToNot(HaveOccurred())

		By("Creating a Service for the NFS server")
		nfsSvc := renderNFSService(nfsServiceName, nfsServerName, privilegedNs)
		nfsSvc, err = virtClient().CoreV1().Services(privilegedNs).Create(context.Background(), nfsSvc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(nfsSvc.Spec.ClusterIP).ToNot(BeEmpty(), "NFS service should have a ClusterIP")

		By("Creating a PV and PVC backed by the NFS server")
		pv, pvc := renderNFSPVandPVC(pvName, pvcName, namespace, nfsSvc.Spec.ClusterIP)
		pv, err = virtClient().CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err := virtClient().CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
			if err != nil {
				Expect(err).To(MatchError(ContainSubstring("not found")))
			}
		})

		pvc, err = virtClient().CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() k8sv1.PersistentVolumeClaimPhase {
			pvc, err := virtClient().CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pvc.Status.Phase
		}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(Equal(k8sv1.ClaimBound))

		By("Starting a VM with the NFS-backed PVC")
		vmi := libvmifact.NewAlpine(libvmi.WithPersistentVolumeClaim("nfs-disk", pvc.Name))
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = virtClient().VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		vmi, err = virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		originalUID := vmi.UID

		By("Blocking NFS traffic by dropping packets on the NFS server")
		_, err = exec.ExecuteCommandOnPod(nfsPod, nfsPod.Spec.Containers[0].Name,
			[]string{"/usr/sbin/iptables-nft", "-A", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", nfsPort), "-j", "DROP"})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VM remains running and is not restarted during NFS unavailability")
		Consistently(func(g Gomega) {
			currentVMI, err := virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(currentVMI.Status.Phase).To(Equal(v1.Running))
			g.Expect(currentVMI.UID).To(Equal(originalUID), "VMI UID changed - VM was restarted")
		}).WithTimeout(3 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

		By("Unblocking NFS traffic")
		_, err = exec.ExecuteCommandOnPod(nfsPod, nfsPod.Spec.Containers[0].Name,
			[]string{"/usr/sbin/iptables-nft", "-D", "INPUT", "-p", "tcp", "--dport", fmt.Sprintf("%d", nfsPort), "-j", "DROP"})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VM is still running after NFS recovery")
		currentVMI, err := virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(currentVMI.UID).To(Equal(originalUID), "VMI UID changed - VM was restarted")
	})

	It("VMI should survive virt-handler restart when launcher socket is unreachable", Serial, func() {
		virtClient := kubevirt.Client
		namespace := testsuite.GetTestNamespace(nil)

		By("Starting an Alpine VMI")
		vmi := libvmifact.NewAlpine()
		vmi, err := virtClient().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		vmi, err = virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		nodeName := vmi.Status.NodeName

		By("Finding the launcher pod and its socket path")
		launcherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, namespace)
		Expect(err).ToNot(HaveOccurred())
		socketPath := fmt.Sprintf("/pods/%s/volumes/kubernetes.io~empty-dir/sockets/launcher-sock", launcherPod.UID)

		By("Replacing the launcher socket with a regular file to simulate unreachable launcher")
		virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
		Expect(err).ToNot(HaveOccurred())
		_, err = exec.ExecuteCommandOnPod(virtHandlerPod, "virt-handler", []string{"mv", socketPath, socketPath + ".bak"})
		Expect(err).ToNot(HaveOccurred())
		_, err = exec.ExecuteCommandOnPod(virtHandlerPod, "virt-handler", []string{"touch", socketPath})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			vh, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
			if err != nil {
				return
			}
			exec.ExecuteCommandOnPod(vh, "virt-handler", []string{"sh", "-c", fmt.Sprintf("test -f %s.bak && rm -f %s && mv %s.bak %s; true", socketPath, socketPath, socketPath, socketPath)})
		})

		By("Deleting the virt-handler pod to trigger informer restart and listAllKnownDomains")
		err = virtClient().CoreV1().Pods(virtHandlerPod.Namespace).Delete(context.Background(), virtHandlerPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the new virt-handler pod to be ready")
		Eventually(func() (*k8sv1.Pod, error) {
			return libnode.GetVirtHandlerPod(virtClient(), nodeName)
		}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

		By("Verifying the VMI is still running - Unknown domain status prevented spurious deletion")
		currentVMI, err := virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(currentVMI.Status.Phase).To(Equal(v1.Running), "VMI should still be Running")

		By("Restoring the original launcher socket")
		Eventually(func() error {
			vh, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
			if err != nil {
				return err
			}
			_, err = exec.ExecuteCommandOnPod(vh, "virt-handler", []string{"sh", "-c", fmt.Sprintf("rm -f %s && mv %s.bak %s", socketPath, socketPath, socketPath)})
			return err
		}).WithTimeout(60 * time.Second).WithPolling(5 * time.Second).Should(Succeed())

		By("Restarting virt-handler to re-discover the restored launcher socket")
		virtHandlerPod, err = libnode.GetVirtHandlerPod(virtClient(), nodeName)
		Expect(err).ToNot(HaveOccurred())
		err = virtClient().CoreV1().Pods(virtHandlerPod.Namespace).Delete(context.Background(), virtHandlerPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() (*k8sv1.Pod, error) {
			return libnode.GetVirtHandlerPod(virtClient(), nodeName)
		}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

		By("Pausing the VMI to prove virt-handler resumes active domain processing")
		err = virtClient().VirtualMachineInstance(namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
	})
}))

func renderNFSServerPod(name, namespace string) *k8sv1.Pod {
	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:  "nfs-server",
					Image: libregistry.GetUtilityImageFromRegistry("test-nfs-server"),
					Ports: []k8sv1.ContainerPort{
						{ContainerPort: nfsPort, Protocol: k8sv1.ProtocolTCP},
					},
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: pointer.P(true),
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:      "nfs-export",
							MountPath: "/exports",
						},
					},
					ReadinessProbe: &k8sv1.Probe{
						ProbeHandler: k8sv1.ProbeHandler{
							TCPSocket: &k8sv1.TCPSocketAction{
								Port: intstr.FromInt32(nfsPort),
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       3,
					},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: "nfs-export",
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func renderNFSService(name, podName, namespace string) *k8sv1.Service {
	return &k8sv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				"app": podName,
			},
			Ports: []k8sv1.ServicePort{
				{
					Port:       nfsPort,
					TargetPort: intstr.FromInt32(nfsPort),
					Protocol:   k8sv1.ProtocolTCP,
				},
			},
		},
	}
}

func renderNFSPVandPVC(pvName, pvcName, namespace, nfsServer string) (*k8sv1.PersistentVolume, *k8sv1.PersistentVolumeClaim) {
	storageSize := resource.MustParse("1Gi")
	volumeMode := k8sv1.PersistentVolumeFilesystem

	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: k8sv1.PersistentVolumeSpec{
			Capacity:         k8sv1.ResourceList{k8sv1.ResourceStorage: storageSize},
			VolumeMode:       &volumeMode,
			StorageClassName: "",
			AccessModes: []k8sv1.PersistentVolumeAccessMode{
				k8sv1.ReadWriteOnce,
			},
			MountOptions: []string{"nfsvers=4"},
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				NFS: &k8sv1.NFSVolumeSource{
					Server: nfsServer,
					Path:   "/",
				},
			},
			ClaimRef: &k8sv1.ObjectReference{
				Namespace: namespace,
				Name:      pvcName,
			},
		},
	}

	pvc := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			VolumeMode:       &volumeMode,
			StorageClassName: pointer.P(""),
			AccessModes: []k8sv1.PersistentVolumeAccessMode{
				k8sv1.ReadWriteOnce,
			},
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{k8sv1.ResourceStorage: storageSize},
			},
			VolumeName: pvName,
		},
	}

	return pv, pvc
}
