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

package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	perfPVCStorageClassName = "kubevirt-perf-local"
	// The claim requests 1Gi. Each loop file is 1Gi (GNU truncate treats G as 1024^3)
	// and is formatted with reserved blocks disabled, so metadata overhead is inside
	// the default 10% lessPVCSpaceToleration and diskimg.Creator shrinks disk.img.
	perfPVCSize       = "1Gi"
	perfLoopDiskBytes = "1G"
	perfLoopBase      = "/var/provision/kubevirt.io/perf-pvc"
	// A 90Mi guest with graphics and serial console disabled requests about
	// 423Mi at the pod. 30 VMs plus a primer fit on one 16Gi worker. The
	// batch is pinned there so one virt-handler runs configureDiskImages for
	// every VM.
	perfPVCVMCount        = 30
	perfDensityLabelKey   = "test.kubevirt.io/density"
	perfDensityLabelValue = "pvc"
	perfPVCBindTimeout    = 3 * time.Minute
	perfPVCCleanupTimeout = 3 * time.Minute
)

const perfLoopSetupScript = `set -euo pipefail
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
base=%q
last=%d
size=%q
modprobe loop >/dev/null 2>&1 || true
mkdir -p "$base"
for i in $(seq 0 "$last"); do
  img="$base/disk-${i}.img"
  mnt="$base/mnt-${i}"
  mkdir -p "$mnt"
  if ! losetup -j "$img" | grep -q .; then
    truncate -s "$size" "$img"
    dev=$(losetup --find --show "$img")
  else
    dev=$(losetup -j "$img" | cut -d: -f1 | head -n1)
  fi
  if ! mountpoint -q "$mnt"; then
    mkfs.ext4 -F -m 0 -E lazy_itable_init=1,lazy_journal_init=1 -q "$dev"
    mount -o noatime "$dev" "$mnt"
  fi
  chmod 777 "$mnt"
  if ! chcon -R unconfined_u:object_r:svirt_sandbox_file_t:s0 "$mnt"; then
    if command -v selinuxenabled >/dev/null 2>&1 && selinuxenabled; then
      echo "chcon failed while SELinux is enabled: $mnt" >&2
      exit 1
    fi
  fi
done
`

const perfLoopCleanupScript = `set -euo pipefail
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
base=%q
if [ ! -d "$base" ]; then
  exit 0
fi
for mnt in "$base"/mnt-*; do
  [ -d "$mnt" ] || continue
  if mountpoint -q "$mnt"; then
    umount "$mnt" || umount -l "$mnt"
  fi
done
for img in "$base"/disk-*.img; do
  [ -e "$img" ] || continue
  # losetup -j exits non-zero when the file has no loop device. pipefail would
  # abort the script before the image is removed.
  dev=$(losetup -j "$img" | cut -d: -f1 || true)
  if [ -n "$dev" ]; then
    losetup -d "$dev"
  fi
  rm -f "$img"
done
for mnt in "$base"/mnt-*; do
  [ -d "$mnt" ] || continue
  if mountpoint -q "$mnt"; then
    echo "still mounted: $mnt" >&2
    exit 1
  fi
done
if losetup -a | grep -q "$base"; then
  echo "loop device still attached under $base" >&2
  losetup -a >&2
  exit 1
fi
rm -rf "$base"
`

// Measures VMI creation-to-Running for VMs whose root disk is an already-bound
// filesystem PVC. Each claim is an ext4 filesystem on its own loop device, mounted
// on one node before the audit window. The VMs do not boot a guest: phase Running
// is the QEMU domain, and virt-handler creates disk.img on the empty filesystem.
// No DataVolume is created.
var _ = Describe(SIG("Control Plane Performance Density Testing with PVC", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		skipIfNoPerformanceTests()
		virtClient = kubevirt.Client()
	})

	Describe("Density test", func() {
		vmCount := perfPVCVMCount
		vmBatchStartupLimit := 5 * time.Minute

		Context(fmt.Sprintf("[small] create a batch of %d running VMs with a pre-bound filesystem PVC on one node", vmCount), func() {
			It("should successfully create all VMs", func() {
				artifactsDir := os.Getenv("ARTIFACTS")
				nodeName, hostname := pickPerfDensityNode(virtClient)
				DeferCleanup(func() {
					cleanupPerfLoopVolumes(virtClient, nodeName)
				})

				By(fmt.Sprintf("Creating %d loop filesystems on node %s", vmCount+1, nodeName))
				// One extra claim is the primer. Its mount finishes before the audit window.
				mounts := createPerfLoopFilesystems(nodeName, vmCount+1)
				By("Creating filesystem PVCs and waiting until they are Bound")
				pvcNames := createBoundLoopPVCs(virtClient, hostname, mounts)

				By("Creating a primer VM")
				createVMWithPVC(virtClient, pvcNames[0])
				waitRunningLabeledVMI(virtClient, 1, primerVMITimeout)
				time.Sleep(2 * PrometheusScrapeInterval)

				startTime := time.Now()
				By("Creating a batch of VMs")
				createBatchRunningVMWithPVC(virtClient, pvcNames[1:])

				By("Waiting for the batch of VMIs to be Running")
				waitRunningLabeledVMI(virtClient, vmCount+1, vmBatchStartupLimit)
				collectMetrics(startTime, filepath.Join(artifactsDir, "VM-pvc-perf-audit-results.json"))
			})
		})
	})
}))

func pickPerfDensityNode(virtClient kubecli.KubevirtClient) (name, hostname string) {
	nodes := libnode.GetAllSchedulableNodes(virtClient)
	Expect(nodes.Items).NotTo(BeEmpty())

	controlPlane := map[string]struct{}{}
	for _, node := range libnode.GetControlPlaneNodes(virtClient).Items {
		controlPlane[node.Name] = struct{}{}
	}

	candidates := make([]k8sv1.Node, 0, len(nodes.Items))
	for i := range nodes.Items {
		if _, ok := controlPlane[nodes.Items[i].Name]; ok {
			continue
		}
		candidates = append(candidates, nodes.Items[i])
	}
	if len(candidates) == 0 {
		candidates = nodes.Items
	}

	best := 0
	var bestFree int64 = -1
	for i := range candidates {
		free := candidates[i].Status.Allocatable.Memory().Value() - nodeRequestedMemory(virtClient, candidates[i].Name)
		if bestFree < 0 || free > bestFree {
			best = i
			bestFree = free
		}
	}
	node := &candidates[best]
	hostname = node.Labels[k8sv1.LabelHostname]
	if hostname == "" {
		hostname = node.Name
	}
	return node.Name, hostname
}

func nodeRequestedMemory(virtClient kubecli.KubevirtClient, nodeName string) int64 {
	pods, err := virtClient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	Expect(err).ToNot(HaveOccurred())
	var total int64
	for i := range pods.Items {
		for _, c := range append(pods.Items[i].Spec.InitContainers, pods.Items[i].Spec.Containers...) {
			if req := c.Resources.Requests.Memory(); req != nil {
				total += req.Value()
			}
		}
	}
	return total
}

func createPerfLoopFilesystems(nodeName string, count int) []string {
	Expect(count).To(BeNumerically(">", 0))
	script := fmt.Sprintf(perfLoopSetupScript, perfLoopBase, count-1, perfLoopDiskBytes)
	_, err := execOnPerfNode(nodeName, script)
	Expect(err).NotTo(HaveOccurred())

	mounts := make([]string, count)
	for i := 0; i < count; i++ {
		mounts[i] = fmt.Sprintf("%s/mnt-%d", perfLoopBase, i)
	}
	return mounts
}

func execOnPerfNode(nodeName, script string) (string, error) {
	return libnode.ExecuteCommandInVirtHandlerPod(nodeName, []string{
		"/usr/bin/virt-chroot", "--mount", "/proc/1/ns/mnt", "exec", "--",
		"/bin/bash", "-c", script,
	})
}

func createBoundLoopPVCs(virtClient kubecli.KubevirtClient, hostname string, mounts []string) []string {
	namespace := testsuite.NamespaceTestDefault
	ensurePerfLoopStorageClass(virtClient, namespace)

	names := make([]string, 0, len(mounts))
	for i, mountPath := range mounts {
		name := fmt.Sprintf("perf-pvc-%d", i)
		createPerfLoopPV(virtClient, name, namespace, hostname, mountPath)
		createPerfPVC(virtClient, name, namespace)
		names = append(names, name)
	}

	Eventually(func() int {
		pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: perfDensityLabelSelector(),
		})
		Expect(err).ToNot(HaveOccurred())
		bound := 0
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == k8sv1.ClaimBound {
				bound++
			}
		}
		return bound
	}, perfPVCBindTimeout, time.Second).Should(Equal(len(mounts)), "timed out waiting for performance PVCs to bind")

	return names
}

func ensurePerfLoopStorageClass(virtClient kubecli.KubevirtClient, namespace string) {
	bindingMode := storagev1.VolumeBindingImmediate
	existing, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), perfPVCStorageClassName, metav1.GetOptions{})
	if err == nil {
		Expect(existing.VolumeBindingMode).ToNot(BeNil())
		Expect(*existing.VolumeBindingMode).To(Equal(bindingMode))
		return
	}
	Expect(err).To(MatchError(k8serrors.IsNotFound, "errors.IsNotFound"))

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: perfPVCStorageClassName,
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(namespace): "",
			},
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		VolumeBindingMode: &bindingMode,
	}
	_, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func createPerfLoopPV(virtClient kubecli.KubevirtClient, name, namespace, hostname, mountPath string) {
	volumeMode := k8sv1.PersistentVolumeFilesystem
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: perfPVName(name, namespace),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(namespace): "",
				perfDensityLabelKey:                      perfDensityLabelValue,
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				k8sv1.ResourceStorage: resource.MustParse(perfPVCSize),
			},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimDelete,
			StorageClassName:              perfPVCStorageClassName,
			VolumeMode:                    &volumeMode,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				Local: &k8sv1.LocalVolumeSource{
					Path: mountPath,
				},
			},
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{{
							Key:      k8sv1.LabelHostname,
							Operator: k8sv1.NodeSelectorOpIn,
							Values:   []string{hostname},
						}},
					}},
				},
			},
		},
	}

	_, err := virtClient.CoreV1().PersistentVolumes().Create(context.Background(), pv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func createPerfPVC(virtClient kubecli.KubevirtClient, name, namespace string) {
	volumeMode := k8sv1.PersistentVolumeFilesystem
	storageClass := perfPVCStorageClassName
	pvc := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				perfDensityLabelKey: perfDensityLabelValue,
			},
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: resource.MustParse(perfPVCSize),
				},
			},
			StorageClassName: &storageClass,
			VolumeMode:       &volumeMode,
			VolumeName:       perfPVName(name, namespace),
		},
	}

	_, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func perfPVName(name, namespace string) string {
	return fmt.Sprintf("%s-%s", name, namespace)
}

func cleanupPerfLoopVolumes(virtClient kubecli.KubevirtClient, nodeName string) {
	By("Deleting performance VMs and detaching loop filesystems")
	namespace := testsuite.NamespaceTestDefault
	grace := int64(0)
	propagation := metav1.DeletePropagationBackground
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
		PropagationPolicy:  &propagation,
	}
	listOptions := metav1.ListOptions{LabelSelector: perfDensityLabelSelector()}

	Expect(virtClient.VirtualMachine(namespace).DeleteCollection(context.Background(), deleteOptions, listOptions)).To(Succeed())
	Eventually(func() int {
		pods, err := virtClient.CoreV1().Pods(namespace).List(context.Background(), listOptions)
		Expect(err).ToNot(HaveOccurred())
		return len(pods.Items)
	}, perfPVCCleanupTimeout, 2*time.Second).Should(Equal(0))

	Expect(virtClient.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(context.Background(), deleteOptions, listOptions)).To(Succeed())
	Eventually(func() int {
		pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), listOptions)
		Expect(err).ToNot(HaveOccurred())
		return len(pvcs.Items)
	}, perfPVCCleanupTimeout, 2*time.Second).Should(Equal(0))

	Expect(virtClient.CoreV1().PersistentVolumes().DeleteCollection(context.Background(), metav1.DeleteOptions{}, listOptions)).To(Succeed())

	script := fmt.Sprintf(perfLoopCleanupScript, perfLoopBase)
	Eventually(func() error {
		_, err := execOnPerfNode(nodeName, script)
		return err
	}, perfPVCCleanupTimeout, 5*time.Second).Should(Succeed())
}

func perfDensityLabelSelector() string {
	return fmt.Sprintf("%s=%s", perfDensityLabelKey, perfDensityLabelValue)
}

func createBatchRunningVMWithPVC(virtClient kubecli.KubevirtClient, pvcNames []string) {
	for _, pvcName := range pvcNames {
		createVMWithPVC(virtClient, pvcName)
		time.Sleep(100 * time.Millisecond)
	}
}

func createVMWithPVC(virtClient kubecli.KubevirtClient, pvcName string) {
	vm := newVMWithPVC(pvcName)
	By(fmt.Sprintf("Creating VM %s", vm.Name))
	_, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func newVMWithPVC(pvcName string) *kvv1.VirtualMachine {
	const (
		cpuLimit = "100m"
		memLimit = "90Mi"
	)
	vmi := libvmi.New(
		libvmi.WithPersistentVolumeClaim("disk0", pvcName),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(kvv1.DefaultPodNetwork()),
		libvmi.WithMemoryRequest(memLimit),
		libvmi.WithMemoryLimit(memLimit),
		libvmi.WithCPURequest(cpuLimit),
		libvmi.WithCPULimit(cpuLimit),
		libvmi.WithAutoattachGraphicsDevice(false),
		libvmi.WithAutoattachSerialConsole(false),
		libvmi.WithLabel(perfDensityLabelKey, perfDensityLabelValue),
	)
	return libvmi.NewVirtualMachine(vmi,
		libvmi.WithRunStrategy(kvv1.RunStrategyAlways),
		libvmi.WithLabels(map[string]string{
			perfDensityLabelKey: perfDensityLabelValue,
		}),
	)
}

func waitRunningLabeledVMI(virtClient kubecli.KubevirtClient, vmiCount int, timeout time.Duration) {
	Eventually(func() int {
		vmis, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{
			LabelSelector: perfDensityLabelSelector(),
		})
		Expect(err).ToNot(HaveOccurred())
		running := 0
		for _, vmi := range vmis.Items {
			if vmi.Status.Phase == kvv1.Running {
				running++
			}
		}
		return running
	}, timeout, 10*time.Second).Should(Equal(vmiCount))
}
