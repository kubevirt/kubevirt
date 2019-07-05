package releases

import (
	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

func init() {
	release := Release{
		Version: "0.12.0",
		Containers: []opv1alpha1.Container{
			opv1alpha1.Container{
				Namespace:  "multus",
				ParentName: "kube-multus-ds-amd64",
				ParentKind: "DaemonSet",
				Name:       "kube-multus",
				Image:      "quay.io/kubevirt/cluster-network-addon-multus:v3.2.0-1.gitbf61002",
			},
			opv1alpha1.Container{
				Namespace:  "linux-bridge",
				ParentName: "bridge-marker",
				ParentKind: "DaemonSet",
				Name:       "bridge-marker",
				Image:      "quay.io/kubevirt/bridge-marker:0.1.0",
			},
			opv1alpha1.Container{
				Namespace:  "linux-bridge",
				ParentName: "kube-cni-linux-bridge-plugin",
				ParentKind: "DaemonSet",
				Name:       "cni-plugins",
				Image:      "quay.io/kubevirt/cni-default-plugins:v0.8.0",
			},
			opv1alpha1.Container{
				Namespace:  "kubemacpool-system",
				ParentName: "kubemacpool-mac-controller-manager",
				ParentKind: "Deployment",
				Name:       "manager",
				Image:      "quay.io/kubevirt/kubemacpool:v0.3.0",
			},
		},
		SupportedSpec: opv1alpha1.NetworkAddonsConfigSpec{
			KubeMacPool: &opv1alpha1.KubeMacPool{},
			LinuxBridge: &opv1alpha1.LinuxBridge{},
			Multus:      &opv1alpha1.Multus{},
		},
		Manifests: []string{
			"operator.yaml",
		},
	}
	releases = append(releases, release)
}
