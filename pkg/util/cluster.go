package util

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync/atomic"

	"github.com/go-logr/logr"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
)

type ClusterInfo interface {
	Init(ctx context.Context, cl client.Client, logger logr.Logger) error
	IsOpenshift() bool
	IsRunningLocally() bool
	GetBaseDomain() string
	IsManagedByOLM() bool
	IsControlPlaneHighlyAvailable() bool
	IsControlPlaneNodeExists() bool
	IsInfrastructureHighlyAvailable() bool
	SetHighAvailabilityMode(ctx context.Context, cl client.Client) error
	IsConsolePluginImageProvided() bool
	IsMonitoringAvailable() bool
	IsDeschedulerAvailable() bool
	IsDeschedulerCRDDeployed(ctx context.Context, cl client.Client) bool
	IsSingleStackIPv6() bool
	GetTLSSecurityProfile(hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile
	RefreshAPIServerCR(ctx context.Context, c client.Client) error
	GetPod() *corev1.Pod
	GetDeployment() *appsv1.Deployment
	GetCSV() *csvv1alpha1.ClusterServiceVersion
}

type ClusterInfoImp struct {
	runningInOpenshift            bool
	managedByOLM                  bool
	runningLocally                bool
	controlPlaneHighlyAvailable   atomic.Bool
	controlPlaneNodeExist         atomic.Bool
	infrastructureHighlyAvailable atomic.Bool
	consolePluginImageProvided    bool
	monitoringAvailable           bool
	deschedulerAvailable          bool
	singlestackipv6               bool
	baseDomain                    string
	ownResources                  *OwnResources
	logger                        logr.Logger
}

var clusterInfo ClusterInfo

var validatedAPIServerTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile

var GetClusterInfo = func() ClusterInfo {
	return clusterInfo
}

// OperatorConditionNameEnvVar - this Env var is set by OLM, so the Operator can discover it's OperatorCondition.
const OperatorConditionNameEnvVar = "OPERATOR_CONDITION_NAME"

func (c *ClusterInfoImp) Init(ctx context.Context, cl client.Client, logger logr.Logger) error {
	c.logger = logger
	err := c.queryCluster(ctx, cl)
	if err != nil {
		return err
	}

	// We assume that this Operator is managed by OLM when this variable is present.
	_, c.managedByOLM = os.LookupEnv(OperatorConditionNameEnvVar)

	if c.runningInOpenshift {
		err = c.initOpenshift(ctx, cl)
	} else {
		err = c.initKubernetes(ctx, cl)
	}
	if err != nil {
		return err
	}
	if c.runningInOpenshift && c.singlestackipv6 {
		metrics.SetHCOMetricSingleStackIPv6True()
	}

	uiPluginVarValue, uiPluginVarExists := os.LookupEnv(KVUIPluginImageEnvV)
	uiProxyVarValue, uiProxyVarExists := os.LookupEnv(KVUIProxyImageEnvV)
	c.consolePluginImageProvided = uiPluginVarExists && len(uiPluginVarValue) > 0 && uiProxyVarExists && len(uiProxyVarValue) > 0

	c.monitoringAvailable = isPrometheusExists(ctx, cl)
	c.deschedulerAvailable = isDeschedulerExists(ctx, cl)
	c.logger.Info("addOns ",
		"monitoring", c.monitoringAvailable,
		"kubeDescheduler", c.deschedulerAvailable,
	)

	err = c.RefreshAPIServerCR(ctx, cl)
	if err != nil {
		return err
	}

	c.ownResources = findOwnResources(ctx, cl, c.logger)
	return nil
}

func (c *ClusterInfoImp) initKubernetes(ctx context.Context, cl client.Client) error {
	return c.SetHighAvailabilityMode(ctx, cl)
}

func (c *ClusterInfoImp) initOpenshift(ctx context.Context, cl client.Client) error {
	clusterInfrastructure := &openshiftconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	err := cl.Get(ctx, client.ObjectKeyFromObject(clusterInfrastructure), clusterInfrastructure)
	if err != nil {
		return err
	}

	c.logger.Info("Cluster Infrastructure",
		"platform", clusterInfrastructure.Status.PlatformStatus.Type,
		"controlPlaneTopology", clusterInfrastructure.Status.ControlPlaneTopology,
		"infrastructureTopology", clusterInfrastructure.Status.InfrastructureTopology,
	)

	err = c.SetHighAvailabilityMode(ctx, cl)
	if err != nil {
		return err
	}

	clusterNetwork := &openshiftconfigv1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	err = cl.Get(ctx, client.ObjectKeyFromObject(clusterNetwork), clusterNetwork)
	if err != nil {
		return err
	}
	cn := clusterNetwork.Status.ClusterNetwork
	for _, i := range cn {
		c.logger.Info("Cluster Network",
			"CIDR", i.CIDR,
			"Host Prefix", i.HostPrefix,
		)
	}
	c.singlestackipv6 = len(cn) == 1 && net.IsIPv6CIDRString(cn[0].CIDR)
	return nil
}

func (c *ClusterInfoImp) IsManagedByOLM() bool {
	return c.managedByOLM
}

func (c *ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}

func (c *ClusterInfoImp) IsConsolePluginImageProvided() bool {
	return c.consolePluginImageProvided
}

func (c *ClusterInfoImp) IsMonitoringAvailable() bool {
	return c.monitoringAvailable
}

func (c *ClusterInfoImp) IsDeschedulerAvailable() bool {
	return c.deschedulerAvailable
}

func (c *ClusterInfoImp) IsDeschedulerCRDDeployed(ctx context.Context, cl client.Client) bool {
	return isCRDExists(ctx, cl, DeschedulerCRDName)
}

func (c *ClusterInfoImp) IsRunningLocally() bool {
	return c.runningLocally
}

func (c *ClusterInfoImp) IsSingleStackIPv6() bool {
	return c.singlestackipv6
}

func (c *ClusterInfoImp) IsControlPlaneHighlyAvailable() bool {
	return c.controlPlaneHighlyAvailable.Load()
}

func (c *ClusterInfoImp) IsControlPlaneNodeExists() bool {
	return c.controlPlaneNodeExist.Load()
}

func (c *ClusterInfoImp) IsInfrastructureHighlyAvailable() bool {
	return c.infrastructureHighlyAvailable.Load()
}

func (c *ClusterInfoImp) SetHighAvailabilityMode(ctx context.Context, cl client.Client) error {
	var err error
	masterNodeCount, workerNodeCount, err := getNodesCount(ctx, cl)
	if err != nil {
		return err
	}

	c.controlPlaneHighlyAvailable.Store(masterNodeCount >= 3)
	c.controlPlaneNodeExist.Store(masterNodeCount >= 1)
	c.infrastructureHighlyAvailable.Store(workerNodeCount >= 2)
	return nil
}

func (c *ClusterInfoImp) GetBaseDomain() string {
	return c.baseDomain
}

func (c *ClusterInfoImp) GetPod() *corev1.Pod {
	return c.ownResources.GetPod()
}

func (c *ClusterInfoImp) GetDeployment() *appsv1.Deployment {
	return c.ownResources.GetDeployment()
}

func (c *ClusterInfoImp) GetCSV() *csvv1alpha1.ClusterServiceVersion {
	return c.ownResources.GetCSV()
}

func getClusterBaseDomain(ctx context.Context, cl client.Client) (string, error) {
	clusterDNS := &openshiftconfigv1.DNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(clusterDNS), clusterDNS); err != nil {
		return "", err
	}
	return clusterDNS.Spec.BaseDomain, nil
}

func isPrometheusExists(ctx context.Context, cl client.Client) bool {
	prometheusRuleCRDExists := isCRDExists(ctx, cl, PrometheusRuleCRDName)
	serviceMonitorCRDExists := isCRDExists(ctx, cl, ServiceMonitorCRDName)

	return prometheusRuleCRDExists && serviceMonitorCRDExists
}

func isDeschedulerExists(ctx context.Context, cl client.Client) bool {
	return isCRDExists(ctx, cl, DeschedulerCRDName)
}

func isCRDExists(ctx context.Context, cl client.Client, crdName string) bool {
	found := &apiextensionsv1.CustomResourceDefinition{}
	key := client.ObjectKey{Name: crdName}
	err := cl.Get(ctx, key, found)
	return err == nil
}

func init() {
	clusterInfo = &ClusterInfoImp{
		runningLocally:     IsRunModeLocal(),
		runningInOpenshift: false,
	}
}

func (c *ClusterInfoImp) queryCluster(ctx context.Context, cl client.Client) error {
	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	if err := cl.Get(ctx, client.ObjectKeyFromObject(clusterVersion), clusterVersion); err != nil {
		var gdferr *discovery.ErrGroupDiscoveryFailed
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) || errors.As(err, &gdferr) {
			// Not on OpenShift
			c.runningInOpenshift = false
			c.logger.Info("Cluster type = kubernetes")
		} else {
			c.logger.Error(err, "Failed to get ClusterVersion")
			return err
		}
	} else {
		c.runningInOpenshift = true
		c.logger.Info("Cluster type = openshift", "version", clusterVersion.Status.Desired.Version)
		c.baseDomain, err = getClusterBaseDomain(ctx, cl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterInfoImp) GetTLSSecurityProfile(hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile {
	if hcoTLSSecurityProfile != nil {
		return hcoTLSSecurityProfile
	} else if validatedAPIServerTLSSecurityProfile != nil {
		return validatedAPIServerTLSSecurityProfile
	}
	return &openshiftconfigv1.TLSSecurityProfile{
		Type:         openshiftconfigv1.TLSProfileIntermediateType,
		Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
	}
}

func (c *ClusterInfoImp) RefreshAPIServerCR(ctx context.Context, cl client.Client) error {
	if c.IsOpenshift() {
		instance := &openshiftconfigv1.APIServer{}

		key := client.ObjectKey{Namespace: UndefinedNamespace, Name: APIServerCRName}
		err := cl.Get(ctx, key, instance)
		if err != nil {
			return err
		}
		validatedAPIServerTLSSecurityProfile = c.validateAPIServerTLSSecurityProfile(instance.Spec.TLSSecurityProfile)
		return nil
	}
	validatedAPIServerTLSSecurityProfile = nil

	return nil
}

func (c *ClusterInfoImp) validateAPIServerTLSSecurityProfile(apiServerTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile {
	if apiServerTLSSecurityProfile == nil || apiServerTLSSecurityProfile.Type != openshiftconfigv1.TLSProfileCustomType {
		return apiServerTLSSecurityProfile
	}
	validatedAPIServerTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileCustomType,
		Custom: &openshiftconfigv1.CustomTLSProfile{
			TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
				Ciphers:       openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].Ciphers,
				MinTLSVersion: openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
			},
		},
	}
	if apiServerTLSSecurityProfile.Custom != nil {
		validatedAPIServerTLSSecurityProfile.Custom.MinTLSVersion = apiServerTLSSecurityProfile.Custom.MinTLSVersion
		validatedAPIServerTLSSecurityProfile.Custom.Ciphers = nil
		for _, cipher := range apiServerTLSSecurityProfile.Custom.Ciphers {
			if isValidCipherName(cipher) {
				validatedAPIServerTLSSecurityProfile.Custom.Ciphers = append(validatedAPIServerTLSSecurityProfile.Custom.Ciphers, cipher)
			} else {
				c.logger.Error(nil, "invalid cipher name on the APIServer CR, ignoring it", "cipher", cipher)
			}
		}
	} else {
		c.logger.Error(nil, "invalid custom configuration for TLSSecurityProfile on the APIServer CR, taking default values", "apiServerTLSSecurityProfile", apiServerTLSSecurityProfile)
	}
	return validatedAPIServerTLSSecurityProfile
}

func isValidCipherName(str string) bool {
	return slices.Contains(openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileOldType].Ciphers, str) ||
		slices.Contains(openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].Ciphers, str) ||
		slices.Contains(openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileModernType].Ciphers, str)
}

func getNodesCount(ctx context.Context, cl client.Client) (int, int, error) {
	nodesList := &corev1.NodeList{}
	err := cl.List(ctx, nodesList)
	if err != nil {
		return 0, 0, err
	}
	workerNodeCount := 0
	masterNodeCount := 0

	for _, node := range nodesList.Items {
		_, workerLabelExists := node.Labels["node-role.kubernetes.io/worker"]
		if workerLabelExists {
			workerNodeCount++
		}
		_, masterLabelExists := node.Labels["node-role.kubernetes.io/master"]
		_, cpLabelExists := node.Labels["node-role.kubernetes.io/control-plane"]
		if masterLabelExists || cpLabelExists {
			masterNodeCount++
		}
	}
	return masterNodeCount, workerNodeCount, nil
}
