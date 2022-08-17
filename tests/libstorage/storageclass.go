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
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

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
	_, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), sc, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util.PanicOnError(err)
	}
}

func DeleteStorageClass(name string) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	_, err = virtClient.StorageV1().StorageClasses().Get(context.Background(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return
	}
	util.PanicOnError(err)

	err = virtClient.StorageV1().StorageClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
	util.PanicOnError(err)
}

func GetSnapshotStorageClass() (string, bool) {
	storageSnapshot := Config.StorageSnapshot
	return storageSnapshot, storageSnapshot != ""
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

func IsStorageClassBindingModeWaitForFirstConsumer(sc string) bool {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	storageClass, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), sc, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return storageClass.VolumeBindingMode != nil &&
		*storageClass.VolumeBindingMode == wffc
}

func CheckNoProvisionerStorageClassPVs(storageClassName string, numExpectedPVs int) {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
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
