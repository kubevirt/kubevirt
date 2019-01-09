package framework

import (
	"strings"

	"github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/containerized-data-importer/tests/utils"
)

// CreatePVCFromDefinition is a wrapper around utils.CreatePVCFromDefinition
func (f *Framework) CreatePVCFromDefinition(def *k8sv1.PersistentVolumeClaim) (*k8sv1.PersistentVolumeClaim, error) {
	return utils.CreatePVCFromDefinition(f.K8sClient, f.Namespace.Name, def)
}

// DeletePVC is a wrapper around utils.DeletePVC
func (f *Framework) DeletePVC(pvc *k8sv1.PersistentVolumeClaim) error {
	return utils.DeletePVC(f.K8sClient, f.Namespace.Name, pvc)
}

// WaitForPersistentVolumeClaimPhase is a wrapper around utils.WaitForPersistentVolumeClaimPhase
func (f *Framework) WaitForPersistentVolumeClaimPhase(phase k8sv1.PersistentVolumeClaimPhase, pvcName string) error {
	return utils.WaitForPersistentVolumeClaimPhase(f.K8sClient, f.Namespace.Name, phase, pvcName)
}

// CreateExecutorPodWithPVC is a wrapper around utils.CreateExecutorPodWithPVC
func (f *Framework) CreateExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error) {
	return utils.CreateExecutorPodWithPVC(f.K8sClient, podName, f.Namespace.Name, pvc)
}

// FindPVC is a wrapper around utils.FindPVC
func (f *Framework) FindPVC(pvcName string) (*k8sv1.PersistentVolumeClaim, error) {
	return utils.FindPVC(f.K8sClient, f.Namespace.Name, pvcName)
}

// VerifyPVCIsEmpty verifies a passed in PVC is empty, returns true if the PVC is empty, false if it is not.
func VerifyPVCIsEmpty(f *Framework, pvc *k8sv1.PersistentVolumeClaim) bool {
	executorPod, err := f.CreateExecutorPodWithPVC("verify-pvc-empty", pvc)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = f.WaitTimeoutForPodReady(executorPod.Name, utils.PodWaitForTime)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, f.Namespace.Name, "ls -1 /pvc | wc -l")
	return strings.Compare("0", output) == 0
}

// CreateAndPopulateSourcePVC Creates and populates a PVC using the provided POD and command
func (f *Framework) CreateAndPopulateSourcePVC(pvcName string, podName string, fillCommand string) *k8sv1.PersistentVolumeClaim {
	// Create the source PVC and populate it with a file, so we can verify the clone.
	sourcePvc, err := f.CreatePVCFromDefinition(utils.NewPVCDefinition(pvcName, "1G", nil, nil))

	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	// why not just call utils.CreatePod() we're not using framework to help with tests so not sure why it's even there
	// it just wraps the calls in utils, doesn't seem to do much more
	pod, err := f.CreatePod(utils.NewPodWithPVC(podName, fillCommand, sourcePvc))
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = f.WaitTimeoutForPodStatus(pod.Name, k8sv1.PodSucceeded, utils.PodWaitForTime)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return sourcePvc
}

// VerifyTargetPVCContent is used to check the contents of a PVC and ensure it matches the provided expected data
func (f *Framework) VerifyTargetPVCContent(namespace *k8sv1.Namespace, pvc *k8sv1.PersistentVolumeClaim, fileName string, expectedData string) bool {
	executorPod, err := utils.CreateExecutorPodWithPVC(f.K8sClient, "verify-pvc-content", namespace.Name, pvc)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = utils.WaitTimeoutForPodReady(f.K8sClient, executorPod.Name, namespace.Name, utils.PodWaitForTime)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, namespace.Name, "cat "+fileName)
	return strings.Compare(expectedData, output) == 0
}

// VerifyTargetPVCContentMD5 provides a function to check the md5 of data on a PVC and ensure it matches that which is provided
func (f *Framework) VerifyTargetPVCContentMD5(namespace *k8sv1.Namespace, pvc *k8sv1.PersistentVolumeClaim, fileName string, expectedHash string) bool {
	executorPod, err := utils.CreateExecutorPodWithPVC(f.K8sClient, "verify-pvc-md5", namespace.Name, pvc)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = utils.WaitTimeoutForPodReady(f.K8sClient, executorPod.Name, namespace.Name, utils.PodWaitForTime)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	output := f.ExecShellInPod(executorPod.Name, namespace.Name, "md5sum "+fileName)
	return strings.Compare(expectedHash, output[:32]) == 0
}
