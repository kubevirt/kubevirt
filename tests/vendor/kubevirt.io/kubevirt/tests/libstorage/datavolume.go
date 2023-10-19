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
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v13 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"
)

func AddDataVolumeDisk(vmi *v13.VirtualMachineInstance, diskName, dataVolumeName string) *v13.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v13.Disk{
		Name: diskName,
		DiskDevice: v13.DiskDevice{
			Disk: &v13.DiskTarget{
				Bus: v13.DiskBusVirtio,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v13.Volume{
		Name: diskName,
		VolumeSource: v13.VolumeSource{
			DataVolume: &v13.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})

	return vmi
}

func AddDataVolumeTemplate(vm *v13.VirtualMachine, dataVolume *v1beta1.DataVolume) {
	dvt := &v13.DataVolumeTemplateSpec{}

	dvt.Spec = *dataVolume.Spec.DeepCopy()
	dvt.ObjectMeta = *dataVolume.ObjectMeta.DeepCopy()

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
}

func AddDataVolume(vm *v13.VirtualMachine, diskName string, dataVolume *v1beta1.DataVolume) {
	vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v13.Disk{
		Name: diskName,
		DiskDevice: v13.DiskDevice{
			Disk: &v13.DiskTarget{
				Bus: v13.DiskBusVirtio,
			},
		},
	})
	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v13.Volume{
		Name: diskName,
		VolumeSource: v13.VolumeSource{
			DataVolume: &v13.DataVolumeSource{
				Name: dataVolume.Name,
			},
		},
	})
}

func EventuallyDV(dv *v1beta1.DataVolume, timeoutSec int, matcher gomegatypes.GomegaMatcher) {
	Expect(dv).ToNot(BeNil())
	EventuallyDVWith(dv.Namespace, dv.Name, timeoutSec, matcher)
}

func EventuallyDVWith(namespace, name string, timeoutSec int, matcher gomegatypes.GomegaMatcher) {
	virtCli := kubevirt.Client()

	if !IsDataVolumeGC(virtCli) {
		Eventually(ThisDVWith(namespace, name), timeoutSec, time.Second).Should(matcher)
		return
	}

	ginkgo.By("Verifying DataVolume garbage collection")
	var dv *v1beta1.DataVolume
	Eventually(func() *v1beta1.DataVolume {
		var err error
		dv, err = ThisDVWith(namespace, name)()
		Expect(err).ToNot(HaveOccurred())
		return dv
	}, timeoutSec, time.Second).Should(Or(BeNil(), matcher))

	if dv != nil {
		if dv.Status.Phase != v1beta1.Succeeded {
			return
		}
		if dv.Annotations["cdi.kubevirt.io/storage.deleteAfterCompletion"] == "true" {
			Eventually(ThisDV(dv), timeoutSec).Should(BeNil())
		}
	}

	Eventually(func() bool {
		pvc, err := ThisPVCWith(namespace, name)()
		Expect(err).ToNot(HaveOccurred())
		return pvc != nil && pvc.Spec.VolumeName != ""
	}, timeoutSec, time.Second).Should(BeTrue())
}

func DeleteDataVolume(dv **v1beta1.DataVolume) {
	Expect(dv).ToNot(BeNil())
	if *dv == nil {
		return
	}
	ginkgo.By("Deleting DataVolume")
	virtCli := kubevirt.Client()

	err := virtCli.CdiClient().CdiV1beta1().DataVolumes((*dv).Namespace).Delete(context.Background(), (*dv).Name, v12.DeleteOptions{})
	if !IsDataVolumeGC(virtCli) {
		Expect(err).ToNot(HaveOccurred())
		*dv = nil
		return
	}
	if err != nil {
		Expect(errors.IsNotFound(err)).To(BeTrue())
	}
	if err = virtCli.CoreV1().PersistentVolumeClaims((*dv).Namespace).Delete(context.Background(), (*dv).Name, v12.DeleteOptions{}); err != nil {
		Expect(errors.IsNotFound(err)).To(BeTrue())
	}
	*dv = nil
}

func SetDataVolumeGC(virtCli kubecli.KubevirtClient, ttlSec *int32) {
	cdi := GetCDI(virtCli)
	if cdi.Spec.Config.DataVolumeTTLSeconds == ttlSec {
		return
	}
	cdi.Spec.Config.DataVolumeTTLSeconds = ttlSec
	_, err := virtCli.CdiClient().CdiV1beta1().CDIs().Update(context.TODO(), cdi, v12.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() *int32 {
		config, err := virtCli.CdiClient().CdiV1beta1().CDIConfigs().Get(context.TODO(), "config", v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return config.Spec.DataVolumeTTLSeconds
	}, 10, time.Second).Should(Equal(ttlSec))
}

func IsDataVolumeGC(virtCli kubecli.KubevirtClient) bool {
	config, err := virtCli.CdiClient().CdiV1beta1().CDIConfigs().Get(context.TODO(), "config", v12.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return config.Spec.DataVolumeTTLSeconds != nil && *config.Spec.DataVolumeTTLSeconds >= 0
}

func GetCDI(virtCli kubecli.KubevirtClient) *v1beta1.CDI {
	cdiList, err := virtCli.CdiClient().CdiV1beta1().CDIs().List(context.Background(), v12.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(cdiList.Items).To(HaveLen(1))

	cdi := &cdiList.Items[0]
	cdi, err = virtCli.CdiClient().CdiV1beta1().CDIs().Get(context.TODO(), cdi.Name, v12.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return cdi
}

func HasDataVolumeCRD() bool {
	virtClient := kubevirt.Client()

	ext, err := clientset.NewForConfig(virtClient.Config())
	util.PanicOnError(err)

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "datavolumes.cdi.kubevirt.io", v12.GetOptions{})

	if err != nil {
		return false
	}
	return true
}

func HasCDI() bool {
	return HasDataVolumeCRD()
}

func GoldenImageRBAC(namespace string) (*rbacv1.Role, *rbacv1.RoleBinding) {
	name := "golden-rbac-" + rand.String(12)
	role := &rbacv1.Role{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"cdi.kubevirt.io",
				},
				Resources: []string{
					"datavolumes/source",
				},
				Verbs: []string{
					"create",
				},
			},
		},
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}
	return role, roleBinding
}
