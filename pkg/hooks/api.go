package hooks

import (
	"encoding/json"

	k8sv1 "k8s.io/api/core/v1"
)

const HookSidecarListAnnotationName = "hookSidecars"
const HookSocketsSharedDirectory = "/var/run/kubevirt-hooks"

type HookSidecarList []HookSidecar

type HookSidecar struct {
	Image           string           `json:"image"`
	ImagePullPolicy k8sv1.PullPolicy `json:"imagePullPolicy"`
}

func UnmarshalHookSidecarList(annotation string) (HookSidecarList, error) {
	var hookSidecarList HookSidecarList
	if err := json.Unmarshal([]byte(annotation), &hookSidecarList); err != nil {
		return nil, err
	}
	return hookSidecarList, nil
}
