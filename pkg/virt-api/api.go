/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package virt_api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/openapi"
	"kubevirt.io/kubevirt/pkg/version"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	mutating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook"
	validating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	// Default port that virt-api listens on.
	defaultPort = 443

	// Default address that virt-api listens on.
	defaultHost = "0.0.0.0"

	// selfsigned cert secret name
	virtApiCertSecretName = "kubevirt-virt-api-certs"

	virtWebhookValidator = "virt-api-validator"
	virtWebhookMutator   = "virt-api-mutator"

	virtApiServiceName = "virt-api"

	vmiCreateValidatePath       = "/virtualmachineinstances-validate-create"
	vmiUpdateValidatePath       = "/virtualmachineinstances-validate-update"
	vmValidatePath              = "/virtualmachines-validate"
	vmirsValidatePath           = "/virtualmachinereplicaset-validate"
	vmipresetValidatePath       = "/vmipreset-validate"
	migrationCreateValidatePath = "/migration-validate-create"
	migrationUpdateValidatePath = "/migration-validate-update"

	vmiMutatePath = "/virtualmachineinstances-mutate"

	certBytesValue        = "cert-bytes"
	keyBytesValue         = "key-bytes"
	signingCertBytesValue = "signing-cert-bytes"
)

type VirtApi interface {
	Compose()
	Run()
	AddFlags()
	ConfigureOpenAPIService()
	Execute()
}

type virtAPIApp struct {
	service.ServiceListen
	SwaggerUI        string
	SubresourcesOnly bool
	virtCli          kubecli.KubevirtClient
	aggregatorClient *aggregatorclient.Clientset
	authorizor       rest.VirtApiAuthorizor
	certsDirectory   string

	signingCertBytes           []byte
	certBytes                  []byte
	keyBytes                   []byte
	clientCABytes              []byte
	requestHeaderClientCABytes []byte
	certFile                   string
	keyFile                    string
	clientCAFile               string
	signingCertFile            string
	namespace                  string
	tlsConfig                  *tls.Config
}

var _ service.Service = &virtAPIApp{}

func NewVirtApi() VirtApi {

	app := &virtAPIApp{}
	app.BindAddress = defaultHost
	app.Port = defaultPort

	return app
}

func (app *virtAPIApp) Execute() {
	virtconfig.Init()

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	authorizor, err := rest.NewAuthorizor()
	if err != nil {
		panic(err)
	}

	config, err := kubecli.GetConfig()
	if err != nil {
		panic(err)
	}

	app.aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

	app.authorizor = authorizor

	app.virtCli = virtCli

	app.certsDirectory, err = ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	app.namespace, err = util.GetNamespace()
	if err != nil {
		panic(err)
	}

	app.Compose()
	app.ConfigureOpenAPIService()
	app.Run()
}

func subresourceAPIGroup() metav1.APIGroup {
	apiGroup := metav1.APIGroup{
		Name: "subresource.kubevirt.io",
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: v1.SubresourceGroupVersion.Group + "/" + v1.SubresourceGroupVersion.Version,
			Version:      v1.SubresourceGroupVersion.Version,
		},
	}
	apiGroup.Versions = append(apiGroup.Versions, metav1.GroupVersionForDiscovery{
		GroupVersion: v1.SubresourceGroupVersion.Group + "/" + v1.SubresourceGroupVersion.Version,
		Version:      v1.SubresourceGroupVersion.Version,
	})
	apiGroup.ServerAddressByClientCIDRs = append(apiGroup.ServerAddressByClientCIDRs, metav1.ServerAddressByClientCIDR{
		ClientCIDR:    "0.0.0.0/0",
		ServerAddress: "",
	})
	apiGroup.Kind = "APIGroup"
	return apiGroup
}

func (app *virtAPIApp) composeSubresources() {

	subresourcesvmGVR := schema.GroupVersionResource{Group: v1.SubresourceGroupVersion.Group, Version: v1.SubresourceGroupVersion.Version, Resource: "virtualmachines"}
	subresourcesvmiGVR := schema.GroupVersionResource{Group: v1.SubresourceGroupVersion.Group, Version: v1.SubresourceGroupVersion.Version, Resource: "virtualmachineinstances"}

	subws := new(restful.WebService)
	subws.Doc("The KubeVirt Subresource API.")
	subws.Path(rest.GroupVersionBasePath(v1.SubresourceGroupVersion))

	subresourceApp := &rest.SubresourceAPIApp{
		VirtCli: app.virtCli,
	}

	subws.Route(subws.PUT(rest.ResourcePath(subresourcesvmGVR)+rest.SubResourcePath("restart")).
		To(subresourceApp.RestartVMRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("restart").
		Doc("Restart a VirtualMachine object.").
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusNotFound, "Not Found", nil))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmiGVR) + rest.SubResourcePath("console")).
		To(subresourceApp.ConsoleRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("console").
		Doc("Open a websocket connection to a serial console on the specified VirtualMachineInstance."))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmiGVR) + rest.SubResourcePath("vnc")).
		To(subresourceApp.VNCRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("vnc").
		Doc("Open a websocket connection to connect to VNC on the specified VirtualMachineInstance."))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmiGVR) + rest.SubResourcePath("test")).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteHeader(http.StatusOK)
		}).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("test").
		Doc("Test endpoint verifying apiserver connectivity."))

	subws.Route(subws.GET(rest.SubResourcePath("version")).Produces(restful.MIME_JSON).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteAsJson(version.Get())
		}).Operation("version"))

	subws.Route(subws.GET(rest.SubResourcePath("healthz")).
		To(healthz.KubeConnectionHealthzFunc).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Operation("checkHealth").
		Doc("Health endpoint").
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusInternalServerError, "Unhealthy", nil))

	// Return empty api resource list.
	// K8s expects to be able to retrieve a resource list for each aggregated
	// app in order to discover what resources it provides. Without returning
	// an empty list here, there's a bug in the k8s resource discovery that
	// breaks kubectl's ability to reference short names for resources.
	subws.Route(subws.GET("/").
		Produces(restful.MIME_JSON).Writes(metav1.APIResourceList{}).
		To(func(request *restful.Request, response *restful.Response) {
			list := &metav1.APIResourceList{}

			list.Kind = "APIResourceList"
			list.GroupVersion = v1.SubresourceGroupVersion.Group + "/" + v1.SubresourceGroupVersion.Version
			list.APIVersion = v1.SubresourceGroupVersion.Version
			list.APIResources = []metav1.APIResource{
				{
					Name:       "virtualmachineinstances/vnc",
					Namespaced: true,
				},
				{
					Name:       "virtualmachineinstances/restart",
					Namespaced: true,
				},
				{
					Name:       "virtualmachineinstances/console",
					Namespaced: true,
				},
			}

			response.WriteAsJson(list)
		}).
		Operation("getAPIResources").
		Doc("Get a KubeVirt API resources").
		Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	restful.Add(subws)

	ws := new(restful.WebService)

	// K8s needs the ability to query the root paths
	ws.Route(ws.GET("/").
		Produces(restful.MIME_JSON).Writes(metav1.RootPaths{}).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteAsJson(&metav1.RootPaths{
				Paths: []string{
					"/apis",
					"/apis/",
					rest.GroupBasePath(v1.SubresourceGroupVersion),
					rest.GroupVersionBasePath(v1.SubresourceGroupVersion),
					"/openapi/v2",
				},
			})
		}).
		Operation("getRootPaths").
		Doc("Get KubeVirt API root paths").
		Returns(http.StatusOK, "OK", metav1.RootPaths{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	// K8s needs the ability to query info about a specific API group
	ws.Route(ws.GET(rest.GroupBasePath(v1.SubresourceGroupVersion)).
		Produces(restful.MIME_JSON).Writes(metav1.APIGroup{}).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteAsJson(subresourceAPIGroup())
		}).
		Operation("getAPIGroup").
		Doc("Get a KubeVirt API Group").
		Returns(http.StatusOK, "OK", metav1.APIGroup{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	// K8s needs the ability to query the list of API groups this endpoint supports
	ws.Route(ws.GET("apis").
		Produces(restful.MIME_JSON).Writes(metav1.APIGroupList{}).
		To(func(request *restful.Request, response *restful.Response) {
			list := &metav1.APIGroupList{}
			list.Kind = "APIGroupList"
			list.Groups = append(list.Groups, subresourceAPIGroup())
			response.WriteAsJson(list)
		}).
		Operation("getAPIGroupList").
		Doc("Get a KubeVirt API GroupList").
		Returns(http.StatusOK, "OK", metav1.APIGroupList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	once := sync.Once{}
	ws.Route(ws.GET("openapi/v2").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		To(func(request *restful.Request, response *restful.Response) {
			var openapispec *spec.Swagger
			once.Do(func() {
				openapispec = openapi.LoadOpenAPISpec([]*restful.WebService{ws, subws})
				openapispec.Info.Version = version.Get().String()
			})
			response.WriteAsJson(openapispec)
		}))

	restful.Add(ws)
}

func (app *virtAPIApp) Compose() {

	app.composeSubresources()

	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())
	restful.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		allowed, reason, err := app.authorizor.Authorize(req)
		if err != nil {

			log.Log.Reason(err).Error("internal error during auth request")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		} else if allowed {
			// request is permitted, so proceed with filter chain.
			chain.ProcessFilter(req, resp)
			return
		}
		resp.WriteErrorString(http.StatusUnauthorized, reason)
	})
}

func (app *virtAPIApp) ConfigureOpenAPIService() {
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(openapi.CreateOpenAPIConfig(restful.RegisteredWebServices())))
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir(app.SwaggerUI))))
}

func deserializeStrings(in string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	var ret []string
	if err := json.Unmarshal([]byte(in), &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (app *virtAPIApp) getClientCert() error {
	authConfigMap, err := app.virtCli.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return err
	}

	clientCA, ok := authConfigMap.Data["client-ca-file"]
	if !ok {
		return fmt.Errorf("client-ca-file value not found in auth config map.")
	}
	app.clientCABytes = []byte(clientCA)

	// request-header-ca-file doesn't always exist in all deployments.
	// set it if the value is set though.
	requestHeaderClientCA, ok := authConfigMap.Data["requestheader-client-ca-file"]
	if ok {
		app.requestHeaderClientCABytes = []byte(requestHeaderClientCA)
	}

	// This config map also contains information about what
	// headers our authorizor should inspect
	headers, ok := authConfigMap.Data["requestheader-username-headers"]
	if ok {
		headerList, err := deserializeStrings(headers)
		if err != nil {
			return err
		}
		app.authorizor.AddUserHeaders(headerList)
	}

	headers, ok = authConfigMap.Data["requestheader-group-headers"]
	if ok {
		headerList, err := deserializeStrings(headers)
		if err != nil {
			return err
		}
		app.authorizor.AddGroupHeaders(headerList)
	}

	headers, ok = authConfigMap.Data["requestheader-extra-headers-prefix"]
	if ok {
		headerList, err := deserializeStrings(headers)
		if err != nil {
			return err
		}
		app.authorizor.AddExtraPrefixHeaders(headerList)
	}
	return nil
}

func (app *virtAPIApp) getSelfSignedCert() error {
	var ok bool

	generateCerts := false
	secret, err := app.virtCli.CoreV1().Secrets(app.namespace).Get(virtApiCertSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			generateCerts = true
		} else {
			return err
		}
	}

	if generateCerts {
		// Generate new certs if secret doesn't already exist
		caKeyPair, _ := triple.NewCA("kubevirt.io")
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			"virt-api."+app.namespace+".pod.cluster.local",
			"virt-api",
			app.namespace,
			"cluster.local",
			nil,
			nil,
		)

		app.keyBytes = cert.EncodePrivateKeyPEM(keyPair.Key)
		app.certBytes = cert.EncodeCertPEM(keyPair.Cert)
		app.signingCertBytes = cert.EncodeCertPEM(caKeyPair.Cert)

		secret := k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      virtApiCertSecretName,
				Namespace: app.namespace,
				Labels: map[string]string{
					v1.AppLabel: "virt-api-aggregator",
				},
			},
			Type: "Opaque",
			Data: map[string][]byte{
				certBytesValue:        app.certBytes,
				keyBytesValue:         app.keyBytes,
				signingCertBytesValue: app.signingCertBytes,
			},
		}
		_, err := app.virtCli.CoreV1().Secrets(app.namespace).Create(&secret)
		if err != nil {
			return err
		}
	} else {
		// retrieve self signed cert info from secret

		app.certBytes, ok = secret.Data[certBytesValue]
		if !ok {
			return fmt.Errorf("%s value not found in %s virt-api secret", certBytesValue, virtApiCertSecretName)
		}
		app.keyBytes, ok = secret.Data[keyBytesValue]
		if !ok {
			return fmt.Errorf("%s value not found in %s virt-api secret", keyBytesValue, virtApiCertSecretName)
		}
		app.signingCertBytes, ok = secret.Data[signingCertBytesValue]
		if !ok {
			return fmt.Errorf("%s value not found in %s virt-api secret", signingCertBytesValue, virtApiCertSecretName)
		}
	}
	return nil
}

func (app *virtAPIApp) createWebhook() error {
	err := app.createValidatingWebhook()
	if err != nil {
		return err
	}
	err = app.createMutatingWebhook()
	if err != nil {
		return err
	}
	return nil
}

func (app *virtAPIApp) validatingWebhooks() []admissionregistrationv1beta1.Webhook {

	vmiPathCreate := vmiCreateValidatePath
	vmiPathUpdate := vmiUpdateValidatePath
	vmPath := vmValidatePath
	vmirsPath := vmirsValidatePath
	vmipresetPath := vmipresetValidatePath
	migrationCreatePath := migrationCreateValidatePath
	migrationUpdatePath := migrationUpdateValidatePath
	failurePolicy := admissionregistrationv1beta1.Fail

	webHooks := []admissionregistrationv1beta1.Webhook{
		{
			Name:          "virtualmachineinstances-create-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstances"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmiPathCreate,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "virtualmachineinstances-update-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Update,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstances"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmiPathUpdate,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "virtualmachine-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
					admissionregistrationv1beta1.Update,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineGroupVersionKind.Version},
					Resources:   []string{"virtualmachines"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmPath,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "virtualmachinereplicaset-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
					admissionregistrationv1beta1.Update,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstancereplicasets"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmirsPath,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "virtualmachinepreset-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
					admissionregistrationv1beta1.Update,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstancePresetGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstancepresets"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmipresetPath,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "migration-create-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceMigrationGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstancemigrations"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &migrationCreatePath,
				},
				CABundle: app.signingCertBytes,
			},
		},
		{
			Name:          "migration-update-validator.kubevirt.io",
			FailurePolicy: &failurePolicy,
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Update,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceMigrationGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstancemigrations"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &migrationUpdatePath,
				},
				CABundle: app.signingCertBytes,
			},
		},
	}

	return webHooks
}

func (app *virtAPIApp) createValidatingWebhook() error {
	registerWebhook := false
	webhookRegistration, err := app.virtCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(virtWebhookValidator, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			fmt.Println(err)
			registerWebhook = true
		} else {
			return err
		}
	}
	webHooks := app.validatingWebhooks()

	if registerWebhook {
		_, err := app.virtCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: virtWebhookValidator,
				Labels: map[string]string{
					v1.AppLabel: virtWebhookValidator,
				},
			},
			Webhooks: webHooks,
		})
		if err != nil {
			return err
		}
	} else {
		for _, webhook := range webhookRegistration.Webhooks {
			if webhook.ClientConfig.Service != nil && webhook.ClientConfig.Service.Namespace != app.namespace {
				return fmt.Errorf("ValidatingAdmissionWebhook [%s] is already registered using services endpoints in a different namespace. Existing webhook registration must be deleted before virt-api can proceed.", virtWebhookValidator)
			}
		}

		// update registered webhook with our data
		webhookRegistration.Webhooks = webHooks

		_, err := app.virtCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Update(webhookRegistration)
		if err != nil {
			return err
		}
	}

	http.HandleFunc(vmiCreateValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMICreate(w, r)
	})
	http.HandleFunc(vmiUpdateValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMIUpdate(w, r)
	})
	http.HandleFunc(vmValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMs(w, r)
	})
	http.HandleFunc(vmirsValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMIRS(w, r)
	})
	http.HandleFunc(vmipresetValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeVMIPreset(w, r)
	})
	http.HandleFunc(migrationCreateValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeMigrationCreate(w, r)
	})
	http.HandleFunc(migrationUpdateValidatePath, func(w http.ResponseWriter, r *http.Request) {
		validating_webhook.ServeMigrationUpdate(w, r)
	})

	return nil
}

func (app *virtAPIApp) mutatingWebhooks() []admissionregistrationv1beta1.Webhook {
	vmiPath := vmiMutatePath
	webHooks := []admissionregistrationv1beta1.Webhook{
		{
			Name: "virtualmachineinstances-mutator.kubevirt.io",
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{v1.GroupName},
					APIVersions: []string{v1.VirtualMachineInstanceGroupVersionKind.Version},
					Resources:   []string{"virtualmachineinstances"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: app.namespace,
					Name:      virtApiServiceName,
					Path:      &vmiPath,
				},
				CABundle: app.signingCertBytes,
			},
		},
	}
	return webHooks
}

func (app *virtAPIApp) createMutatingWebhook() error {
	registerWebhook := false

	webhookRegistration, err := app.virtCli.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(virtWebhookMutator, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			registerWebhook = true
		} else {
			return err
		}
	}
	webHooks := app.mutatingWebhooks()

	if registerWebhook {
		_, err := app.virtCli.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(&admissionregistrationv1beta1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: virtWebhookMutator,
				Labels: map[string]string{
					v1.AppLabel: virtWebhookMutator,
				},
			},
			Webhooks: webHooks,
		})
		if err != nil {
			return err
		}
	} else {

		for _, webhook := range webhookRegistration.Webhooks {
			if webhook.ClientConfig.Service != nil && webhook.ClientConfig.Service.Namespace != app.namespace {
				return fmt.Errorf("MutatingAdmissionWebhook [%s] is already registered using services endpoints in a different namespace. Existing webhook registration must be deleted before virt-api can proceed.", virtWebhookMutator)
			}
		}

		// update registered webhook with our data
		webhookRegistration.Webhooks = webHooks

		_, err := app.virtCli.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Update(webhookRegistration)
		if err != nil {
			return err
		}
	}

	http.HandleFunc(vmiMutatePath, func(w http.ResponseWriter, r *http.Request) {
		mutating_webhook.ServeVMIs(w, r)
	})
	return nil
}

func (app *virtAPIApp) subresourceApiservice() *apiregistrationv1beta1.APIService {

	subresourceAggregatedApiName := v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName

	return &apiregistrationv1beta1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subresourceAggregatedApiName,
			Namespace: app.namespace,
			Labels: map[string]string{
				v1.AppLabel: "virt-api-aggregator",
			},
		},
		Spec: apiregistrationv1beta1.APIServiceSpec{
			Service: &apiregistrationv1beta1.ServiceReference{
				Namespace: app.namespace,
				Name:      virtApiServiceName,
			},
			Group:                v1.SubresourceGroupName,
			Version:              v1.SubresourceGroupVersion.Version,
			CABundle:             app.signingCertBytes,
			GroupPriorityMinimum: 1000,
			VersionPriority:      15,
		},
	}

}

func (app *virtAPIApp) createSubresourceApiservice() error {

	subresourceApiservice := app.subresourceApiservice()

	registerApiService := false

	apiService, err := app.aggregatorClient.ApiregistrationV1beta1().APIServices().Get(subresourceApiservice.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			registerApiService = true
		} else {
			return err
		}
	}

	if registerApiService {
		_, err = app.aggregatorClient.ApiregistrationV1beta1().APIServices().Create(app.subresourceApiservice())
		if err != nil {
			return err
		}
	} else {
		if apiService.Spec.Service != nil && apiService.Spec.Service.Namespace != app.namespace {
			return fmt.Errorf("apiservice [%s] is already registered in a different namespace. Existing apiservice registration must be deleted before virt-api can proceed.", subresourceApiservice.Name)
		}

		// Always update spec to latest.
		apiService.Spec = app.subresourceApiservice().Spec
		_, err := app.aggregatorClient.ApiregistrationV1beta1().APIServices().Update(apiService)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *virtAPIApp) setupTLS(fs Filesystem) error {

	app.keyFile = filepath.Join(app.certsDirectory, "/key.pem")
	app.certFile = filepath.Join(app.certsDirectory, "/cert.pem")
	app.signingCertFile = filepath.Join(app.certsDirectory, "/signingCert.pem")
	app.clientCAFile = filepath.Join(app.certsDirectory, "/clientCA.crt")

	// Write the certs to disk
	err := fs.WriteFile(app.clientCAFile, app.clientCABytes, 0600)
	if err != nil {
		return err
	}

	if len(app.requestHeaderClientCABytes) != 0 {
		f, err := fs.OpenFile(app.clientCAFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(app.requestHeaderClientCABytes)
		if err != nil {
			return err
		}
	}

	err = fs.WriteFile(app.keyFile, app.keyBytes, 0600)
	if err != nil {
		return err
	}
	err = fs.WriteFile(app.certFile, app.certBytes, 0600)
	if err != nil {
		return err
	}
	err = fs.WriteFile(app.signingCertFile, app.signingCertBytes, 0600)
	if err != nil {
		return err
	}

	// create the client CA pool.
	// This ensures we're talking to the k8s api server
	pool, err := cert.NewPool(app.clientCAFile)
	if err != nil {
		return err
	}

	app.tlsConfig = &tls.Config{
		ClientCAs: pool,
		// A VerifyClientCertIfGiven request means we're not guaranteed
		// a client has been authenticated unless they provide a peer
		// cert.
		//
		// Make sure to verify in subresource endpoint that peer cert
		// was provided before processing request. If the peer cert is
		// given on the connection, then we can be guaranteed that it
		// was signed by the client CA in our pool.
		//
		// There is another ClientAuth type called 'RequireAndVerifyClientCert'
		// We can't use this type here because during the aggregated api status
		// check it attempts to hit '/' on our api endpoint to verify an http
		// response is given. That status request won't send a peer cert regardless
		// if the TLS handshake requests it. As a result, the TLS handshake fails
		// and our aggregated endpoint never becomes available.
		ClientAuth: tls.VerifyClientCertIfGiven,
	}
	app.tlsConfig.BuildNameToCertificate()
	return nil
}

func (app *virtAPIApp) startTLS() error {

	errors := make(chan error)

	err := app.setupTLS(IOUtil{})
	if err != nil {
		return err
	}

	// start TLS server
	go func() {
		http.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:      fmt.Sprintf("%s:%d", app.BindAddress, app.Port),
			TLSConfig: app.tlsConfig,
		}

		errors <- server.ListenAndServeTLS(app.certFile, app.keyFile)
	}()

	// wait for server to exit
	return <-errors
}

func (app *virtAPIApp) Run() {
	// get client Cert
	err := app.getClientCert()
	if err != nil {
		panic(err)
	}

	// Get/Set selfsigned cert
	err = app.getSelfSignedCert()
	if err != nil {
		panic(err)
	}

	// Verify/create aggregator endpoint.
	err = app.createSubresourceApiservice()
	if err != nil {
		panic(err)
	}

	// Run informers for webhooks usage
	webhookInformers := webhooks.GetInformers()

	stopChan := make(chan struct{}, 1)
	defer close(stopChan)
	go webhookInformers.VMIInformer.Run(stopChan)
	go webhookInformers.VMIPresetInformer.Run(stopChan)
	go webhookInformers.NamespaceLimitsInformer.Run(stopChan)

	cache.WaitForCacheSync(stopChan,
		webhookInformers.VMIInformer.HasSynced,
		webhookInformers.VMIPresetInformer.HasSynced,
		webhookInformers.NamespaceLimitsInformer.HasSynced)

	// Verify/create webhook endpoint.
	err = app.createWebhook()
	if err != nil {
		panic(err)
	}

	// start TLS server
	err = app.startTLS()
	if err != nil {
		panic(err)
	}
}

func (app *virtAPIApp) AddFlags() {
	app.InitFlags()

	app.AddCommonFlags()

	flag.StringVar(&app.SwaggerUI, "swagger-ui", "third_party/swagger-ui",
		"swagger-ui location")
	flag.BoolVar(&app.SubresourcesOnly, "subresources-only", false,
		"Only serve subresource endpoints")
}
