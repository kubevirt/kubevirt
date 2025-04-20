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
 */

package matcher

import (
	"context"

	policyv1 "k8s.io/api/policy/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	k8sv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var getClient = kubevirt.Client

// ThisPod fetches the latest state of the pod. If the object does not exist, nil is returned.
func ThisPod(pod *v1.Pod) func() (*v1.Pod, error) {
	return ThisPodWith(pod.Namespace, pod.Name)
}

// ThisPodWith fetches the latest state of the pod based on namespace and name. If the object does not exist, nil is returned.
func ThisPodWith(namespace string, name string) func() (*v1.Pod, error) {
	return func() (*v1.Pod, error) {
		virtClient := getClient()
		p, err := virtClient.CoreV1().Pods(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		// Since https://github.com/kubernetes/client-go/issues/861 we manually add the Kind
		p.Kind = "Pod"
		return p, nil
	}
}

// ThisVMI fetches the latest state of the VirtualMachineInstance. If the object does not exist, nil is returned.
func ThisVMI(vmi *virtv1.VirtualMachineInstance) func() (*virtv1.VirtualMachineInstance, error) {
	return ThisVMIWith(vmi.Namespace, vmi.Name)
}

// ThisVMIWith fetches the latest state of the VirtualMachineInstance based on namespace and name. If the object does not exist, nil is returned.
func ThisVMIWith(namespace string, name string) func() (*virtv1.VirtualMachineInstance, error) {
	return func() (*virtv1.VirtualMachineInstance, error) {
		virtClient := getClient()
		p, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return p, nil
	}
}

// ThisVM fetches the latest state of the VirtualMachine. If the object does not exist, nil is returned.
func ThisVM(vm *virtv1.VirtualMachine) func() (*virtv1.VirtualMachine, error) {
	return ThisVMWith(vm.Namespace, vm.Name)
}

// ThisVMWith fetches the latest state of the VirtualMachine based on namespace and name. If the object does not exist, nil is returned.
func ThisVMWith(namespace string, name string) func() (*virtv1.VirtualMachine, error) {
	return func() (*virtv1.VirtualMachine, error) {
		virtClient := getClient()
		p, err := virtClient.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return p, nil
	}
}

// AllVMI fetches the latest state of all VMIs in a namespace.
func AllVMIs(namespace string) func() ([]virtv1.VirtualMachineInstance, error) {
	return func() (p []virtv1.VirtualMachineInstance, err error) {
		virtClient := getClient()
		list, err := virtClient.VirtualMachineInstance(namespace).List(context.Background(), k8smetav1.ListOptions{})
		return list.Items, err
	}
}

func AllPDBs(namespace string) func() ([]policyv1.PodDisruptionBudget, error) {
	return func() (p []policyv1.PodDisruptionBudget, err error) {
		virtClient := getClient()
		list, err := virtClient.PolicyV1().PodDisruptionBudgets(namespace).List(context.Background(), k8smetav1.ListOptions{})
		return list.Items, err
	}
}

// ThisDV fetches the latest state of the DataVolume. If the object does not exist, nil is returned.
func ThisDV(dv *v1beta1.DataVolume) func() (*v1beta1.DataVolume, error) {
	return ThisDVWith(dv.Namespace, dv.Name)
}

// ThisDVWith fetches the latest state of the DataVolume based on namespace and name. If the object does not exist, nil is returned.
func ThisDVWith(namespace string, name string) func() (*v1beta1.DataVolume, error) {
	return func() (*v1beta1.DataVolume, error) {
		virtClient := getClient()
		p, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		//Since https://github.com/kubernetes/client-go/issues/861 we manually add the Kind
		p.Kind = "DataVolume"
		return p, nil
	}
}

// ThisPVC fetches the latest state of the PVC. If the object does not exist, nil is returned.
func ThisPVC(pvc *v1.PersistentVolumeClaim) func() (*v1.PersistentVolumeClaim, error) {
	return ThisPVCWith(pvc.Namespace, pvc.Name)
}

// ThisPVCWith fetches the latest state of the PersistentVolumeClaim based on namespace and name. If the object does not exist, nil is returned.
func ThisPVCWith(namespace string, name string) func() (*v1.PersistentVolumeClaim, error) {
	return func() (*v1.PersistentVolumeClaim, error) {
		virtClient := getClient()
		p, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		//Since https://github.com/kubernetes/client-go/issues/861 we manually add the Kind
		p.Kind = "PersistentVolumeClaim"
		return p, nil
	}
}

// ThisMigration fetches the latest state of the Migration. If the object does not exist, nil is returned.
func ThisMigration(migration *virtv1.VirtualMachineInstanceMigration) func() (*virtv1.VirtualMachineInstanceMigration, error) {
	return ThisMigrationWith(migration.Namespace, migration.Name)
}

// ThisMigrationWith fetches the latest state of the Migration based on namespace and name. If the object does not exist, nil is returned.
func ThisMigrationWith(namespace string, name string) func() (*virtv1.VirtualMachineInstanceMigration, error) {
	return func() (*virtv1.VirtualMachineInstanceMigration, error) {
		virtClient := getClient()
		p, err := virtClient.VirtualMachineInstanceMigration(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return p, nil
	}
}

// ThisDeploymentWith fetches the latest state of the Deployment based on namespace and name. If the object does not exist, nil is returned.
func ThisDeploymentWith(namespace string, name string) func() (*k8sv1.Deployment, error) {
	return func() (*k8sv1.Deployment, error) {
		virtClient := getClient()
		p, err := virtClient.AppsV1().Deployments(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		//Since https://github.com/kubernetes/client-go/issues/861 we manually add the Kind
		p.Kind = "Deployment"
		return p, nil
	}
}
