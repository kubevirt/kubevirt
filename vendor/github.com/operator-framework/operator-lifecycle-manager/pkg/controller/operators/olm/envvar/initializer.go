package envvar

import (
	"fmt"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorlister"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/proxy"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// NewDeploymentInitializer returns a function that accepts a Deployment object
// and initializes it with env variables specified in operator configuration.
func NewDeploymentInitializer(logger *logrus.Logger, querier proxy.Querier, lister operatorlister.OperatorLister) *DeploymentInitializer {
	return &DeploymentInitializer{
		logger:  logger,
		querier: querier,
		config: &operatorConfig{
			lister: lister,
			logger: logger,
		},
	}
}

type DeploymentInitializer struct {
	logger  *logrus.Logger
	querier proxy.Querier
	config  *operatorConfig
}

func (d *DeploymentInitializer) GetDeploymentInitializer(ownerCSV ownerutil.Owner) install.DeploymentInitializerFunc {
	return func(spec *appsv1.Deployment) error {
		err := d.initialize(ownerCSV, spec)
		return err
	}
}

// Initialize initializes a deployment object with appropriate global cluster
// level proxy env variable(s).
func (d *DeploymentInitializer) initialize(ownerCSV ownerutil.Owner, deployment *appsv1.Deployment) error {
	var podConfigEnvVar, proxyEnvVar, merged []corev1.EnvVar
	var err error

	podConfigEnvVar, err = d.config.GetOperatorConfig(ownerCSV)
	if err != nil {
		err = fmt.Errorf("failed to get subscription pod configuration - %v", err)
		return err
	}

	if !proxy.IsOverridden(podConfigEnvVar) {
		proxyEnvVar, err = d.querier.QueryProxyConfig()
		if err != nil {
			err = fmt.Errorf("failed to query cluster proxy configuration - %v", err)
			return err
		}
	}

	merged = append(podConfigEnvVar, proxyEnvVar...)

	if len(merged) == 0 {
		d.logger.Debugf("no env var to inject into csv=%s", ownerCSV.GetName())
	}

	podSpec := deployment.Spec.Template.Spec
	if err := InjectEnvIntoDeployment(&podSpec, merged); err != nil {
		return fmt.Errorf("failed to inject proxy env variable(s) into deployment spec name=%s - %v", deployment.Name, err)
	}

	return nil
}
