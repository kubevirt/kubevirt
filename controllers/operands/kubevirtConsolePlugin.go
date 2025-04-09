package operands

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"

	"k8s.io/utils/ptr"

	log "github.com/go-logr/logr"
	consolev1 "github.com/openshift/api/console/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	kvUIPluginName            = "kubevirt-plugin"
	kvUIPluginDeploymentName  = string(hcoutil.AppComponentUIPlugin)
	kvUIProxyDeploymentName   = string(hcoutil.AppComponentUIProxy)
	kvUIPluginSvcName         = kvUIPluginDeploymentName + "-service"
	kvUIProxySvcName          = kvUIProxyDeploymentName + "-service"
	kvUIPluginServingCertName = "plugin-serving-cert"
	kvUIProxyServingCertName  = "console-proxy-serving-cert"
	kvUIPluginServingCertPath = "/var/serving-cert"
	kvUIProxyServingCertPath  = "/app/cert"
	nginxConfigMapName        = "nginx-conf"
	kvUIUserSettingsCMName    = "kubevirt-user-settings"
	kvUIFeaturesCMName        = "kubevirt-ui-features"
	kvUIConfigReaderRoleName  = "kubevirt-ui-config-reader"
	kvUIConfigReaderRBName    = "kubevirt-ui-config-reader-rolebinding"
)

// **** Kubevirt UI Plugin Deployment Handler ****
func newKvUIPluginDeploymentHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newDeploymentHandler(Client, Scheme, NewKvUIPluginDeployment, hc), nil
}

// **** Kubevirt UI apiserver proxy Deployment Handler ****
func newKvUIProxyDeploymentHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newDeploymentHandler(Client, Scheme, NewKvUIProxyDeployment, hc), nil
}

// **** nginx config map Handler ****
func newKvUINginxCMHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newCmHandler(Client, Scheme, NewKVUINginxCM(hc)), nil
}

// **** UI user settings config map Handler ****
func newKvUIUserSettingsCMHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newCmHandler(Client, Scheme, NewKvUIUserSettingsCM(hc)), nil
}

// **** UI features config map Handler ****
func newKvUIFeaturesCMHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newCmHandler(Client, Scheme, NewKvUIFeaturesCM(hc)), nil
}

// **** Kubevirt UI Console Plugin Custom Resource Handler ****
func newKvUIPluginCRHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newConsolePluginHandler(Client, Scheme, NewKVConsolePlugin(hc)), nil
}

func NewKvUIPluginDeployment(hc *hcov1beta1.HyperConverged) *appsv1.Deployment {
	// The env var was validated prior to handler creation
	kvUIPluginImage, _ := os.LookupEnv(hcoutil.KVUIPluginImageEnvV)
	deployment := getKvUIDeployment(hc, kvUIPluginDeploymentName, kvUIPluginImage,
		kvUIPluginServingCertName, kvUIPluginServingCertPath, hcoutil.UIPluginServerPort, hcoutil.AppComponentUIPlugin)

	nginxVolumeMount := corev1.VolumeMount{
		Name:      nginxConfigMapName,
		MountPath: "/etc/nginx/nginx.conf",
		SubPath:   "nginx.conf",
		ReadOnly:  true,
	}

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		nginxVolumeMount,
	)

	nginxVolume := corev1.Volume{
		Name: nginxConfigMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: nginxConfigMapName,
				},
			},
		},
	}

	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, nginxVolume)

	return deployment
}

func NewKvUIProxyDeployment(hc *hcov1beta1.HyperConverged) *appsv1.Deployment {
	// The env var was validated prior to handler creation
	kvUIProxyImage, _ := os.LookupEnv(hcoutil.KVUIProxyImageEnvV)
	return getKvUIDeployment(hc, kvUIProxyDeploymentName, kvUIProxyImage, kvUIProxyServingCertName,
		kvUIProxyServingCertPath, hcoutil.UIProxyServerPort, hcoutil.AppComponentUIProxy)
}

func getKvUIDeployment(hc *hcov1beta1.HyperConverged, deploymentName string, image string,
	servingCertName string, servingCertPath string, port int32, componentName hcoutil.AppComponent) *appsv1.Deployment {
	labels := getLabels(hc, componentName)
	infrastructureHighlyAvailable := hcoutil.GetClusterInfo().IsInfrastructureHighlyAvailable()
	var replicas int32
	if infrastructureHighlyAvailable {
		replicas = int32(2)
	} else {
		replicas = int32(1)
	}

	affinity := getPodAntiAffinity(labels[hcoutil.AppLabelComponent], infrastructureHighlyAvailable)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Labels:    labels,
			Namespace: hc.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"openshift.io/required-scc": "restricted-v2",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					SecurityContext:    components.GetStdPodSecurityContext(),
					Containers: []corev1.Container{
						{
							Name:            deploymentName,
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							Ports: []corev1.ContainerPort{{
								ContainerPort: port,
								Protocol:      corev1.ProtocolTCP,
							}},
							SecurityContext:          components.GetStdContainerSecurityContext(),
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      servingCertName,
									MountPath: servingCertPath,
									ReadOnly:  true,
								},
							},
						},
					},
					PriorityClassName: kvPriorityClass,
					Volumes: []corev1.Volume{
						{
							Name: servingCertName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  servingCertName,
									DefaultMode: ptr.To(int32(420)),
								},
							},
						},
					},
				},
			},
		},
	}

	if hc.Spec.Infra.NodePlacement != nil {
		if hc.Spec.Infra.NodePlacement.NodeSelector != nil {
			deployment.Spec.Template.Spec.NodeSelector = maps.Clone(hc.Spec.Infra.NodePlacement.NodeSelector)
		} else {
			deployment.Spec.Template.Spec.NodeSelector = nil
		}

		if hc.Spec.Infra.NodePlacement.Affinity != nil {
			deployment.Spec.Template.Spec.Affinity = hc.Spec.Infra.NodePlacement.Affinity.DeepCopy()
		} else {
			deployment.Spec.Template.Spec.Affinity = affinity
		}

		if hc.Spec.Infra.NodePlacement.Tolerations != nil {
			deployment.Spec.Template.Spec.Tolerations = make([]corev1.Toleration, len(hc.Spec.Infra.NodePlacement.Tolerations))
			copy(deployment.Spec.Template.Spec.Tolerations, hc.Spec.Infra.NodePlacement.Tolerations)
		} else {
			deployment.Spec.Template.Spec.Tolerations = nil
		}
	} else {
		deployment.Spec.Template.Spec.NodeSelector = nil
		deployment.Spec.Template.Spec.Affinity = affinity
		deployment.Spec.Template.Spec.Tolerations = nil
	}
	return deployment
}

func getPodAntiAffinity(componentLabel string, infrastructureHighlyAvailable bool) *corev1.Affinity {
	if infrastructureHighlyAvailable {
		return &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
					{
						Weight: 90,
						PodAffinityTerm: corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      hcoutil.AppLabelComponent,
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{componentLabel},
									},
								},
							},
							TopologyKey: corev1.LabelHostname,
						},
					},
				},
			},
		}
	}

	return nil
}

func NewKvUIPluginSvc(hc *hcov1beta1.HyperConverged) *corev1.Service {
	servicePorts := []corev1.ServicePort{
		{
			Port:       hcoutil.UIPluginServerPort,
			Name:       kvUIPluginDeploymentName + "-port",
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.UIPluginServerPort},
		},
	}

	spec := corev1.ServiceSpec{
		Ports:    servicePorts,
		Selector: map[string]string{hcoutil.AppLabelComponent: string(hcoutil.AppComponentUIPlugin)},
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kvUIPluginSvcName,
			Labels: getLabels(hc, hcoutil.AppComponentUIPlugin),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": kvUIPluginServingCertName,
			},
			Namespace: hc.Namespace,
		},
		Spec: spec,
	}
}

func NewKvUIProxySvc(hc *hcov1beta1.HyperConverged) *corev1.Service {
	servicePorts := []corev1.ServicePort{
		{
			Port:       hcoutil.UIProxyServerPort,
			Name:       kvUIProxyDeploymentName + "-port",
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.UIProxyServerPort},
		},
	}

	spec := corev1.ServiceSpec{
		Ports:    servicePorts,
		Selector: map[string]string{hcoutil.AppLabelComponent: string(hcoutil.AppComponentUIProxy)},
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kvUIProxySvcName,
			Labels: getLabels(hc, hcoutil.AppComponentUIProxy),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": kvUIProxyServingCertName,
			},
			Namespace: hc.Namespace,
		},
		Spec: spec,
	}
}

var nginxConfig = fmt.Sprintf(`error_log /dev/stdout info;
events {}
http {
	access_log         /dev/stdout;
	include            /etc/nginx/mime.types;
	default_type       application/octet-stream;
	keepalive_timeout  65;
	add_header X-Content-Type-Options nosniff;
		server {
			listen              %d ssl;
			ssl_certificate     /var/serving-cert/tls.crt;
			ssl_certificate_key /var/serving-cert/tls.key;
			root                /usr/share/nginx/html;

			# Prevent caching for plugin-manifest.json and plugin-entry.js
			# to avoid "Unexpected end of JSON input" error
			location = /plugin-manifest.json {
			  add_header Cache-Control 'no-cache, no-store, must-revalidate, proxy-revalidate, max-age=0';
			  add_header Pragma 'no-cache';
			  add_header Expires '0';
			}
			location = /plugin-entry.js {
			  add_header Cache-Control 'no-cache, no-store, must-revalidate, proxy-revalidate, max-age=0';
			  add_header Pragma 'no-cache';
			  add_header Expires '0';
			}
        }
	}
`, hcoutil.UIPluginServerPort)

func NewKVUINginxCM(hc *hcov1beta1.HyperConverged) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxConfigMapName,
			Labels:    getLabels(hc, hcoutil.AppComponentUIPlugin),
			Namespace: hc.Namespace,
		},
		Data: map[string]string{
			"nginx.conf": nginxConfig,
		},
	}
}

func NewKvUIUserSettingsCM(hc *hcov1beta1.HyperConverged) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvUIUserSettingsCMName,
			Labels:    getLabels(hc, hcoutil.AppComponentUIConfig),
			Namespace: hc.Namespace,
		},
		Data: map[string]string{},
	}
}

var UIFeaturesConfig = map[string]string{
	"automaticSubscriptionActivationKey":  "",
	"automaticSubscriptionOrganizationId": "",
	"disabledGuestSystemLogsAccess":       "false",
	"kubevirtApiserverProxy":              "true",
	"loadBalancerEnabled":                 "true",
	"nodePortAddress":                     "",
	"nodePortEnabled":                     "false",
}

func NewKvUIFeaturesCM(hc *hcov1beta1.HyperConverged) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvUIFeaturesCMName,
			Labels:    getLabels(hc, hcoutil.AppComponentUIConfig),
			Namespace: hc.Namespace,
		},
		Data: UIFeaturesConfig,
	}
}

func NewKVConsolePlugin(hc *hcov1beta1.HyperConverged) *consolev1.ConsolePlugin {
	return &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kvUIPluginName,
			Labels: getLabels(hc, hcoutil.AppComponentUIPlugin),
		},
		Spec: consolev1.ConsolePluginSpec{
			DisplayName: "Kubevirt Console Plugin",
			Backend: consolev1.ConsolePluginBackend{
				Type: consolev1.Service,
				Service: &consolev1.ConsolePluginService{
					Name:      kvUIPluginSvcName,
					Namespace: hc.Namespace,
					Port:      hcoutil.UIPluginServerPort,
					BasePath:  "/",
				},
			},
			Proxy: []consolev1.ConsolePluginProxy{{
				Alias:         kvUIProxyDeploymentName,
				Authorization: consolev1.UserToken,
				Endpoint: consolev1.ConsolePluginProxyEndpoint{
					Type: consolev1.ProxyTypeService,
					Service: &consolev1.ConsolePluginProxyServiceConfig{
						Name:      kvUIProxySvcName,
						Namespace: hc.Namespace,
						Port:      hcoutil.UIProxyServerPort,
					},
				},
			}},
		},
	}
}

func newConsolePluginHandler(Client client.Client, Scheme *runtime.Scheme, required *consolev1.ConsolePlugin) Operand {
	h := &genericOperand{
		Client: Client,
		Scheme: Scheme,
		crType: "ConsolePlugin",
		hooks:  &consolePluginHooks{required: required},
	}

	return h
}

// **** UI configuration (user settings and features) ConfigMap Role Handler ****
func newKvUIConfigReaderRoleHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newRoleHandler(Client, Scheme, NewKvUIConfigCMReaderRole(hc)), nil
}

// **** UI configuration (user settings and features) ConfigMap RoleBinding Handler ****
func newKvUIConfigReaderRoleBindingHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) (Operand, error) {
	return newRoleBindingHandler(Client, Scheme, NewKvUIConfigCMReaderRoleBinding(hc)), nil
}

func NewKvUIConfigCMReaderRole(hc *hcov1beta1.HyperConverged) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvUIConfigReaderRoleName,
			Labels:    getLabels(hc, hcoutil.AppComponentUIPlugin),
			Namespace: hc.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{kvUIUserSettingsCMName},
				Verbs:         []string{"get", "update", "patch"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{kvUIFeaturesCMName},
				Verbs:         []string{"get"},
			},
		},
	}
}

func NewKvUIConfigCMReaderRoleBinding(hc *hcov1beta1.HyperConverged) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvUIConfigReaderRBName,
			Labels:    getLabels(hc, hcoutil.AppComponentUIPlugin),
			Namespace: hc.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     kvUIConfigReaderRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			},
		},
	}
}

type consolePluginHooks struct {
	required *consolev1.ConsolePlugin
}

func (h consolePluginHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h consolePluginHooks) getEmptyCr() client.Object {
	return &consolev1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}

func (consolePluginHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (h consolePluginHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*consolev1.ConsolePlugin)

	if !ok {
		return false, false, errors.New("can't convert to ConsolePlugin")
	}

	if !reflect.DeepEqual(h.required.Spec, found.Spec) ||
		!hcoutil.CompareLabels(h.required, found) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsolePlugin to new opinionated values", "name", h.required.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated ConsolePlugin to its opinionated values", "name", h.required.Name)
		}
		hcoutil.MergeLabels(&h.required.ObjectMeta, &found.ObjectMeta)
		h.required.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

type consoleHandler struct {
	// K8s client
	Client client.Client
}

func (h consoleHandler) ensure(req *common.HcoRequest) *EnsureResult {
	// Enable console plugin for kubevirt if not already enabled
	consoleKey := client.ObjectKey{Namespace: hcoutil.UndefinedNamespace, Name: "cluster"}
	consoleObj := &operatorv1.Console{}
	err := h.Client.Get(req.Ctx, consoleKey, consoleObj)
	if err != nil {
		req.Logger.Error(err, fmt.Sprintf("Could not find resource - APIVersion: %s, Kind: %s, Name: %s",
			consoleObj.APIVersion, consoleObj.Kind, consoleObj.Name))
		return &EnsureResult{
			Err: nil,
		}
	}

	if !slices.Contains(consoleObj.Spec.Plugins, kvUIPluginName) {
		req.Logger.Info("Enabling kubevirt plugin in Console")
		consoleObj.Spec.Plugins = append(consoleObj.Spec.Plugins, kvUIPluginName)
		err := h.Client.Update(req.Ctx, consoleObj)
		if err != nil {
			req.Logger.Error(err, fmt.Sprintf("Could not update resource - APIVersion: %s, Kind: %s, Name: %s",
				consoleObj.APIVersion, consoleObj.Kind, consoleObj.Name))
			return &EnsureResult{
				Err: err,
			}
		}

		return &EnsureResult{
			Err:         nil,
			Updated:     true,
			UpgradeDone: true,
		}
	}
	return &EnsureResult{
		Err:         nil,
		Updated:     false,
		UpgradeDone: true,
	}
}

func (consoleHandler) reset() { /* no implementation */ }

func newConsoleHandler(Client client.Client) Operand {
	h := &consoleHandler{
		Client: Client,
	}
	return h
}
