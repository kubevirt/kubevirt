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

package libstorage

import (
	"context"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/util"
)

var wffc = storagev1.VolumeBindingWaitForFirstConsumer

func CreateStorageClass(name string, bindingMode *storagev1.VolumeBindingMode) {
	virtClient := kubevirt.Client()

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				kubevirtIoTest: name,
			},
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		VolumeBindingMode: bindingMode,
	}
	_, err := virtClient.StorageV1().StorageClasses().Create(context.Background(), sc, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func DeleteStorageClass(name string) {
	virtClient := kubevirt.Client()

	_, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return
	}
	util.PanicOnError(err)

	err = virtClient.StorageV1().StorageClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
	util.PanicOnError(err)
}

func GetSnapshotStorageClass(client kubecli.KubevirtClient) (string, error) {
	if Config != nil && Config.StorageSnapshot != "" {
		return Config.StorageSnapshot, nil
	}

	crd, err := client.
		ExtensionsClient().
		ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.Background(), "volumesnapshotclasses.snapshot.storage.k8s.io", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}

		return "", err
	}

	hasV1 := false
	for _, v := range crd.Spec.Versions {
		if v.Name == "v1" && v.Served {
			hasV1 = true
		}
	}

	if !hasV1 {
		return "", nil
	}

	volumeSnapshotClasses, err := client.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(volumeSnapshotClasses.Items) == 0 {
		return "", nil
	}
	defaultSnapClass := volumeSnapshotClasses.Items[0]
	for _, snapClass := range volumeSnapshotClasses.Items {
		if snapClass.Annotations["snapshot.storage.kubernetes.io/is-default-class"] == "true" {
			defaultSnapClass = snapClass
		}
	}

	storageClasses, err := client.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	var storageClass string

	for _, sc := range storageClasses.Items {
		if sc.Provisioner == defaultSnapClass.Driver {
			storageClass = sc.Name
			break
		}
	}

	if Config != nil {
		Config.StorageSnapshot = storageClass
	}

	return storageClass, nil
}

func GetSnapshotClass(scName string, client kubecli.KubevirtClient) (string, error) {
	crd, err := client.
		ExtensionsClient().
		ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.Background(), "volumesnapshotclasses.snapshot.storage.k8s.io", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}

		return "", err
	}

	hasV1 := false
	for _, v := range crd.Spec.Versions {
		if v.Name == "v1" && v.Served {
			hasV1 = true
		}
	}

	if !hasV1 {
		return "", nil
	}

	volumeSnapshotClasses, err := client.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(volumeSnapshotClasses.Items) == 0 {
		return "", nil
	}
	sc, err := client.StorageV1().StorageClasses().Get(context.Background(), scName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, snapClass := range volumeSnapshotClasses.Items {
		// Validate association between snapshot class and storage class
		if snapClass.Driver == sc.Provisioner {
			return snapClass.Name, nil
		}
	}

	return "", nil
}

func GetRWXFileSystemStorageClass() (string, bool) {
	storageRWXFileSystem := Config.StorageRWXFileSystem
	return storageRWXFileSystem, storageRWXFileSystem != ""
}

func GetRWOFileSystemStorageClass() (string, bool) {
	storageRWOFileSystem := Config.StorageRWOFileSystem
	return storageRWOFileSystem, storageRWOFileSystem != ""
}

func GetRWOBlockStorageClass() (string, bool) {
	storageRWOBlock := Config.StorageRWOBlock
	return storageRWOBlock, storageRWOBlock != ""
}

func GetRWXBlockStorageClass() (string, bool) {
	storageRWXBlock := Config.StorageRWXBlock
	return storageRWXBlock, storageRWXBlock != ""
}

func GetBlockStorageClass(accessMode k8sv1.PersistentVolumeAccessMode) (string, bool) {
	sc, foundSC := GetRWOBlockStorageClass()
	if accessMode == k8sv1.ReadWriteMany {
		sc, foundSC = GetRWXBlockStorageClass()
	}

	return sc, foundSC
}

// GetNoVolumeSnapshotStorageClass goes over all the existing storage classes
// and returns one which doesnt have volume snapshot ability
// if the preference storage class exists and is without snapshot
// ability it will be returned
func GetNoVolumeSnapshotStorageClass(preference string) string {
	virtClient := kubevirt.Client()
	scs, err := virtClient.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	vscs, err := virtClient.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshotClasses().List(context.Background(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return ""
	}
	Expect(err).ToNot(HaveOccurred())
	vscsDrivers := make(map[string]bool)
	for _, vsc := range vscs.Items {
		vscsDrivers[vsc.Driver] = true
	}

	candidate := ""
	for _, sc := range scs.Items {
		if _, ok := vscsDrivers[sc.Provisioner]; !ok {
			if sc.Name == preference {
				return sc.Name
			}
			if candidate == "" {
				candidate = sc.Name
			}
		}
	}

	return candidate
}

func IsStorageClassBindingModeWaitForFirstConsumer(sc string) bool {
	virtClient := kubevirt.Client()
	storageClass, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), sc, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return storageClass.VolumeBindingMode != nil &&
		*storageClass.VolumeBindingMode == wffc
}

func CheckNoProvisionerStorageClassPVs(storageClassName string, numExpectedPVs int) {
	virtClient := kubevirt.Client()
	sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), storageClassName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	if sc.Provisioner != "" && sc.Provisioner != "kubernetes.io/no-provisioner" {
		return
	}

	// Verify we have at least `numExpectedPVs` available file system PVs
	pvList, err := virtClient.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	if countLocalStoragePVAvailableForUse(pvList, storageClassName) < numExpectedPVs {
		Skip("Not enough available filesystem local storage PVs available, expected: %d", numExpectedPVs)
	}
}

func countLocalStoragePVAvailableForUse(pvList *k8sv1.PersistentVolumeList, storageClassName string) int {
	count := 0
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == storageClassName && isLocalPV(pv) && isPVAvailable(pv) {
			count++
		}
	}
	return count
}

func isLocalPV(pv k8sv1.PersistentVolume) bool {
	return pv.Spec.NodeAffinity != nil &&
		pv.Spec.NodeAffinity.Required != nil &&
		len(pv.Spec.NodeAffinity.Required.NodeSelectorTerms) > 0 &&
		(pv.Spec.VolumeMode == nil || *pv.Spec.VolumeMode != k8sv1.PersistentVolumeBlock)
}

func isPVAvailable(pv k8sv1.PersistentVolume) bool {
	return pv.Spec.ClaimRef == nil
}
