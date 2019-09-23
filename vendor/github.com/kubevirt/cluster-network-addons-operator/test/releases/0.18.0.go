package releases

import (
	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

func init() {
	release := Release{
		Version: "0.18.0",
		Containers: []opv1alpha1.Container{
			opv1alpha1.Container{
				ParentName: "kube-multus-ds-amd64",
				ParentKind: "DaemonSet",
				Name:       "kube-multus",
				Image:      "quay.io/kubevirt/cluster-network-addon-multus:v3.2.0-1.gitbf61002",
			},
			opv1alpha1.Container{
				ParentName: "bridge-marker",
				ParentKind: "DaemonSet",
				Name:       "bridge-marker",
				Image:      "quay.io/kubevirt/bridge-marker:0.2.0",
			},
			opv1alpha1.Container{
				ParentName: "kube-cni-linux-bridge-plugin",
				ParentKind: "DaemonSet",
				Name:       "cni-plugins",
				Image:      "quay.io/kubevirt/cni-default-plugins:v0.8.1",
			},
			opv1alpha1.Container{
				ParentName: "kubemacpool-mac-controller-manager",
				ParentKind: "Deployment",
				Name:       "manager",
				Image:      "quay.io/kubevirt/kubemacpool:v0.7.0",
			},
			opv1alpha1.Container{
				ParentName: "nmstate-handler",
				ParentKind: "DaemonSet",
				Name:       "nmstate-handler",
				Image:      "quay.io/nmstate/kubernetes-nmstate-handler:v0.8.0",
			},
			opv1alpha1.Container{
				ParentName: "ovs-cni-amd64",
				ParentKind: "DaemonSet",
				Name:       "ovs-cni-plugin",
				Image:      "quay.io/kubevirt/ovs-cni-plugin:v0.7.0",
			},
			opv1alpha1.Container{
				ParentName: "ovs-cni-amd64",
				ParentKind: "DaemonSet",
				Name:       "ovs-cni-marker",
				Image:      "quay.io/kubevirt/ovs-cni-marker:v0.7.0",
			},
		},
		SupportedSpec: opv1alpha1.NetworkAddonsConfigSpec{
			KubeMacPool: &opv1alpha1.KubeMacPool{},
			LinuxBridge: &opv1alpha1.LinuxBridge{},
			Multus:      &opv1alpha1.Multus{},
			NMState:     &opv1alpha1.NMState{},
			Ovs:         &opv1alpha1.Ovs{},
		},
		Manifests: []string{
			"operator.yaml",
		},
	}
	releases = append(releases, release)
}
