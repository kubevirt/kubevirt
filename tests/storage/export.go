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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	routev1 "github.com/openshift/api/route/v1"
	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	caBundleKey          = "ca-bundle"
	caCertPath           = "/cacerts"
	dataPath             = "/data"
	diskImage            = "disk.img"
	blockVolumeMountPath = "/dev/volume"

	// annContentType is an annotation on a PVC indicating the content type. This is populated by CDI.
	annContentType = "cdi.kubevirt.io/storage.contentType"

	kubevirtcontentUrlTemplate   = "%s?x-kubevirt-export-token=%s"
	archiveDircontentUrlTemplate = "%s/disk.img?x-kubevirt-export-token=%s"

	certificates = "certificates"

	pvcNotFoundReason = "PVCNotFound"
	podReadyReason    = "PodReady"
	inUseReason       = "InUse"

	proxyUrlBase = "https://virt-exportproxy.%s.svc/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s%s"

	tlsKey           = "tls.key"
	tlsCert          = "tls.crt"
	testKey          = "test"
	testHostName     = "vmexport-proxy.test.net"
	subjectAltNameId = "2.5.29.17"

	exportPrefix = "virt-export"
)

var (
	podReadyCondition = MatchConditionIgnoreTimeStamp(exportv1.Condition{
		Type:   exportv1.ConditionReady,
		Status: k8sv1.ConditionTrue,
		Reason: podReadyReason,
	})
)

var _ = SIGDescribe("Export", func() {
	var err error
	var token *k8sv1.Secret
	var virtClient kubecli.KubevirtClient
	var qemuGid = int64(107)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		testsuite.WaitExportProxyReady()
	})

	AfterEach(func() {
		if token != nil {
			err := virtClient.CoreV1().Secrets(token.Namespace).Delete(context.Background(), token.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			token = nil
		}
	})

	addBlockVolume := func(pod *k8sv1.Pod, volumeName string) *k8sv1.Pod {
		pod.Spec.Containers[0].VolumeDevices = append(pod.Spec.Containers[0].VolumeDevices, k8sv1.VolumeDevice{
			Name:       volumeName,
			DevicePath: "/dev/volume",
		})
		return pod
	}

	addFilesystemVolume := func(pod *k8sv1.Pod, volumeName string) *k8sv1.Pod {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
			Name:      volumeName,
			MountPath: "/data",
		})
		return pod
	}

	addCertVolume := func(pod *k8sv1.Pod) *k8sv1.Pod {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
			Name:      "cacerts",
			ReadOnly:  true,
			MountPath: "/cacerts",
		})
		return pod
	}

	createDownloadPod := func(caConfigMap *k8sv1.ConfigMap) *k8sv1.Pod {
		podName := "download-pod"
		pod := tests.RenderPod(podName, []string{"/bin/sh", "-c", "sleep 360"}, []string{})
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.FSGroup = &qemuGid
		pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
			Name: "cacerts",
			VolumeSource: k8sv1.VolumeSource{
				ConfigMap: &k8sv1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{
						Name: caConfigMap.Name,
					},
				},
			},
		})
		addCertVolume(pod)
		return pod
	}

	createDownloadPodForPvc := func(pvc *k8sv1.PersistentVolumeClaim, caConfigMap *k8sv1.ConfigMap) *k8sv1.Pod {
		volumeName := pvc.GetName()
		pod := createDownloadPod(caConfigMap)
		pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
			Name: volumeName,
			VolumeSource: k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.GetName(),
				},
			},
		})

		volumeMode := pvc.Spec.VolumeMode
		if volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock {
			addBlockVolume(pod, volumeName)
		} else {
			addFilesystemVolume(pod, volumeName)
		}
		return tests.RunPod(pod)
	}

	createSourcePodChecker := func(pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
		volumeName := pvc.GetName()
		podName := "download-pod"
		pod := tests.RenderPod(podName, []string{"/bin/sh", "-c", "sleep 360"}, []string{})
		pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
			Name: volumeName,
			VolumeSource: k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.GetName(),
				},
			},
		})
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.FSGroup = &qemuGid

		volumeMode := pvc.Spec.VolumeMode
		if volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock {
			addBlockVolume(pod, volumeName)
		} else {
			addFilesystemVolume(pod, volumeName)
		}
		return tests.RunPod(pod)
	}

	createExportTokenSecret := func(name, namespace string) *k8sv1.Secret {
		var err error
		secret := &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      fmt.Sprintf("export-token-%s", name),
			},
			StringData: map[string]string{
				"token": name,
			},
		}
		token, err = virtClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return token
	}

	createCaConfigMap := func(name, namespace, data string) *k8sv1.ConfigMap {
		dst := &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string]string{
				caBundleKey: data,
			},
		}

		err = virtClient.CoreV1().ConfigMaps(dst.Namespace).Delete(context.TODO(), dst.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		cm, err := virtClient.CoreV1().ConfigMaps(dst.Namespace).Create(context.TODO(), dst, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		return cm
	}

	createCaConfigMapInternal := func(name, namespace string, export *exportv1.VirtualMachineExport) *k8sv1.ConfigMap {
		return createCaConfigMap(name, namespace, export.Status.Links.Internal.Cert)
	}

	createCaConfigMapProxy := func(name, namespace string, _ *exportv1.VirtualMachineExport) *k8sv1.ConfigMap {
		cm, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.TODO(), "kubevirt-ca", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return createCaConfigMap(name, namespace, cm.Data["ca-bundle"])
	}

	md5Command := func(fileName string) []string {
		return []string{
			"md5sum",
			fileName,
		}
	}

	populateKubeVirtContent := func(sc string, volumeMode k8sv1.PersistentVolumeMode) (*k8sv1.PersistentVolumeClaim, string) {
		By("Creating source volume")
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeMode(volumeMode)),
			libdv.WithForceBindAnnotation(),
		)

		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		var pvc *k8sv1.PersistentVolumeClaim
		Eventually(func() error {
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(dv)).Get(context.Background(), dv.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(BeNil(), "persistent volume associated with DV should be created")

		By("Making sure the DV is successful")
		libstorage.EventuallyDV(dv, 90, HaveSucceeded())

		pod := createSourcePodChecker(pvc)

		fileName := filepath.Join(dataPath, diskImage)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileName = blockVolumeMountPath
		}
		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, md5Command(fileName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(HaveLen(32))

		err = virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: pointer.Int64(0),
		})
		Expect(err).ToNot(HaveOccurred())
		return pvc, md5sum
	}

	populateArchiveContent := func(sc string, volumeMode k8sv1.PersistentVolumeMode) (*k8sv1.PersistentVolumeClaim, string) {
		pvc, md5sum := populateKubeVirtContent(sc, volumeMode)

		patchData, err := patch.GeneratePatchPayload(
			patch.PatchOperation{
				Op:    patch.PatchAddOp,
				Path:  "/metadata/annotations/" + patch.EscapeJSONPointer(annContentType),
				Value: "archive",
			},
			patch.PatchOperation{
				Op:    patch.PatchAddOp,
				Path:  "/metadata/ownerReferences",
				Value: []metav1.OwnerReference{},
			},
		)
		Expect(err).ToNot(HaveOccurred())
		pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.Background(), pvc.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		return pvc, md5sum
	}

	verifyKubeVirtRawContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(HaveLen(32))
		Expect(md5sum).To(Equal(expectedMD5))
	}

	verifyKubeVirtGzContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		command := []string{
			"/usr/bin/gzip",
			"-d",
			filepath.Join(dataPath, fileName),
		}
		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)

		fileName = strings.Replace(fileName, ".gz", "", 1)
		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(HaveLen(32))
		Expect(md5sum).To(Equal(expectedMD5))
	}

	verifyArchiveGzContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		extractedFileName := strings.ReplaceAll(fileName, ".tar.gz", ".img")
		command := []string{
			"/usr/bin/tar",
			"--strip-components",
			"1",
			"-xzvf",
			filepath.Join(dataPath, fileName),
			"-C",
			filepath.Join(dataPath),
			"./" + extractedFileName,
		}
		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)

		fileAndPathName := filepath.Join(dataPath, extractedFileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(HaveLen(32))
		Expect(md5sum).To(Equal(expectedMD5))
	}

	getExporterPod := func(vmExport *exportv1.VirtualMachineExport) *k8sv1.Pod {
		var pod *k8sv1.Pod
		var err error
		Eventually(func() error {
			pod, err = virtClient.CoreV1().Pods(vmExport.Namespace).Get(context.TODO(), fmt.Sprintf("virt-export-%s", vmExport.Name), metav1.GetOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil(), "unable to find pod %s", fmt.Sprintf("virt-export-%s", vmExport.Name))
		return pod
	}

	getExportService := func(vmExport *exportv1.VirtualMachineExport) *k8sv1.Service {
		var service *k8sv1.Service
		var err error
		Eventually(func() error {
			service, err = virtClient.CoreV1().Services(vmExport.Namespace).Get(context.TODO(), fmt.Sprintf("virt-export-%s", vmExport.Name), metav1.GetOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil(), "unable to find service %s", fmt.Sprintf("virt-export-%s", vmExport.Name))
		return service
	}

	urlGeneratorInternal := func(expectedFormat exportv1.ExportVolumeFormat, pvcName, template, token string, export *exportv1.VirtualMachineExport) (string, string) {
		downloadUrl := ""
		fileName := ""
		for _, volume := range export.Status.Links.Internal.Volumes {
			if volume.Name == pvcName {
				for _, format := range volume.Formats {
					if format.Format == expectedFormat {
						downloadUrl = fmt.Sprintf(template, format.Url, token)
						fileName = filepath.Base(format.Url)
					}
				}
			}
		}
		return downloadUrl, fileName
	}

	urlGeneratorProxy := func(expectedFormat exportv1.ExportVolumeFormat, pvcName, template, token string, export *exportv1.VirtualMachineExport) (string, string) {
		downloadUrl := ""
		fileName := ""
		for _, volume := range export.Status.Links.Internal.Volumes {
			if volume.Name == pvcName {
				for _, format := range volume.Formats {
					if format.Format == expectedFormat {
						i := strings.Index(format.Url, ".svc/")
						if i >= 0 {
							uri := fmt.Sprintf(template, format.Url[i+4:], token)
							downloadUrl = fmt.Sprintf(proxyUrlBase, flags.KubeVirtInstallNamespace, export.Namespace, export.Name, uri)
							fileName = filepath.Base(format.Url)
						}
					}
				}
			}
		}
		return downloadUrl, fileName
	}

	waitForReadyExport := func(export *exportv1.VirtualMachineExport) *exportv1.VirtualMachineExport {
		Eventually(func() []exportv1.Condition {
			export, err = virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status == nil {
				return nil
			}
			return export.Status.Conditions
		}, 180*time.Second, 1*time.Second).Should(ContainElement(podReadyCondition), "export %s/%s is expected to become ready %v", export.Namespace, export.Name, export)
		return export
	}

	checkExportSecretRef := func(vmExport *exportv1.VirtualMachineExport) {
		By("Making sure vmexport status contains the right secretRef")
		Expect(vmExport.Spec.TokenSecretRef).ToNot(BeNil())
		Expect(vmExport.Status.TokenSecretRef).ToNot(BeNil())
		Expect(*vmExport.Spec.TokenSecretRef).To(Equal(*vmExport.Status.TokenSecretRef))
		Expect(*vmExport.Status.TokenSecretRef).ToNot(BeEmpty())
	}

	verifyDefaultRequestLimits := func(export *exportv1.VirtualMachineExport) {
		By("Verifying the exporter pod has default request/limits")
		exporterPod := getExporterPod(export)
		Expect(exporterPod.Spec.Containers[0].Resources.Requests.Cpu()).ToNot(BeNil())
		Expect(exporterPod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(100)))
		Expect(exporterPod.Spec.Containers[0].Resources.Limits.Cpu()).ToNot(BeNil())
		Expect(exporterPod.Spec.Containers[0].Resources.Limits.Cpu().Value()).To(Equal(int64(1)))
		Expect(exporterPod.Spec.Containers[0].Resources.Requests.Memory()).ToNot(BeNil())
		Expect(exporterPod.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(int64(200 * 1024 * 1024)))
		Expect(exporterPod.Spec.Containers[0].Resources.Limits.Memory()).ToNot(BeNil())
		Expect(exporterPod.Spec.Containers[0].Resources.Limits.Memory().Value()).To(Equal(int64(1024 * 1024 * 1024)))
	}

	type populateFunction func(string, k8sv1.PersistentVolumeMode) (*k8sv1.PersistentVolumeClaim, string)
	type verifyFunction func(string, string, *k8sv1.Pod, k8sv1.PersistentVolumeMode)
	type storageClassFunction func() (string, bool)
	type caBundleGenerator func(string, string, *exportv1.VirtualMachineExport) *k8sv1.ConfigMap
	type urlGenerator func(exportv1.ExportVolumeFormat, string, string, string, *exportv1.VirtualMachineExport) (string, string)

	DescribeTable("should make a PVC export available", func(populateFunction populateFunction, verifyFunction verifyFunction,
		storageClassFunction storageClassFunction, caBundleGenerator caBundleGenerator, urlGenerator urlGenerator,
		expectedFormat exportv1.ExportVolumeFormat, urlTemplate string, volumeMode k8sv1.PersistentVolumeMode) {
		sc, exists := storageClassFunction()
		if !exists {
			Skip("Skip test when right storage is not present")
		}
		pvc, comparison := populateFunction(sc, volumeMode)
		By("Creating the export token, we can export volumes using this token")
		// For testing the token is the name of the source pvc.
		token := createExportTokenSecret(pvc.Name, pvc.Namespace)

		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: pvc.Namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: &token.Name,
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     pvc.Name,
				},
			},
		}
		By("Creating VMExport we can start exporting the volume")
		export, err := virtClient.VirtualMachineExport(pvc.Namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		export = waitForReadyExport(export)
		checkExportSecretRef(export)
		Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
		verifyDefaultRequestLimits(export)

		By("Creating download pod, so we can download image")
		targetPvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("target-pvc-%s", rand.String(12)),
				Namespace: pvc.Namespace,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				AccessModes:      pvc.Spec.AccessModes,
				StorageClassName: pvc.Spec.StorageClassName,
				Resources:        pvc.Spec.Resources,
				VolumeMode:       pvc.Spec.VolumeMode,
			},
		}
		By("Creating target PVC, so we can inspect if the export worked")
		targetPvc, err = virtClient.CoreV1().PersistentVolumeClaims(targetPvc.Namespace).Create(context.Background(), targetPvc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		caConfigMap := caBundleGenerator("export-cacerts", targetPvc.Namespace, export)

		downloadPod := createDownloadPodForPvc(targetPvc, caConfigMap)

		downloadUrl, fileName := urlGenerator(expectedFormat, pvc.Name, urlTemplate, pvc.Name, export)
		Expect(downloadUrl).ToNot(BeEmpty())
		Expect(fileName).ToNot(BeEmpty())

		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		command := []string{
			"curl",
			"-L",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			downloadUrl,
			"--output",
			fileAndPathName,
		}
		By(fmt.Sprintf("Downloading from URL: %s", downloadUrl))
		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)

		verifyFunction(fileName, comparison, downloadPod, volumeMode)
	},
		// "internal" tests
		Entry("with RAW kubevirt content type", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapInternal, urlGeneratorInternal, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW gzipped kubevirt content type", populateKubeVirtContent, verifyKubeVirtGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapInternal, urlGeneratorInternal, exportv1.KubeVirtGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive content type", populateArchiveContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapInternal, urlGeneratorInternal, exportv1.Dir, archiveDircontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive tarred gzipped content type", populateArchiveContent, verifyArchiveGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapInternal, urlGeneratorInternal, exportv1.ArchiveGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW kubevirt content type block", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOBlockStorageClass, createCaConfigMapInternal, urlGeneratorInternal, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeBlock),
		// "proxy" tests
		Entry("with RAW kubevirt content type PROXY", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW gzipped kubevirt content type PROXY", populateKubeVirtContent, verifyKubeVirtGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive content type PROXY", populateArchiveContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.Dir, archiveDircontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive tarred gzipped content type PROXY", populateArchiveContent, verifyArchiveGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.ArchiveGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW kubevirt content type block PROXY", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOBlockStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeBlock),
	)

	createPVCExportObject := func(name, namespace string, token *k8sv1.Secret) *exportv1.VirtualMachineExport {
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: &token.Name,
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     name,
				},
			},
		}
		By("Creating VMExport we can start exporting the volume")
		export, err := virtClient.VirtualMachineExport(namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return export
	}

	createPVCExportObjectWithoutSecret := func(name, namespace string) *exportv1.VirtualMachineExport {
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     name,
				},
			},
		}
		By("Creating VMExport we can start exporting the volume")
		export, err := virtClient.VirtualMachineExport(namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return export
	}

	createVMSnapshotExportObject := func(name, namespace string, token *k8sv1.Secret) *exportv1.VirtualMachineExport {
		apiGroup := "snapshot.kubevirt.io"
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: pointer.String(token.Name),
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &apiGroup,
					Kind:     "VirtualMachineSnapshot",
					Name:     name,
				},
			},
		}
		By("Creating VMExport we can start exporting the volume")
		export, err := virtClient.VirtualMachineExport(namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return export
	}

	createVMExportObject := func(name, namespace string, token *k8sv1.Secret) *exportv1.VirtualMachineExport {
		apiGroup := "kubevirt.io"
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: &token.Name,
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &apiGroup,
					Kind:     "VirtualMachine",
					Name:     name,
				},
			},
		}
		By("Creating VMExport we can start exporting the volume")
		export, err := virtClient.VirtualMachineExport(namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return export
	}

	createRunningPVCExport := func(sc string, volumeMode k8sv1.PersistentVolumeMode) *exportv1.VirtualMachineExport {
		pvc, _ := populateKubeVirtContent(sc, volumeMode)
		By("Creating the export token, we can export volumes using this token")
		// For testing the token is the name of the source pvc.
		token := createExportTokenSecret(pvc.Name, pvc.Namespace)

		export := createPVCExportObject(pvc.Name, pvc.Namespace, token)

		return waitForReadyExport(export)
	}

	createRunningVMSnapshotExport := func(snapshot *snapshotv1.VirtualMachineSnapshot) *exportv1.VirtualMachineExport {
		// For testing the token is the name of the source snapshot.
		token := createExportTokenSecret(snapshot.Name, snapshot.Namespace)
		export := createVMSnapshotExportObject(snapshot.Name, snapshot.Namespace, token)
		return waitForReadyExport(export)
	}

	matchesCNOrAlt := func(cert *x509.Certificate, hostName string) bool {
		logToGinkgoWritter("CN: %s, hostname: %s\n", cert.Subject.CommonName, hostName)
		if strings.Contains(cert.Subject.CommonName, hostName) {
			return true
		}
		for _, extension := range cert.Extensions {
			logToGinkgoWritter("ExtensionID: %s, subjectAltNameId: %s, value: %s, hostname: %s\n", extension.Id.String(), subjectAltNameId, string(extension.Value), hostName)
			if extension.Id.String() == subjectAltNameId && strings.Contains(string(extension.Value), hostName) {
				return true
			}
		}

		return false
	}

	It("Should recreate the exporter pod and secret if the pod fails", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
		checkExportSecretRef(vmExport)
		By("looking up the exporter pod and secret name")
		exporterPod := getExporterPod(vmExport)
		Expect(exporterPod).ToNot(BeNil())
		By(fmt.Sprintf("pod name %s", exporterPod.Name))
		var exporterSecretName string
		for _, volume := range exporterPod.Spec.Volumes {
			By(volume.Name)
			if volume.Name == certificates {
				exporterSecretName = volume.Secret.SecretName
			}
		}
		Expect(exporterSecretName).ToNot(BeEmpty())
		secret, err := virtClient.CoreV1().Secrets(vmExport.Namespace).Get(context.Background(), exporterSecretName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(secret).ToNot(BeNil())
		podUID := exporterPod.GetUID()

		By("Simulating the deadline expiring in the exporter")
		command := []string{
			"/bin/bash",
			"-c",
			"kill 1",
		}
		_, _, _ = exec.ExecuteCommandOnPodWithResults(virtClient, exporterPod, exporterPod.Spec.Containers[0].Name, command)
		By("Verifying the pod is killed and a new secret created")
		Eventually(func() types.UID {
			exporterPod = getExporterPod(vmExport)
			return exporterPod.UID
		}, 30*time.Second, 1*time.Second).ShouldNot(BeEquivalentTo(podUID))
		for _, volume := range exporterPod.Spec.Volumes {
			if volume.Name == certificates {
				exporterSecretName = volume.Secret.SecretName
			}
		}
		Expect(exporterSecretName).ToNot(Equal(secret.Name))
	})

	It("Should recreate the exporter pod if the pod is deleted", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
		checkExportSecretRef(vmExport)
		By("looking up the exporter pod and secret name")
		exporterPod := getExporterPod(vmExport)
		Expect(exporterPod).ToNot(BeNil())
		podUID := exporterPod.GetUID()
		err := virtClient.CoreV1().Pods(exporterPod.Namespace).Delete(context.Background(), exporterPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() types.UID {
			exporterPod = getExporterPod(vmExport)
			return exporterPod.UID
		}, 30*time.Second, 1*time.Second).ShouldNot(BeEquivalentTo(podUID))
	})

	It("Should recreate the service if the service is deleted", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
		checkExportSecretRef(vmExport)
		By("looking up the exporter pod and secret name")
		exporterService := getExportService(vmExport)
		Expect(exporterService).ToNot(BeNil())
		serviceUID := exporterService.GetUID()
		err := virtClient.CoreV1().Services(exporterService.Namespace).Delete(context.Background(), exporterService.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() types.UID {
			exporterService = getExportService(vmExport)
			return exporterService.UID
		}, 30*time.Second, 1*time.Second).ShouldNot(BeEquivalentTo(serviceUID))
	})

	It("Should handle no pvc existing when export created, then creating and populating the pvc", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
			libdv.WithForceBindAnnotation(),
		)

		name := dv.Name
		token := createExportTokenSecret(name, testsuite.GetTestNamespace(nil))
		export := createPVCExportObject(name, testsuite.GetTestNamespace(nil), token)
		expectedCond := MatchConditionIgnoreTimeStamp(exportv1.Condition{
			Type:    exportv1.ConditionPVC,
			Status:  k8sv1.ConditionFalse,
			Reason:  pvcNotFoundReason,
			Message: fmt.Sprintf("pvc %s/%s not found", testsuite.GetTestNamespace(nil), name),
		})

		Eventually(func() []exportv1.Condition {
			export, err = virtClient.VirtualMachineExport(testsuite.GetTestNamespace(export)).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status == nil {
				return nil
			}
			return export.Status.Conditions
		}, 60*time.Second, 1*time.Second).Should(ContainElement(expectedCond), "export should report missing pvc")

		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
		Eventually(func() error {
			_, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(dv)).Get(context.Background(), dv.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(BeNil(), "persistent volume associated with DV should be created")

		By("Making sure the DV is successful")
		libstorage.EventuallyDV(dv, 90, HaveSucceeded())

		By("Making sure the export becomes ready")
		waitForReadyExport(export)
		checkExportSecretRef(export)
		Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
	})

	It("should be possibe to observe exportserver pod exiting", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
		checkExportSecretRef(vmExport)
		By("looking up the exporter pod")
		exporterPod := getExporterPod(vmExport)
		Expect(exporterPod).ToNot(BeNil())
		By("creating new exporterpod")
		newExportPod := exporterPod.DeepCopy()
		newExportPod.ObjectMeta = metav1.ObjectMeta{
			Name:      exporterPod.Name + "-xxx",
			Namespace: exporterPod.Namespace,
		}
		newExportPod.Status = k8sv1.PodStatus{}
		deadline := time.Now().Add(10 * time.Second).Format(time.RFC3339)
		for i, e := range newExportPod.Spec.Containers[0].Env {
			if e.Name == "DEADLINE" {
				newExportPod.Spec.Containers[0].Env[i].Value = deadline
				break
			}
		}
		newExportPod, err := virtClient.CoreV1().Pods(newExportPod.Namespace).Create(context.TODO(), newExportPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			err = virtClient.CoreV1().Pods(newExportPod.Namespace).Delete(context.Background(), newExportPod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}()
		Eventually(func() bool {
			p, err := virtClient.CoreV1().Pods(exporterPod.Namespace).Get(context.TODO(), newExportPod.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return p.Status.Phase == k8sv1.PodSucceeded
		}, 90*time.Second, 1*time.Second).Should(BeTrue())
	})

	It("Should handle populating an export without a previously defined tokenSecretRef", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}

		pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
		export := createPVCExportObjectWithoutSecret(pvc.Name, pvc.Namespace)
		By("Making sure the export becomes ready")
		waitForReadyExport(export)

		By("Making sure the default secret is created")
		export, err = virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(export.Status.TokenSecretRef).ToNot(BeNil())

		token, err = virtClient.CoreV1().Secrets(export.Namespace).Get(context.Background(), *export.Status.TokenSecretRef, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(token.Name).To(Equal(*export.Status.TokenSecretRef))
		Expect(*export.Status.TokenSecretRef).ToNot(BeEmpty())
	})

	It("Should honor TTL by cleaning up the the VMExport altogether", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}

		pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
		ttl := &metav1.Duration{Duration: 2 * time.Minute}
		export := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: pvc.Namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     pvc.Name,
				},
				TTLDuration: ttl,
			},
		}
		export, err := virtClient.VirtualMachineExport(export.Namespace).Create(context.Background(), export, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// VMExport sticks around exactly until TTL expiration time is reached
		// Take a couple of seconds off so we don't start flaking because of races
		safeTimeout := ttl.Duration - 2*time.Second
		Consistently(func() error {
			_, err := virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			return err
		}, safeTimeout, time.Second).Should(Succeed())
		// Now gets cleaned up
		Eventually(func() error {
			_, err := virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			return err
		}, 10*time.Second, 1*time.Second).Should(
			SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())),
			"The VM export should have been cleaned up according to TTL by now",
		)
	})

	Context("[Serial]Ingress", Serial, func() {
		const (
			tlsSecretName = "test-tls"
		)

		AfterEach(func() {
			err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Delete(context.Background(), tlsSecretName, metav1.DeleteOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			err = virtClient.NetworkingV1().Ingresses(flags.KubeVirtInstallNamespace).Delete(context.Background(), "export-proxy-ingress", metav1.DeleteOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		})

		generateTestCert := func(hostName string) (string, error) {
			key, err := certutil.NewECDSAPrivateKey()
			if err != nil {
				return "", err
			}

			config := certutil.Config{
				CommonName: "blah blah",
			}
			config.AltNames.DNSNames = []string{"hahaha.wwoo", hostName, "fgdgd.dfsgdf"}

			cert, err := certutil.NewSelfSignedCACert(config, key, time.Hour)
			Expect(err).ToNot(HaveOccurred())
			pemOut := strings.Builder{}
			if err := pem.Encode(&pemOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
				return "", err
			}
			return strings.TrimSpace(pemOut.String()), nil
		}

		createIngressTLSSecret := func(name string) (string, error) {
			testCert, err := generateTestCert(testHostName)
			if err != nil {
				return "", err
			}
			secret := &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: flags.KubeVirtInstallNamespace,
				},
				StringData: map[string]string{
					tlsKey:  testKey,
					tlsCert: testCert,
				},
			}
			_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}
			return testCert, nil
		}

		createIngress := func(tlsSecretName string) *networkingv1.Ingress {
			prefix := networkingv1.PathTypePrefix
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "export-proxy-ingress",
					Namespace: flags.KubeVirtInstallNamespace,
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: pointer.String("ingress-class-name"),
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "virt-exportproxy",
							Port: networkingv1.ServiceBackendPort{
								Number: int32(443),
							},
						},
					},
					TLS: []networkingv1.IngressTLS{
						{
							Hosts: []string{
								testHostName,
							},
							SecretName: tlsSecretName,
						},
					},
					Rules: []networkingv1.IngressRule{
						{
							Host: testHostName,
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: &prefix,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "virt-exportproxy",
													Port: networkingv1.ServiceBackendPort{
														Number: int32(443),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			ingress, err := virtClient.NetworkingV1().Ingresses(flags.KubeVirtInstallNamespace).Create(context.Background(), ingress, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return ingress
		}

		It("should populate external links and cert and contain ingress host", func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			testCert, err := createIngressTLSSecret(tlsSecretName)
			Expect(err).NotTo(HaveOccurred())
			ingress := createIngress(tlsSecretName)
			vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
			checkExportSecretRef(vmExport)
			Expect(vmExport.Status.Links.External.Cert).To(Equal(testCert))
			certs, err := certutil.ParseCertsPEM([]byte(vmExport.Status.Links.External.Cert))
			Expect(err).ToNot(HaveOccurred())
			Expect(certs).ToNot(BeEmpty())
			prefix := fmt.Sprintf("%s-%s", components.VirtExportProxyServiceName, flags.KubeVirtInstallNamespace)
			domainName := strings.TrimPrefix(ingress.Spec.Rules[0].Host, prefix)
			matchesCNOrAltName := false
			for _, cert := range certs {
				if matchesCNOrAlt(cert, domainName) {
					matchesCNOrAltName = true
				}
			}
			Expect(matchesCNOrAltName).To(BeTrue())
			Expect(vmExport.Status.Links.External.Volumes[0].Formats[0].Url).To(ContainSubstring(ingress.Spec.Rules[0].Host))
		})
	})

	Context("Route", func() {
		getExportRoute := func() *routev1.Route {
			route, err := virtClient.RouteClient().Routes(flags.KubeVirtInstallNamespace).Get(context.Background(), components.VirtExportProxyServiceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return route
		}

		It("should populate external links and cert and contain route host", func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			if !checks.IsOpenShift() {
				Skip("Not on openshift")
			}
			vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
			checkExportSecretRef(vmExport)
			certs, err := certutil.ParseCertsPEM([]byte(vmExport.Status.Links.External.Cert))
			Expect(err).ToNot(HaveOccurred())
			Expect(certs).ToNot(BeEmpty())
			route := getExportRoute()
			host := ""
			if len(route.Status.Ingress) > 0 {
				host = route.Status.Ingress[0].Host
			}
			Expect(host).ToNot(BeEmpty())
			prefix := fmt.Sprintf("%s-%s", components.VirtExportProxyServiceName, flags.KubeVirtInstallNamespace)
			domainName := strings.TrimPrefix(host, prefix)
			matchesCNOrAltName := false
			for _, cert := range certs {
				if matchesCNOrAlt(cert, domainName) {
					matchesCNOrAltName = true
				}
			}
			Expect(matchesCNOrAltName).To(BeTrue())
			Expect(vmExport.Status.Links.External.Volumes[0].Formats[0].Url).To(ContainSubstring(host))

		})
	})

	waitForDisksComplete := func(vm *virtv1.VirtualMachine) {
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if volume.DataVolume != nil {
				libstorage.EventuallyDVWith(vm.Namespace, volume.DataVolume.Name, 360, HaveSucceeded())
			}
		}
	}

	checkVMNameInStatus := func(name string, export *exportv1.VirtualMachineExport) {
		Eventually(func() string {
			export, err := virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status == nil || export.Status.VirtualMachineName == nil {
				return ""
			}
			return *export.Status.VirtualMachineName
		}, 30*time.Second, time.Second).Should(Equal(name))
	}

	createDataVolume := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return dv
	}

	createVMI := func(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachineInstance {
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())
		for _, volume := range vmi.Spec.Volumes {
			if volume.DataVolume != nil {
				libstorage.EventuallyDVWith(vmi.Namespace, volume.DataVolume.Name, 180, HaveSucceeded())
			}
		}
		return vmi
	}

	createVM := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
		vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
		Expect(err).ToNot(HaveOccurred())
		waitForDisksComplete(vm)
		return vm
	}

	stopVM := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
		vmName := vm.Name
		vmNamespace := vm.Namespace
		var err error
		Eventually(func() error {
			vm, err = virtClient.VirtualMachine(vmNamespace).Get(context.Background(), vmName, &metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Running = pointer.Bool(false)
			vm, err = virtClient.VirtualMachine(vmNamespace).Update(context.Background(), vm)
			return err
		}, 15*time.Second, time.Second).Should(BeNil())
		return vm
	}

	deleteVMI := func(vmi *virtv1.VirtualMachineInstance) {
		err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	startVM := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
		vmName := vm.Name
		vmNamespace := vm.Namespace
		Eventually(func() error {
			vm, err = virtClient.VirtualMachine(vmNamespace).Get(context.Background(), vmName, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Running = pointer.Bool(true)
			vm, err = virtClient.VirtualMachine(vmNamespace).Update(context.Background(), vm)
			return err
		}, 15*time.Second, time.Second).Should(Succeed())
		return vm
	}

	newSnapshot := func(vm *virtv1.VirtualMachine) *snapshotv1.VirtualMachineSnapshot {
		apiGroup := "kubevirt.io"
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-" + vm.Name,
				Namespace: vm.Namespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &apiGroup,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
			},
		}
	}

	deleteSnapshot := func(snapshot *snapshotv1.VirtualMachineSnapshot) {
		err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	waitSnapshotReady := func(snapshot *snapshotv1.VirtualMachineSnapshot) {
		Eventually(func() bool {
			snapshot, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
		}, 180*time.Second, time.Second).Should(BeTrue())
	}

	createAndVerifyVMSnapshot := func(vm *virtv1.VirtualMachine) *snapshotv1.VirtualMachineSnapshot {
		snapshot := newSnapshot(vm)

		_, err := virtClient.VirtualMachineSnapshot(vm.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		waitSnapshotReady(snapshot)
		snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(snapshot.Status.SourceUID).ToNot(BeNil())
		Expect(*snapshot.Status.SourceUID).To(Equal(vm.UID))

		contentName := *snapshot.Status.VirtualMachineSnapshotContentName
		content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
		Expect(content.Spec.Source.VirtualMachine.UID).ToNot(BeEmpty())
		Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))
		return snapshot
	}

	verifyLinksInternal := func(vmExport *exportv1.VirtualMachineExport, expectedVolumeFormats ...exportv1.VirtualMachineExportVolumeFormat) {
		Expect(vmExport.Status).ToNot(BeNil())
		Expect(vmExport.Status.Links).ToNot(BeNil())
		Expect(vmExport.Status.Links.Internal).NotTo(BeNil())
		Expect(vmExport.Status.Links.Internal.Cert).NotTo(BeEmpty())
		Expect(vmExport.Status.Links.Internal.Volumes).To(HaveLen(len(expectedVolumeFormats) / 2))
		for _, volume := range vmExport.Status.Links.Internal.Volumes {
			Expect(volume.Formats).To(HaveLen(2))
			Expect(expectedVolumeFormats).To(ContainElements(volume.Formats))
		}
	}

	verifyMultiKubevirtInternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName1, volumeName2 string) {
		verifyLinksInternal(vmExport,
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName1),
			},
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName1),
			},
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName2),
			},
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName2),
			})
	}

	verifyKubevirtInternal := func(vmExport *exportv1.VirtualMachineExport, exportName, namespace, volumeName string) {
		verifyLinksInternal(vmExport,
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName),
			},
			exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    fmt.Sprintf("https://%s.%s.svc/volumes/%s/disk.img.gz", fmt.Sprintf("%s-%s", exportPrefix, exportName), namespace, volumeName),
			})
	}

	It("should create export from VMSnapshot", func() {
		sc, err := libstorage.GetSnapshotStorageClass(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if sc == "" {
			Skip("Skip test when storage with snapshot is not present")
		}

		vm := createVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
			cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
			testsuite.GetTestNamespace(nil),
			bashHelloScript,
			sc))
		snapshot := createAndVerifyVMSnapshot(vm)
		Expect(snapshot).ToNot(BeNil())
		defer deleteSnapshot(snapshot)
		export := createRunningVMSnapshotExport(snapshot)
		Expect(export).ToNot(BeNil())
		checkExportSecretRef(export)
		restoreName := fmt.Sprintf("%s-%s", export.Name, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)
		verifyKubevirtInternal(export, export.Name, export.Namespace, restoreName)
	})

	addDataVolumeDisk := func(vm *virtv1.VirtualMachine, diskName, dataVolumeName string) *virtv1.VirtualMachine {
		vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, virtv1.Disk{
			Name: diskName,
			DiskDevice: virtv1.DiskDevice{
				Disk: &virtv1.DiskTarget{
					Bus: virtv1.DiskBusVirtio,
				},
			},
		})
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
			Name: diskName,
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: dataVolumeName,
				},
			},
		})

		return vm
	}

	waitForExportPhase := func(export *exportv1.VirtualMachineExport, expectedPhase exportv1.VirtualMachineExportPhase) *exportv1.VirtualMachineExport {
		Eventually(func() exportv1.VirtualMachineExportPhase {
			export, err = virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status == nil {
				return ""
			}
			return export.Status.Phase
		}, 30*time.Second, time.Second).Should(Equal(expectedPhase))
		return export
	}

	waitForExportCondition := func(export *exportv1.VirtualMachineExport, condMatcher gomegatypes.GomegaMatcher, message string) *exportv1.VirtualMachineExport {
		Eventually(func() []exportv1.Condition {
			export, err = virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status == nil {
				return nil
			}
			return export.Status.Conditions
		}, 60*time.Second, 1*time.Second).Should(ContainElement(condMatcher), message)
		return export
	}

	It("should create export from VMSnapshot with multiple volumes", func() {
		sc, err := libstorage.GetSnapshotStorageClass(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if sc == "" {
			Skip("Skip test when storage with snapshot is not present")
		}

		blankDv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
		)

		vm := tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
			cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
			testsuite.GetTestNamespace(nil),
			bashHelloScript,
			sc)
		libstorage.AddDataVolumeTemplate(vm, blankDv)
		addDataVolumeDisk(vm, "blankdisk", blankDv.Name)
		vm = createVM(vm)
		stopVM(vm)
		snapshot := createAndVerifyVMSnapshot(vm)
		Expect(snapshot).ToNot(BeNil())
		defer deleteSnapshot(snapshot)
		export := createRunningVMSnapshotExport(snapshot)
		Expect(export).ToNot(BeNil())
		checkVMNameInStatus(vm.Name, export)
		checkExportSecretRef(export)
		restoreName := fmt.Sprintf("%s-%s", export.Name, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)
		// [1] is the cloud init
		restoreName2 := fmt.Sprintf("%s-%s", export.Name, vm.Spec.Template.Spec.Volumes[2].DataVolume.Name)
		verifyMultiKubevirtInternal(export, export.Name, export.Namespace, restoreName, restoreName2)
	})

	It("should mark the status phase skipped on VMSnapshot without volumes", func() {
		vm := tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
		vm = createVM(vm)
		snapshot := createAndVerifyVMSnapshot(vm)
		Expect(snapshot).ToNot(BeNil())
		defer deleteSnapshot(snapshot)
		// For testing the token is the name of the source snapshot.
		token := createExportTokenSecret(snapshot.Name, snapshot.Namespace)
		export := createVMSnapshotExportObject(snapshot.Name, snapshot.Namespace, token)
		Expect(export).ToNot(BeNil())
		waitForExportPhase(export, exportv1.Skipped)
	})

	expectedVMRunningCondition := func(name, namespace string) gomegatypes.GomegaMatcher {
		return MatchConditionIgnoreTimeStamp(exportv1.Condition{
			Type:    exportv1.ConditionReady,
			Status:  k8sv1.ConditionFalse,
			Reason:  inUseReason,
			Message: fmt.Sprintf("Virtual Machine %s/%s is running", namespace, name),
		})
	}

	expectedPVCInUseCondition := func(name, namespace string) gomegatypes.GomegaMatcher {
		return MatchConditionIgnoreTimeStamp(exportv1.Condition{
			Type:    exportv1.ConditionReady,
			Status:  k8sv1.ConditionFalse,
			Reason:  inUseReason,
			Message: fmt.Sprintf("pvc %s/%s is in use", namespace, name),
		})
	}

	expectedPVCPopulatingCondition := func(name, namespace string) gomegatypes.GomegaMatcher {
		return MatchConditionIgnoreTimeStamp(exportv1.Condition{
			Type:    exportv1.ConditionReady,
			Status:  k8sv1.ConditionFalse,
			Reason:  inUseReason,
			Message: fmt.Sprintf("Not all volumes in the Virtual Machine %s/%s are populated", namespace, name),
		})
	}

	It("should report export pending if VM is running, and start the VM export if the VM is not running, then stop again once VM started", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(
			cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
			testsuite.GetTestNamespace(nil),
			sc,
			k8sv1.ReadWriteOnce)
		vm.Spec.Running = pointer.Bool(true)
		vm = createVM(vm)
		Eventually(func() virtv1.VirtualMachineInstancePhase {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return ""
			}
			Expect(err).ToNot(HaveOccurred())
			return vmi.Status.Phase
		}, 180*time.Second, time.Second).Should(Equal(virtv1.Running))
		// For testing the token is the name of the source VM.
		token := createExportTokenSecret(vm.Name, vm.Namespace)
		export := createVMExportObject(vm.Name, vm.Namespace, token)
		Expect(export).ToNot(BeNil())
		waitForExportPhase(export, exportv1.Pending)
		checkVMNameInStatus(vm.Name, export)

		waitForExportCondition(export, expectedVMRunningCondition(vm.Name, vm.Namespace), "export should report VM running")

		By("Stopping VM, we should get the export ready eventually")
		vm = stopVM(vm)
		export = waitForReadyExport(export)
		checkExportSecretRef(export)
		Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
		verifyKubevirtInternal(export, export.Name, export.Namespace, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)
		By("Starting VM, the export should return to pending")
		vm = startVM(vm)
		waitForExportPhase(export, exportv1.Pending)
		waitForExportCondition(export, expectedVMRunningCondition(vm.Name, vm.Namespace), "export should report VM running")
	})

	Context("with limit range", func() {
		var (
			lr             *k8sv1.LimitRange
			originalConfig virtv1.KubeVirtConfiguration
		)

		updateKubeVirtExportRequestLimit := func(cpuRequest, cpuLimit, memRequest, memLimit *resource.Quantity) {
			By("Updating hotplug and container disks ratio to the specified ratio")
			resources := k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    *cpuRequest,
					k8sv1.ResourceMemory: *memRequest,
				},
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    *cpuLimit,
					k8sv1.ResourceMemory: *memLimit,
				},
			}
			config := originalConfig.DeepCopy()
			config.SupportContainerResources = []virtv1.SupportContainerResources{
				{
					Type:      virtv1.VMExport,
					Resources: resources,
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(*config)
		}

		createLimitRangeInNamespace := func(namespace string, memRatio, cpuRatio float64) {
			lr = &k8sv1.LimitRange{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      fmt.Sprintf("%s-lr", namespace),
				},
				Spec: k8sv1.LimitRangeSpec{
					Limits: []k8sv1.LimitRangeItem{
						{
							Type: k8sv1.LimitTypeContainer,
							MaxLimitRequestRatio: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse(fmt.Sprintf("%f", memRatio)),
								k8sv1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%f", cpuRatio)),
							},
							Max: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("2Gi"),
								k8sv1.ResourceCPU:    resource.MustParse("2"),
							},
							Min: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("1Mi"),
								k8sv1.ResourceCPU:    resource.MustParse("1m"),
							},
						},
					},
				},
			}
			lr, err = virtClient.CoreV1().LimitRanges(namespace).Create(context.Background(), lr, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Ensuring LimitRange exists")
			Eventually(func() error {
				lr, err = virtClient.CoreV1().LimitRanges(namespace).Get(context.Background(), lr.Name, metav1.GetOptions{})
				return err
			}, 30*time.Second, 1*time.Second).Should(BeNil())
		}

		removeLimitRangeFromNamespace := func() {
			if lr != nil {
				err = virtClient.CoreV1().LimitRanges(lr.Namespace).Delete(context.Background(), lr.Name, metav1.DeleteOptions{})
				if !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
				lr = nil
			}
			tests.UpdateKubeVirtConfigValueAndWait(originalConfig)
		}

		BeforeEach(func() {
			originalConfig = *util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
		})

		AfterEach(func() {
			removeLimitRangeFromNamespace()
		})

		It("[Serial] should report export pending if PVC is in use because of VMI using it, and start the VM export if the PVC is not in use, then stop again once pvc in use again", Serial, func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			cpu := resource.MustParse("500m")
			mem := resource.MustParse("1240Mi")
			updateKubeVirtExportRequestLimit(&cpu, &cpu, &mem, &mem)
			dataVolume := libdv.NewDataVolume(
				libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
			)
			dataVolume = createDataVolume(dataVolume)
			vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
			vmi = createVMI(vmi)
			Eventually(func() virtv1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return ""
				}
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase
			}, 180*time.Second, time.Second).Should(Equal(virtv1.Running))
			createLimitRangeInNamespace(testsuite.GetTestNamespace(nil), float64(1), float64(1))
			// For testing the token is the name of the source VM.
			token := createExportTokenSecret(vmi.Name, vmi.Namespace)
			pvcName := ""
			for _, volume := range vmi.Spec.Volumes {
				if volume.DataVolume != nil {
					pvcName = volume.DataVolume.Name
				}
			}
			Expect(pvcName).ToNot(BeEmpty())
			export := createPVCExportObject(pvcName, vmi.Namespace, token)
			Expect(export).ToNot(BeNil())
			waitForExportPhase(export, exportv1.Pending)
			waitForExportCondition(export, expectedPVCInUseCondition(dataVolume.Name, dataVolume.Namespace), "export should report pvc in use")

			By("Deleting VMI, we should get the export ready eventually")
			deleteVMI(vmi)
			export = waitForReadyExport(export)
			checkExportSecretRef(export)
			Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
			verifyKubevirtInternal(export, export.Name, export.Namespace, vmi.Spec.Volumes[0].DataVolume.Name)
			By("Verifying the ratio is proper for the exporter pod")
			exporterPod := getExporterPod(export)
			Expect(exporterPod.Spec.Containers[0].Resources.Requests.Cpu()).ToNot(BeNil())
			Expect(exporterPod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(500)))
			Expect(exporterPod.Spec.Containers[0].Resources.Limits.Cpu()).ToNot(BeNil())
			Expect(exporterPod.Spec.Containers[0].Resources.Limits.Cpu().MilliValue()).To(Equal(int64(500)))
			Expect(exporterPod.Spec.Containers[0].Resources.Requests.Memory()).ToNot(BeNil())
			Expect(exporterPod.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(int64(1240 * 1024 * 1024)))
			Expect(exporterPod.Spec.Containers[0].Resources.Limits.Memory()).ToNot(BeNil())
			Expect(exporterPod.Spec.Containers[0].Resources.Limits.Memory().Value()).To(Equal(int64(1240 * 1024 * 1024)))
			// Remove limit range to avoid having to configure proper VMI ratio for VMI.
			removeLimitRangeFromNamespace()
			By("Starting VMI, the export should return to pending")
			vmi = tests.NewRandomVMIWithDataVolume(dataVolume.Name)
			vmi = createVMI(vmi)
			waitForExportPhase(export, exportv1.Pending)
			waitForExportCondition(export, expectedPVCInUseCondition(dataVolume.Name, dataVolume.Namespace), "export should report pvc in use")
		})
	})

	getManifestUrl := func(manifests []exportv1.VirtualMachineExportManifest, manifestType exportv1.ExportManifestType) string {
		for _, manifest := range manifests {
			if manifest.Type == manifestType {
				return manifest.Url
			}
		}
		return ""
	}

	cleanMacAddresses := func(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
		if len(vm.Spec.Template.Spec.Domain.Devices.Interfaces) > 0 {
			By("Clearing out any mac addresses")
			for i := range vm.Spec.Template.Spec.Domain.Devices.Interfaces {
				vm.Spec.Template.Spec.Domain.Devices.Interfaces[i].MacAddress = ""
			}
		}
		return vm
	}

	checkWithYamlOutput := func(pod *k8sv1.Pod, export *exportv1.VirtualMachineExport, vm *virtv1.VirtualMachine) {
		By("Getting export VM definition yaml")
		url := fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests), token.Data["token"])
		command := []string{
			"curl",
			"--header",
			"Accept:application/yaml",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}

		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		split := strings.Split(out, "\n---\n")
		Expect(split).To(HaveLen(3))
		resCM := &k8sv1.ConfigMap{}
		err = yaml.Unmarshal([]byte(split[0]), resCM)
		Expect(err).ToNot(HaveOccurred())
		resVM := &virtv1.VirtualMachine{}
		err = yaml.Unmarshal([]byte(split[1]), resVM)
		Expect(err).ToNot(HaveOccurred())
		resVM.SetName(fmt.Sprintf("%s-clone", resVM.Name))
		Expect(resVM.Spec.DataVolumeTemplates).To(HaveLen(1))
		resVM.Spec.DataVolumeTemplates[0].SetName(fmt.Sprintf("%s-clone", resVM.Spec.DataVolumeTemplates[0].Name))
		Expect(resVM.Spec.Template).ToNot(BeNil())
		Expect(resVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
		Expect(resVM.Spec.Template.Spec.Volumes[0].DataVolume).ToNot(BeNil())
		resVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = resVM.Spec.DataVolumeTemplates[0].Name
		resVM = cleanMacAddresses(resVM)
		By("Getting token secret header")
		url = fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader), token.Data["token"])
		command = []string{
			"curl",
			"--header",
			"Accept:application/yaml",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}
		out, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		split = strings.Split(out, "\n---\n")
		Expect(split).To(HaveLen(2))
		resSecret := &k8sv1.Secret{}
		err = yaml.Unmarshal([]byte(split[0]), resSecret)
		Expect(err).ToNot(HaveOccurred())
		resSecret, err = virtClient.CoreV1().Secrets(vm.Namespace).Create(context.Background(), resSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resSecret).ToNot(BeNil())
		resCM, err = virtClient.CoreV1().ConfigMaps(vm.Namespace).Create(context.Background(), resCM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resCM).ToNot(BeNil())
		Expect(resVM.Spec.Running).ToNot(BeNil())
		*resVM.Spec.Running = true
		resVM, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), resVM)
		Expect(err).ToNot(HaveOccurred())
		Expect(resVM).ToNot(BeNil())
		waitForDisksComplete(resVM)
	}

	checkWithJsonOutput := func(pod *k8sv1.Pod, export *exportv1.VirtualMachineExport, vm *virtv1.VirtualMachine) {
		By("Getting export VM definition yaml")
		url := fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests), token.Data["token"])
		command := []string{
			"curl",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}

		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		list := &k8sv1.List{}
		err = json.Unmarshal([]byte(out), list)
		Expect(err).ToNot(HaveOccurred())
		Expect(list.Items).To(HaveLen(2))

		resCM := &k8sv1.ConfigMap{}
		err = yaml.Unmarshal(list.Items[0].Raw, resCM)
		Expect(err).ToNot(HaveOccurred())
		resCM.Name = fmt.Sprintf("%s-clone-json", resCM.Name)
		resVM := &virtv1.VirtualMachine{}
		err = yaml.Unmarshal(list.Items[1].Raw, resVM)
		Expect(err).ToNot(HaveOccurred())
		resVM.SetName(fmt.Sprintf("%s-clone-json", resVM.Name))
		Expect(resVM.Spec.DataVolumeTemplates).To(HaveLen(1))
		resVM.Spec.DataVolumeTemplates[0].SetName(fmt.Sprintf("%s-clone-json", resVM.Spec.DataVolumeTemplates[0].Name))
		resVM.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.CertConfigMap = resCM.Name
		Expect(resVM.Spec.Template).ToNot(BeNil())
		Expect(resVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
		Expect(resVM.Spec.Template.Spec.Volumes[0].DataVolume).ToNot(BeNil())
		resVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = resVM.Spec.DataVolumeTemplates[0].Name
		resVM = cleanMacAddresses(resVM)
		By("Getting token secret header")
		url = fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader), token.Data["token"])
		command = []string{
			"curl",
			"--header",
			"Accept:application/yaml",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}
		out, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		resSecret := &k8sv1.Secret{}
		err = yaml.Unmarshal([]byte(out), resSecret)
		Expect(err).ToNot(HaveOccurred())
		resSecret.Name = fmt.Sprintf("%s-clone-json", resSecret.Name)
		resVM.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.SecretExtraHeaders = []string{resSecret.Name}
		resSecret, err = virtClient.CoreV1().Secrets(vm.Namespace).Create(context.Background(), resSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resSecret).ToNot(BeNil())
		resCM, err = virtClient.CoreV1().ConfigMaps(vm.Namespace).Create(context.Background(), resCM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resCM).ToNot(BeNil())
		Expect(resVM.Spec.Running).ToNot(BeNil())
		*resVM.Spec.Running = true
		resVM, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), resVM)
		Expect(err).ToNot(HaveOccurred())
		Expect(resVM).ToNot(BeNil())
		waitForDisksComplete(resVM)
	}

	It("should generate updated DataVolumeTemplates on http endpoint when exporting", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, sc, k8sv1.ReadWriteOnce)
		vm.Spec.Running = pointer.Bool(true)
		vm = createVM(vm)
		Expect(vm).ToNot(BeNil())
		vm = stopVM(vm)
		token := createExportTokenSecret(vm.Name, vm.Namespace)
		export := createVMExportObject(vm.Name, vm.Namespace, token)
		Expect(export).ToNot(BeNil())
		export = waitForReadyExport(export)
		checkExportSecretRef(export)
		Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
		verifyKubevirtInternal(export, export.Name, export.Namespace, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)
		Expect(export.Status).ToNot(BeNil())
		Expect(export.Status.Links).ToNot(BeNil())
		Expect(export.Status.Links.Internal).ToNot(BeNil())
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/all", fmt.Sprintf("virt-export-%s", export.Name), export.Namespace)))
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/secret", fmt.Sprintf("virt-export-%s", export.Name), export.Namespace)))
		Expect(err).ToNot(HaveOccurred())
		caConfigMap := createCaConfigMapInternal("export-cacerts", vm.Namespace, export)
		Expect(caConfigMap).ToNot(BeNil())
		pod := createDownloadPod(caConfigMap)
		pod = tests.RunPod(pod)
		checkWithYamlOutput(pod, export, vm)
		checkWithJsonOutput(pod, export, vm)
	})

	It("should generate updated DataVolumeTemplates on http endpoint when exporting snapshot", func() {
		virtClient, err := kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		sc, err := libstorage.GetSnapshotStorageClass(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if sc == "" {
			Skip("Skip test when storage with snapshot is not present")
		}

		vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, sc, k8sv1.ReadWriteOnce)
		vm.Spec.Running = pointer.Bool(true)
		vm = createVM(vm)
		Expect(vm).ToNot(BeNil())
		vm = stopVM(vm)
		snapshot := createAndVerifyVMSnapshot(vm)
		export := createRunningVMSnapshotExport(snapshot)
		Expect(export).ToNot(BeNil())
		export = waitForReadyExport(export)
		verifyKubevirtInternal(export, export.Name, export.Namespace, fmt.Sprintf("%s-%s", export.Name, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name))
		Expect(export.Status).ToNot(BeNil())
		Expect(export.Status.Links).ToNot(BeNil())
		Expect(export.Status.Links.Internal).ToNot(BeNil())
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/all", fmt.Sprintf("virt-export-%s", export.Name), export.Namespace)))
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/secret", fmt.Sprintf("virt-export-%s", export.Name), export.Namespace)))
		Expect(err).ToNot(HaveOccurred())
		caConfigMap := createCaConfigMapInternal("export-cacerts", vm.Namespace, export)
		Expect(caConfigMap).ToNot(BeNil())
		pod := createDownloadPod(caConfigMap)
		pod = tests.RunPod(pod)
		checkWithYamlOutput(pod, export, vm)
		checkWithJsonOutput(pod, export, vm)
	})

	It("Should generate DVs and expanded VM definition on http endpoint with multiple volumes", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		clusterInstancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineClusterInstancetype",
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "export-test-cluster-instancetype",
			},
			Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1beta1.CPUInstancetype{
					Guest: uint32(4),
				},
			},
		}

		_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			err = virtClient.VirtualMachineClusterInstancetype().Delete(context.Background(), clusterInstancetype.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}()

		imageUrl := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)
		dataVolume := libdv.NewDataVolume(
			libdv.WithRegistryURLSourceAndPullMethod(imageUrl, cdiv1.RegistryPullNode),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.CirrosVolumeSize)),
		)
		dataVolume.SetNamespace(testsuite.GetTestNamespace(dataVolume))
		dataVolume = createDataVolume(dataVolume)
		blankDv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
		)
		blankDv.SetNamespace(testsuite.GetTestNamespace(blankDv))
		blankDv = createDataVolume(blankDv)

		vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
		tests.AddUserData(vmi, "cloud-init", bashHelloScript)
		vm := tests.NewRandomVirtualMachine(vmi, false)
		addDataVolumeDisk(vm, "blankdisk", blankDv.Name)
		vm.Spec.Running = pointer.Bool(true)
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			Name: clusterInstancetype.Name,
		}
		// If I don't remove this, it clashes with the instance type.
		delete(vm.Spec.Template.Spec.Domain.Resources.Requests, k8sv1.ResourceMemory)
		vm = createVM(vm)
		Expect(vm).ToNot(BeNil())
		vm = stopVM(vm)
		token := createExportTokenSecret(vm.Name, vm.Namespace)
		export := createVMExportObject(vm.Name, vm.Namespace, token)
		Expect(export).ToNot(BeNil())
		export = waitForReadyExport(export)
		checkExportSecretRef(export)
		Expect(*export.Status.TokenSecretRef).To(Equal(token.Name))
		Expect(vm).ToNot(BeNil())
		Expect(vm.Spec.Template).ToNot(BeNil())
		Expect(vm.Spec.Template.Spec.Volumes).ToNot(BeEmpty())
		// index 1 is for cloud init
		verifyMultiKubevirtInternal(export, export.Name, export.Namespace, vm.Spec.Template.Spec.Volumes[0].DataVolume.Name, vm.Spec.Template.Spec.Volumes[2].DataVolume.Name)
		Expect(export.Status).ToNot(BeNil())
		Expect(export.Status.Links).ToNot(BeNil())
		Expect(export.Status.Links.Internal).ToNot(BeNil())
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/all", fmt.Sprintf("virt-export-%s", export.Name), vm.Namespace)))
		Expect(getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader)).To(Equal(fmt.Sprintf("https://%s.%s.svc/internal/manifests/secret", fmt.Sprintf("virt-export-%s", export.Name), vm.Namespace)))
		Expect(err).ToNot(HaveOccurred())
		caConfigMap := createCaConfigMapInternal("export-cacerts", vm.Namespace, export)
		Expect(caConfigMap).ToNot(BeNil())
		pod := createDownloadPod(caConfigMap)
		pod = tests.RunPod(pod)
		By("Getting export VM definition yaml")
		url := fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AllManifests), token.Data["token"])
		command := []string{
			"curl",
			"--header",
			"Accept:application/yaml",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}

		out, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		split := strings.Split(out, "\n---\n")
		Expect(split).To(HaveLen(5))
		resCM := &k8sv1.ConfigMap{}
		err = yaml.Unmarshal([]byte(split[0]), resCM)
		Expect(err).ToNot(HaveOccurred())
		resVM := &virtv1.VirtualMachine{}
		err = yaml.Unmarshal([]byte(split[1]), resVM)
		Expect(err).ToNot(HaveOccurred())
		resVM.SetName(fmt.Sprintf("%s-clone", resVM.Name))
		Expect(resVM.Spec.DataVolumeTemplates).To(BeEmpty())
		Expect(resVM.Spec.Template).ToNot(BeNil())
		Expect(resVM.Spec.Template.Spec.Domain.CPU.Sockets).To(Equal(uint32(4)))
		Expect(resVM.Spec.Template.Spec.Volumes).To(HaveLen(3))
		Expect(resVM.Spec.Template.Spec.Volumes[0].DataVolume).ToNot(BeNil())
		resVM = cleanMacAddresses(resVM)
		resVM.Spec.Template.Spec.Volumes[0].DataVolume.Name = fmt.Sprintf("%s-clone", resVM.Spec.Template.Spec.Volumes[0].DataVolume.Name)
		resVM.Spec.Template.Spec.Volumes[2].DataVolume.Name = fmt.Sprintf("%s-clone", resVM.Spec.Template.Spec.Volumes[2].DataVolume.Name)
		diskDV := &cdiv1.DataVolume{}
		err = yaml.Unmarshal([]byte(split[2]), diskDV)
		Expect(err).ToNot(HaveOccurred())
		diskDV.Name = fmt.Sprintf("%s-clone", diskDV.Name)
		diskDV.Spec.PVC.StorageClassName = pointer.String(sc)
		Expect(diskDV.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage]).To(BeEquivalentTo(resource.MustParse(cd.CirrosVolumeSize)))
		blankDv = &cdiv1.DataVolume{}
		err = yaml.Unmarshal([]byte(split[3]), blankDv)
		Expect(err).ToNot(HaveOccurred())
		blankDv.Name = fmt.Sprintf("%s-clone", blankDv.Name)
		blankDv.Spec.PVC.StorageClassName = pointer.String(sc)
		Expect(blankDv.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage]).To(BeEquivalentTo(resource.MustParse(cd.BlankVolumeSize)))

		By("Getting token secret header")
		url = fmt.Sprintf("%s?x-kubevirt-export-token=%s", getManifestUrl(export.Status.Links.Internal.Manifests, exportv1.AuthHeader), token.Data["token"])
		command = []string{
			"curl",
			"--header",
			"Accept:application/yaml",
			"--cacert",
			filepath.Join(caCertPath, caBundleKey),
			url,
		}
		out, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		split = strings.Split(out, "\n---\n")
		Expect(split).To(HaveLen(2))
		resSecret := &k8sv1.Secret{}
		err = yaml.Unmarshal([]byte(split[0]), resSecret)
		Expect(err).ToNot(HaveOccurred())
		resSecret, err = virtClient.CoreV1().Secrets(vm.Namespace).Create(context.Background(), resSecret, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resSecret).ToNot(BeNil())
		diskDV = createDataVolume(diskDV)
		Expect(diskDV).ToNot(BeNil())
		blankDv = createDataVolume(blankDv)
		Expect(blankDv).ToNot(BeNil())
		resCM, err = virtClient.CoreV1().ConfigMaps(vm.Namespace).Create(context.Background(), resCM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(resCM).ToNot(BeNil())
		Expect(resVM.Spec.Running).ToNot(BeNil())
		*resVM.Spec.Running = true
		resVM, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), resVM)
		Expect(err).ToNot(HaveOccurred())
		Expect(resVM).ToNot(BeNil())
		waitForDisksComplete(resVM)
	})

	It("should mark the status phase skipped on VM without volumes", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vm := tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
		vm = createVM(vm)
		// For testing the token is the name of the source VM.
		token := createExportTokenSecret(vm.Name, vm.Namespace)
		export := createVMExportObject(vm.Name, vm.Namespace, token)
		Expect(export).ToNot(BeNil())
		waitForExportPhase(export, exportv1.Skipped)

		dv := libdv.NewDataVolume(
			libdv.WithNamespace(vm.Namespace),
			libdv.WithBlankImageSource(),
			libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
		)
		dv = createDataVolume(dv)
		Eventually(ThisPVCWith(vm.Namespace, dv.Name), 160).Should(Exist())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		libstorage.AddDataVolume(vm, "blank-disk", dv)
		vm, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
		Expect(err).ToNot(HaveOccurred())
		if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(sc) {
			// With WFFC we expect the volume to not be populated and the condition to be not ready and reason populating
			// start the VM which triggers the populating, and then it should become ready.
			waitForExportPhase(export, exportv1.Pending)
			waitForExportCondition(export, expectedPVCPopulatingCondition(vm.Name, vm.Namespace), "export should report PVCs in VM populating")
			vm = startVM(vm)
			waitForDisksComplete(vm)
			stopVM(vm)
			waitForExportPhase(export, exportv1.Ready)
		} else {
			// With non WFFC storage we expect the disk to populate immediately and thus the export to become ready
			waitForDisksComplete(vm)
			waitForExportPhase(export, exportv1.Ready)
		}
	})

	Context("[Serial] with potential KubeVirt CR update", Serial, func() {
		var beforeCertParams *virtv1.KubeVirtCertificateRotateStrategy

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)
			beforeCertParams = kv.Spec.CertificateRotationStrategy.DeepCopy()
		})

		AfterEach(func() {
			kv := util.GetCurrentKv(virtClient)
			if equality.Semantic.DeepEqual(beforeCertParams, &kv.Spec.CertificateRotationStrategy) {
				return
			}
			kv.Spec.CertificateRotationStrategy = *beforeCertParams
			_, err := virtClient.KubeVirt(kv.Namespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())
		})

		updateCertConfig := func() {
			kv := util.GetCurrentKv(virtClient)
			kv.Spec.CertificateRotationStrategy.SelfSigned = &virtv1.KubeVirtSelfSignConfiguration{
				CA: &virtv1.CertConfig{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 3 * time.Hour},
				},
				Server: &virtv1.CertConfig{
					Duration:    &metav1.Duration{Duration: 2 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
				},
			}
			_, err := virtClient.KubeVirt(kv.Namespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())
		}

		It("should recreate exportserver pod when KubeVirt cert params updated", func() {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
			checkExportSecretRef(vmExport)
			By("looking up the exporter pod")
			exporterPod := getExporterPod(vmExport)
			Expect(exporterPod).ToNot(BeNil())
			podUID := exporterPod.GetUID()
			preCertParamms := exporterPod.Annotations["kubevirt.io/export.certParameters"]
			Expect(preCertParamms).ToNot(BeEmpty())

			By("updating cert configuration")
			updateCertConfig()

			By("Verifying the pod is killed and a new one created")
			Eventually(func() types.UID {
				exporterPod = getExporterPod(vmExport)
				return exporterPod.UID
			}, 30*time.Second, 1*time.Second).ShouldNot(BeEquivalentTo(podUID))

			postCertParamms := exporterPod.Annotations["kubevirt.io/export.certParameters"]
			Expect(postCertParamms).ToNot(BeEmpty())
			Expect(postCertParamms).ToNot(Equal(preCertParamms))
		})
	})

	var _ = Describe("virtctl vmexport command", func() {
		const (
			commandName    = "vmexport"
			defaultOutput  = "/tmp/test-disk-%s.img"
			defaultVMEName = "vme-test-%s"
		)

		var (
			sc         string
			vmeName    string
			outputFile string
		)

		checkForReadyExport := func(name string) {
			vmexport, err := virtClient.VirtualMachineExport(testsuite.GetTestNamespace(nil)).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			waitForReadyExport(vmexport)
		}

		BeforeEach(func() {
			storageClass, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			sc = storageClass
			vmeName = fmt.Sprintf(defaultVMEName, rand.String(12))
		})

		AfterEach(func() {
			By("Deleting VirtualMachineExport")
			vmexport, err := virtClient.VirtualMachineExport(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmeName, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
				err = virtClient.VirtualMachineExport(testsuite.GetTestNamespace(vmexport)).Delete(context.Background(), vmexport.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("Create succeeds using PVC source", func() {
			pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "create", vmeName, "--pvc", pvc.Name, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
			checkForReadyExport(vmeName)
		})

		It("Create succeeds using Snapshot source", func() {
			sc, err := libstorage.GetSnapshotStorageClass(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if sc == "" {
				Skip("Skip test when storage with snapshot is not present")
			}
			// Create a populated Snapshot
			blankDv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
			)

			vm := tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
				cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
				testsuite.GetTestNamespace(nil),
				bashHelloScript,
				sc)
			libstorage.AddDataVolumeTemplate(vm, blankDv)
			addDataVolumeDisk(vm, "blankdisk", blankDv.Name)
			vm = createVM(vm)
			snapshot := createAndVerifyVMSnapshot(vm)
			Expect(snapshot).ToNot(BeNil())
			defer deleteSnapshot(snapshot)

			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "create", vmeName, "--snapshot", snapshot.Name, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
			checkForReadyExport(vmeName)
		})

		It("Create succeeds using VM source", func() {
			// Create a populated VM
			vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(
				cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
				testsuite.GetTestNamespace(nil),
				sc,
				k8sv1.ReadWriteOnce)
			vm.Spec.Running = pointer.Bool(true)
			vm = createVM(vm)
			Eventually(func() virtv1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return ""
				}
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase
			}, 180*time.Second, time.Second).Should(Equal(virtv1.Running))

			By("Stopping VM, we should get the export ready eventually")
			vm = stopVM(vm)

			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "create", vmeName, "--vm", vm.Name, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
			checkForReadyExport(vmeName)
		})

		It("Create fails when the vmexport already exists", func() {
			vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
			vmeName = vmExport.Name
			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "create", vmeName, "--pvc", vmExport.Spec.Source.Name, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).To(HaveOccurred())
			errString := fmt.Sprintf("VirtualMachineExport '%s/%s' already exists", vmExport.Namespace, vmeName)
			Expect(err.Error()).Should(Equal(errString))
		})

		It("Delete succeeds", func() {
			vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
			vmeName = vmExport.Name
			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "delete", vmeName, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
			By("Verifying the vmexport was deleted")
			Eventually(func() bool {
				_, err := virtClient.VirtualMachineExport(testsuite.GetTestNamespace(vmExport)).Get(context.Background(), vmExport.Name, metav1.GetOptions{})
				if err == nil {
					return false
				}
				return errors.IsNotFound(err)
			}, 180*time.Second, time.Second).Should(BeTrue())
		})

		It("Delete succeeds when vmexport doesn't exist", func() {
			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "delete", vmeName, "--namespace", testsuite.GetTestNamespace(nil))
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Create with TTL", func() {
			ttl := &metav1.Duration{Duration: 2 * time.Minute}
			pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
			// Run vmexport
			By("Running vmexport command")
			virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName, "create", vmeName, "--pvc", pvc.Name, "--namespace", testsuite.GetTestNamespace(pvc), "--ttl", ttl.Duration.String())
			err = virtctlCmd()
			Expect(err).ToNot(HaveOccurred())
			export, err := virtClient.VirtualMachineExport(testsuite.GetTestNamespace(pvc)).Get(context.Background(), vmeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(export.Spec.TTLDuration).To(Equal(ttl))
		})

		Context("Download a volume with vmexport", func() {
			BeforeEach(func() {
				outputFile = fmt.Sprintf(defaultOutput, rand.String(12))
			})

			AfterEach(func() {
				if err := os.Remove(outputFile); err != nil && !goerrors.Is(err, os.ErrNotExist) {
					Fail(err.Error())
				}
			})

			It("Download succeeds creating and downloading a vmexport using PVC source", func() {
				// Create populated PVC
				pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
				// Run vmexport
				By("Running vmexport command")
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--pvc", pvc.Name,
					"--output", outputFile,
					"--volume", pvc.Name,
					"--insecure",
					"--namespace", testsuite.GetTestNamespace(pvc),
				}

				if !checks.IsOpenShift() {
					cmdArgs = append(cmdArgs, "--port-forward")
				}

				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)
				err = virtctlCmd()
				Expect(err).ToNot(HaveOccurred())
				_, err := os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Download succeeds creating and downloading a vmexport using Snapshot source", func() {
				sc, err := libstorage.GetSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())
				if sc == "" {
					Skip("Skip test when storage with snapshot is not present")
				}

				// Create a populated Snapshot
				blankDv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
				)
				vm := tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
					sc)
				libstorage.AddDataVolumeTemplate(vm, blankDv)
				addDataVolumeDisk(vm, "blankdisk", blankDv.Name)
				vm = createVM(vm)
				snapshot := createAndVerifyVMSnapshot(vm)
				Expect(snapshot).ToNot(BeNil())
				defer deleteSnapshot(snapshot)

				// We create the vmexport object in advance to get the volume name
				export := createRunningVMSnapshotExport(snapshot)
				Expect(export).ToNot(BeNil())
				checkExportSecretRef(export)
				vmeName = export.Name

				// Run vmexport
				By("Running vmexport command")
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--snapshot", snapshot.Name,
					"--output", outputFile,
					"--volume", vm.Spec.Template.Spec.Volumes[0].DataVolume.Name,
					"--insecure",
					"--namespace", testsuite.GetTestNamespace(export),
				}

				if !checks.IsOpenShift() {
					cmdArgs = append(cmdArgs, "--port-forward")
				}

				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)
				err = virtctlCmd()
				Expect(err).ToNot(HaveOccurred())
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Download succeeds creating and downloading a vmexport using VM source", func() {
				// Create a populated VM
				vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					sc,
					k8sv1.ReadWriteOnce)
				vm.Spec.Running = pointer.Bool(true)
				vm = createVM(vm)
				Eventually(func() virtv1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return ""
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase
				}, 180*time.Second, time.Second).Should(Equal(virtv1.Running))

				By("Stopping VM, we should get the export ready eventually")
				vm = stopVM(vm)

				// Run vmexport
				By("Running vmexport command")
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--vm", vm.Name,
					"--output", outputFile,
					"--volume", vm.Spec.Template.Spec.Volumes[0].DataVolume.Name,
					"--insecure",
					"--namespace", testsuite.GetTestNamespace(vm),
				}

				if !checks.IsOpenShift() {
					cmdArgs = append(cmdArgs, "--port-forward")
				}

				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)
				err = virtctlCmd()
				Expect(err).ToNot(HaveOccurred())
				_, err := os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Download succeeds with an already existing vmexport", func() {
				vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
				vmeName = vmExport.Name
				// Run vmexport
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--output", outputFile,
					"--insecure",
					"--namespace", testsuite.GetTestNamespace(vmExport),
				}
				export, err := virtClient.VirtualMachineExport(vmExport.Namespace).Get(context.Background(), vmExport.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if export.Status.Links.External == nil {
					localPort := fmt.Sprintf("%d", 37548+rand.Intn(6000))
					cmdArgs = append(cmdArgs, "--volume", vmExport.Status.Links.Internal.Volumes[0].Name, "--port-forward", "--local-port", localPort, "--keep-vme")
				} else {
					cmdArgs = append(cmdArgs, "--volume", vmExport.Status.Links.External.Volumes[0].Name)
				}
				By("Running vmexport command")
				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)

				Eventually(func() error {
					return virtctlCmd()
				}, 30*time.Second, time.Second).Should(BeNil())
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Download succeeds with a vmexport without user-defined TokenSecretRef", func() {
				pvc, _ := populateKubeVirtContent(sc, k8sv1.PersistentVolumeFilesystem)
				export := createPVCExportObjectWithoutSecret(pvc.Name, pvc.Namespace)
				By("Making sure the export becomes ready")
				waitForReadyExport(export)

				By("Making sure the default secret is created")
				export, err = virtClient.VirtualMachineExport(export.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(export.Status.TokenSecretRef).ToNot(BeNil())

				vmeName = export.Name
				// Run vmexport
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--output", outputFile,
					"--insecure",
					"--namespace", testsuite.GetTestNamespace(export),
				}
				if export.Status.Links.External == nil {
					localPort := fmt.Sprintf("%d", 37548+rand.Intn(6000))
					cmdArgs = append(cmdArgs, "--port-forward", "--local-port", localPort,
						"--insecure", "--keep-vme")
				}
				By("Running vmexport command")
				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)
				Eventually(func() error {
					return virtctlCmd()
				}, 30*time.Second, time.Second).Should(BeNil())
				_, err := os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Download succeeds and keeps the vmexport after finishing the download", func() {
				vmExport := createRunningPVCExport(sc, k8sv1.PersistentVolumeFilesystem)
				vmeName = vmExport.Name
				vme, err := virtClient.VirtualMachineExport(vmExport.Namespace).Get(context.Background(), vmeName, metav1.GetOptions{})
				// Run vmexport
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--output", outputFile,
					"--insecure",
					"--keep-vme",
					"--namespace", testsuite.GetTestNamespace(vmExport),
				}
				Expect(err).ToNot(HaveOccurred())
				if vme.Status.Links.External == nil {
					localPort := fmt.Sprintf("%d", 37548+rand.Intn(6000))
					cmdArgs = append(cmdArgs, "--volume", vmExport.Status.Links.Internal.Volumes[0].Name, "--port-forward", "--local-port", localPort)
				} else {
					cmdArgs = append(cmdArgs, "--volume", vmExport.Status.Links.External.Volumes[0].Name)
				}
				By("Running vmexport command")
				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(cmdArgs...)
				Eventually(func() error {
					return virtctlCmd()
				}, 30*time.Second, time.Second).Should(BeNil())

				Expect(err).ToNot(HaveOccurred())
				checkForReadyExport(vmeName)
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Get export manifest from vmexport", func() {
			DescribeTable("manifest should be successfully retrieved on running VM export", func(expectedObjects int, extraArgs ...string) {
				// Create a populated VM
				vm := tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					sc,
					k8sv1.ReadWriteOnce)
				vm.Spec.Running = pointer.Bool(true)
				vm = createVM(vm)
				Eventually(func() virtv1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return ""
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase
				}, 180*time.Second, time.Second).Should(Equal(virtv1.Running))

				By("Stopping VM, we should get the export ready eventually")
				vm = stopVM(vm)

				// Run vmexport
				By("Running vmexport create command")
				virtctlCmd := clientcmd.NewRepeatableVirtctlCommand(commandName,
					"create",
					vmeName,
					"--vm", vm.Name,
					"--namespace", testsuite.GetTestNamespace(vm))
				err = virtctlCmd()
				Expect(err).ToNot(HaveOccurred())

				exportReady := exportv1.Ready
				Eventually(func() *exportv1.VirtualMachineExportPhase {
					vme, err := virtClient.VirtualMachineExport(vm.Namespace).Get(context.Background(), vmeName, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return nil
					}
					Expect(err).ToNot(HaveOccurred())
					if vme.Status == nil {
						return nil
					}
					return &vme.Status.Phase
				}, 180*time.Second, time.Second).Should(Equal(&exportReady))
				vme, err := virtClient.VirtualMachineExport(vm.Namespace).Get(context.Background(), vmeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				cmdArgs := []string{
					commandName,
					"download",
					vmeName,
					"--manifest",
					"--namespace",
					testsuite.GetTestNamespace(vm),
				}
				if vme.Status.Links.External == nil {
					localPort := fmt.Sprintf("%d", 37548+rand.Intn(6000))
					cmdArgs = append(cmdArgs, "--service-url", fmt.Sprintf("127.0.0.1:%s", localPort), "--port-forward", "--local-port", localPort)
				}
				cmdArgs = append(cmdArgs, extraArgs...)
				virtctlCmdOut := clientcmd.NewRepeatableVirtctlCommandWithOut(cmdArgs...)

				var out []byte
				By("Running vmexport manifest command")
				Eventually(func() error {
					out, err = virtctlCmdOut()
					return err
				}, 30*time.Second, time.Second).Should(BeNil())
				Expect(out).NotTo(BeEmpty())
				split := strings.Split(string(out), "\n---\n")
				// Add one for the --- at the end.
				Expect(split).To(HaveLen(expectedObjects + 1))
				resVM := &virtv1.VirtualMachine{}
				err = yaml.Unmarshal([]byte(split[1]), resVM)
				Expect(err).ToNot(HaveOccurred())
			},
				Entry("without --include-secret", 2),
				Entry("with --include-secret", 3, "--include-secret"),
			)
		})
	})
})

func logToGinkgoWritter(format string, parameters ...interface{}) {
	_, _ = fmt.Fprintf(GinkgoWriter, format, parameters...)
}

func MatchConditionIgnoreTimeStamp(expected interface{}) gomegatypes.GomegaMatcher {
	return &ConditionNoTimeMatcher{
		Cond: expected,
	}
}

type ConditionNoTimeMatcher struct {
	Cond interface{}
}

func (matcher *ConditionNoTimeMatcher) Match(actual interface{}) (success bool, err error) {
	cond, isCond := actual.(exportv1.Condition)
	if !isCond {
		return false, fmt.Errorf("ConditionNoTimeMatch expects an export condition")
	}
	expectedCond, isCond := matcher.Cond.(exportv1.Condition)
	if !isCond {
		return false, fmt.Errorf("ConditionNoTimeMatch expects an export condition")
	}
	return cond.Type == expectedCond.Type && cond.Status == expectedCond.Status && cond.Reason == expectedCond.Reason && cond.Message == expectedCond.Message, nil
}

func (matcher *ConditionNoTimeMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to match without time", matcher.Cond)
}

func (matcher *ConditionNoTimeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to match without time", matcher.Cond)
}
