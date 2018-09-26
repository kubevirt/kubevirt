package framework

import (
	"strings"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/containerized-data-importer/tests/utils"
)

func (f *Framework) CreatePVCFromDefinition(def *k8sv1.PersistentVolumeClaim) (*k8sv1.PersistentVolumeClaim, error) {
	return utils.CreatePVCFromDefinition(f.K8sClient, f.Namespace.Name, def)
}

func (f *Framework) DeletePVC(pvc *k8sv1.PersistentVolumeClaim) error {
	return utils.DeletePVC(f.K8sClient, f.Namespace.Name, pvc)
}

func (f *Framework) WaitForPersistentVolumeClaimPhase(phase k8sv1.PersistentVolumeClaimPhase, pvcName string) error {
	return utils.WaitForPersistentVolumeClaimPhase(f.K8sClient, f.Namespace.Name, phase, pvcName)
}

func (f *Framework) CreateExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error) {
	return utils.CreateExecutorPodWithPVC(f.K8sClient, podName, f.Namespace.Name, pvc)
}

func (f *Framework) FindPVC(pvcName string) (*k8sv1.PersistentVolumeClaim, error) {
	return utils.FindPVC(f.K8sClient, f.Namespace.Name, pvcName)
}

// Verify passed in PVC is empty, returns true if the PVC is empty, false if it is not.
func VerifyPVCIsEmpty(f *Framework, pvc *k8sv1.PersistentVolumeClaim) bool {
	executorPod, err := f.CreateExecutorPodWithPVC("verify-pvc-empty", pvc)
	Expect(err).ToNot(HaveOccurred())
	err = f.WaitTimeoutForPodReady(executorPod.Name, utils.PodWaitForTime)
	Expect(err).ToNot(HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, f.Namespace.Name, "ls -1 /pvc | wc -l")
	f.DeletePod(executorPod)
	return strings.Compare("0", output) == 0
}

func (f *Framework) CreateAndPopulateSourcePVC(pvcName string, podName string, fillCommand string) *k8sv1.PersistentVolumeClaim {
	// Create the source PVC and populate it with a file, so we can verify the clone.
	sourcePvc, err := f.CreatePVCFromDefinition(utils.NewPVCDefinition(pvcName, "1G", nil, nil))

	Expect(err).ToNot(HaveOccurred())
	pod, err := f.CreatePod(utils.NewPodWithPVC(podName, fillCommand, sourcePvc))
	Expect(err).ToNot(HaveOccurred())
	err = f.WaitTimeoutForPodStatus(pod.Name, k8sv1.PodSucceeded, utils.PodWaitForTime)
	Expect(err).ToNot(HaveOccurred())
	return sourcePvc
}

func (f *Framework) VerifyTargetPVCContent(namespace *k8sv1.Namespace, pvc *k8sv1.PersistentVolumeClaim, fileName string, expectedData string) bool {
	executorPod, err := utils.CreateExecutorPodWithPVC(f.K8sClient, "verify-pvc-content", namespace.Name, pvc)
	Expect(err).ToNot(HaveOccurred())
	err = utils.WaitTimeoutForPodReady(f.K8sClient, executorPod.Name, namespace.Name, utils.PodWaitForTime)
	Expect(err).ToNot(HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, namespace.Name, "cat "+fileName)
	f.DeletePod(executorPod)
	return strings.Compare(expectedData, output) == 0
}

func (f *Framework) VerifyTargetPVCContentMD5(namespace *k8sv1.Namespace, pvc *k8sv1.PersistentVolumeClaim, fileName string, expectedHash string) bool {
	executorPod, err := utils.CreateExecutorPodWithPVC(f.K8sClient, "verify-pvc-content", namespace.Name, pvc)
	Expect(err).ToNot(HaveOccurred())
	err = utils.WaitTimeoutForPodReady(f.K8sClient, executorPod.Name, namespace.Name, utils.PodWaitForTime)
	Expect(err).ToNot(HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, namespace.Name, "md5sum "+fileName)
	f.DeletePod(executorPod)
	return strings.Compare(expectedHash, output[:32]) == 0
}
