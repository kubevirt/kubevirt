package apiserver

import (
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/util/cert/triple"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/emicklei/go-restful"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	datavolumev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	uploadv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/uploadcontroller/v1alpha1"
	validatingwebhook "kubevirt.io/containerized-data-importer/pkg/apiserver/webhooks/validating-webhook"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/pkg/keys"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

const (
	// selfsigned cert secret name
	apiCertSecretName       = "cdi-api-server-cert"
	apiSigningKeySecretName = "cdi-api-signing-key"

	uploadTokenGroup   = "upload.cdi.kubevirt.io"
	uploadTokenVersion = "v1alpha1"

	apiServiceName = "cdi-api"

	apiWebhookValidator = "cdi-api-validator"

	dvCreateValidatePath = "/datavolume-validate-create"
)

// CdiAPIServer is the public interface to the CDI API
type CdiAPIServer interface {
	Start() error
}

type uploadPossibleFunc func(*v1.PersistentVolumeClaim) error

type cdiAPIApp struct {
	bindAddress string
	bindPort    uint

	client           kubernetes.Interface
	aggregatorClient aggregatorclient.Interface

	authorizer CdiAPIAuthorizer

	signingCertBytes           []byte
	certBytes                  []byte
	keyBytes                   []byte
	clientCABytes              []byte
	requestHeaderClientCABytes []byte

	privateSigningKey *rsa.PrivateKey

	container *restful.Container

	// test hook
	uploadPossible uploadPossibleFunc
}

// UploadTokenRequestAPI returns web service for swagger generation
func UploadTokenRequestAPI() []*restful.WebService {
	app := cdiAPIApp{}
	app.composeUploadTokenAPI()
	return app.container.RegisteredWebServices()
}

// NewCdiAPIServer returns an initialized CDI api server
func NewCdiAPIServer(bindAddress string,
	bindPort uint,
	client kubernetes.Interface,
	aggregatorClient aggregatorclient.Interface,
	authorizor CdiAPIAuthorizer) (CdiAPIServer, error) {
	var err error
	app := &cdiAPIApp{
		bindAddress:      bindAddress,
		bindPort:         bindPort,
		client:           client,
		aggregatorClient: aggregatorClient,
		authorizer:       authorizor,
		uploadPossible:   controller.UploadPossibleForPVC,
	}
	err = app.getClientCert()
	if err != nil {
		return nil, errors.Errorf("Unable to get client cert: %v\n", errors.WithStack(err))
	}

	err = app.getSelfSignedCert()
	if err != nil {
		return nil, errors.Errorf("Unable to get self signed cert: %v\n", errors.WithStack(err))
	}

	err = app.createAPIService()
	if err != nil {
		return nil, errors.Errorf("Unable to register aggregated api service: %v\n", errors.WithStack(err))
	}

	app.composeUploadTokenAPI()

	app.container.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		var username = "-"
		if req.Request.URL.User != nil {
			if name := req.Request.URL.User.Username(); name != "" {
				username = name
			}
		}
		chain.ProcessFilter(req, resp)
		glog.V(1).Infof("----------------------------")
		glog.V(1).Infof("remoteAddress:%s", strings.Split(req.Request.RemoteAddr, ":")[0])
		glog.V(1).Infof("username: %s", username)
		glog.V(1).Infof("method: %s", req.Request.Method)
		glog.V(1).Infof("url: %s", req.Request.URL.RequestURI())
		glog.V(1).Infof("proto: %s", req.Request.Proto)
		glog.V(1).Infof("headers: %v", req.Request.Header)
		glog.V(1).Infof("statusCode: %d", resp.StatusCode())
		glog.V(1).Infof("contentLength: %d", resp.ContentLength())

	})

	err = app.createWebhook()
	if err != nil {
		return nil, errors.Errorf("failed to create webhook: %s", err)
	}

	return app, nil
}

func (app *cdiAPIApp) Start() error {
	return app.startTLS()
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

func (app *cdiAPIApp) getClientCert() error {
	authConfigMap, err := app.client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return err
	}

	clientCA, ok := authConfigMap.Data["client-ca-file"]
	if !ok {
		return errors.Errorf("client-ca-file value not found in auth config map.")
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
		app.authorizer.AddUserHeaders(headerList)
	}

	headers, ok = authConfigMap.Data["requestheader-group-headers"]
	if ok {
		headerList, err := deserializeStrings(headers)
		if err != nil {
			return err
		}
		app.authorizer.AddGroupHeaders(headerList)
	}

	headers, ok = authConfigMap.Data["requestheader-extra-headers-prefix"]
	if ok {
		headerList, err := deserializeStrings(headers)
		if err != nil {
			return err
		}
		app.authorizer.AddExtraPrefixHeaders(headerList)
	}

	return nil
}

func (app *cdiAPIApp) getSelfSignedCert() error {
	namespace := util.GetNamespace()
	caKeyPair, err := triple.NewCA("api.cdi.kubevirt.io")
	if err != nil {
		return errors.Wrap(err, "Error creating CA")
	}

	keyPairAndCert, err := keys.GetOrCreateServerKeyPairAndCert(app.client,
		namespace,
		apiCertSecretName,
		caKeyPair,
		caKeyPair.Cert,
		apiServiceName+"."+namespace,
		apiServiceName,
		nil,
	)
	if err != nil {
		return errors.Wrapf(err, "Error getting/creating secret %s", apiCertSecretName)
	}

	app.keyBytes = cert.EncodePrivateKeyPEM(keyPairAndCert.KeyPair.Key)
	app.certBytes = cert.EncodeCertPEM(keyPairAndCert.KeyPair.Cert)
	app.signingCertBytes = cert.EncodeCertPEM(keyPairAndCert.CACert)

	privateKey, err := keys.GetOrCreatePrivateKey(app.client, namespace, apiSigningKeySecretName)
	if err != nil {
		return errors.Wrap(err, "Error getting/creating signing key")
	}

	app.privateSigningKey = privateKey

	return nil
}

func (app *cdiAPIApp) startTLS() error {
	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		return err
	}
	defer os.RemoveAll(certsDirectory)

	keyFile := filepath.Join(certsDirectory, "key.pem")
	certFile := filepath.Join(certsDirectory, "cert.pem")
	signingCertFile := filepath.Join(certsDirectory, "signingCert.pem")
	clientCAFile := filepath.Join(certsDirectory, "clientCA.crt")

	// Write the certs to disk
	err = ioutil.WriteFile(clientCAFile, app.clientCABytes, 0600)
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

	errChan := make(chan error)

	// create the client CA pool.
	// This ensures we're talking to the k8s api server
	pool, err := cert.NewPool(clientCAFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		ClientCAs:  pool,
		ClientAuth: tls.RequestClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	go func() {
		server := &http.Server{
			Addr:      fmt.Sprintf("%s:%d", app.bindAddress, app.bindPort),
			TLSConfig: tlsConfig,
			Handler:   app.container,
		}

		errChan <- server.ListenAndServeTLS(certFile, keyFile)
	}()

	// wait for server to exit
	return <-errChan
}

func (app *cdiAPIApp) uploadHandler(request *restful.Request, response *restful.Response) {

	allowed, reason, err := app.authorizer.Authorize(request)
	if err != nil {
		glog.Error(err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	} else if !allowed {
		glog.Infof("Rejected Request: %s", reason)
		response.WriteErrorString(http.StatusUnauthorized, reason)
		return
	}

	namespace := request.PathParameter("namespace")
	defer request.Request.Body.Close()
	body, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		glog.Error(err)
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	uploadToken := &uploadv1alpha1.UploadTokenRequest{}
	err = json.Unmarshal(body, uploadToken)
	if err != nil {
		glog.Error(err)
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	pvcName := uploadToken.Spec.PvcName
	pvc, err := app.client.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			glog.Infof("Rejecting request for PVC %s that doesn't exist", pvcName)
			response.WriteError(http.StatusBadRequest, err)
			return
		}
		glog.Error(err)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	if err = app.uploadPossible(pvc); err != nil {
		response.WriteError(http.StatusServiceUnavailable, err)
		return
	}

	tokenData, _ := GenerateToken(pvcName, namespace, app.privateSigningKey)

	uploadToken.Status.Token = tokenData
	response.WriteAsJson(uploadToken)

}

func uploadTokenAPIGroup() metav1.APIGroup {
	apiGroup := metav1.APIGroup{
		Name: uploadTokenGroup,
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: uploadTokenGroup + "/" + uploadTokenVersion,
			Version:      uploadTokenVersion,
		},
	}
	apiGroup.Versions = append(apiGroup.Versions, metav1.GroupVersionForDiscovery{
		GroupVersion: uploadTokenGroup + "/" + uploadTokenVersion,
		Version:      uploadTokenVersion,
	})
	apiGroup.ServerAddressByClientCIDRs = append(apiGroup.ServerAddressByClientCIDRs, metav1.ServerAddressByClientCIDR{
		ClientCIDR:    "0.0.0.0/0",
		ServerAddress: "",
	})
	apiGroup.Kind = "APIGroup"
	apiGroup.APIVersion = "v1"
	return apiGroup
}

func (app *cdiAPIApp) composeUploadTokenAPI() {
	objPointer := &uploadv1alpha1.UploadTokenRequest{}
	objExample := reflect.ValueOf(objPointer).Elem().Interface()
	objKind := "UploadTokenRequest"
	resource := "uploadtokenrequests"

	groupPath := fmt.Sprintf("/apis/%s", uploadTokenGroup)
	resourcePath := fmt.Sprintf("/apis/%s/%s", uploadTokenGroup, uploadTokenVersion)
	createPath := fmt.Sprintf("/namespaces/{namespace:[a-z0-9][a-z0-9\\-]*}/%s", resource)

	app.container = restful.NewContainer()

	uploadTokenWs := new(restful.WebService)
	uploadTokenWs.Doc("The CDI Upload API.")
	uploadTokenWs.Path(resourcePath)

	uploadTokenWs.Route(uploadTokenWs.POST(createPath).
		Produces("application/json").
		Consumes("application/json").
		Operation("createNamespaced"+objKind).
		To(app.uploadHandler).Reads(objExample).Writes(objExample).
		Doc("Create an UploadTokenRequest object.").
		Returns(http.StatusOK, "OK", objExample).
		Returns(http.StatusCreated, "Created", objExample).
		Returns(http.StatusAccepted, "Accepted", objExample).
		Returns(http.StatusUnauthorized, "Unauthorized", nil).
		Param(uploadTokenWs.PathParameter("namespace", "Object name and auth scope, such as for teams and projects").Required(true)))

	// Return empty api resource list.
	// K8s expects to be able to retrieve a resource list for each aggregated
	// app in order to discover what resources it provides. Without returning
	// an empty list here, there's a bug in the k8s resource discovery that
	// breaks kubectl's ability to reference short names for resources.
	uploadTokenWs.Route(uploadTokenWs.GET("/").
		Produces("application/json").Writes(metav1.APIResourceList{}).
		To(func(request *restful.Request, response *restful.Response) {
			list := &metav1.APIResourceList{}

			list.Kind = "APIResourceList"
			list.APIVersion = "v1" // this is the version of the resource list
			list.GroupVersion = uploadTokenGroup + "/" + uploadTokenVersion
			list.APIResources = append(list.APIResources, metav1.APIResource{
				Name:         "uploadtokenrequests",
				SingularName: "UploadtokenRequest",
				Namespaced:   true,
				Group:        uploadTokenGroup,
				Version:      uploadTokenVersion,
				Kind:         "UploadTokenRequest",
				Verbs:        []string{"create"},
				ShortNames:   []string{"utr", "utrs"},
			})
			response.WriteAsJson(list)
		}).
		Operation("getAPIResources").
		Doc("Get a CDI API resources").
		Returns(http.StatusOK, "OK", metav1.APIResourceList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	app.container.Add(uploadTokenWs)

	ws := new(restful.WebService)

	// K8s needs the ability to query info about a specific API group
	ws.Route(ws.GET(groupPath).
		Produces("application/json").Writes(metav1.APIGroup{}).
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteAsJson(uploadTokenAPIGroup())
		}).
		Operation("getAPIGroup").
		Doc("Get a CDI API Group").
		Returns(http.StatusOK, "OK", metav1.APIGroup{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	// K8s needs the ability to query the list of API groups this endpoint supports
	ws.Route(ws.GET("apis").
		Produces("application/json").Writes(metav1.APIGroupList{}).
		To(func(request *restful.Request, response *restful.Response) {
			list := &metav1.APIGroupList{}
			list.Kind = "APIGroupList"
			list.APIVersion = "v1"
			list.Groups = append(list.Groups, uploadTokenAPIGroup())
			response.WriteAsJson(list)
		}).
		Operation("getAPIGroup").
		Doc("Get a CDI API GroupList").
		Returns(http.StatusOK, "OK", metav1.APIGroupList{}).
		Returns(http.StatusNotFound, "Not Found", nil))

	app.container.Add(ws)
}

func (app *cdiAPIApp) createAPIService() error {
	namespace := util.GetNamespace()
	apiName := uploadTokenVersion + "." + uploadTokenGroup

	registerAPIService := false

	apiService, err := app.aggregatorClient.ApiregistrationV1beta1().APIServices().Get(apiName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			registerAPIService = true
		} else {
			return err
		}
	}

	newAPIService := &apiregistrationv1beta1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: apiName,
			Labels: map[string]string{
				common.CDIComponentLabel: apiServiceName,
			},
		},
		Spec: apiregistrationv1beta1.APIServiceSpec{
			Service: &apiregistrationv1beta1.ServiceReference{
				Namespace: namespace,
				Name:      apiServiceName,
			},
			Group:                uploadTokenGroup,
			Version:              uploadTokenVersion,
			CABundle:             app.signingCertBytes,
			GroupPriorityMinimum: 1000,
			VersionPriority:      15,
		},
	}

	if registerAPIService {
		_, err = app.aggregatorClient.ApiregistrationV1beta1().APIServices().Create(newAPIService)
		if err != nil {
			return err
		}
	} else {
		if apiService.Spec.Service != nil && apiService.Spec.Service.Namespace != namespace {
			return fmt.Errorf("apiservice [%s] is already registered in a different namespace. Existing apiservice registration must be deleted before virt-api can proceed", apiName)
		}

		// Always update spec to latest.
		apiService.Spec = newAPIService.Spec
		_, err := app.aggregatorClient.ApiregistrationV1beta1().APIServices().Update(apiService)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *cdiAPIApp) createWebhook() error {
	dvPathCreate := dvCreateValidatePath
	namespace := util.GetNamespace()
	registerWebhook := false
	webhookRegistration, err := app.client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(apiWebhookValidator, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			registerWebhook = true
		} else {
			return err
		}
	}

	webHooks := []admissionregistrationv1beta1.Webhook{
		{
			Name: "datavolume-create-validator.cdi.kubevirt.io",
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{
					admissionregistrationv1beta1.Create,
				},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{datavolumev1alpha1.SchemeGroupVersion.Group},
					APIVersions: []string{datavolumev1alpha1.SchemeGroupVersion.Version},
					Resources:   []string{"datavolumes"},
				},
			}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: namespace,
					Name:      apiServiceName,
					Path:      &dvPathCreate,
				},
				CABundle: app.signingCertBytes,
			},
		},
	}

	if registerWebhook {
		_, err := app.client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(&admissionregistrationv1beta1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: apiWebhookValidator,
			},
			Webhooks: webHooks,
		})
		if err != nil {
			return err
		}
	} else {
		for _, webhook := range webhookRegistration.Webhooks {
			if webhook.ClientConfig.Service != nil && webhook.ClientConfig.Service.Namespace != namespace {
				return fmt.Errorf("ValidatingAdmissionWebhook [%s] is already registered using services endpoints in a different namespace. Existing webhook registration must be deleted before cdi-api can proceed", apiWebhookValidator)
			}
		}

		// update registered webhook with our data
		webhookRegistration.Webhooks = webHooks

		_, err := app.client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Update(webhookRegistration)
		if err != nil {
			return err
		}
	}

	app.container.ServeMux.HandleFunc(
		dvCreateValidatePath, func(w http.ResponseWriter, r *http.Request) {
			validatingwebhook.ServeDVs(w, r)
		},
	)
	return nil
}
