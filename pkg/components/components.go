package components

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/blang/semver/v4"
	csvVersion "github.com/operator-framework/api/pkg/lib/version"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"golang.org/x/tools/go/packages"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	crdgen "sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	cnaoapi "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	kvapi "kubevirt.io/api/core"
	aaqapi "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core"
	cdiapi "kubevirt.io/containerized-data-importer-api/pkg/apis/core"
	sspapi "kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const DisableOperandDeletionAnnotation = "console.openshift.io/disable-operand-delete"

const (
	crName              = util.HyperConvergedName
	packageName         = util.HyperConvergedName
	hcoName             = "hyperconverged-cluster-operator"
	hcoNameWebhook      = "hyperconverged-cluster-webhook"
	hcoDeploymentName   = "hco-operator"
	hcoWhDeploymentName = "hco-webhook"
	certVolume          = "apiservice-cert"

	cliDownloadsName = "hyperconverged-cluster-cli-download"

	kubevirtProjectName = "KubeVirt project"
	rbacVersionV1       = "rbac.authorization.k8s.io/v1"
)

var deploymentType = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

type DeploymentOperatorParams struct {
	Namespace              string
	Image                  string
	WebhookImage           string
	CliDownloadsImage      string
	KVUIPluginImage        string
	KVUIProxyImage         string
	ImagePullPolicy        string
	ConversionContainer    string
	VmwareContainer        string
	VirtIOWinContainer     string
	Smbios                 string
	Machinetype            string
	Amd64MachineType       string
	Arm64MachineType       string
	HcoKvIoVersion         string
	KubevirtVersion        string
	KvVirtLancherOsVersion string
	CdiVersion             string
	CnaoVersion            string
	SspVersion             string
	HppoVersion            string
	MtqVersion             string
	AaqVersion             string
	Env                    []corev1.EnvVar
}

func GetDeploymentOperator(params *DeploymentOperatorParams) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: deploymentType,
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoName,
			Labels: map[string]string{
				"name": hcoName,
			},
		},
		Spec: GetDeploymentSpecOperator(params),
	}
}

func GetDeploymentWebhook(namespace, image, imagePullPolicy, hcoKvIoVersion string, env []corev1.EnvVar) appsv1.Deployment {
	deploy := appsv1.Deployment{
		TypeMeta: deploymentType,
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoNameWebhook,
			Labels: map[string]string{
				"name": hcoNameWebhook,
			},
		},
		Spec: GetDeploymentSpecWebhook(namespace, image, imagePullPolicy, hcoKvIoVersion, env),
	}

	InjectVolumesForWebHookCerts(&deploy)
	return deploy
}

func GetDeploymentCliDownloads(params *DeploymentOperatorParams) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: deploymentType,
		ObjectMeta: metav1.ObjectMeta{
			Name: cliDownloadsName,
			Labels: map[string]string{
				"name": cliDownloadsName,
			},
		},
		Spec: GetDeploymentSpecCliDownloads(params),
	}
}

func GetServiceWebhook() corev1.Service {
	return corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoNameWebhook + "-service",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"name": hcoNameWebhook,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       strconv.Itoa(util.WebhookPort),
					Port:       util.WebhookPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(util.WebhookPort),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func GetDeploymentSpecOperator(params *DeploymentOperatorParams) appsv1.DeploymentSpec {
	envs := buildEnvVars(params)

	return appsv1.DeploymentSpec{
		Replicas: ptr.To[int32](1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"name": hcoName,
			},
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: getLabels(hcoName, params.HcoKvIoVersion),
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: hcoName,
				SecurityContext:    GetStdPodSecurityContext(),
				Containers: []corev1.Container{
					{
						Name:            hcoName,
						Image:           params.Image,
						ImagePullPolicy: corev1.PullPolicy(params.ImagePullPolicy),
						Command:         stringListToSlice(hcoName),
						ReadinessProbe:  getReadinessProbe(util.ReadinessEndpointName, util.HealthProbePort),
						LivenessProbe:   getLivenessProbe(util.LivenessEndpointName, util.HealthProbePort),
						Env:             envs,
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("96Mi"),
							},
						},
						SecurityContext: GetStdContainerSecurityContext(),
					},
				},
				PriorityClassName: "system-cluster-critical",
			},
		},
	}
}

func buildEnvVars(params *DeploymentOperatorParams) []corev1.EnvVar {
	envs := append([]corev1.EnvVar{
		{
			// deprecated: left here for CI test.
			Name:  util.OperatorWebhookModeEnv,
			Value: "false",
		},
		{
			Name:  util.ContainerAppName,
			Value: util.ContainerOperatorApp,
		},
		{
			Name:  "KVM_EMULATION",
			Value: "",
		},
		{
			Name:  "OPERATOR_IMAGE",
			Value: params.Image,
		},
		{
			Name:  "OPERATOR_NAME",
			Value: hcoName,
		},
		{
			Name:  "OPERATOR_NAMESPACE",
			Value: params.Namespace,
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "VIRTIOWIN_CONTAINER",
			Value: params.VirtIOWinContainer,
		},
		{
			Name:  "SMBIOS",
			Value: params.Smbios,
		},
		{
			Name:  "MACHINETYPE",
			Value: params.Machinetype,
		},
		{
			Name:  "AMD64_MACHINETYPE",
			Value: params.Amd64MachineType,
		},
		{
			Name:  "ARM64_MACHINETYPE",
			Value: params.Arm64MachineType,
		},
		{
			Name:  util.HcoKvIoVersionName,
			Value: params.HcoKvIoVersion,
		},
		{
			Name:  util.KubevirtVersionEnvV,
			Value: params.KubevirtVersion,
		},
		{
			Name:  util.CdiVersionEnvV,
			Value: params.CdiVersion,
		},
		{
			Name:  util.CnaoVersionEnvV,
			Value: params.CnaoVersion,
		},
		{
			Name:  util.SspVersionEnvV,
			Value: params.SspVersion,
		},
		{
			Name:  util.HppoVersionEnvV,
			Value: params.HppoVersion,
		},
		{
			Name:  util.AaqVersionEnvV,
			Value: params.AaqVersion,
		},
		{
			Name:  util.KVUIPluginImageEnvV,
			Value: params.KVUIPluginImage,
		},
		{
			Name:  util.KVUIProxyImageEnvV,
			Value: params.KVUIProxyImage,
		},
	}, params.Env...)

	if params.KvVirtLancherOsVersion != "" {
		envs = append(envs, corev1.EnvVar{
			Name:  util.KvVirtLauncherOSVersionEnvV,
			Value: params.KvVirtLancherOsVersion,
		})
	}

	return envs
}

func GetDeploymentSpecCliDownloads(params *DeploymentOperatorParams) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: ptr.To[int32](1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"name": cliDownloadsName,
			},
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: getLabels(cliDownloadsName, params.HcoKvIoVersion),
			},
			Spec: corev1.PodSpec{
				ServiceAccountName:           cliDownloadsName,
				AutomountServiceAccountToken: ptr.To(false),
				SecurityContext:              GetStdPodSecurityContext(),
				Containers: []corev1.Container{
					{
						Name:            "server",
						Image:           params.CliDownloadsImage,
						ImagePullPolicy: corev1.PullPolicy(params.ImagePullPolicy),
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("96Mi"),
							},
						},
						Ports: []corev1.ContainerPort{
							{
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: int32(8080),
							},
						},
						SecurityContext: GetStdContainerSecurityContext(),
						ReadinessProbe:  getReadinessProbe("/health", 8080),
						LivenessProbe:   getLivenessProbe("/health", 8080),
					},
				},
				PriorityClassName: "system-cluster-critical",
			},
		},
	}
}

func getLabels(name, hcoKvIoVersion string) map[string]string {
	return map[string]string{
		"name":                 name,
		util.AppLabelVersion:   hcoKvIoVersion,
		util.AppLabelPartOf:    util.HyperConvergedCluster,
		util.AppLabelComponent: string(util.AppComponentDeployment),
	}
}

func GetStdPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: ptr.To(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func GetStdContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// Currently we are abusing the pod readiness to signal to OLM that HCO is not ready
// for an upgrade. This has a lot of side effects, one of this is the validating webhook
// being not able to receive traffic when exposed by a pod that is not reporting ready=true.
// This can cause a lot of side effects if not deadlocks when the system reach a status where,
// for any possible reason, HCO pod cannot be ready and so HCO pod cannot validate any further update or
// delete request on HCO CR.
// A proper solution is properly use the readiness probe only to report the pod readiness and communicate
// status to OLM via conditions once OLM will be ready for:
// https://github.com/operator-framework/enhancements/blob/master/enhancements/operator-conditions.md
// in the meanwhile a quick (but dirty!) solution is to expose the same hco binary on two distinct pods:
// the first one will run only the controller and the second one (almost always ready) just the validating
// webhook one.
func GetDeploymentSpecWebhook(namespace, image, imagePullPolicy, hcoKvIoVersion string, env []corev1.EnvVar) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: ptr.To[int32](1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"name": hcoNameWebhook,
			},
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: getLabels(hcoNameWebhook, hcoKvIoVersion),
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: hcoName,
				SecurityContext:    GetStdPodSecurityContext(),
				Containers: []corev1.Container{
					{
						Name:            hcoNameWebhook,
						Image:           image,
						ImagePullPolicy: corev1.PullPolicy(imagePullPolicy),
						Command:         stringListToSlice(hcoNameWebhook),
						ReadinessProbe:  getReadinessProbe(util.ReadinessEndpointName, util.HealthProbePort),
						LivenessProbe:   getLivenessProbe(util.LivenessEndpointName, util.HealthProbePort),
						Env: append([]corev1.EnvVar{
							{
								// deprecated: left here for CI test.
								Name:  util.OperatorWebhookModeEnv,
								Value: "true",
							},
							{
								Name:  util.ContainerAppName,
								Value: util.ContainerWebhookApp,
							},
							{
								Name:  "OPERATOR_IMAGE",
								Value: image,
							},
							{
								Name:  "OPERATOR_NAME",
								Value: hcoNameWebhook,
							},
							{
								Name:  "OPERATOR_NAMESPACE",
								Value: namespace,
							},
							{
								Name: "POD_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
						}, env...),
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU:    resource.MustParse("5m"),
								corev1.ResourceMemory: resource.MustParse("48Mi"),
							},
						},
						SecurityContext: GetStdContainerSecurityContext(),
					},
				},
				PriorityClassName: "system-node-critical",
			},
		},
	}
}

func GetClusterRole() rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacVersionV1,
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoName,
			Labels: map[string]string{
				"name": hcoName,
			},
		},
		Rules: GetClusterPermissions(),
	}
}

var (
	emptyAPIGroup = []string{""}
)

func GetClusterPermissions() []rbacv1.PolicyRule {
	const configOpenshiftIO = "config.openshift.io"
	const operatorOpenshiftIO = "operator.openshift.io"
	return []rbacv1.PolicyRule{
		{
			APIGroups: stringListToSlice(util.APIVersionGroup),
			Resources: stringListToSlice("hyperconvergeds"),
			Verbs:     stringListToSlice("get", "list", "update", "watch"),
		},
		{
			APIGroups: stringListToSlice(util.APIVersionGroup),
			Resources: stringListToSlice("hyperconvergeds/finalizers", "hyperconvergeds/status"),
			Verbs:     stringListToSlice("get", "list", "create", "update", "watch"),
		},
		roleWithAllPermissions(kvapi.GroupName, stringListToSlice("kubevirts", "kubevirts/finalizers")),
		roleWithAllPermissions(cdiapi.GroupName, stringListToSlice("cdis", "cdis/finalizers")),
		roleWithAllPermissions(sspapi.GroupVersion.Group, stringListToSlice("ssps", "ssps/finalizers")),
		roleWithAllPermissions(cnaoapi.GroupVersion.Group, stringListToSlice("networkaddonsconfigs", "networkaddonsconfigs/finalizers")),
		roleWithAllPermissions(aaqapi.GroupName, stringListToSlice("aaqs", "aaqs/finalizers")),
		roleWithAllPermissions("", stringListToSlice("configmaps")),
		{
			APIGroups: emptyAPIGroup,
			Resources: stringListToSlice("events"),
			Verbs:     stringListToSlice("get", "list", "watch", "create", "patch"),
		},
		roleWithAllPermissions("", stringListToSlice("services")),
		{
			APIGroups: emptyAPIGroup,
			Resources: stringListToSlice("pods", "nodes"),
			Verbs:     stringListToSlice("get", "list", "watch"),
		},
		{
			APIGroups: emptyAPIGroup,
			Resources: stringListToSlice("secrets"),
			Verbs:     stringListToSlice("get", "list", "watch", "create", "update"),
		},
		{
			APIGroups: emptyAPIGroup,
			Resources: stringListToSlice("endpoints"),
			Verbs:     stringListToSlice("get", "list", "delete", "watch"),
		},
		{
			APIGroups: emptyAPIGroup,
			Resources: stringListToSlice("namespaces"),
			Verbs:     stringListToSlice("get", "list", "watch", "patch", "update"),
		},
		{
			APIGroups: stringListToSlice("apps"),
			Resources: stringListToSlice("deployments", "replicasets"),
			Verbs:     stringListToSlice("get", "list", "watch", "create", "update", "delete"),
		},
		roleWithAllPermissions("rbac.authorization.k8s.io", stringListToSlice("roles", "rolebindings")),
		{
			APIGroups: stringListToSlice("apiextensions.k8s.io"),
			Resources: stringListToSlice("customresourcedefinitions"),
			Verbs:     stringListToSlice("get", "list", "watch", "delete"),
		},
		{
			APIGroups: stringListToSlice("apiextensions.k8s.io"),
			Resources: stringListToSlice("customresourcedefinitions/status"),
			Verbs:     stringListToSlice("get", "list", "watch", "patch", "update"),
		},
		roleWithAllPermissions("monitoring.coreos.com", stringListToSlice("servicemonitors", "prometheusrules")),
		{
			APIGroups: stringListToSlice("operators.coreos.com"),
			Resources: stringListToSlice("clusterserviceversions"),
			Verbs:     stringListToSlice("get", "list", "watch", "update", "patch"),
		},
		{
			APIGroups: stringListToSlice("scheduling.k8s.io"),
			Resources: stringListToSlice("priorityclasses"),
			Verbs:     stringListToSlice("get", "list", "watch", "create", "delete", "patch"),
		},
		{
			APIGroups: stringListToSlice("admissionregistration.k8s.io"),
			Resources: stringListToSlice("validatingwebhookconfigurations"),
			Verbs:     stringListToSlice("list", "watch", "update", "patch"),
		},
		roleWithAllPermissions("console.openshift.io", stringListToSlice("consoleclidownloads", "consolequickstarts")),
		{
			APIGroups: stringListToSlice(configOpenshiftIO),
			Resources: stringListToSlice("clusterversions", "infrastructures", "networks"),
			Verbs:     stringListToSlice("get", "list"),
		},
		{
			APIGroups: stringListToSlice(configOpenshiftIO),
			Resources: stringListToSlice("ingresses"),
			Verbs:     stringListToSlice("get", "list", "watch"),
		},
		{
			APIGroups: stringListToSlice(configOpenshiftIO),
			Resources: stringListToSlice("ingresses/status"),
			Verbs:     stringListToSlice("update"),
		},
		{
			APIGroups: stringListToSlice(configOpenshiftIO),
			Resources: stringListToSlice("apiservers"),
			Verbs:     stringListToSlice("get", "list", "watch"),
		},
		{
			APIGroups: stringListToSlice(operatorOpenshiftIO),
			Resources: stringListToSlice("kubedeschedulers"),
			Verbs:     stringListToSlice("get", "list", "watch"),
		},
		{
			APIGroups: stringListToSlice(configOpenshiftIO),
			Resources: stringListToSlice("dnses"),
			Verbs:     stringListToSlice("get"),
		},
		roleWithAllPermissions("coordination.k8s.io", stringListToSlice("leases")),
		roleWithAllPermissions("route.openshift.io", stringListToSlice("routes")),
		{
			APIGroups: stringListToSlice("route.openshift.io"),
			Resources: stringListToSlice("routes/custom-host"),
			Verbs:     stringListToSlice("create", "update", "patch"),
		},
		{
			APIGroups: stringListToSlice("operators.coreos.com"),
			Resources: stringListToSlice("operatorconditions"),
			Verbs:     stringListToSlice("get", "list", "watch", "update", "patch"),
		},
		roleWithAllPermissions("image.openshift.io", stringListToSlice("imagestreams")),
		roleWithAllPermissions("console.openshift.io", stringListToSlice("consoleplugins")),
		{
			APIGroups: stringListToSlice("operator.openshift.io"),
			Resources: stringListToSlice("consoles"),
			Verbs:     stringListToSlice("get", "list", "watch", "update"),
		},
		{
			APIGroups: stringListToSlice("monitoring.coreos.com"),
			Resources: stringListToSlice("alertmanagers", "alertmanagers/api"),
			Verbs:     stringListToSlice("get", "list", "create", "delete"),
		},
	}
}

func roleWithAllPermissions(apiGroup string, resources []string) rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		APIGroups: stringListToSlice(apiGroup),
		Resources: resources,
		Verbs:     stringListToSlice("get", "list", "watch", "create", "update", "delete", "patch"),
	}
}

func GetServiceAccount(namespace string) corev1.ServiceAccount {
	return createServiceAccount(namespace, hcoName)
}

func GetCLIDownloadServiceAccount(namespace string) corev1.ServiceAccount {
	return createServiceAccount(namespace, cliDownloadsName)
}

func createServiceAccount(namespace, name string) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"name": name,
			},
		},
	}
}

func GetClusterRoleBinding(namespace string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacVersionV1,
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoName,
			Labels: map[string]string{
				"name": hcoName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     hcoName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      hcoName,
				Namespace: namespace,
			},
		},
	}
}

func packageErrors(pkg *loader.Package, filterKinds ...packages.ErrorKind) error {
	toSkip := make(map[packages.ErrorKind]struct{})
	for _, errKind := range filterKinds {
		toSkip[errKind] = struct{}{}
	}
	var outErr error
	packages.Visit([]*packages.Package{pkg.Package}, nil, func(pkgRaw *packages.Package) {
		for _, err := range pkgRaw.Errors {
			if _, skip := toSkip[err.Kind]; skip {
				continue
			}
			outErr = err
		}
	})
	return outErr
}

const objectType = "object"

func GetOperatorCRD(relPath string) *extv1.CustomResourceDefinition {
	pkgs, err := loader.LoadRoots(relPath)
	if err != nil {
		panic(err)
	}
	reg := &markers.Registry{}
	panicOnError(crdmarkers.Register(reg))

	parser := &crdgen.Parser{
		Collector:                  &markers.Collector{Registry: reg},
		Checker:                    &loader.TypeChecker{},
		GenerateEmbeddedObjectMeta: true,
	}

	crdgen.AddKnownTypes(parser)
	if len(pkgs) == 0 {
		panic("Failed identifying packages")
	}
	for _, p := range pkgs {
		parser.NeedPackage(p)
	}
	groupKind := schema.GroupKind{Kind: util.HyperConvergedKind, Group: util.APIVersionGroup}
	parser.NeedCRDFor(groupKind, nil)
	for _, p := range pkgs {
		err = packageErrors(p, packages.TypeError)
		if err != nil {
			panic(err)
		}
	}
	c := parser.CustomResourceDefinitions[groupKind]
	// enforce validation of CR name to prevent multiple CRs
	for _, v := range c.Spec.Versions {
		v.Schema.OpenAPIV3Schema.Properties["metadata"] = extv1.JSONSchemaProps{
			Type: objectType,
			Properties: map[string]extv1.JSONSchemaProps{
				"name": {
					Type:    "string",
					Pattern: hcov1beta1.HyperConvergedName,
				},
			},
		}
	}
	return &c
}

func GetOperatorCR() *hcov1beta1.HyperConverged {
	defaultScheme := runtime.NewScheme()
	_ = hcov1beta1.AddToScheme(defaultScheme)
	_ = hcov1beta1.RegisterDefaults(defaultScheme)
	defaultHco := &hcov1beta1.HyperConverged{
		TypeMeta: metav1.TypeMeta{
			APIVersion: util.APIVersion,
			Kind:       util.HyperConvergedKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		}}
	defaultScheme.Default(defaultHco)
	return defaultHco
}

// GetInstallStrategyBase returns the basics of an HCO InstallStrategy
func GetInstallStrategyBase(params *DeploymentOperatorParams) *csvv1alpha1.StrategyDetailsDeployment {
	return &csvv1alpha1.StrategyDetailsDeployment{

		DeploymentSpecs: []csvv1alpha1.StrategyDeploymentSpec{
			{
				Name:  hcoDeploymentName,
				Spec:  GetDeploymentSpecOperator(params),
				Label: getLabels(hcoName, params.HcoKvIoVersion),
			},
			{
				Name:  hcoWhDeploymentName,
				Spec:  GetDeploymentSpecWebhook(params.Namespace, params.WebhookImage, params.ImagePullPolicy, params.HcoKvIoVersion, params.Env),
				Label: getLabels(hcoNameWebhook, params.HcoKvIoVersion),
			},
			{
				Name:  cliDownloadsName,
				Spec:  GetDeploymentSpecCliDownloads(params),
				Label: getLabels(cliDownloadsName, params.HcoKvIoVersion),
			},
		},
		Permissions: []csvv1alpha1.StrategyDeploymentPermissions{},
		ClusterPermissions: []csvv1alpha1.StrategyDeploymentPermissions{
			{
				ServiceAccountName: hcoName,
				Rules:              GetClusterPermissions(),
			},
			{
				ServiceAccountName: cliDownloadsName,
				Rules:              []rbacv1.PolicyRule{},
			},
		},
	}
}

type CSVBaseParams struct {
	Name            string
	Namespace       string
	DisplayName     string
	MetaDescription string
	Description     string
	Image           string
	Replaces        string
	Version         semver.Version
	CrdDisplay      string
}

// GetCSVBase returns a base HCO CSV without an InstallStrategy
func GetCSVBase(params *CSVBaseParams) *csvv1alpha1.ClusterServiceVersion {
	almExamples, _ := json.Marshal(
		map[string]interface{}{
			"apiVersion": util.APIVersion,
			"kind":       util.HyperConvergedKind,
			"metadata": map[string]interface{}{
				"name":      packageName,
				"namespace": params.Namespace,
				"annotations": map[string]string{
					"deployOVS": "false",
				},
			},
			"spec": map[string]interface{}{},
		})

	// Explicitly fail on unvalidated (for any reason) requests:
	// this can make removing HCO CR harder if HCO webhook is not able
	// to really validate the requests.
	// In that case the user can only directly remove the
	// ValidatingWebhookConfiguration object first (eventually bypassing the OLM if needed).
	// so failurePolicy = admissionregistrationv1.Fail

	validatingWebhook := csvv1alpha1.WebhookDescription{
		GenerateName:            util.HcoValidatingWebhook,
		Type:                    csvv1alpha1.ValidatingAdmissionWebhook,
		DeploymentName:          hcoWhDeploymentName,
		ContainerPort:           util.WebhookPort,
		AdmissionReviewVersions: stringListToSlice("v1beta1", "v1"),
		SideEffects:             ptr.To(admissionregistrationv1.SideEffectClassNone),
		FailurePolicy:           ptr.To(admissionregistrationv1.Fail),
		TimeoutSeconds:          ptr.To[int32](10),
		Rules: []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
					admissionregistrationv1.Delete,
					admissionregistrationv1.Update,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   stringListToSlice(util.APIVersionGroup),
					APIVersions: stringListToSlice(util.APIVersionAlpha, util.APIVersionBeta),
					Resources:   stringListToSlice("hyperconvergeds"),
				},
			},
		},
		WebhookPath: ptr.To(util.HCOWebhookPath),
	}

	mutatingNamespaceWebhook := csvv1alpha1.WebhookDescription{
		GenerateName:            util.HcoMutatingWebhookNS,
		Type:                    csvv1alpha1.MutatingAdmissionWebhook,
		DeploymentName:          hcoWhDeploymentName,
		ContainerPort:           util.WebhookPort,
		AdmissionReviewVersions: stringListToSlice("v1beta1", "v1"),
		SideEffects:             ptr.To(admissionregistrationv1.SideEffectClassNoneOnDryRun),
		FailurePolicy:           ptr.To(admissionregistrationv1.Fail),
		TimeoutSeconds:          ptr.To[int32](10),
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{util.KubernetesMetadataName: params.Namespace},
		},
		Rules: []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Delete,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: stringListToSlice("v1"),
					Resources:   stringListToSlice("namespaces"),
				},
			},
		},
		WebhookPath: ptr.To(util.HCONSWebhookPath),
	}

	mutatingHyperConvergedWebhook := csvv1alpha1.WebhookDescription{
		GenerateName:            util.HcoMutatingWebhookHyperConverged,
		Type:                    csvv1alpha1.MutatingAdmissionWebhook,
		DeploymentName:          hcoWhDeploymentName,
		ContainerPort:           util.WebhookPort,
		AdmissionReviewVersions: stringListToSlice("v1beta1", "v1"),
		SideEffects:             ptr.To(admissionregistrationv1.SideEffectClassNoneOnDryRun),
		FailurePolicy:           ptr.To(admissionregistrationv1.Fail),
		TimeoutSeconds:          ptr.To[int32](10),
		Rules: []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
					admissionregistrationv1.Update,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   stringListToSlice(util.APIVersionGroup),
					APIVersions: stringListToSlice(util.APIVersionAlpha, util.APIVersionBeta),
					Resources:   stringListToSlice("hyperconvergeds"),
				},
			},
		},
		WebhookPath: ptr.To(util.HCOMutatingWebhookPath),
	}

	return &csvv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operators.coreos.com/v1alpha1",
			Kind:       "ClusterServiceVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.v%v", params.Name, params.Version.String()),
			Namespace: params.Namespace,
			Annotations: map[string]string{
				"alm-examples":                   string(almExamples),
				"capabilities":                   "Deep Insights",
				"certified":                      "false",
				"categories":                     "OpenShift Optional",
				"containerImage":                 params.Image,
				DisableOperandDeletionAnnotation: "true",
				"createdAt":                      time.Now().Format("2006-01-02 15:04:05"),
				"description":                    params.MetaDescription,
				"repository":                     "https://github.com/kubevirt/hyperconverged-cluster-operator",
				"support":                        "false",
				"operatorframework.io/suggested-namespace":         params.Namespace,
				"operatorframework.io/initialization-resource":     string(almExamples),
				"operators.openshift.io/infrastructure-features":   `["disconnected","proxy-aware"]`, // TODO: deprecated, remove once all the tools support "features.operators.openshift.io/*"
				"features.operators.openshift.io/disconnected":     "true",
				"features.operators.openshift.io/fips-compliant":   "false",
				"features.operators.openshift.io/proxy-aware":      "true",
				"features.operators.openshift.io/cnf":              "false",
				"features.operators.openshift.io/cni":              "true",
				"features.operators.openshift.io/csi":              "true",
				"features.operators.openshift.io/tls-profiles":     "true",
				"features.operators.openshift.io/token-auth-aws":   "false",
				"features.operators.openshift.io/token-auth-azure": "false",
				"features.operators.openshift.io/token-auth-gcp":   "false",
				"openshift.io/required-scc":                        "restricted-v2",
			},
		},
		Spec: csvv1alpha1.ClusterServiceVersionSpec{
			DisplayName: params.DisplayName,
			Description: params.Description,
			Keywords:    stringListToSlice("KubeVirt", "Virtualization"),
			Version:     csvVersion.OperatorVersion{Version: params.Version},
			Replaces:    params.Replaces,
			Maintainers: []csvv1alpha1.Maintainer{
				{
					Name:  kubevirtProjectName,
					Email: "kubevirt-dev@googlegroups.com",
				},
			},
			Maturity: "alpha",
			Provider: csvv1alpha1.AppLink{
				Name: kubevirtProjectName,
				// https://github.com/operator-framework/operator-courier/issues/173
				// URL:  "https://kubevirt.io",
			},
			Links: []csvv1alpha1.AppLink{
				{
					Name: kubevirtProjectName,
					URL:  "https://kubevirt.io",
				},
				{
					Name: "Source Code",
					URL:  "https://github.com/kubevirt/hyperconverged-cluster-operator",
				},
			},
			Icon: []csvv1alpha1.Icon{
				{
					MediaType: "image/svg+xml",
					Data:      "PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA3MDQgNzA3Ij48ZGVmcy8+PGcgZmlsbD0ibm9uZSIgZmlsbC1ydWxlPSJldmVub2RkIj48cGF0aCBkPSJNODguMzMgMTQwLjg5bC4zOC0uNC0uMzguNHpNNzQuMTggMTY3LjcyYy45Ni0zLjMgMi44Ny02LjU4IDUuNzQtOS44Ny0yLjg3IDMuMy00Ljc4IDYuNTgtNS43NCA5Ljg3ek0yMjcuNTIgNjkwLjcxYy0yLjk0IDAtNi42MiAwLTkuNTYtLjk5IDMuNjcgMCA2LjYyLjk5IDkuNTYuOTl6Ii8+PHBhdGggZmlsbD0iIzAwQUFCMiIgZmlsbC1ydWxlPSJub256ZXJvIiBkPSJNNjA2Ljg0IDEzNi45NEwzNzEuMjkgMjAuNTRsLTIuMy0xLjE4YTE4LjUgMTguNSAwIDAwLTQuOTYtMS41OGMtMS41My0uNC0zLjA2LS43OS00LjYtLjc5LTIuMjktLjM5LTQuOTYtLjM5LTcuMjYtLjM5LTQuOTcgMC05LjU2IDAtMTQuNTMuNzktMS41My40LTMuMDYuNC00Ljk3Ljc5TDk3LjEyIDEzNS4zNmEzNC45MSAzNC45MSAwIDAwLTguMDMgNS4xM2wtLjM4LjQgMTIxLjk4IDI1My4zLTkxLjc3LTE5My4zM0gyNzMuOGw2MS45NCAxMTcuOTctMjEuNDEgNDEuMDQtLjc3LjQtNjIuNy0xMTkuOTVIMTgyLjRsMTA3LjgzIDIzNS45NSAxNS4zLTMwYzQuOTYtOS40NiAxNi40NC0xMy40IDI2LTguMjhhMjAuMzMgMjAuMzMgMCAwMTguMDIgMjYuODNsLTI3LjkgNTQuMDYtMjEuNDIgNDEuMDMgNjIuNyAxMjkuODFMNDEyLjIyIDU2OWMtNi4xMiA4LjY4LTE4LjM2IDEwLjY1LTI2Ljc3IDQuMzQtNy42NS01LjUzLTkuOTQtMTYuMTgtNS43NC0yNC40N2wxMy43Ny0yOC40YzUuMzUtOS40NyAxNi44My0xMi42MyAyNi03LjEgOC4wMyA0LjczIDExLjQ3IDE0Ljk5IDguNDEgMjQuMDZsMjcuOTItNTYuODFjLTYuMTIgOC42OC0xOC4zNiAxMC42NS0yNi43NyAzLjk0YTE5LjkzIDE5LjkzIDAgMDEtNS43My0yNC40NmwyNy41My01Ni40MmM0LjU5LTkuODcgMTYuMDYtMTMuODEgMjUuNjItOC42OGExOS42NSAxOS42NSAwIDAxOC40MSAyNi40M2wtNi44OCAxMy44MSAzNS4xOC03MS44MWMtNi4xMiA5LjA4LTE3Ljk4IDExLjQ0LTI2LjM5IDUuMTNhMTkuNzggMTkuNzggMCAwMS02LjUtMjQuODZsMjcuMTUtNTYuNDJjNC41OS05Ljg2IDE2LjA2LTEzLjgxIDI1LjYyLTguNjggOS41NiA0LjczIDEzLjM4IDE2LjU3IDguNDEgMjYuNDRsLTE1LjMgMzEuOTUgNDMuNi04OC43N2MtNS4zNiA5LjQ3LTE2LjgzIDEzLjAyLTI2IDcuNWEyMC4wMyAyMC4wMyAwIDAxLTkuMTgtMTMuMDNoLTIyLjk0Yy0xMC43MSAwLTE5LjEyLTguNjgtMTkuMTItMTkuNzIgMC0xMS4wNSA4LjQxLTE5LjczIDE5LjEyLTE5LjczaDc5LjkxbC0xOS4xMiAzOS4wNiA0Ny4wNC05NS44OGE0MC44MiA0MC44MiAwIDAwLTQuNi0zLjk0IDQxLjg1IDQxLjg1IDAgMDAtOC4wMi01LjUzek00MDUuNyAzNDQuMWwtMjguNjggNTUuNjNjLTQuOTcgOS40Ny0xNi40NCAxMy40Mi0yNiA4LjI5LTkuNTYtNS4xMy0xMy0xNi45Ny04LjAzLTI2LjgzbDI4LjY3LTU1LjY0YzQuOTgtOS40NyAxNi40NS0xMy40MSAyNi04LjI4IDkuNTcgNS4xMyAxMyAxNy4zNiA4LjA0IDI2Ljgzem01OC44OC0xMTUuMjJsLTI4LjY4IDU2LjAzYy00Ljk3IDkuNDctMTYuNDQgMTMuNDItMjYgOC4yOWEyMC4zMyAyMC4zMyAwIDAxLTguMDMtMjYuODNsMzMuNjUtNjYuMjloMTIuMjRjMy4wNiAwIDYuMTIuOCA4LjggMi4zN2ExOS42MiAxOS42MiAwIDAxOC4wMiAyNi40M3oiLz48cGF0aCBmaWxsPSIjRkZGIiBmaWxsLXJ1bGU9Im5vbnplcm8iIGQ9Ik04OS4xIDE0MC41YTkxLjA1IDkxLjA1IDAgMDE4LjAyLTUuMTRMMzMyLjY3IDE4LjE4YzEuNTMtLjQgMy4wNi0uOCA0LjU5LS44IDQuOTctLjc4IDkuOTQtLjc4IDE0LjkxLS43OCAyLjMgMCA0Ljk3IDAgNy4yNy40IDEuNTMuMzkgMy40NC4zOSA0Ljk3Ljc4IDEuNTMuNCAzLjQ0LjggNC45NyAxLjU4bDIuMyAxLjE5IDIzNS41NCAxMTYuNGE0MS44NSA0MS44NSAwIDAxOC4wMyA1LjUyIDQwLjgyIDQwLjgyIDAgMDE0LjU5IDMuOTRsNy4yNy0xNC42Yy0zLjgzLTMuMTUtOC4wMy02LjMxLTEyLjYyLTguNjhoLS43N0wzNzguMTggNi4zNGE1NC4zIDU0LjMgMCAwMC0yNi01LjUyYy03LjY1LS40LTE1LjMuNC0yMi45NSAxLjk3bC0xLjUzLjQtMS41My43OEw5MC42MiAxMjEuMTZhNjYuOTkgNjYuOTkgMCAwMC04Ljc5IDUuNTJsNi44OCAxNC4yLjM4LS4zOXpNNzA1LjUgNDI1LjM3bC0uMzktMS41OC01OC44OS0yNjAuNDF2LTEuMTljLTMuNDQtMTEuNDQtMTAuMzItMjIuMS0xOS41LTI5LjU5bC03LjI2IDE0LjZjNC4yIDQuMzQgOC4wMyA5LjQ3IDEwLjMyIDE1YTIyLjc0IDIyLjc0IDAgMDExLjUzIDQuNzNsNTguNSAyNjAuNDFhOTIgOTIgMCAwMS4zOSAxMC4yNmMwIDMuMTUtLjc3IDUuOTItMS4xNSA5LjA3IDAgLjgtLjM4IDEuNTgtLjM4IDIuMzdhNTYuMjMgNTYuMjMgMCAwMS03LjY1IDE2Ljk3bC03MC4zNiA4OS45Ni05Mi41MyAxMTcuOTdjLTYuMTIgOC42OC0xNS42OCAxNC4yLTI2IDE1Ljc4LTMuMDYuNC02LjUuOC05LjU2LjhIMzUyLjk0bDU4Ljg4IDE1Ljc4aDcwLjc1YzIwLjI2IDAgMzcuMDktOC4yOSA0Ny44LTIyLjg5bDE2Mi44OS0yMDcuOTMuMzgtLjQuMzgtLjRjOS41Ni0xNC42IDEzLjc3LTMxLjk1IDExLjQ3LTQ5LjMxek0yMjIuOTMgNjkwLjEyYy0xLjUzIDAtMy40NC0uNC00Ljk3LS40LTIuMy0uNC00LjItLjc5LTYuNS0xLjU3bC0zLjQ0LTEuMTljLTIuMy0uNzktNC42LTEuOTctNi41LTIuNzZhNjAuMDEgNjAuMDEgMCAwMS05LjE4LTUuOTJjLTEuOTEtMS41OC0zLjgzLTMuMTYtNS4zNi00LjczbC01NC4zLTY5LjQ1LTEwOC4yLTEzOC44OGE1My40MiA1My40MiAwIDAxLTguOC0yMy4yOGMtLjM4LTEuNTgtLjM4LTMuMTYtLjM4LTQuNzQgMC0zLjU1IDAtNi43Ljc2LTEwLjI1bDU4LjEyLTI2MmEyNS42NCAyNS42NCAwIDAxMi4zLTcuMWMyLjY3LTYuNyA2Ljg4LTEyLjIzIDEyLjIzLTE2Ljk2bC02Ljg4LTE0LjJhNTcuNTMgNTcuNTMgMCAwMC0yMi41NiAzNC43MWwtNTguMTIgMjYydi43OWMtMy4wNiAxNy43NSAxLjE0IDM1LjUgMTEuMDkgNTAuMWwuMzguNC4zOC40TDE3NS41MSA2ODMuNGwuMzkuNzkuNzYuNGE2OS44MiA2OS44MiAwIDAwNDUuNSAyMS4zaDEzMC43OHYtMTUuNzhIMjIyLjkzeiIvPjxwYXRoIGZpbGw9IiNGRkYiIGZpbGwtcnVsZT0ibm9uemVybyIgZD0iTTM1Mi45NCA2OTAuMTJ2MTUuNzhoNTguODh6Ii8+PHBhdGggZmlsbD0iIzAwNzk3RiIgZmlsbC1ydWxlPSJub256ZXJvIiBkPSJNMjg5Ljg1IDU2MS4xbC03OS4xNi0xNjYuNUw4OC4zMyAxNDAuODhhNDEuNjggNDEuNjggMCAwMC0xMi4yNCAxNi45NmwtMi4yOSA3LjEtNTcuNzQgMjYyYy0uNzYgMy41NS0uNzYgNi43LS43NiAxMC4yNSAwIDEuNTggMCAzLjE2LjM4IDUuMTNhNTcuNDMgNTcuNDMgMCAwMDguOCAyMy4yOEwxMzMuMDYgNjA0LjVsNTQuMyA2OC42NWMxLjUzIDEuNTggMy40NCAzLjE2IDUuMzUgNC43NGEzNy4wOCAzNy4wOCAwIDAwOS4xOCA1LjkyYzIuMyAxLjE4IDQuMiAxLjk3IDYuNSAyLjc2bDMuNDQgMS4xOGMxLjkxLjc5IDQuMiAxLjE4IDYuNSAxLjU4IDEuNTMuNCAzLjQ0LjQgNC45Ny40aDEzMC4wMUwyOTAuOTkgNTU5LjlsLTEuMTQgMS4xOXoiLz48cGF0aCBkPSJNMTUuMyA0MzcuMmMwLTMuNTUgMC02LjcgMS45LTEwLjI1Ii8+PHBhdGggZmlsbD0iIzAwNzk3RiIgZmlsbC1ydWxlPSJub256ZXJvIiBkPSJNMTk2LjkzIDY4My40MWMtMy40Mi0zLjI5LTYuODMtNi41OC05LjU2LTkuODYgMi43MyAzLjI4IDYuMTQgNi41NyA5LjU2IDkuODZ6Ii8+PHBhdGggZD0iTTIwMi4yOCA2ODcuNzVhNjguNyA2OC43IDAgMDEtOS41Ni05Ljg2TTE4Ny4zNyA2NzMuMTVsLTU0LjMtNjkuMDVNMjE3IDY4OS45MmwtOC42LTIuOTYiLz48cGF0aCBmaWxsPSIjMDA3OTdGIiBmaWxsLXJ1bGU9Im5vbnplcm8iIGQ9Ik0yMTEuNDYgNjkxLjFjLTMuMzgtMS45Ny02Ljc1LTQuOTMtOS41Ni02LjkgMi44IDEuOTcgNi4xOCA0LjkzIDkuNTYgNi45eiIvPjxwYXRoIGZpbGw9IiMzQUNDQzUiIGZpbGwtcnVsZT0ibm9uemVybyIgZD0iTTU3MC4xMyAyNDcuNDJsLTQzLjYgODguNzgtMTEuODQgMjQuNDZhOC42OCA4LjY4IDAgMDEtMS4xNSAxLjk3bC0zNS4xOCA3MS40Mi0yMC42NSA0Mi42MWMtLjM4Ljc5LTEuMTUgMS45Ny0xLjUzIDIuNzZsLTI3LjkxIDU3LjIxYzAgLjQgMCAuNC0uMzkuOGwtMTMuNzYgMjguNGMtLjM4Ljc5LTEuMTUgMS45Ny0xLjUzIDIuNzZsLTU5LjI3IDEyMC43NGgxMjkuNjNjMy4wNiAwIDYuNS0uNCA5LjU2LS43OWEzOS44IDM5LjggMCAwMDI2LTE1Ljc4bDkyLjU0LTExNy45OCA3MC4zNS04OS45NmE1Mi4yIDUyLjIgMCAwMDcuNjUtMTYuOTZjLjM4LS44LjM4LTEuNTguMzgtMi4zN2EzNi45IDM2LjkgMCAwMDEuMTUtOS4wOGMwLTMuNTUgMC02LjctLjM4LTEwLjI1bC01OC41LTI1OS42M2MtLjM5LTEuNTctMS4xNS0zLjE1LTEuNTQtNC43My0yLjI5LTUuNTItNi4xMS0xMC42NS0xMC4zMi0xNWwtNDcuMDMgOTUuNDktMi42OCA1LjEzeiIvPjxwYXRoIGQ9Ik02OTIuMyA0MzcuMmMwIDMuNDMtMS45MSA2LjQ0LTIuODcgOS44N002OTIuNjggNDQ4LjY1Yy0xLjkgMy40NS0zLjgyIDYuOS02LjY5IDkuODZNNDkyLjEyIDY4OS4zM2EzOS44IDM5LjggMCAwMDI2LTE1Ljc4bDkyLjU0LTExNy45OE02OTAuNTggNDI2Ljk1Yy45NiAzLjU1Ljk2IDYuNy45NiAxMC4yNSIvPjxwYXRoIGZpbGw9IiNGRkYiIGZpbGwtcnVsZT0ibm9uemVybyIgZD0iTTM5Ny42OCAzMTcuMjZjLTkuMTgtNS4xMy0yMS4wMy0xLjU4LTI2IDguMjhMMzQzIDM4MS4xOGMtNC45NyA5LjQ3LTEuNTMgMjEuNyA4LjAzIDI2LjgzIDkuMTcgNS4xMyAyMS4wMyAxLjU3IDI2LTguMjlsMjguNjgtNTUuNjNjNC45Ny05LjQ3IDEuMTQtMjEuNy04LjAzLTI2Ljgzek00MTkuMDkgNTExLjM4YTE5LjAzIDE5LjAzIDAgMDAtMjUuNjIgOC42OGwtMTMuNzcgMjguNDFjLTQuNTggOS44Ni0uNzYgMjEuNyA4LjggMjYuNDQgOC40MSA0LjM0IDE4LjM1IDEuNTcgMjMuNy01LjkybDE1LjY4LTMxLjk2YzQuMjEtOS40Ny4zOS0yMC45MS04Ljc5LTI1LjY1ek00MjcuODggNTM3LjgyYzAtLjQgMC0uNC4zOS0uOEw0MTIuNTkgNTY5YTYuNDEgNi40MSAwIDAwMS41My0yLjc2bDEzLjc2LTI4LjQxeiIvPjxwYXRoIGZpbGw9IiNGRkYiIGZpbGwtcnVsZT0ibm9uemVybyIgZD0iTTMxMS42NCA1MTkuMjdsMjcuOTEtNTQuMDVjNC45OC05LjQ3IDEuNTMtMjEuNy04LjAzLTI2LjgzLTkuMTctNS4xMy0yMS4wMy0xLjU4LTI2IDguMjhsLTE1LjMgMjkuOTlMMTgyLjQgMjQwLjMyaDY4LjQ0bDYyLjcxIDExOS45NC43Ny0uNCAyMS4wMy00MC42My02MS45NS0xMTguMzdIMTE4LjU0TDIxMC4zIDM5NC4ybDc5LjkyIDE2NS43MSAyMS40MS00MC42NHoiLz48cGF0aCBmaWxsPSIjRkZGIiBmaWxsLXJ1bGU9Im5vbnplcm8iIGQ9Ik0yOTAuMjMgNTYwLjMxbC03OS41NC0xNjUuNzIgNzkuMTYgMTY2LjUxek01OTEuNTQgMjAzLjIzaC03OS45MWMtMTAuNzEgMC0xOS4xMiA4LjY4LTE5LjEyIDE5LjczIDAgMTEuMDQgOC40MSAxOS43MiAxOS4xMiAxOS43MmgyMi45NGMyLjMgMTAuNjYgMTIuNjIgMTcuMzcgMjIuOTQgMTVhMTkuNSAxOS41IDAgMDAxMi42Mi05LjQ3bDIuNjgtNS41MyAxOC43My0zOS40NXpNNTc2LjgyIDI0Mi4yOWwtNi42OSA5Ljg2Ljk2LS43ek01NDEuODMgMzA0LjYzYzQuOTgtOS44NiAxLjE1LTIxLjctOC40LTI2LjQ0LTkuNTctNC43My0yMS4wNC0xLjE4LTI1LjYzIDguNjl2LjM5bC0yNy4xNSA1Ni40MmMtNC41OSA5Ljg3LS43NiAyMS43IDguOCAyNi40NCA4LjQxIDQuMzQgMTguNzMgMS41OCAyNC4wOS02LjcxbDEzLTI2LjgzIDE1LjMtMzEuOTZ6Ii8+PHBhdGggZmlsbD0iI0ZGRiIgZmlsbC1ydWxlPSJub256ZXJvIiBkPSJNNTI2LjU0IDMzNi41OWwtMTMgMjYuODNjLjM4LS43OS43Ni0xLjU4IDEuMTUtMS45N2wxMS44NS0yNC44NnpNNDg0Ljg2IDQyMS4wM2M0LjU5LTkuODcuNzYtMjEuNy04LjQxLTI2LjQ0LTkuMTgtNC43My0yMS4wMy0uNzktMjUuNjIgOC42OGwtMjcuMTUgNTYuNDJjLTQuNTkgOS44Ny0uNzcgMjEuNyA4LjggMjYuNDQgOC40IDQuMzQgMTguMzUgMS41OCAyMy43LTUuOTJsMjIuMTgtNDUuMzcgNi41LTEzLjgxeiIvPjxwYXRoIGZpbGw9IiNGRkYiIGZpbGwtcnVsZT0ibm9uemVybyIgZD0iTTQ3OC4zNiA0MzQuODRsLTIyLjE4IDQ1LjM3Yy43Ny0uNzkgMS4xNS0xLjk3IDEuNTMtMi43NmwyMC42NS00Mi42MXpNNDU2LjU2IDIwMi40NGExNy4zNCAxNy4zNCAwIDAwLTguOC0yLjM3aC0xMS44NWwtMzMuNjQgNjYuMjlhMjAuMTUgMjAuMTUgMCAwMDQuOTcgMjcuNjJjOC44IDYuMzEgMjAuNjQgMy45NCAyNi43Ni01LjEzLjc3LTEuMTkgMS41My0yLjM3IDEuOTEtMy45NWwyOC42OC01NS42M2M0Ljk3LTkuODYgMS41My0yMS43LTguMDMtMjYuODMuMzkgMCAwIDAgMCAweiIvPjwvZz48L3N2Zz4=",
				},
			},
			Labels: map[string]string{
				"alm-owner-kubevirt": packageName,
				"operated-by":        packageName,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"alm-owner-kubevirt": packageName,
					"operated-by":        packageName,
				},
			},
			InstallModes: []csvv1alpha1.InstallMode{
				{
					Type:      csvv1alpha1.InstallModeTypeOwnNamespace,
					Supported: false,
				},
				{
					Type:      csvv1alpha1.InstallModeTypeSingleNamespace,
					Supported: false,
				},
				{
					Type:      csvv1alpha1.InstallModeTypeMultiNamespace,
					Supported: false,
				},
				{
					Type:      csvv1alpha1.InstallModeTypeAllNamespaces,
					Supported: true,
				},
			},
			// Skip this in favor of having a separate function to get
			// the actual StrategyDetailsDeployment when merging CSVs
			InstallStrategy: csvv1alpha1.NamedInstallStrategy{},
			WebhookDefinitions: []csvv1alpha1.WebhookDescription{
				validatingWebhook,
				mutatingNamespaceWebhook,
				mutatingHyperConvergedWebhook,
			},
			CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
				Owned: []csvv1alpha1.CRDDescription{
					{
						Name:        "hyperconvergeds.hco.kubevirt.io",
						Version:     util.CurrentAPIVersion,
						Kind:        util.HyperConvergedKind,
						DisplayName: params.CrdDisplay + " Deployment",
						Description: "Represents the deployment of " + params.CrdDisplay,
						// TODO: move this to annotations on hyperconverged_types.go once kubebuilder
						// properly supports SpecDescriptors as the operator-sdk already does
						SpecDescriptors: []csvv1alpha1.SpecDescriptor{
							{
								DisplayName: "Infra components node affinity",
								Description: "nodeAffinity describes node affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.nodeAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:nodeAffinity",
								),
							},
							{
								DisplayName: "Infra components pod affinity",
								Description: "podAffinity describes pod affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.podAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:podAffinity",
								),
							},
							{
								DisplayName: "Infra components pod anti-affinity",
								Description: "podAntiAffinity describes pod anti affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.podAntiAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:podAntiAffinity",
								),
							},
							{
								DisplayName: "Workloads components node affinity",
								Description: "nodeAffinity describes node affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.nodeAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:nodeAffinity",
								),
							},
							{
								DisplayName: "Workloads components pod affinity",
								Description: "podAffinity describes pod affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.podAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:podAffinity",
								),
							},
							{
								DisplayName: "Workloads components pod anti-affinity",
								Description: "podAntiAffinity describes pod anti affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.podAntiAffinity",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:podAntiAffinity",
								),
							},
							{
								DisplayName: "HIDDEN FIELDS - operator version",
								Description: "HIDDEN FIELDS - operator version.",
								Path:        "version",
								XDescriptors: stringListToSlice(
									"urn:alm:descriptor:com.tectonic.ui:hidden",
								),
							},
						},
						StatusDescriptors: []csvv1alpha1.StatusDescriptor{},
					},
				},
				Required: []csvv1alpha1.CRDDescription{},
			},
		},
	}
}

func InjectVolumesForWebHookCerts(deploy *appsv1.Deployment) {
	// check if there is already a volume for api certificates
	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.Name == certVolume {
			return
		}
	}

	volume := corev1.Volume{
		Name: certVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  deploy.Name + "-service-cert",
				DefaultMode: ptr.To[int32](420),
				Items: []corev1.KeyToPath{
					{
						Key:  "tls.crt",
						Path: util.WebhookCertName,
					},
					{
						Key:  "tls.key",
						Path: util.WebhookKeyName,
					},
				},
			},
		},
	}
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)

	for index, container := range deploy.Spec.Template.Spec.Containers {
		deploy.Spec.Template.Spec.Containers[index].VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      certVolume,
				MountPath: util.DefaultWebhookCertDir,
			})
	}
}

func getReadinessProbe(endpoint string, port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: endpoint,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: port,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    1,
	}
}

func getLivenessProbe(endpoint string, port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: endpoint,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: port,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       5,
		FailureThreshold:    1,
	}
}

func stringListToSlice(words ...string) []string {
	return words
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
