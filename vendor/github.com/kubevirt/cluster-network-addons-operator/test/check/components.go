package check

type Component struct {
	ComponentName              string
	Namespace                  string
	ClusterRole                string
	ClusterRoleBinding         string
	SecurityContextConstraints string
	DaemonSets                 []string
	Deployments                []string
}

var (
	KubeMacPoolComponent = Component{
		ComponentName:      "KubeMacPool",
		Namespace:          "kubemacpool-system",
		ClusterRole:        "kubemacpool-manager-role",
		ClusterRoleBinding: "kubemacpool-manager-rolebinding",
		Deployments:        []string{"kubemacpool-mac-controller-manager"},
	}
	LinuxBridgeComponent = Component{
		ComponentName:              "Linux Bridge",
		Namespace:                  "linux-bridge",
		ClusterRole:                "bridge-marker-cr",
		ClusterRoleBinding:         "bridge-marker-crb",
		SecurityContextConstraints: "linux-bridge",
		DaemonSets: []string{
			"bridge-marker",
			"kube-cni-linux-bridge-plugin",
		},
	}
	MultusComponent = Component{
		ComponentName:              "Multus",
		Namespace:                  "multus",
		ClusterRole:                "multus",
		ClusterRoleBinding:         "multus",
		SecurityContextConstraints: "multus",
		DaemonSets:                 []string{"kube-multus-ds-amd64"},
	}
	NMStateComponent = Component{
		ComponentName:              "NMState",
		Namespace:                  "nmstate",
		ClusterRoleBinding:         "nmstate-handler",
		ClusterRole:                "nmstate-handler",
		SecurityContextConstraints: "nmstate",
		DaemonSets:                 []string{"nmstate-handler"},
	}

	OvsComponent = Component{
		ComponentName:              "Ovs",
		Namespace:                  "ovs",
		ClusterRole:                "ovs-cni-marker-cr",
		ClusterRoleBinding:         "ovs-cni-marker-crb",
		SecurityContextConstraints: "ovs-cni-marker",
		DaemonSets: []string{
			"ovs-cni-amd64",
		},
	}
	AllComponents = []Component{
		KubeMacPoolComponent,
		LinuxBridgeComponent,
		MultusComponent,
		NMStateComponent,
		OvsComponent,
	}
)
