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

package libstorage

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
)

func AddDataVolumeTemplate(vm *v1.VirtualMachine, dataVolume *v1beta1.DataVolume) {
	dvt := &v1.DataVolumeTemplateSpec{}

	dvt.Spec = *dataVolume.Spec.DeepCopy()
	dvt.ObjectMeta = *dataVolume.ObjectMeta.DeepCopy()

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
}

func AddDataVolume(vm *v1.VirtualMachine, diskName string, dataVolume *v1beta1.DataVolume) {
	vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusVirtio,
			},
		},
	})
	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
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
	Eventually(ThisDVWith(namespace, name), timeoutSec, time.Second).Should(matcher)
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

func HasCDI() bool {
	virtClient := kubevirt.Client()

	ext, err := clientset.NewForConfig(virtClient.Config())
	Expect(err).ToNot(HaveOccurred())

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "datavolumes.cdi.kubevirt.io", v12.GetOptions{})
	return err == nil
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

func RenderVMIWithDataVolume(dvName, ns string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	defaultOptions := []libvmi.Option{
		libvmi.WithDataVolume("disk0", dvName),
		// This default can be optimized further to 128Mi on certain setups
		libvmi.WithResourceMemory("256Mi"),
		libvmi.WithNamespace(ns),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	}
	return libvmi.New(append(defaultOptions, opts...)...)
}

func RenderVMWithDataVolumeTemplate(dv *v1beta1.DataVolume, opts ...libvmi.VMOption) *v1.VirtualMachine {
	defaultOptions := []libvmi.VMOption{libvmi.WithDataVolumeTemplate(dv)}
	return libvmi.NewVirtualMachine(
		RenderVMIWithDataVolume(dv.Name, dv.Namespace),
		append(defaultOptions, opts...)...,
	)
}
