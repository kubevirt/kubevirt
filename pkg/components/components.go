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
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	crdgen "sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
)

const (
	crName              = util.HyperConvergedName
	packageName         = util.HyperConvergedName
	hcoName             = "hyperconverged-cluster-operator"
	hcoNameWebhook      = "hyperconverged-cluster-webhook"
	hcoDeploymentName   = "hco-operator"
	hcoWhDeploymentName = "hco-webhook"
	certVolume          = "apiservice-cert"

	kubevirtProjectName = "KubeVirt project"
)

type DeploymentOperatorParams struct {
	Namespace           string
	Image               string
	WebhookImage        string
	ImagePullPolicy     string
	ConversionContainer string
	VmwareContainer     string
	VirtIOWinContainer  string
	Smbios              string
	Machinetype         string
	HcoKvIoVersion      string
	KubevirtVersion     string
	CdiVersion          string
	CnaoVersion         string
	SspVersion          string
	NmoVersion          string
	HppoVersion         string
	VMImportVersion     string
	Env                 []corev1.EnvVar
}

func GetDeploymentOperator(params *DeploymentOperatorParams) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
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

func GetServiceWebhook(namespace string) v1.Service {
	return v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: hcoNameWebhook + "-service",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"name": hcoNameWebhook,
			},
			Ports: []v1.ServicePort{
				{
					Name:       strconv.Itoa(util.WebhookPort),
					Port:       util.WebhookPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(util.WebhookPort),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func GetDeploymentSpecOperator(params *DeploymentOperatorParams) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: int32Ptr(1),
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
				Containers: []corev1.Container{
					{
						Name:            hcoName,
						Image:           params.Image,
						ImagePullPolicy: corev1.PullPolicy(params.ImagePullPolicy),
						// command being name is artifact of operator-sdk usage
						Command:        []string{hcoName},
						ReadinessProbe: getReadinessProbe(),
						LivenessProbe:  getLivenessProbe(),
						Env: append([]corev1.EnvVar{
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
								Name:  "WATCH_NAMESPACE",
								Value: "",
							},
							{
								Name:  "CONVERSION_CONTAINER",
								Value: params.ConversionContainer,
							},
							{
								Name:  "VMWARE_CONTAINER",
								Value: params.VmwareContainer,
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
								Name:  util.NmoVersionEnvV,
								Value: params.NmoVersion,
							},
							{
								Name:  util.HppoVersionEnvV,
								Value: params.HppoVersion,
							},
							{
								Name:  util.VMImportEnvV,
								Value: params.VMImportVersion,
							},
						}, params.Env...),
						Resources: v1.ResourceRequirements{
							Requests: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU:    resource.MustParse("10m"),
								v1.ResourceMemory: resource.MustParse("96Mi"),
							},
						},
					},
				},
				PriorityClassName: "system-cluster-critical",
			},
		},
	}
}

func getLabels(name, hcoKvIoVersion string) map[string]string {
	return map[string]string{
		"name":                    name,
		hcoutil.AppLabelVersion:   hcoKvIoVersion,
		hcoutil.AppLabelPartOf:    hcoutil.HyperConvergedCluster,
		hcoutil.AppLabelComponent: string(hcoutil.AppComponentDeployment),
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
		Replicas: int32Ptr(1),
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
				Containers: []corev1.Container{
					{
						Name:            hcoNameWebhook,
						Image:           image,
						ImagePullPolicy: corev1.PullPolicy(imagePullPolicy),
						Command:         []string{hcoNameWebhook},
						ReadinessProbe:  getReadinessProbe(),
						LivenessProbe:   getLivenessProbe(),
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
							{
								Name:  "WATCH_NAMESPACE",
								Value: "",
							},
						}, env...),
						Resources: v1.ResourceRequirements{
							Requests: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU:    resource.MustParse("5m"),
								v1.ResourceMemory: resource.MustParse("48Mi"),
							},
						},
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
			APIVersion: "rbac.authorization.k8s.io/v1",
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
	AnyResource = []string{"*"}
	anyVerb     = []string{"*"}
)

func getAnyPolicy(apiGroups []string) rbacv1.PolicyRule {
	return rbacv1.PolicyRule{
		APIGroups: apiGroups,
		Resources: AnyResource,
		Verbs:     anyVerb,
	}
}

func GetClusterPermissions() []rbacv1.PolicyRule {
	emptyAPIGroup := []string{""}

	return []rbacv1.PolicyRule{
		getAnyPolicy([]string{util.APIVersionGroup}),
		getAnyPolicy([]string{"kubevirt.io"}),
		getAnyPolicy([]string{"cdi.kubevirt.io"}),
		getAnyPolicy([]string{"ssp.kubevirt.io"}),
		getAnyPolicy([]string{"networkaddonsoperator.network.kubevirt.io"}),
		getAnyPolicy([]string{"v2v.kubevirt.io"}),
		{
			APIGroups: []string{
				"machineremediation.kubevirt.io",
			},
			Resources: []string{
				"machineremediationoperators",
				"machineremediationoperators/status",
			},
			Verbs: []string{
				rbacv1.VerbAll,
			},
		},
		{
			APIGroups: emptyAPIGroup,
			Resources: getPolicyRules("pods", "services", "services/finalizers", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets", "serviceaccounts"),
			Verbs:     anyVerb,
		},
		{
			APIGroups: emptyAPIGroup,
			Resources: []string{
				"nodes",
			},
			Verbs: getPolicyRules("get", "list"),
		},
		{
			APIGroups: []string{
				"apps",
			},
			Resources: getPolicyRules("deployments", "deployments/finalizers", "daemonsets", "replicasets"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete", "update"),
		},
		{
			APIGroups: []string{
				"batch",
			},
			Resources: getPolicyRules("jobs"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete"),
		},
		{
			APIGroups: []string{
				"rbac.authorization.k8s.io",
			},
			Resources: getPolicyRules("clusterroles", "clusterrolebindings", "roles", "rolebindings"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete", "update"),
		},
		{
			APIGroups: []string{
				"apiextensions.k8s.io",
			},
			Resources: getPolicyRules("customresourcedefinitions"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete", "patch", "update"),
		},
		{
			APIGroups: []string{
				"security.openshift.io",
			},
			Resources: getPolicyRules("securitycontextconstraints"),
			Verbs:     getPolicyRules("get", "list", "watch"),
		},
		{
			APIGroups: []string{
				"security.openshift.io",
			},
			Resources: getPolicyRules("securitycontextconstraints"),
			ResourceNames: []string{
				"privileged",
			},
			Verbs: getPolicyRules("get", "patch", "update"),
		},
		{
			APIGroups: []string{
				"monitoring.coreos.com",
			},
			Resources: getPolicyRules("servicemonitors", "prometheusrules"),
			Verbs:     anyVerb,
		},
		{
			APIGroups: []string{
				"operators.coreos.com",
			},
			Resources: getPolicyRules("clusterserviceversions"),
			Verbs:     getPolicyRules("get", "list", "watch"),
		},
		{
			APIGroups: []string{"scheduling.k8s.io"},
			Resources: getPolicyRules("priorityclasses"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete"),
		},
		{
			APIGroups: []string{
				"admissionregistration.k8s.io",
			},
			Resources: getPolicyRules("validatingwebhookconfigurations"),
			Verbs:     getPolicyRules("list", "watch", "update", "patch"),
		},
		{
			APIGroups: []string{
				"console.openshift.io",
			},
			Resources: getPolicyRules("consoleclidownloads", "consolequickstarts"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete", "update"),
		},
		{
			APIGroups: []string{
				"config.openshift.io",
			},
			Resources: getPolicyRules("clusterversions"),
			Verbs:     getPolicyRules("get", "list"),
		},
		{
			APIGroups: []string{
				"coordination.k8s.io",
			},
			Resources: getPolicyRules("leases"),
			Verbs:     getPolicyRules("get", "list", "watch", "create", "delete", "update"),
		},
	}
}

func GetServiceAccount(namespace string) v1.ServiceAccount {
	return v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoName,
			Namespace: namespace,
			Labels: map[string]string{
				"name": hcoName,
			},
		},
	}
}

func GetClusterRoleBinding(namespace string) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
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
			rbacv1.Subject{
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
		Collector: &markers.Collector{Registry: reg},
		Checker:   &loader.TypeChecker{},
	}
	crdgen.AddKnownTypes(parser)
	if len(pkgs) == 0 {
		panic("Failed identifying packages")
	}
	for _, p := range pkgs {
		parser.NeedPackage(p)
	}
	groupKind := schema.GroupKind{Kind: "HyperConverged", Group: "hco.kubevirt.io"}
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

var crdMeta = metav1.TypeMeta{
	APIVersion: "apiextensions.k8s.io/v1",
	Kind:       "CustomResourceDefinition",
}

func getSchemaInitialProps() map[string]extv1.JSONSchemaProps {
	return map[string]extv1.JSONSchemaProps{
		"apiVersion": {
			Type: "string",
			Description: `APIVersion defines the versioned schema of this representation
                        of an object. Servers should convert recognized schemas to the latest
                        internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources`,
		},
		"kind": {
			Type: "string",
			Description: `Kind is a string value representing the REST resource this
                        object represents. Servers may infer this from the endpoint the client
                        submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds`,
		},
		"metadata": {Type: objectType},
	}
}

// TODO: remove once VMware provider is removed from HCO
// GetV2VCRD creates CRD for v2v VMWare provider
func GetV2VCRD() *extv1.CustomResourceDefinition {
	versionSchema := &extv1.CustomResourceValidation{
		OpenAPIV3Schema: &extv1.JSONSchemaProps{
			Description: "V2VVmware is the Schema for the v2vvmwares API",
			Type:        objectType,
			Properties:  getSchemaInitialProps(),
		},
	}

	versionSchema.OpenAPIV3Schema.Properties["spec"] = extv1.JSONSchemaProps{
		Description: "V2VVmwareSpec defines the desired state of V2VVmware",
		Type:        objectType,
		Properties: map[string]extv1.JSONSchemaProps{
			"connection": {Type: "string"},
			"thumbprint": {Type: "string"},
			"timeToLive": {Type: "string"},
			"vms": {
				Items: &extv1.JSONSchemaPropsOrArray{
					Schema: &extv1.JSONSchemaProps{
						Type: objectType,
						Properties: map[string]extv1.JSONSchemaProps{
							"detail": {
								Type: objectType,
								Properties: map[string]extv1.JSONSchemaProps{
									"hostPath": {
										Type: "string",
									},
									"raw": {
										Type:        "string",
										Description: "TODO: list required details",
									},
								},
								Required: []string{"hostPath"},
							},
							"detailRequest": {Type: "boolean"},
							"name":          {Type: "string"},
						},
						Required: []string{"name"},
					},
				},
				Type: "array",
			},
		},
	}
	versionSchema.OpenAPIV3Schema.Properties["status"] = extv1.JSONSchemaProps{
		Description: "V2VVmwareStatus defines the observed state of V2VVmware",
		Type:        objectType,
		Properties: map[string]extv1.JSONSchemaProps{
			"phase": {
				Type: "string",
			},
		},
	}

	names := extv1.CustomResourceDefinitionNames{
		Plural:   "v2vvmwares",
		Singular: "v2vvmware",
		Kind:     "V2VVmware",
		ListKind: "V2VVmwareList",
	}

	return getCrd("v2vvmwares."+vmimportv1beta1.SchemeGroupVersion.Group, versionSchema, names)
}

// TODO: remove once oVirt provider  is removed from HCO
// GetV2VOvirtProviderCRD creates CRD for v2v oVirt provider
func GetV2VOvirtProviderCRD() *extv1.CustomResourceDefinition {
	versionSchema := &extv1.CustomResourceValidation{
		OpenAPIV3Schema: &extv1.JSONSchemaProps{
			Description: "OVirtProvider is the Schema for the ovirtproviders API",
			Type:        objectType,
			Properties: map[string]extv1.JSONSchemaProps{
				"apiVersion": {
					Type: "string",
					Description: `APIVersion defines the versioned schema of this representation
                        of an object. Servers should convert recognized schemas to the latest
                        internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources`,
				},
				"kind": {
					Type: "string",
					Description: `Kind is a string value representing the REST resource this
                        object represents. Servers may infer this from the endpoint the client
                        submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds`,
				},
				"metadata": {Type: objectType},
			},
		},
	}

	versionSchema.OpenAPIV3Schema.Properties["spec"] = extv1.JSONSchemaProps{
		Description: "OVirtProviderSpec defines the desired state of OVirtProvider",
		Type:        objectType,
		Properties: map[string]extv1.JSONSchemaProps{
			"connection": {Type: "string"},
			"timeToLive": {Type: "string"},
			"vms": {
				Items: &extv1.JSONSchemaPropsOrArray{
					Schema: &extv1.JSONSchemaProps{
						Description: "OVirtVM aligns with maintained UI interface",
						Type:        objectType,
						Properties: map[string]extv1.JSONSchemaProps{
							"cluster": {Type: "string"},
							"detail": {
								Description: "OVirtVMDetail contains ovirt vm details as json string",
								Type:        objectType,
								Properties: map[string]extv1.JSONSchemaProps{
									"raw": {Type: "string"},
								},
							},
							"detailRequest": {Type: "boolean"},
							"id":            {Type: "string"},
							"name":          {Type: "string"},
						},
						Required: []string{"cluster", "id", "name"},
					},
				},
				Type: "array",
			},
		},
	}

	versionSchema.OpenAPIV3Schema.Properties["status"] = extv1.JSONSchemaProps{
		Description: "OVirtProviderStatus defines the observed state of OVirtProvider",
		Type:        objectType,
		Properties: map[string]extv1.JSONSchemaProps{
			"phase": {
				Description: "VirtualMachineProviderPhase defines provider phase",
				Type:        "string",
			},
		},
	}

	names := extv1.CustomResourceDefinitionNames{
		Plural:   "ovirtproviders",
		Singular: "ovirtprovider",
		Kind:     "OVirtProvider",
		ListKind: "OVirtProviderList",
	}

	return getCrd("ovirtproviders."+vmimportv1beta1.SchemeGroupVersion.Group, versionSchema, names)
}

func GetOperatorCR() *hcov1beta1.HyperConverged {
	// TODO: better handle defaults
	// on a real cluster the defaulting mechanism is properly
	// ensured by the APIServer according to defaults set
	// in the OpenAPIv3 specification on the CRD.
	// With unit tests on a mock client or locally generating
	// templates we cannot relay on that mechanism and we
	// have to keep this up to date.

	bandwidthPerMigration := "64Mi"
	completionTimeoutPerGiB := int64(800)
	parallelMigrationsPerCluster := uint32(5)
	parallelOutboundMigrationsPerNode := uint32(2)
	progressTimeout := int64(150)

	batchEvictionSize := 10
	batchEvictionInterval := metav1.Duration{Duration: 1 * time.Minute}

	return &hcov1beta1.HyperConverged{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hco.kubevirt.io/v1beta1",
			Kind:       "HyperConverged",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
		Spec: hcov1beta1.HyperConvergedSpec{
			CertConfig: hcov1beta1.HyperConvergedCertConfig{
				CA: hcov1beta1.CertRotateConfigCA{
					Duration:    metav1.Duration{Duration: 48 * time.Hour},
					RenewBefore: metav1.Duration{Duration: 24 * time.Hour},
				},
				Server: hcov1beta1.CertRotateConfigServer{
					Duration:    metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: metav1.Duration{Duration: 12 * time.Hour},
				},
			},
			FeatureGates: hcov1beta1.HyperConvergedFeatureGates{
				WithHostPassthroughCPU: false,
				SRIOVLiveMigration:     true,
			},
			LiveMigrationConfig: hcov1beta1.LiveMigrationConfigurations{
				BandwidthPerMigration:             &bandwidthPerMigration,
				CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
				ParallelMigrationsPerCluster:      &parallelMigrationsPerCluster,
				ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNode,
				ProgressTimeout:                   &progressTimeout,
			},
			WorkloadUpdateStrategy: &hcov1beta1.HyperConvergedWorkloadUpdateStrategy{
				WorkloadUpdateMethods: []string{"LiveMigrate", "Evict"},
				BatchEvictionSize:     &batchEvictionSize,
				BatchEvictionInterval: &batchEvictionInterval,
			},
			LocalStorageClassName: "",
		},
	}
}

// GetInstallStrategyBase returns the basics of an HCO InstallStrategy
func GetInstallStrategyBase(params *DeploymentOperatorParams) *csvv1alpha1.StrategyDetailsDeployment {
	return &csvv1alpha1.StrategyDetailsDeployment{

		DeploymentSpecs: []csvv1alpha1.StrategyDeploymentSpec{
			csvv1alpha1.StrategyDeploymentSpec{
				Name:  hcoDeploymentName,
				Spec:  GetDeploymentSpecOperator(params),
				Label: getLabels(hcoName, params.HcoKvIoVersion),
			},
			csvv1alpha1.StrategyDeploymentSpec{
				Name:  hcoWhDeploymentName,
				Spec:  GetDeploymentSpecWebhook(params.Namespace, params.WebhookImage, params.ImagePullPolicy, params.HcoKvIoVersion, params.Env),
				Label: getLabels(hcoNameWebhook, params.HcoKvIoVersion),
			},
		},
		Permissions: []csvv1alpha1.StrategyDeploymentPermissions{},
		ClusterPermissions: []csvv1alpha1.StrategyDeploymentPermissions{
			csvv1alpha1.StrategyDeploymentPermissions{
				ServiceAccountName: hcoName,
				Rules:              GetClusterPermissions(),
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
			"apiVersion": "hco.kubevirt.io/v1beta1",
			"kind":       "HyperConverged",
			"metadata": map[string]interface{}{
				"name":      packageName,
				"namespace": params.Namespace,
				"annotations": map[string]string{
					"deployOVS": "false",
				},
			},
			"spec": map[string]interface{}{},
		})

	sideEffect := admissionregistrationv1.SideEffectClassNone
	// Explicitly fail on unvalidated (for any reason) requests:
	// this can make removing HCO CR harder if HCO webhook is not able
	// to really validate the requests.
	// In that case the user can only directly remove the
	// ValidatingWebhookConfiguration object first (eventually bypassing the OLM if needed).
	failurePolicy := admissionregistrationv1.Fail
	webhookPath := util.HCOWebhookPath
	// TODO: temporary workaround for https://bugzilla.redhat.com/1868712
	// currently OLM is going to periodically kill HCO, due to that some request can got lost with a timeout error
	// using a really high timeout can mitigate it giving more time to a new HCO instance.
	// Please remove this once https://bugzilla.redhat.com/1868712 is not biting us anymore
	var webhookTimeout int32 = 30

	validatingWebhook := csvv1alpha1.WebhookDescription{
		GenerateName:            util.HcoValidatingWebhook,
		Type:                    csvv1alpha1.ValidatingAdmissionWebhook,
		DeploymentName:          hcoWhDeploymentName,
		ContainerPort:           util.WebhookPort,
		AdmissionReviewVersions: []string{"v1beta1", "v1"},
		SideEffects:             &sideEffect,
		FailurePolicy:           &failurePolicy,
		TimeoutSeconds:          &webhookTimeout,
		Rules: []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
					admissionregistrationv1.Delete,
					admissionregistrationv1.Update,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{util.APIVersionGroup},
					APIVersions: []string{util.APIVersionAlpha, util.APIVersionBeta},
					Resources:   []string{"hyperconvergeds"},
				},
			},
		},
		WebhookPath: &webhookPath,
	}

	mutatingWebhookSideEffects := admissionregistrationv1.SideEffectClassNoneOnDryRun
	mutatingWebhookPath := util.HCONSWebhookPath

	mutatingWebhook := csvv1alpha1.WebhookDescription{
		GenerateName:            util.HcoMutatingWebhookNS,
		Type:                    csvv1alpha1.MutatingAdmissionWebhook,
		DeploymentName:          hcoWhDeploymentName,
		ContainerPort:           util.WebhookPort,
		AdmissionReviewVersions: []string{"v1beta1", "v1"},
		SideEffects:             &mutatingWebhookSideEffects,
		FailurePolicy:           &failurePolicy,
		TimeoutSeconds:          &webhookTimeout,
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"name": params.Namespace},
		},
		Rules: []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Delete,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"namespaces"},
				},
			},
		},
		WebhookPath: &mutatingWebhookPath,
	}

	return &csvv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operators.coreos.com/v1alpha1",
			Kind:       "ClusterServiceVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.v%v", params.Name, params.Version.String()),
			Namespace: "placeholder",
			Annotations: map[string]string{
				"alm-examples":   string(almExamples),
				"capabilities":   "Full Lifecycle",
				"certified":      "false",
				"categories":     "OpenShift Optional",
				"containerImage": params.Image,
				"createdAt":      time.Now().Format("2006-01-02 15:04:05"),
				"description":    params.MetaDescription,
				"repository":     "https://github.com/kubevirt/hyperconverged-cluster-operator",
				"support":        "false",
				"operatorframework.io/suggested-namespace":       params.Namespace,
				"operators.openshift.io/infrastructure-features": `["Disconnected"]`,
				"operatorframework.io/initialization-resource":   string(almExamples),
			},
		},
		Spec: csvv1alpha1.ClusterServiceVersionSpec{
			DisplayName: params.DisplayName,
			Description: params.Description,
			Keywords:    []string{"KubeVirt", "Virtualization"},
			Version:     csvVersion.OperatorVersion{Version: params.Version},
			Replaces:    params.Replaces,
			Maintainers: []csvv1alpha1.Maintainer{
				csvv1alpha1.Maintainer{
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
				csvv1alpha1.AppLink{
					Name: kubevirtProjectName,
					URL:  "https://kubevirt.io",
				},
				csvv1alpha1.AppLink{
					Name: "Source Code",
					URL:  "https://github.com/kubevirt/hyperconverged-cluster-operator",
				},
			},
			Icon: []csvv1alpha1.Icon{
				csvv1alpha1.Icon{
					MediaType: "image/png",
					Data:      "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",
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
				csvv1alpha1.InstallMode{
					Type:      csvv1alpha1.InstallModeTypeOwnNamespace,
					Supported: false,
				},
				csvv1alpha1.InstallMode{
					Type:      csvv1alpha1.InstallModeTypeSingleNamespace,
					Supported: false,
				},
				csvv1alpha1.InstallMode{
					Type:      csvv1alpha1.InstallModeTypeMultiNamespace,
					Supported: false,
				},
				csvv1alpha1.InstallMode{
					Type:      csvv1alpha1.InstallModeTypeAllNamespaces,
					Supported: true,
				},
			},
			// Skip this in favor of having a separate function to get
			// the actual StrategyDetailsDeployment when merging CSVs
			InstallStrategy:    csvv1alpha1.NamedInstallStrategy{},
			WebhookDefinitions: []csvv1alpha1.WebhookDescription{validatingWebhook, mutatingWebhook},
			CustomResourceDefinitions: csvv1alpha1.CustomResourceDefinitions{
				Owned: []csvv1alpha1.CRDDescription{
					{
						Name:        "hyperconvergeds.hco.kubevirt.io",
						Version:     util.CurrentAPIVersion,
						Kind:        "HyperConverged",
						DisplayName: params.CrdDisplay + " Deployment",
						Description: "Represents the deployment of " + params.CrdDisplay,
						// TODO: move this to annotations on hyperconverged_types.go once kubebuilder
						// properly supports SpecDescriptors as the operator-sdk already does
						SpecDescriptors: []csvv1alpha1.SpecDescriptor{
							{
								DisplayName: "Infra components node affinity",
								Description: "nodeAffinity describes node affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.nodeAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:nodeAffinity",
								},
							},
							{
								DisplayName: "Infra components pod affinity",
								Description: "podAffinity describes pod affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.podAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:podAffinity",
								},
							},
							{
								DisplayName: "Infra components pod anti-affinity",
								Description: "podAntiAffinity describes pod anti affinity scheduling rules for the infra pods.",
								Path:        "infra.nodePlacement.affinity.podAntiAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:podAntiAffinity",
								},
							},
							{
								DisplayName: "Workloads components node affinity",
								Description: "nodeAffinity describes node affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.nodeAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:nodeAffinity",
								},
							},
							{
								DisplayName: "Workloads components pod affinity",
								Description: "podAffinity describes pod affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.podAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:podAffinity",
								},
							},
							{
								DisplayName: "Workloads components pod anti-affinity",
								Description: "podAntiAffinity describes pod anti affinity scheduling rules for the workloads pods.",
								Path:        "workloads.nodePlacement.affinity.podAntiAffinity",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:podAntiAffinity",
								},
							},
							{
								DisplayName: "HIDDEN FIELDS - operator version",
								Description: "HIDDEN FIELDS - operator version.",
								Path:        "version",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:hidden",
								},
							},
						},
						StatusDescriptors: []csvv1alpha1.StatusDescriptor{},
					},
					// TODO: remove once oVirt and VMware providers are removed from HCO
					{
						Name:        "v2vvmwares.v2v.kubevirt.io",
						Version:     "v1alpha1",
						Kind:        "V2VVmware",
						DisplayName: "V2V Vmware",
						Description: "V2V Vmware",
					},
					{
						Name:        "ovirtproviders.v2v.kubevirt.io",
						Version:     "v1alpha1",
						Kind:        "OVirtProvider",
						DisplayName: "V2V oVirt",
						Description: "V2V oVirt",
					},
				},
				Required: []csvv1alpha1.CRDDescription{},
			},
		},
	}
}

func InjectVolumesForWebHookCerts(deploy *appsv1.Deployment) {
	defaultMode := int32(420)
	volume := v1.Volume{
		Name: certVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  deploy.Name + "-service-cert",
				DefaultMode: &defaultMode,
				Items: []corev1.KeyToPath{
					{
						Key:  "tls.crt",
						Path: hcov1beta1.WebhookCertName,
					},
					{
						Key:  "tls.key",
						Path: hcov1beta1.WebhookKeyName,
					},
				},
			},
		},
	}
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)

	for index, container := range deploy.Spec.Template.Spec.Containers {
		deploy.Spec.Template.Spec.Containers[index].VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      "apiservice-cert",
				MountPath: hcov1beta1.DefaultWebhookCertDir,
			})
	}
}

func getReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: hcoutil.ReadinessEndpointName,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: hcoutil.HealthProbePort,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    1,
	}
}

func getLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: hcoutil.LivenessEndpointName,
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: hcoutil.HealthProbePort,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       5,
		FailureThreshold:    1,
	}
}

func getPolicyRules(words ...string) []string {
	return words
}

func getCrd(crdName string, versionSchema *extv1.CustomResourceValidation, names extv1.CustomResourceDefinitionNames) *extv1.CustomResourceDefinition {
	return &extv1.CustomResourceDefinition{
		TypeMeta: crdMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: getCrdSpec(versionSchema, names),
	}
}

func getCrdSpec(schema *extv1.CustomResourceValidation, names extv1.CustomResourceDefinitionNames) extv1.CustomResourceDefinitionSpec {
	return extv1.CustomResourceDefinitionSpec{
		Group: vmimportv1beta1.SchemeGroupVersion.Group,
		Scope: "Namespaced",
		Versions: []extv1.CustomResourceDefinitionVersion{
			{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Subresources: &extv1.CustomResourceSubresources{
					Status: &extv1.CustomResourceSubresourceStatus{},
				},
				Schema: schema,
			},
		},
		Names: names,
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
