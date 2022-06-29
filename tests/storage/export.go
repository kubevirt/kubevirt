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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
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

	pvcNotFoundReason = "pvcNotFound"

	proxyUrlBase = "https://virt-exportproxy.%s.svc/api/export.kubevirt.io/v1alpha1/namespaces/%s/virtualmachineexports/%s%s"
)

var _ = SIGDescribe("Export", func() {
	var err error
	var token *k8sv1.Secret
	var virtClient kubecli.KubevirtClient

	waitExportProxyReady := func() {
		Eventually(func() bool {
			d, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.TODO(), "virt-exportproxy", metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false
			}
			Expect(err).ToNot(HaveOccurred())
			return d.Status.AvailableReplicas > 0
		}, 90*time.Second, 1*time.Second).Should(Equal(true))
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		waitExportProxyReady()
	})

	AfterEach(func() {
		err := virtClient.CoreV1().Secrets(token.Namespace).Delete(context.Background(), token.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
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

	createDownloadPodForPvc := func(pvc *k8sv1.PersistentVolumeClaim, caConfigMap *k8sv1.ConfigMap) *k8sv1.Pod {
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
		}, k8sv1.Volume{
			Name: "cacerts",
			VolumeSource: k8sv1.VolumeSource{
				ConfigMap: &k8sv1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{
						Name: caConfigMap.Name,
					},
				},
			},
		})

		volumeMode := pvc.Spec.VolumeMode
		if volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock {
			addBlockVolume(pod, volumeName)
		} else {
			addFilesystemVolume(pod, volumeName)
		}
		addCertVolume(pod)
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

		volumeMode := pvc.Spec.VolumeMode
		if volumeMode != nil && *volumeMode == k8sv1.PersistentVolumeBlock {
			addBlockVolume(pod, volumeName)
		} else {
			addFilesystemVolume(pod, volumeName)
		}
		return tests.RunPod(pod)
	}

	createTriggerPodForPvc := func(pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
		volumeName := pvc.GetName()
		podName := fmt.Sprintf("bind-%s", volumeName)
		pod := tests.RenderPod(podName, []string{"/bin/sh", "-c", "sleep 1"}, []string{})
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
		return tests.RunPodAndExpectCompletion(pod)
	}

	isWaitForFirstConsumer := func(storageClassName string) bool {
		sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer
	}

	ensurePVCBound := func(pvc *k8sv1.PersistentVolumeClaim) {
		namespace := pvc.Namespace
		if !isWaitForFirstConsumer(*pvc.Spec.StorageClassName) {
			By("Checking for bound claim on non-WFFC storage")
			// Not WFFC, pvc will be bound
			Eventually(func() k8sv1.PersistentVolumeClaimPhase {
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return pvc.Status.Phase
			}, 15*time.Second, 1*time.Second).Should(Equal(k8sv1.ClaimBound))
			return
		}
		By("Checking the PVC is pending for WFFC storage")
		Eventually(func() k8sv1.PersistentVolumeClaimPhase {
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pvc.Status.Phase
		}, 15*time.Second, 1*time.Second).Should(Equal(k8sv1.ClaimPending))

		By("Creating trigger pod to bind WFFC storage")
		triggerPod := createTriggerPodForPvc(pvc)
		By("Checking the PVC was bound")
		Eventually(func() k8sv1.PersistentVolumeClaimPhase {
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pvc.Status.Phase
		}, 30*time.Second, 1*time.Second).Should(Equal(k8sv1.ClaimBound))
		By("Deleting the trigger pod")
		immediate := int64(0)
		virtClient.CoreV1().Pods(triggerPod.Namespace).Delete(context.Background(), triggerPod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &immediate,
		})
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

	createCaConfigMapProxy := func(name, namespace string, export *exportv1.VirtualMachineExport) *k8sv1.ConfigMap {
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
		dv := libstorage.NewRandomDataVolumeWithRegistryImportInStorageClass(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault, sc, k8sv1.ReadWriteOnce, volumeMode)
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
		var pvc *k8sv1.PersistentVolumeClaim
		Eventually(func() error {
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(BeNil(), "persistent volume associated with DV should be created")
		ensurePVCBound(pvc)

		By("Making sure the DV is successful")
		Eventually(func() cdiv1.DataVolumePhase {
			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			return dv.Status.Phase
		}, 90*time.Second, 1*time.Second).Should(Equal(cdiv1.Succeeded))

		pod := createSourcePodChecker(pvc)

		fileName := filepath.Join(dataPath, diskImage)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileName = blockVolumeMountPath
		}
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, pod, pod.Spec.Containers[0].Name, md5Command(fileName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]

		err = virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: pointer.Int64(0),
		})
		Expect(err).ToNot(HaveOccurred())
		return pvc, md5sum
	}

	populateArchiveContent := func(sc string, volumeMode k8sv1.PersistentVolumeMode) (*k8sv1.PersistentVolumeClaim, string) {
		pvc, md5sum := populateKubeVirtContent(sc, volumeMode)
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		pvc.Annotations[annContentType] = "archive"
		pvc.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}
		pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		log.DefaultLogger().Infof("Calculated MD5 %s", md5sum)
		return pvc, md5sum
	}

	verifyKubeVirtRawContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(Equal(expectedMD5))
	}

	verifyKubeVirtGzContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		command := []string{
			"/usr/bin/gzip",
			"-d",
			filepath.Join(dataPath, fileName),
		}
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)

		fileName = strings.Replace(fileName, ".gz", "", 1)
		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err = tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
		Expect(md5sum).To(Equal(expectedMD5))
	}

	verifyArchiveGzContent := func(fileName, expectedMD5 string, downloadPod *k8sv1.Pod, volumeMode k8sv1.PersistentVolumeMode) {
		command := []string{
			"/usr/bin/tar",
			"-xzvf",
			filepath.Join(dataPath, fileName),
			"-C",
			dataPath,
		}
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), out, stderr)

		fileName = strings.ReplaceAll(fileName, ".tar.gz", ".img")
		fileAndPathName := filepath.Join(dataPath, fileName)
		if volumeMode == k8sv1.PersistentVolumeBlock {
			fileAndPathName = blockVolumeMountPath
		}
		out, stderr, err = tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, md5Command(fileAndPathName))
		Expect(err).ToNot(HaveOccurred(), out, stderr)
		md5sum := strings.Split(out, " ")[0]
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
				TokenSecretRef: token.Name,
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

		Eventually(func() bool {
			export, err = virtClient.VirtualMachineExport(pvc.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			condReady := false
			if export.Status != nil {
				for _, cond := range export.Status.Conditions {
					if cond.Type == exportv1.ConditionReady && cond.Status == k8sv1.ConditionTrue {
						condReady = true
					}
				}
			}
			return condReady
		}, 60*time.Second, 1*time.Second).Should(BeTrue(), "export is expected to become ready")

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
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, downloadPod, downloadPod.Spec.Containers[0].Name, command)
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
		Entry("with RAW kubevirt content type (proxy)", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW gzipped kubevirt content type (proxy)", populateKubeVirtContent, verifyKubeVirtGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive content type (proxy)", populateArchiveContent, verifyKubeVirtRawContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.Dir, archiveDircontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with archive tarred gzipped content type (proxy)", populateArchiveContent, verifyArchiveGzContent, libstorage.GetRWOFileSystemStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.ArchiveGz, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeFilesystem),
		Entry("with RAW kubevirt content type block (proxy)", populateKubeVirtContent, verifyKubeVirtRawContent, libstorage.GetRWOBlockStorageClass, createCaConfigMapProxy, urlGeneratorProxy, exportv1.KubeVirtRaw, kubevirtcontentUrlTemplate, k8sv1.PersistentVolumeBlock),
	)

	createExportObject := func(name, namespace string, token *k8sv1.Secret) *exportv1.VirtualMachineExport {
		vmExport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-export-%s", rand.String(12)),
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: token.Name,
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

	createRunningExport := func(sc string, volumeMode k8sv1.PersistentVolumeMode) *exportv1.VirtualMachineExport {
		pvc, _ := populateKubeVirtContent(sc, volumeMode)
		By("Creating the export token, we can export volumes using this token")
		// For testing the token is the name of the source pvc.
		token := createExportTokenSecret(pvc.Name, pvc.Namespace)

		export := createExportObject(pvc.Name, pvc.Namespace, token)

		Eventually(func() bool {
			export, err = virtClient.VirtualMachineExport(pvc.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			condReady := false
			if export.Status != nil {
				for _, cond := range export.Status.Conditions {
					if cond.Type == exportv1.ConditionReady && cond.Status == k8sv1.ConditionTrue {
						condReady = true
					}
				}
			}
			return condReady
		}, 30*time.Second, 1*time.Second).Should(BeTrue(), "export is expected to become ready")
		return export
	}

	It("Should recreate the exporter pod and secret if the pod fails", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningExport(sc, k8sv1.PersistentVolumeFilesystem)
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
		out, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, exporterPod, exporterPod.Spec.Containers[0].Name, command)
		Expect(err).ToNot(HaveOccurred(), "out[%s], err[%s]", out, stderr)
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
		vmExport := createRunningExport(sc, k8sv1.PersistentVolumeFilesystem)
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
		vmExport := createRunningExport(sc, k8sv1.PersistentVolumeFilesystem)
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
		dv := libstorage.NewRandomDataVolumeWithRegistryImportInStorageClass(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault, sc, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
		name := dv.Name
		namespace := dv.Namespace
		token := createExportTokenSecret(name, namespace)
		export := createExportObject(name, namespace, token)
		Eventually(func() string {
			export, err = virtClient.VirtualMachineExport(namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if export.Status != nil {
				for _, cond := range export.Status.Conditions {
					if cond.Type == exportv1.ConditionPVC {
						return cond.Reason
					}
				}
			}
			return ""
		}, 60*time.Second, 1*time.Second).Should(BeEquivalentTo(pvcNotFoundReason), "export should report missing pvc")
		dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
		var pvc *k8sv1.PersistentVolumeClaim
		Eventually(func() error {
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(BeNil(), "persistent volume associated with DV should be created")
		ensurePVCBound(pvc)

		By("Making sure the DV is successful")
		Eventually(func() cdiv1.DataVolumePhase {
			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			return dv.Status.Phase
		}, 90*time.Second, 1*time.Second).Should(Equal(cdiv1.Succeeded))
		By("Making sure the export becomes ready")
		Eventually(func() bool {
			export, err = virtClient.VirtualMachineExport(pvc.Namespace).Get(context.Background(), export.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			condReady := false
			if export.Status != nil {
				for _, cond := range export.Status.Conditions {
					if cond.Type == exportv1.ConditionReady && cond.Status == k8sv1.ConditionTrue {
						condReady = true
					}
				}
			}
			return condReady
		}, 60*time.Second, 1*time.Second).Should(BeTrue(), "export is expected to become ready")
	})

	It("should be possibe to observe exportserver pod exiting", func() {
		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Skip("Skip test when Filesystem storage is not present")
		}
		vmExport := createRunningExport(sc, k8sv1.PersistentVolumeFilesystem)
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
})
