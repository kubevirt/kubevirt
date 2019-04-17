package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/render"
	osv1 "github.com/openshift/api/operator/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/network/cni"
)

// code below is copied from openshift/cluster-network-operator:pkg/network/sriov.go
type NetConfSRIOV struct {
	// ...
	Type string `json:"type"`
	// ...
}

func isOpenShiftSRIOV(conf *osv1.AdditionalNetworkDefinition) bool {
	cni := NetConfSRIOV{}
	err := json.Unmarshal([]byte(conf.RawCNIConfig), &cni)
	if err != nil {
		// log.Printf("WARNING: Could not determine if network %q is SR-IOV: %v", conf.Name, err)
		return false
	}
	return cni.Type == "sriov"
} // end of copied code

func isOpenShiftSRIOVEnabled(openshiftNetworkConfig *osv1.Network) bool {
	if openshiftNetworkConfig == nil {
		return false
	}
	for _, network := range openshiftNetworkConfig.Spec.AdditionalNetworks {
		if isOpenShiftSRIOV(&network) {
			return true
		}
	}
	return false
}

func validateSriov(conf *opv1alpha1.NetworkAddonsConfigSpec, openshiftNetworkConfig *osv1.Network) []error {
	if conf.Sriov == nil {
		return []error{}
	}

	if isOpenShiftSRIOVEnabled(openshiftNetworkConfig) {
		return []error{errors.Errorf("SR-IOV has been requested, but it's not compatible with " +
			"OpenShift Cluster Network Operator SR-IOV support")}
	}

	return []error{}
}

func changeSafeSriov(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.Sriov != nil && !reflect.DeepEqual(prev.Sriov, next.Sriov) {
		return []error{errors.Errorf("cannot modify Sriov configuration once it is deployed")}
	}
	return nil
}

func getRootDevicesConfigString(rootDevices string) string {
	devices := make([]string, 0)
	for _, id := range strings.Split(rootDevices, ",") {
		if id != "" {
			devices = append(devices, fmt.Sprintf("\"%s\"", id))
		}
	}
	return strings.Join(devices, ",")
}

// renderSriov generates the manifests of SR-IOV plugins
func renderSriov(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, clusterInfo *ClusterInfo) ([]*unstructured.Unstructured, error) {
	if conf.Sriov == nil {
		return nil, nil
	}

	// render the manifests on disk
	data := render.MakeRenderData()
	data.Data["SriovRootDevices"] = getRootDevicesConfigString(os.Getenv("SRIOV_ROOT_DEVICES"))
	data.Data["SriovDpImage"] = os.Getenv("SRIOV_DP_IMAGE")
	data.Data["SriovCniImage"] = os.Getenv("SRIOV_CNI_IMAGE")
	data.Data["SriovNetworkName"] = os.Getenv("SRIOV_NETWORK_NAME")
	data.Data["SriovNetworkType"] = os.Getenv("SRIOV_NETWORK_TYPE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	if clusterInfo.OpenShift4 {
		data.Data["CNIBinDir"] = cni.BinDirOpenShift4
	} else {
		data.Data["CNIBinDir"] = cni.BinDir
	}
	data.Data["EnableSCC"] = clusterInfo.SCCAvailable

	objs, err := render.RenderDir(filepath.Join(manifestDir, "sriov"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render sriov manifests")
	}

	return objs, nil
}
