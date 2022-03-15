package operands

import (
	"errors"
	"os"
	filepath "path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"

	log "github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

// Dashboard ConfigMaps contain json definitions of OCP UI
const (
	dashboardManifestLocationVarName = "DASHBOARD_FILES_LOCATION"
	dashboardManifestLocationDefault = "./dashboard"
)

func getDashboardHandlers(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	filesLocation := util.GetManifestDirPath(dashboardManifestLocationVarName, dashboardManifestLocationDefault)

	err := util.ValidateManifestDir(filesLocation)
	if err != nil {
		return nil, errors.Unwrap(err) // if not wrapped, then it's not an error that stops processing, and it return nil
	}

	return createDashboardHandlersFromFiles(logger, Client, Scheme, hc, filesLocation)
}

func createDashboardHandlersFromFiles(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged, filesLocation string) ([]Operand, error) {
	var handlers []Operand
	err := filepath.Walk(filesLocation, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		qs, err := processDashboardConfigMapFile(path, info, logger, hc, Client, Scheme)
		if err != nil {
			return err
		}

		if qs != nil {
			handlers = append(handlers, qs)
		}

		return nil
	})

	return handlers, err
}

func processDashboardConfigMapFile(path string, info os.FileInfo, logger log.Logger, hc *hcov1beta1.HyperConverged, Client client.Client, Scheme *runtime.Scheme) (Operand, error) {
	if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
		file, err := os.Open(path)
		if err != nil {
			logger.Error(err, "Can't open the dashboard yaml file", "file name", path)
			return nil, err
		}

		cm := &v1.ConfigMap{}
		err = util.UnmarshalYamlFileToObject(file, cm)
		if err != nil {
			logger.Error(err, "Can't generate a Configmap object from yaml file", "file name", path)
		} else {
			for k, v := range getLabels(hc, util.AppComponentCompute) {
				cm.Labels[k] = v
			}
			return newCmHandler(Client, Scheme, cm), nil
		}
	}
	return nil, nil
}
