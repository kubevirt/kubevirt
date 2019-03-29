package network

import (
	"os"
	"path/filepath"
	"reflect"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/render"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

func changeSafeLinuxBridge(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.LinuxBridge != nil && !reflect.DeepEqual(prev.LinuxBridge, next.LinuxBridge) {
		return []error{errors.Errorf("cannot modify Linux Bridge configuration once it is deployed")}
	}
	return nil
}

// renderLinuxBridge generates the manifests of Linux Bridge
func renderLinuxBridge(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, enableSCC bool) ([]*unstructured.Unstructured, error) {
	if conf.LinuxBridge == nil {
		return nil, nil
	}

	// render the manifests on disk
	data := render.MakeRenderData()
	data.Data["LinuxBridgeImage"] = os.Getenv("LINUX_BRIDGE_IMAGE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	data.Data["EnableSCC"] = enableSCC

	objs, err := render.RenderDir(filepath.Join(manifestDir, "linux-bridge"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render linux-bridge manifests")
	}

	return objs, nil
}
