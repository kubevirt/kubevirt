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
	"k8s.io/utils/ptr"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/downloadhost"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	descriptionText = "The virtctl client is a supplemental command-line utility for managing virtualization resources from the command line."
	displayName     = "virtctl - KubeVirt command line interface"
)

// **** Handler for ConsoleCliDownload ****
type cliDownloadHandler genericOperand

func newCliDownloadHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadHandler {
	return &cliDownloadHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "ConsoleCLIDownload",
		setControllerReference: false,
		hooks:                  &cliDownloadHooks{},
	}
}

type cliDownloadHooks struct{}

func (*cliDownloadHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewConsoleCLIDownload(hc), nil
}

func (*cliDownloadHooks) getEmptyCr() client.Object {
	return &consolev1.ConsoleCLIDownload{}
}

func (*cliDownloadHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	ccd, ok1 := required.(*consolev1.ConsoleCLIDownload)
	found, ok2 := exists.(*consolev1.ConsoleCLIDownload)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ConsoleCLIDownload")
	}
	if !reflect.DeepEqual(found.Spec, ccd.Spec) ||
		!util.CompareLabels(ccd, found) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing ConsoleCLIDownload's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated ConsoleCLIDownload's Spec to its opinionated values")
		}
		util.MergeLabels(&ccd.ObjectMeta, &found.ObjectMeta)
		ccd.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (*cliDownloadHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewConsoleCLIDownload(hc *hcov1beta1.HyperConverged) *consolev1.ConsoleCLIDownload {
	host := string(downloadhost.Get().CurrentHost)
	baseURL := "https://" + host

	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "virtctl-clidownloads-" + hc.Name,
			Labels: getLabels(hc, util.AppComponentCompute),
		},

		Spec: consolev1.ConsoleCLIDownloadSpec{
			Description: descriptionText,
			DisplayName: displayName,
			Links: []consolev1.CLIDownloadLink{
				{
					Href: baseURL + "/amd64/linux/virtctl.tar.gz",
					Text: "Download virtctl for Linux for x86_64",
				},
				{
					Href: baseURL + "/arm64/linux/virtctl.tar.gz",
					Text: "Download virtctl for Linux for ARM 64",
				},
				{
					Href: baseURL + "/s390x/linux/virtctl.tar.gz",
					Text: "Download virtctl for Linux for IBM Z",
				},
				{
					Href: baseURL + "/amd64/mac/virtctl.zip",
					Text: "Download virtctl for Mac for x86_64",
				},
				{
					Href: baseURL + "/arm64/mac/virtctl.zip",
					Text: "Download virtctl for Mac for ARM 64",
				},
				{
					Href: baseURL + "/amd64/windows/virtctl.zip",
					Text: "Download virtctl for Windows for x86_64",
				},
				{
					Href: baseURL + "/arm64/windows/virtctl.zip",
					Text: "Download virtctl for Windows for ARM 64",
				},
			},
		},
	}
}

// **** Handler for Service ****

// NewCliDownloadsService creates a service object for the CLI downloads
func NewCliDownloadsService(hc *hcov1beta1.HyperConverged) *corev1.Service {

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      downloadhost.CLIDownloadsServiceName,
			Namespace: hc.Namespace,
			Labels:    getLabels(hc, util.AppComponentCompute),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"name": downloadhost.CLIDownloadsServiceName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       strconv.Itoa(util.CliDownloadsServerPort),
					Port:       util.CliDownloadsServerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(util.CliDownloadsServerPort),
				},
			},
		},
	}
}

// **** Handler for route ****
type cliDownloadRouteHandler genericOperand

func newCliDownloadsRouteHandler(Client client.Client, Scheme *runtime.Scheme) *cliDownloadRouteHandler {
	return &cliDownloadRouteHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "Route",
		setControllerReference: true,
		hooks:                  &cliDownloadsRouteHooks{},
	}
}

type cliDownloadsRouteHooks struct{}

func (cliDownloadsRouteHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewCliDownloadsRoute(hc), nil
}

func (cliDownloadsRouteHooks) getEmptyCr() client.Object {
	return &routev1.Route{}
}

func (cliDownloadsRouteHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
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
		util.MergeLabels(&route.ObjectMeta, &found.ObjectMeta)
		route.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (cliDownloadsRouteHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewCliDownloadsRoute(hc *hcov1beta1.HyperConverged) *routev1.Route {
	host := downloadhost.Get()

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      downloadhost.CLIDownloadsServiceName,
			Namespace: hc.Namespace,
			Labels:    getLabels(hc, util.AppComponentCompute),
		},
		Spec: routev1.RouteSpec{
			Host: string(downloadhost.Get().CurrentHost),
			Port: &routev1.RoutePort{
				TargetPort: intstr.IntOrString{IntVal: util.CliDownloadsServerPort},
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   downloadhost.CLIDownloadsServiceName,
				Weight: ptr.To[int32](100),
			},
		},
	}

	if len(host.Cert) > 0 && len(host.Key) > 0 {
		route.Spec.TLS.Certificate = host.Cert
		route.Spec.TLS.Key = host.Key
	}

	return route
}

// We need to check only certain fields of Route object. Since there
// are some fields in the Spec that are set by k8s like "host". When
// we compare current spec with expected spec by using reflect.DeepEqual, it
// never returns true.
func hasRouteRightFields(found *routev1.Route, required *routev1.Route) bool {
	return reflect.DeepEqual(found.Labels, required.Labels) &&
		reflect.DeepEqual(found.Spec.Port, required.Spec.Port) &&
		reflect.DeepEqual(found.Spec.TLS, required.Spec.TLS) &&
		reflect.DeepEqual(found.Spec.To, required.Spec.To) &&
		found.Spec.Host == required.Spec.Host
}
