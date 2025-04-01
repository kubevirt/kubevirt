package tests

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	openshiftroutev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const tempRouteName = "prom-route"

type HCOPrometheusClient struct {
	url   string
	token string
	cli   *http.Client
}

var (
	hcoClient     *HCOPrometheusClient
	hcoClientOnce sync.Once
)

func GetHCOPrometheusClient(ctx context.Context, cli client.Client) (*HCOPrometheusClient, error) {
	var err error
	hcoClientOnce.Do(func() {
		hcoClient, err = newHCOPrometheusClient(ctx, cli)
	})

	if err != nil {
		return nil, err
	}

	if hcoClient == nil {
		return nil, fmt.Errorf("HCO client wasn't initiated")
	}

	return hcoClient, nil
}

func newHCOPrometheusClient(ctx context.Context, cli client.Client) (*HCOPrometheusClient, error) {
	secret := &corev1.Secret{}
	err := cli.Get(ctx, client.ObjectKey{Namespace: InstallNamespace, Name: "hco-bearer-auth"}, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to read the secret; %w", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			tempRouteHost, err := getTempRouteHost(ctx, cli)
			if err != nil {
				continue
			}

			httpClient, err := rest.HTTPClientFor(GetClientConfig())
			if err != nil {
				return nil, fmt.Errorf("can't create HTTP client; %w", err)
			}

			return &HCOPrometheusClient{
				url:   fmt.Sprintf("https://%s/metrics", tempRouteHost),
				token: string(secret.Data["token"]),
				cli:   httpClient,
			}, nil

		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for HCO Prometheus metrics route to be available")
		}
	}
}

func (hcoCli HCOPrometheusClient) GetHCOMetric(ctx context.Context, query string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, hcoCli.url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(hcoCli.token)))

	resp, err := hcoCli.cli.Do(req.WithContext(ctx))
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
			parts := strings.Fields(line)
			if len(parts) < 2 {
				return 0, fmt.Errorf("metric line does not contain a value")
			}
			res, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err != nil {
				return 0, fmt.Errorf("error converting %s to int: %v", line, err)
			}
			return res, nil
		}
	}
	return 0, nil
}

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

func getTempRouteHost(ctx context.Context, cli client.Client) (string, error) {
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
