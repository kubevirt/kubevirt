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

func changeSafeNMState(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.NMState != nil && !reflect.DeepEqual(prev.NMState, next.NMState) {
		return []error{errors.Errorf("cannot modify NMState state handler configuration once it is deployed")}
	}
	return nil
}

// renderNMState generates the manifests of NMState handler
func renderNMState(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, clusterInfo *ClusterInfo) ([]*unstructured.Unstructured, error) {
	if conf.NMState == nil {
		return nil, nil
	}

	// render the manifests on disk
	data := render.MakeRenderData()
	data.Data["NMStateStateHandlerImage"] = os.Getenv("NMSTATE_STATE_HANDLER_IMAGE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	data.Data["EnableSCC"] = clusterInfo.SCCAvailable

	objs, err := render.RenderDir(filepath.Join(manifestDir, "nmstate"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render nmstate state handler manifests")
	}

	return objs, nil
}
