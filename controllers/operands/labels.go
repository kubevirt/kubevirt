package operands

import (
	"encoding/json"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/patch"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func getLabels(hc *hcov1beta1.HyperConverged, component hcoutil.AppComponent) map[string]string {
	hcoName := hcov1beta1.HyperConvergedName

	if hc.Name != "" {
		hcoName = hc.Name
	}

	return hcoutil.GetLabels(hcoName, component)
}

func getLabelPatch(dest, src map[string]string) ([]byte, error) {
	const labelPath = "/metadata/labels/"
	var patches []patch.JSONPatchAction

	for k, v := range src {
		op := "replace"
		lbl, ok := dest[k]

		if !ok {
			op = "add"
		} else if lbl == v {
			continue
		}

		patches = append(patches, patch.JSONPatchAction{
			Op:    op,
			Path:  labelPath + patch.EscapeJSONPointer(k),
			Value: v,
		})
	}

	return json.Marshal(patches)
}
