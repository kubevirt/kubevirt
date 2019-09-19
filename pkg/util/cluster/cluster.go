package cluster

import (
	"encoding/json"
	"strconv"

	"kubevirt.io/client-go/kubecli"

	"k8s.io/apimachinery/pkg/api/errors"
)

const OpenShift3Major = 3

type OpenShiftVersion struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
}

func IsOnOpenShift3() (bool, error) {
	var osVersion OpenShiftVersion

	clientset, err := kubecli.GetKubevirtClient()
	if err != nil {
		return false, err
	}

	obj, err := clientset.CoreV1().RESTClient().Get().RequestURI("/version/openshift").Do().Raw()
	if err != nil {
		if errors.IsForbidden(err) || errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(obj, &osVersion); err != nil {
		return false, err
	}

	v, err := strconv.Atoi(osVersion.Major)
	if err != nil {
		return false, err
	}

	if v == OpenShift3Major {
		return true, nil
	}
	return false, nil
}
