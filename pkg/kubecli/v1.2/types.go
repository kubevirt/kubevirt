package v1_2

import "time"

type Container struct {
	Name                   string   `json:"name"`
	Image                  string   `json:"image"`
	Command                []string `json:"command"`
	TerminationMessagePath string   `json:"terminationMessagePath"`
	ImagePullPolicy        string   `json:"imagePullPolicy"`
}

type ContainerStatus struct {
	Name         string `json:"name"`
	GenerateName string `json:"generateName"`
	// State        map[string]map[string]*time.Time `json:"state"`
	LastState    map[string]interface{} `json:"lastState"`
	Ready        bool                   `json:"ready"`
	RestartCount int                    `json:"restartCount"`
	Image        string                 `json:"image"`
	ImageID      string                 `json:"imageID"`
	ContainerID  string                 `json:"containerID"`
}

type Condition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	// LastProbeTime      *time.Time `json:"lastProbeTime"`
	// LastTransitionTime *time.Time `json:"lastTransitionTime"`
}

type Metadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	SelfLink          string            `json:"selfLink"`
	UID               string            `json:"uid"`
	ResourceVersion   string            `json:"resourceVersion"`
	CreationTimestamp *time.Time        `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
}

type Status struct {
	Phase      string      `json:"phase"`
	Conditions []Condition `json:"conditions"`
	HostIP     string      `json:"hostIP"`
	PodIP      string      `json:"podIP"`
	// StartTime         *time.Time        `json:"startTime"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses"`
}

type Spec struct {
	Containers                    []Container `json:"containers"`
	RestartPolicy                 string      `json:"restartPolicy"`
	TerminationGracePeriodSeconds int         `json:"terminationGracePeriodSeconds"`
	DNSPolicy                     string      `json:"dnsPolicy"`
	Host                          string      `json:"host"`
	NodeName                      string      `json:"nodeName"`
	HostNetwork                   bool        `json:"hostNetwork"`
}

type Pod struct {
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
	Status     Status   `json:"status"`
}

type PodList struct {
	Kind  string `json:"kind"`
	Items []Pod  `json:"items"`
}
