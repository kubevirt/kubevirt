package tests

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	openshiftroutev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const tempRouteName = "prom-route"

// CreateTempRoute creates a route to the HCO prometheus endpoint, to allow reading the metrics.
func CreateTempRoute(ctx context.Context, cli client.Client) error {
	err := openshiftroutev1.AddToScheme(cli.Scheme())
	if err != nil {
		return err
	}

	route := &openshiftroutev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempRouteName,
			Namespace: InstallNamespace,
		},
		Spec: openshiftroutev1.RouteSpec{
			Port: &openshiftroutev1.RoutePort{
				TargetPort: intstr.FromString("http-metrics"),
			},
			TLS: &openshiftroutev1.TLSConfig{
				Termination:                   openshiftroutev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: openshiftroutev1.InsecureEdgeTerminationPolicyRedirect,
			},
			To: openshiftroutev1.RouteTargetReference{
				Kind:   "Service",
				Name:   "kubevirt-hyperconverged-operator-metrics",
				Weight: ptr.To[int32](100),
			},
			WildcardPolicy: openshiftroutev1.WildcardPolicyNone,
		},
	}

	return cli.Create(ctx, route)
}

func DeleteTempRoute(ctx context.Context, cli client.Client) error {
	route := &openshiftroutev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempRouteName,
			Namespace: InstallNamespace,
		},
	}
	return cli.Delete(ctx, route)
}

func GetTempRouteHost(ctx context.Context, cli client.Client) (string, error) {
	route := &openshiftroutev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempRouteName,
			Namespace: InstallNamespace,
		},
	}
	err := cli.Get(ctx, client.ObjectKeyFromObject(route), route)
	if err != nil {
		return "", fmt.Errorf("failed to read the temp router; %w", err)
	}

	if len(route.Status.Ingress) == 0 {
		return "", fmt.Errorf("failed to read the temp route status")
	}

	return route.Status.Ingress[0].Host, nil
}

func GetHCOMetric(ctx context.Context, url, query string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := GetHTTPClient().Do(req.WithContext(ctx))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to read the temp route status: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, query) {
			res, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, query)), 64)
			if err != nil {
				return 0, fmt.Errorf("error converting %s to int: %v\n", line, err)
			}
			return res, nil
		}
	}
	return 0, nil
}

// makes http calls to http endpoints in the cluster
var httpClient *http.Client

func GetHTTPClient() *http.Client {
	once := &sync.Once{}
	once.Do(func() {
		var err error
		httpClient, err = rest.HTTPClientFor(GetClientConfig())
		if err != nil {
			panic("can't create HTTP client;" + err.Error())
		}
	})

	return httpClient
}
