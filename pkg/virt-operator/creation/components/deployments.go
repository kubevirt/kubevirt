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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package components

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func CreateControllers(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt, config util.KubeVirtDeploymentConfig, stores util.Stores, expectations *util.Expectations) (int, error) {

	objectsAdded := 0
	core := clientset.CoreV1()
	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return 0, err
	}

	services := []*corev1.Service{
		NewPrometheusService(kv.Namespace),
		NewApiServerService(kv.Namespace),
	}
	for _, service := range services {
		if _, exists, _ := stores.ServiceCache.Get(service); !exists {
			expectations.Service.RaiseExpectations(kvkey, 1, 0)
			_, err := core.Services(kv.Namespace).Create(service)
			if err != nil {
				expectations.Service.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create service %+v: %v", service, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("service %v already exists", service.GetName())
		}
	}

	apps := clientset.AppsV1()

	// TODO make verbosity part of the KubeVirt CRD spec?
	verbosity := "2"
	api, err := NewApiServerDeployment(kv.Namespace, config.ImageRegistry, config.ImageTag, kv.Spec.ImagePullPolicy, verbosity)
	if err != nil {
		return objectsAdded, err
	}
	controller, err := NewControllerDeployment(kv.Namespace, config.ImageRegistry, config.ImageTag, kv.Spec.ImagePullPolicy, verbosity)
	if err != nil {
		return objectsAdded, err
	}

	deployments := []*appsv1.Deployment{api, controller}
	for _, deployment := range deployments {
		if _, exists, _ := stores.DeploymentCache.Get(deployment); !exists {
			expectations.Deployment.RaiseExpectations(kvkey, 1, 0)
			_, err := apps.Deployments(kv.Namespace).Create(deployment)
			if err != nil {
				expectations.Deployment.LowerExpectations(kvkey, 1, 0)
				return objectsAdded, fmt.Errorf("unable to create deployment %+v: %v", deployment, err)
			} else if err == nil {
				objectsAdded++
			}
		} else {
			log.Log.V(4).Infof("deployment %v already exists", deployment.GetName())
		}
	}

	handler, err := NewHandlerDaemonSet(kv.Namespace, config.ImageRegistry, config.ImageTag, kv.Spec.ImagePullPolicy, verbosity)
	if err != nil {
		return objectsAdded, err
	}

	if _, exists, _ := stores.DaemonSetCache.Get(handler); !exists {
		expectations.DaemonSet.RaiseExpectations(kvkey, 1, 0)
		_, err = apps.DaemonSets(kv.Namespace).Create(handler)
		if err != nil {
			expectations.DaemonSet.LowerExpectations(kvkey, 1, 0)
			return objectsAdded, fmt.Errorf("unable to create daemonset %+v: %v", handler, err)
		} else if err == nil {
			objectsAdded++
		}
	} else {
		log.Log.V(4).Infof("daemonset %v already exists", handler.GetName())
	}

	return objectsAdded, nil

}

func NewPrometheusService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-prometheus-metrics",
			Labels: map[string]string{
				virtv1.AppLabel:          "",
				virtv1.ManagedByLabel:    virtv1.ManagedByLabelOperatorValue,
				"prometheus.kubevirt.io": "",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"prometheus.kubevirt.io": "",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "metrics",
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func NewApiServerService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "virt-api",
			Labels: map[string]string{
				virtv1.AppLabel:       "virt-api",
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				virtv1.AppLabel: "virt-api",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8443,
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func newPodTemplateSpec(name string, repository string, version string, pullPolicy corev1.PullPolicy) (*corev1.PodTemplateSpec, error) {

	tolerations := []corev1.Toleration{
		{
			Key:      "CriticalAddonsOnly",
			Operator: corev1.TolerationOpExists,
		},
	}
	tolerationsStr, err := json.Marshal(tolerations)

	if err != nil {
		return nil, fmt.Errorf("unable to create service: %v", err)
	}
	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				virtv1.AppLabel:          name,
				"prometheus.kubevirt.io": "",
			},
			Annotations: map[string]string{
				"scheduler.alpha.kubernetes.io/critical-pod": "",
				"scheduler.alpha.kubernetes.io/tolerations":  string(tolerationsStr),
			},
			Name: name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            name,
					Image:           fmt.Sprintf("%s/%s:%s", repository, name, version),
					ImagePullPolicy: pullPolicy,
				},
			},
		},
	}, nil
}

func newBaseDeployment(name string, namespace string, repository string, version string, pullPolicy corev1.PullPolicy) (*appsv1.Deployment, error) {

	podTemplateSpec, err := newPodTemplateSpec(name, repository, version, pullPolicy)
	if err != nil {
		return nil, err
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				virtv1.AppLabel:       name,
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io": name,
				},
			},
			Template: *podTemplateSpec,
		},
	}, nil
}

func NewApiServerDeployment(namespace string, repository string, version string, pullPolicy corev1.PullPolicy, verbosity string) (*appsv1.Deployment, error) {
	deployment, err := newBaseDeployment("virt-api", namespace, repository, version, pullPolicy)
	if err != nil {
		return nil, err
	}

	pod := &deployment.Spec.Template.Spec
	pod.ServiceAccountName = "kubevirt-apiserver"
	pod.SecurityContext = &corev1.PodSecurityContext{
		RunAsNonRoot: boolPtr(true),
	}

	container := &deployment.Spec.Template.Spec.Containers[0]
	container.Command = []string{
		"virt-api",
		"--port",
		"8443",
		"--subresources-only",
		"-v",
		verbosity,
	}
	container.Ports = []corev1.ContainerPort{
		{
			Name:          "virt-api",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
		{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
	}
	container.ReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
				Path: "/apis/subresources.kubevirt.io/" + virtv1.GroupVersion.Version + "/healthz",
			},
		},
		InitialDelaySeconds: 15,
		PeriodSeconds:       10,
	}
	return deployment, nil
}

func NewControllerDeployment(namespace string, repository string, version string, pullPolicy corev1.PullPolicy, verbosity string) (*appsv1.Deployment, error) {
	deployment, err := newBaseDeployment("virt-controller", namespace, repository, version, pullPolicy)
	if err != nil {
		return nil, err
	}

	pod := &deployment.Spec.Template.Spec
	pod.ServiceAccountName = "kubevirt-controller"
	pod.SecurityContext = &corev1.PodSecurityContext{
		RunAsNonRoot: boolPtr(true),
	}

	container := &deployment.Spec.Template.Spec.Containers[0]
	container.Command = []string{
		"virt-controller",
		"--launcher-image",
		fmt.Sprintf("%s/%s:%s", repository, "virt-launcher", version),
		"--port",
		"8443",
		"-v",
		verbosity,
	}
	container.Ports = []corev1.ContainerPort{
		{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
	}
	container.LivenessProbe = &corev1.Probe{
		FailureThreshold: 8,
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
				Path: "/healthz",
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      10,
	}
	container.ReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8443,
				},
				Path: "/leader",
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      10,
	}
	return deployment, nil
}

func NewHandlerDaemonSet(namespace string, repository string, version string, pullPolicy corev1.PullPolicy, verbosity string) (*appsv1.DaemonSet, error) {

	podTemplateSpec, err := newPodTemplateSpec("virt-handler", repository, version, pullPolicy)
	if err != nil {
		return nil, err
	}

	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "virt-handler",
			Labels: map[string]string{
				virtv1.AppLabel:       "virt-handler",
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io": "virt-handler",
				},
			},
			Template: *podTemplateSpec,
		},
	}

	pod := &daemonset.Spec.Template.Spec
	pod.ServiceAccountName = "kubevirt-handler"
	pod.HostPID = true

	container := &pod.Containers[0]
	container.Command = []string{
		"virt-handler",
		"--port",
		"8443",
		"--hostname-override",
		"$(NODE_NAME)",
		"--pod-ip-address",
		"$(MY_POD_IP)",
		"-v",
		verbosity,
	}
	container.Ports = []corev1.ContainerPort{
		{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: 8443,
		},
	}
	container.SecurityContext = &corev1.SecurityContext{
		Privileged: boolPtr(true),
	}
	container.Env = []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}

	container.VolumeMounts = []corev1.VolumeMount{}
	pod.Volumes = []corev1.Volume{}

	type volume struct {
		name string
		path string
	}

	volumes := []volume{
		{"libvirt-runtimes", "/var/run/kubevirt-libvirt-runtimes"},
		{"virt-share-dir", "/var/run/kubevirt"},
		{"virt-private-dir", "/var/run/kubevirt-private"},
		{"device-plugin", "/var/lib/kubelet/device-plugins"},
	}

	for _, volume := range volumes {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volume.name,
			MountPath: volume.path,
		})
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: volume.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volume.path,
				},
			},
		})
	}

	return daemonset, nil

}

func int32Ptr(i int32) *int32 {
	return &i
}
func boolPtr(b bool) *bool {
	return &b
}
