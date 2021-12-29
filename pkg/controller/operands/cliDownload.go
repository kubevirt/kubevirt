package operands

import (
	"errors"
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	cliDownloadsServiceName = "hyperconverged-cluster-cli-download"
	descriptionText         = "The virtctl client is a supplemental command-line utility for managing virtualization resources from the command line."
	displayName             = "virtctl - KubeVirt command line interface"
)

// **** Handler for ConsoleCliDownload ****
type cliDownloadHandler genericOperand

func newCliDownloadHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadHandler {
	return &cliDownloadHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "ConsoleCLIDownload",
		removeExistingOwner:    false,
		setControllerReference: false,
		hooks:                  &cliDownloadHooks{},
	}
}

type cliDownloadHooks struct{}

func (h cliDownloadHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewConsoleCLIDownload(hc), nil
}

func (h cliDownloadHooks) getEmptyCr() client.Object {
	return &consolev1.ConsoleCLIDownload{}
}

func (h cliDownloadHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*consolev1.ConsoleCLIDownload).ObjectMeta
}

func (h *cliDownloadHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ccd, ok1 := required.(*consolev1.ConsoleCLIDownload)
	found, ok2 := exists.(*consolev1.ConsoleCLIDownload)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ConsoleCLIDownload")
	}
	if !reflect.DeepEqual(found.Spec, ccd.Spec) ||
		!reflect.DeepEqual(found.Labels, ccd.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsoleCLIDownload's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated ConsoleCLIDownload's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&ccd.ObjectMeta, &found.ObjectMeta)
		ccd.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewConsoleCLIDownload(hc *hcov1beta1.HyperConverged) *consolev1.ConsoleCLIDownload {
	baseUrl := "https://" + cliDownloadsServiceName + "-" + hc.Namespace + "." + hcoutil.GetClusterInfo().GetDomain()

	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "virtctl-clidownloads-" + hc.Name,
			Labels: getLabels(hc, hcoutil.AppComponentCompute),
		},

		Spec: consolev1.ConsoleCLIDownloadSpec{
			Description: descriptionText,
			DisplayName: displayName,
			Links: []consolev1.CLIDownloadLink{
				{
					Href: baseUrl + "/amd64/linux/virtctl.tar.gz",
					Text: "Download virtctl for Linux for x86_64",
				},
				{
					Href: baseUrl + "/amd64/mac/virtctl.zip",
					Text: "Download virtctl for Mac for x86_64",
				},
				{
					Href: baseUrl + "/amd64/windows/virtctl.zip",
					Text: "Download virtctl for Windows for x86_64",
				},
			},
		},
	}
}

// **** Handler for Service ****
type cliDownloadServiceHandler genericOperand

func newCliDownloadsServiceHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadServiceHandler {
	return &cliDownloadServiceHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "Service",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &cliDownloadsServiceHooks{},
	}
}

type cliDownloadsServiceHooks struct{}

func (h cliDownloadsServiceHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewCliDownloadsService(hc), nil
}

func (h cliDownloadsServiceHooks) getEmptyCr() client.Object {
	return &corev1.Service{}
}

func (h cliDownloadsServiceHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*corev1.Service).ObjectMeta
}

func (h *cliDownloadsServiceHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	service, ok1 := required.(*corev1.Service)
	found, ok2 := exists.(*corev1.Service)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Service")
	}
	if !hasServiceRightFields(found, service) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Service Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated Service's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&service.ObjectMeta, &found.ObjectMeta)
		service.Spec.ClusterIP = found.Spec.ClusterIP
		service.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewCliDownloadsService(hc *hcov1beta1.HyperConverged) *corev1.Service {

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cliDownloadsServiceName,
			Namespace: hc.Namespace,
			Labels:    getLabels(hc, hcoutil.AppComponentCompute),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"name": cliDownloadsServiceName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       strconv.Itoa(util.CliDownloadsServerPort),
					Port:       util.CliDownloadsServerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(util.CliDownloadsServerPort),
				},
			},
		},
	}
}

// We need to check only certain fields of Service object. Since there
// are some fields in the Spec that are set by k8s like "clusterIP", "ipFamilyPolicy", etc.
// When we compare current spec with expected spec by using reflect.DeepEqual, it
// never returns true.
func hasServiceRightFields(found *corev1.Service, required *corev1.Service) bool {
	return reflect.DeepEqual(found.Labels, required.Labels) &&
		reflect.DeepEqual(found.Spec.Selector, required.Spec.Selector) &&
		reflect.DeepEqual(found.Spec.Ports, required.Spec.Ports)
}

// **** Handler for Service ****
type cliDownloadRouteHandler genericOperand

func newCliDownloadsRouteHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadRouteHandler {
	return &cliDownloadRouteHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "Route",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &cliDownloadsRouteHooks{},
	}
}

type cliDownloadsRouteHooks struct{}

func (h cliDownloadsRouteHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewCliDownloadsRoute(hc), nil
}

func (h cliDownloadsRouteHooks) getEmptyCr() client.Object {
	return &routev1.Route{}
}

func (h cliDownloadsRouteHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*routev1.Route).ObjectMeta
}

func (h *cliDownloadsRouteHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	route, ok1 := required.(*routev1.Route)
	found, ok2 := exists.(*routev1.Route)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Route")
	}
	if !hasRouteRightFields(found, route) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Route Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated Route Spec to its opinionated values")
		}
		util.DeepCopyLabels(&route.ObjectMeta, &found.ObjectMeta)
		route.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewCliDownloadsRoute(hc *hcov1beta1.HyperConverged) *routev1.Route {
	weight := int32(100)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cliDownloadsServiceName,
			Namespace: hc.Namespace,
			Labels:    getLabels(hc, hcoutil.AppComponentCompute),
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.IntOrString{IntVal: util.CliDownloadsServerPort},
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   cliDownloadsServiceName,
				Weight: &weight,
			},
		},
	}
}

// We need to check only certain fields of Route object. Since there
// are some fields in the Spec that are set by k8s like "host". When
// we compare current spec with expected spec by using reflect.DeepEqual, it
// never returns true.
func hasRouteRightFields(found *routev1.Route, required *routev1.Route) bool {
	return reflect.DeepEqual(found.Labels, required.Labels) &&
		reflect.DeepEqual(found.Spec.Port, required.Spec.Port) &&
		reflect.DeepEqual(found.Spec.TLS, required.Spec.TLS) &&
		reflect.DeepEqual(found.Spec.To, required.Spec.To)
}
