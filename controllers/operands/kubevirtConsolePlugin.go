package operands

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"k8s.io/utils/pointer"

	log "github.com/go-logr/logr"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/cmd/cmdcommon"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	kvUIPluginName     = "kubevirt-plugin"
	kvUIPluginSvcName  = kvUIPluginName + "-service"
	kvUIPluginNameEnv  = "UI_PLUGIN_NAME"
	kvServingCertName  = "plugin-serving-cert"
	nginxConfigMapName = "nginx-conf"
)

// **** Kubevirt UI Plugin Deployment Handler ****
func newKvUiPluginDplymntHandler(logger log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	kvUiPluginDeplymnt, err := NewKvUiPluginDeplymnt(hc)
	if err != nil {
		return nil, err
	}
	return []Operand{newDeploymentHandler(Client, Scheme, kvUiPluginDeplymnt)}, nil
}

// **** nginx config map Handler ****
func newKvUiNginxCmHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	kvUiNginxCm := NewKvUiNginxCm(hc)

	return []Operand{newCmHandler(Client, Scheme, kvUiNginxCm)}, nil
}

// **** Kubevirt UI Console Plugin Custom Resource Handler ****
func newKvUiPluginCRHandler(_ log.Logger, Client client.Client, Scheme *runtime.Scheme, hc *hcov1beta1.HyperConverged) ([]Operand, error) {
	kvUiConsolePluginCR := NewKvConsolePlugin(hc)

	return []Operand{newConsolePluginHandler(Client, Scheme, kvUiConsolePluginCR)}, nil
}

func NewKvUiPluginDeplymnt(hc *hcov1beta1.HyperConverged) (*appsv1.Deployment, error) {
	// The env var was validated prior to handler creation
	kvUiPluginImage, _ := os.LookupEnv(hcoutil.KvUiPluginImageEnvV)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvUIPluginName,
			Labels:    getLabels(hc, hcoutil.AppComponentDeployment),
			Namespace: hc.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": kvUIPluginName,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": kvUIPluginName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					Containers: []corev1.Container{
						{
							Name:            kvUIPluginName,
							Image:           kvUiPluginImage,
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{{
								ContainerPort: hcoutil.UiPluginServerPort,
								Protocol:      corev1.ProtocolTCP,
							}},
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      kvServingCertName,
									MountPath: "/var/serving-cert",
									ReadOnly:  true,
								},
								{
									Name:      nginxConfigMapName,
									MountPath: "/etc/nginx/nginx.conf",
									SubPath:   "nginx.conf",
									ReadOnly:  true,
								},
							},
						},
					},
					PriorityClassName: "kubevirt-cluster-critical",
					Volumes: []corev1.Volume{
						{
							Name: kvServingCertName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  kvServingCertName,
									DefaultMode: pointer.Int32Ptr(420),
								},
							},
						},
						{
							Name: nginxConfigMapName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: nginxConfigMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func NewKvUiPluginSvc(hc *hcov1beta1.HyperConverged) *corev1.Service {
	servicePorts := []corev1.ServicePort{
		{Port: hcoutil.UiPluginServerPort, Name: kvUIPluginName + "-port", Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.UiPluginServerPort}},
	}
	pluginName := kvUIPluginName
	val, ok := os.LookupEnv(kvUIPluginNameEnv)
	if ok && val != "" {
		pluginName = val
	}
	labelSelect := map[string]string{"app": pluginName}

	spec := corev1.ServiceSpec{
		Ports:    servicePorts,
		Selector: labelSelect,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kvUIPluginSvcName,
			Labels: getLabels(hc, hcoutil.AppComponentDeployment),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": kvServingCertName,
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
		server {
			listen              %d ssl;
			ssl_certificate     /var/serving-cert/tls.crt;
			ssl_certificate_key /var/serving-cert/tls.key;
			root                /usr/share/nginx/html;
		}
	}
`, hcoutil.UiPluginServerPort)

func NewKvUiNginxCm(hc *hcov1beta1.HyperConverged) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginxConfigMapName,
			Labels:    getLabels(hc, hcoutil.AppComponentDeployment),
			Namespace: hc.Namespace,
		},
		Data: map[string]string{
			"nginx.conf": nginxConfig,
		},
	}
}

func NewKvConsolePlugin(hc *hcov1beta1.HyperConverged) *consolev1alpha1.ConsolePlugin {
	return &consolev1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:   kvUIPluginName,
			Labels: getLabels(hc, hcoutil.AppComponentDeployment),
		},
		Spec: consolev1alpha1.ConsolePluginSpec{
			DisplayName: "Kubevirt Console Plugin",
			Service: consolev1alpha1.ConsolePluginService{
				Name:      kvUIPluginSvcName,
				Namespace: hc.Namespace,
				Port:      hcoutil.UiPluginServerPort,
				BasePath:  "/",
			},
		},
	}
}

func newConsolePluginHandler(Client client.Client, Scheme *runtime.Scheme, required *consolev1alpha1.ConsolePlugin) Operand {
	h := &genericOperand{
		Client:              Client,
		Scheme:              Scheme,
		crType:              "ConsolePlugin",
		removeExistingOwner: false,
		hooks:               &consolePluginHooks{required: required},
	}

	return h
}

type consolePluginHooks struct {
	required *consolev1alpha1.ConsolePlugin
}

func (h consolePluginHooks) getFullCr(_ *hcov1beta1.HyperConverged) (client.Object, error) {
	return h.required.DeepCopy(), nil
}

func (h consolePluginHooks) getEmptyCr() client.Object {
	return &consolev1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.required.Name,
		},
	}
}

func (h consolePluginHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*consolev1alpha1.ConsolePlugin).ObjectMeta
}

func (h consolePluginHooks) reset() { /* no implementation */ }

func (h consolePluginHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func (h consolePluginHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, _ runtime.Object) (bool, bool, error) {
	found, ok := exists.(*consolev1alpha1.ConsolePlugin)

	if !ok {
		return false, false, errors.New("can't convert to ConsolePlugin")
	}

	if !reflect.DeepEqual(found.Spec, h.required.Spec) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsolePlugin to new opinionated values", "name", h.required.Name)
		} else {
			req.Logger.Info("Reconciling an externally updated ConsolePlugin to its opinionated values", "name", h.required.Name)
		}
		hcoutil.DeepCopyLabels(&h.required.ObjectMeta, &found.ObjectMeta)
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

	if !cmdcommon.StringInSlice(kvUIPluginName, consoleObj.Spec.Plugins) {
		req.Logger.Info("Enabling kubevirt plugin in Console")
		consoleObj.Spec.Plugins = append(consoleObj.Spec.Plugins, kvUIPluginName)
		err := h.Client.Update(req.Ctx, consoleObj)
		if err != nil {
			req.Logger.Error(err, fmt.Sprintf("Could not update resource - APIVersion: %s, Kind: %s, Name: %s",
				consoleObj.APIVersion, consoleObj.Kind, consoleObj.Name))
			return &EnsureResult{
				Err: err,
			}
		} else {
			return &EnsureResult{
				Err:         nil,
				Updated:     true,
				UpgradeDone: true,
			}
		}
	}
	return &EnsureResult{
		Err:         nil,
		Updated:     false,
		UpgradeDone: true,
	}
}

func (h consoleHandler) reset() { /* no implementation */ }

func newConsoleHandler(Client client.Client) Operand {
	h := &consoleHandler{
		Client: Client,
	}
	return h
}
