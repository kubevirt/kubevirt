package network

import (
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/render"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

// ValidateMultus validates the combination of DisableMultiNetwork and AddtionalNetworks
func validateKubeMacPool(conf *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if conf.KubeMacPool == nil {
		return []error{}
	}

	// If the range is not configured by the administrator we generate a random range.
	// This random range spans from 02:XX:XX:00:00:00 to 02:XX:XX:FF:FF:FF,
	// where 02 makes the address local unicast and XX:XX is a random prefix.
	if conf.KubeMacPool.StartPoolRange == "" && conf.KubeMacPool.EndPoolRange == "" {
		return []error{}
	}

	if (conf.KubeMacPool.StartPoolRange == "" && conf.KubeMacPool.EndPoolRange != "") ||
		(conf.KubeMacPool.StartPoolRange != "" && conf.KubeMacPool.EndPoolRange == "") {
		return []error{errors.Errorf("both or none of the KubeMacPool ranges needs to be configured")}
	}

	if _, err := net.ParseMAC(conf.KubeMacPool.StartPoolRange); err != nil {
		return []error{errors.Errorf("failed to parse startPoolRange invalid mac address")}
	}

	if _, err := net.ParseMAC(conf.KubeMacPool.EndPoolRange); err != nil {
		return []error{errors.Errorf("failed to parse endPoolRange invalid mac address")}
	}

	return []error{}
}

func changeSafeKubeMacPool(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.KubeMacPool != nil && !reflect.DeepEqual(prev.KubeMacPool, next.KubeMacPool) {
		return []error{errors.Errorf("cannot modify KubeMacPool configuration once it is deployed")}
	}
	return nil
}

// renderLinuxBridge generates the manifests of Linux Bridge
func renderKubeMacPool(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string) ([]*unstructured.Unstructured, error) {
	if conf.KubeMacPool == nil {
		return nil, nil
	}

	if conf.KubeMacPool.StartPoolRange == "" || conf.KubeMacPool.EndPoolRange == "" {
		prefix, err := generateRandomMacPrefix()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate random mac address prefix")
		}

		startPoolRange := net.HardwareAddr(append(prefix, 0x00, 0x00, 0x00))
		conf.KubeMacPool.StartPoolRange = startPoolRange.String()

		endPoolRange := net.HardwareAddr(append(prefix, 0xFF, 0xFF, 0xFF))
		conf.KubeMacPool.EndPoolRange = endPoolRange.String()
	}

	// render the manifests on disk
	data := render.MakeRenderData()
	data.Data["KubeMacPoolImage"] = os.Getenv("KUBEMACPOOL_IMAGE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	data.Data["StartPoolRange"] = conf.KubeMacPool.StartPoolRange
	data.Data["EndPoolRange"] = conf.KubeMacPool.EndPoolRange

	objs, err := render.RenderDir(filepath.Join(manifestDir, "kubemacpool"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render kubemacpool manifests")
	}

	return objs, nil
}

func generateRandomMacPrefix() ([]byte, error) {
	suffix := make([]byte, 2)
	_, err := rand.Read(suffix)
	if err != nil {
		return []byte{}, err
	}

	prefix := append([]byte{0x02}, suffix...)

	return prefix, nil
}
