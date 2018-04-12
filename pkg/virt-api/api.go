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
	"reflect"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	openapispec "github.com/go-openapi/spec"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/context"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const (
	// Default port that virt-api listens on.
	defaultPort = 443

	// Default address that virt-api listens on.
	defaultHost = "0.0.0.0"

	// selfsigned cert secret name
	virtApiCertSecretName = "kubevirt-virt-api-certs"

	certBytesValue        = "cert-bytes"
	keyBytesValue         = "key-bytes"
	signingCertBytesValue = "signing-cert-bytes"
)

type VirtApi interface {
	Compose()
	Run()
	AddFlags()
	ConfigureOpenAPIService()
}

type virtAPIApp struct {
	service.ServiceListen
	SwaggerUI        string
	SubresourcesOnly bool
	virtCli          kubecli.KubevirtClient
	authorizor       rest.VirtApiAuthorizor
	certsDirectory   string

	signingCertBytes           []byte
	certBytes                  []byte
	keyBytes                   []byte
	clientCABytes              []byte
	requestHeaderClientCABytes []byte
}

var _ service.Service = &virtAPIApp{}

func NewVirtApi() VirtApi {

	app := &virtAPIApp{}

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	authorizor, err := rest.NewAuthorizor()
	if err != nil {
		panic(err)
	}

	app.authorizor = authorizor

	app.virtCli = virtCli
	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.certsDirectory, err = ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}

	return app
}

func (app *virtAPIApp) composeResources(ctx context.Context) {

	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	vmrsGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinereplicasets"}
	vmpGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinepresets"}
	ovmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "offlinevirtualmachines"}

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VirtualMachine{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachineList{})
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmrsGVR, &v1.VirtualMachineReplicaSet{}, v1.VMReplicaSetGroupVersionKind.Kind, &v1.VirtualMachineReplicaSetList{})
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmpGVR, &v1.VirtualMachinePreset{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachinePresetList{})
	if err != nil {
		panic(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, ovmGVR, &v1.OfflineVirtualMachine{}, v1.OfflineVirtualMachineGroupVersionKind.Kind, &v1.OfflineVirtualMachineList{})
	if err != nil {
		panic(err)
	}

	restful.Add(ws)

	ws.Route(ws.GET("/healthz").
		To(healthz.KubeConnectionHealthzFunc).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Operation("checkHealth").
		Doc("Health endpoint").
		Returns(http.StatusOK, "OK", nil).
		Returns(http.StatusInternalServerError, "Unhealthy", nil))
	ws, err = rest.ResourceProxyAutodiscovery(ctx, vmGVR)
	if err != nil {
		panic(err)
	}

	restful.Add(ws)
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

func (app *virtAPIApp) composeSubresources(ctx context.Context) {

	subresourcesvmGVR := schema.GroupVersionResource{Group: v1.SubresourceGroupVersion.Group, Version: v1.SubresourceGroupVersion.Version, Resource: "virtualmachines"}

	subws := new(restful.WebService)
	subws.Doc("The KubeVirt Subresource API.")
	subws.Path(rest.GroupVersionBasePath(v1.SubresourceGroupVersion))

	subresourceApp := &rest.SubresourceAPIApp{
		VirtCli: app.virtCli,
	}

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("console")).
		To(subresourceApp.ConsoleRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("console").
		Doc("Open a websocket connection to a serial console on the specified VM."))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("vnc")).
		To(subresourceApp.VNCRequestHandler).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("vnc").
		Doc("Open a websocket connection to connect to VNC on the specified VM."))

	subws.Route(subws.GET(rest.ResourcePath(subresourcesvmGVR) + rest.SubResourcePath("test")).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteHeader(http.StatusOK)
		}).
		Param(rest.NamespaceParam(subws)).Param(rest.NameParam(subws)).
		Operation("test").
		Doc("Test endpoint verifying apiserver connectivity."))

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

			response.WriteAsJson(list)
		}).
		Operation("getAPIResources").
		Doc("Get a KubeVirt API resources").
		Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	restful.Add(subws)

	ws := new(restful.WebService)

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
		Operation("getAPIGroup").
		Doc("Get a KubeVirt API GroupList").
		Returns(http.StatusOK, "OK", metav1.APIGroupList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	restful.Add(ws)
}

func (app *virtAPIApp) Compose() {
	ctx := context.Background()

	if !app.SubresourcesOnly {
		app.composeResources(ctx)
	}
	app.composeSubresources(ctx)

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
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(CreateOpenAPIConfig()))
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir(app.SwaggerUI))))
}

func CreateOpenAPIConfig() restfulspec.Config {
	return restfulspec.Config{
		WebServices:    restful.RegisteredWebServices(),
		WebServicesURL: "",
		APIPath:        "/swaggerapi",
		PostBuildSwaggerObjectHandler: addInfoToSwaggerObject,
	}
}

func addInfoToSwaggerObject(swo *openapispec.Swagger) {
	swo.Info = &openapispec.Info{
		InfoProps: openapispec.InfoProps{
			Title:       "KubeVirt API",
			Description: "This is KubeVirt API an add-on for Kubernetes.",
			Contact: &openapispec.ContactInfo{
				Name:  "kubevirt-dev",
				Email: "kubevirt-dev@googlegroups.com",
				URL:   "https://github.com/kubevirt/kubevirt",
			},
			License: &openapispec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0",
			},
		},
	}
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

func getNamespace() string {
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return metav1.NamespaceSystem
}

func (app *virtAPIApp) getSelfSignedCert() error {
	var ok bool

	namespace := getNamespace()
	generateCerts := false
	secret, err := app.virtCli.CoreV1().Secrets(namespace).Get(virtApiCertSecretName, metav1.GetOptions{})
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
			"virt-api."+namespace+".pod.cluster.local",
			"virt-api",
			namespace,
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
				Namespace: namespace,
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
		_, err := app.virtCli.CoreV1().Secrets(namespace).Create(&secret)
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

func (app *virtAPIApp) createSubresourceApiservice() error {
	namespace := getNamespace()
	config, err := kubecli.GetConfig()
	if err != nil {
		return err
	}
	aggregatorClient := aggregatorclient.NewForConfigOrDie(config)

	subresourceAggregatedApiName := v1.SubresourceGroupVersion.Version + "." + v1.SubresourceGroupName

	registerApiService := false

	apiService, err := aggregatorClient.ApiregistrationV1beta1().APIServices().Get(subresourceAggregatedApiName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			registerApiService = true
		} else {
			return err
		}
	}

	if registerApiService {
		_, err = aggregatorClient.ApiregistrationV1beta1().APIServices().Create(&apiregistrationv1beta1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      subresourceAggregatedApiName,
				Namespace: namespace,
				Labels: map[string]string{
					v1.AppLabel: "virt-api-aggregator",
				},
			},
			Spec: apiregistrationv1beta1.APIServiceSpec{
				Service: &apiregistrationv1beta1.ServiceReference{
					Namespace: namespace,
					Name:      "virt-api",
				},
				Group:                v1.SubresourceGroupName,
				Version:              v1.SubresourceGroupVersion.Version,
				CABundle:             app.signingCertBytes,
				GroupPriorityMinimum: 1000,
				VersionPriority:      15,
			},
		})
		if err != nil {
			return err
		}
	} else if reflect.DeepEqual(apiService.Spec.CABundle, app.signingCertBytes) == false {
		// Update apiService if CA bundle doesn't match ours
		apiService.Spec.CABundle = app.signingCertBytes
		_, err := aggregatorClient.ApiregistrationV1beta1().APIServices().Update(apiService)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *virtAPIApp) startTLS() error {

	errors := make(chan error)

	keyFile := filepath.Join(app.certsDirectory, "/key.pem")
	certFile := filepath.Join(app.certsDirectory, "/cert.pem")
	signingCertFile := filepath.Join(app.certsDirectory, "/signingCert.pem")
	clientCAFile := filepath.Join(app.certsDirectory, "/clientCA.crt")

	// Write the certs to disk
	err := ioutil.WriteFile(clientCAFile, app.clientCABytes, 0600)
	if err != nil {
		return err
	}

	if len(app.requestHeaderClientCABytes) != 0 {
		f, err := os.OpenFile(clientCAFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(app.requestHeaderClientCABytes)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(keyFile, app.keyBytes, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(certFile, app.certBytes, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(signingCertFile, app.signingCertBytes, 0600)
	if err != nil {
		return err
	}

	// create the client CA pool.
	// This ensures we're talking to the k8s api server
	pool, err := cert.NewPool(clientCAFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		ClientCAs: pool,
		// A RequestClientCert request means we're not guaranteed
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
		ClientAuth: tls.RequestClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	go func() {
		server := &http.Server{
			Addr:      fmt.Sprintf("%s:%d", app.BindAddress, app.Port),
			TLSConfig: tlsConfig,
		}

		errors <- server.ListenAndServeTLS(certFile, keyFile)
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
