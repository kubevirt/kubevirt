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

package virt_operator

import (
	"context"
	"crypto/tls"
	"fmt"
	golog "log"
	"net/http"
	"os"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"k8s.io/client-go/util/certificate"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"

	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	operator_webhooks "kubevirt.io/kubevirt/pkg/virt-operator/webhooks"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/prometheus/client_golang/prometheus"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/monitoring/configuration"
	"kubevirt.io/kubevirt/pkg/monitoring/profiler"
	"kubevirt.io/kubevirt/pkg/service"
	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	install "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const VirtOperator = "virt-operator"

const (
	controllerThreads = 3

	// Default port that virt-operator listens on.
	defaultPort = 8186

	// Default address that virt-operator listens on.
	defaultHost = "0.0.0.0"
)

type VirtOperatorApp struct {
	service.ServiceListen

	clientSet       kubecli.KubevirtClient
	restClient      *clientrest.RESTClient
	informerFactory controller.KubeInformerFactory

	kubeVirtController *KubeVirtController
	kubeVirtRecorder   record.EventRecorder

	operatorNamespace string

	kubeVirtInformer cache.SharedIndexInformer
	kubeVirtCache    cache.Store

	crdInformer cache.SharedIndexInformer

	stores    util.Stores
	informers util.Informers

	LeaderElection      leaderelectionconfig.Configuration
	aggregatorClient    aggregatorclient.Interface
	operatorCertManager certificate.Manager

	clusterConfig *virtconfig.ClusterConfig
	host          string

	ctx context.Context

	reInitChan chan string
}

var (
	_ service.Service = &VirtOperatorApp{}

	leaderGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubevirt_virt_operator_leading_status",
			Help: "Indication for an operating virt-operator.",
		},
	)

	readyGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kubevirt_virt_operator_ready_status",
			Help: "Indication for a virt-operator that is ready to take the lead.",
		},
	)
)

func init() {
	prometheus.MustRegister(leaderGauge)
	prometheus.MustRegister(readyGauge)
}

func Execute() {
	var err error
	app := VirtOperatorApp{}

	dumpInstallStrategy := pflag.Bool("dump-install-strategy", false, "Dump install strategy to configmap and exit")

	service.Setup(&app)

	log.InitializeLogging(VirtOperator)

	host, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}
	app.host = host

	err = util.VerifyEnv()
	if err != nil {
		golog.Fatal(err)
	}

	// apply any passthrough environment to this operator as well
	for k, v := range util.GetPassthroughEnv() {
		os.Setenv(k, v)
	}

	config, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		panic(err)
	}

	app.aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	app.operatorNamespace, err = clientutil.GetNamespace()
	if err != nil {
		golog.Fatalf("Error searching for namespace: %v", err)
	}

	if *dumpInstallStrategy {
		err = install.DumpInstallStrategyToConfigMap(app.clientSet, app.operatorNamespace)
		if err != nil {
			golog.Fatal(err)
		}
		os.Exit(0)
	}

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet, app.aggregatorClient, app.operatorNamespace)

	app.kubeVirtInformer = app.informerFactory.KubeVirt()
	app.kubeVirtCache = app.kubeVirtInformer.GetStore()

	app.informers = util.Informers{
		ServiceAccount:           app.informerFactory.OperatorServiceAccount(),
		ClusterRole:              app.informerFactory.OperatorClusterRole(),
		ClusterRoleBinding:       app.informerFactory.OperatorClusterRoleBinding(),
		Role:                     app.informerFactory.OperatorRole(),
		RoleBinding:              app.informerFactory.OperatorRoleBinding(),
		Crd:                      app.informerFactory.OperatorCRD(),
		Service:                  app.informerFactory.OperatorService(),
		Deployment:               app.informerFactory.OperatorDeployment(),
		DaemonSet:                app.informerFactory.OperatorDaemonSet(),
		ValidationWebhook:        app.informerFactory.OperatorValidationWebhook(),
		MutatingWebhook:          app.informerFactory.OperatorMutatingWebhook(),
		APIService:               app.informerFactory.OperatorAPIService(),
		InstallStrategyConfigMap: app.informerFactory.OperatorInstallStrategyConfigMaps(),
		InstallStrategyJob:       app.informerFactory.OperatorInstallStrategyJob(),
		InfrastructurePod:        app.informerFactory.OperatorPod(),
		PodDisruptionBudget:      app.informerFactory.OperatorPodDisruptionBudget(),
		Namespace:                app.informerFactory.Namespace(),
		Secrets:                  app.informerFactory.Secrets(),
		ConfigMap:                app.informerFactory.OperatorConfigMap(),
	}

	app.stores = util.Stores{
		ServiceAccountCache:           app.informerFactory.OperatorServiceAccount().GetStore(),
		ClusterRoleCache:              app.informerFactory.OperatorClusterRole().GetStore(),
		ClusterRoleBindingCache:       app.informerFactory.OperatorClusterRoleBinding().GetStore(),
		RoleCache:                     app.informerFactory.OperatorRole().GetStore(),
		RoleBindingCache:              app.informerFactory.OperatorRoleBinding().GetStore(),
		CrdCache:                      app.informerFactory.OperatorCRD().GetStore(),
		ServiceCache:                  app.informerFactory.OperatorService().GetStore(),
		DeploymentCache:               app.informerFactory.OperatorDeployment().GetStore(),
		DaemonSetCache:                app.informerFactory.OperatorDaemonSet().GetStore(),
		ValidationWebhookCache:        app.informerFactory.OperatorValidationWebhook().GetStore(),
		MutatingWebhookCache:          app.informerFactory.OperatorMutatingWebhook().GetStore(),
		APIServiceCache:               app.informerFactory.OperatorAPIService().GetStore(),
		InstallStrategyConfigMapCache: app.informerFactory.OperatorInstallStrategyConfigMaps().GetStore(),
		InstallStrategyJobCache:       app.informerFactory.OperatorInstallStrategyJob().GetStore(),
		InfrastructurePodCache:        app.informerFactory.OperatorPod().GetStore(),
		PodDisruptionBudgetCache:      app.informerFactory.OperatorPodDisruptionBudget().GetStore(),
		NamespaceCache:                app.informerFactory.Namespace().GetStore(),
		SecretCache:                   app.informerFactory.Secrets().GetStore(),
		ConfigMapCache:                app.informerFactory.OperatorConfigMap().GetStore(),
	}

	app.crdInformer = app.informerFactory.CRD()

	onOpenShift, err := clusterutil.IsOnOpenShift(app.clientSet)
	if err != nil {
		golog.Fatalf("Error determining cluster type: %v", err)
	}
	if onOpenShift {
		log.Log.Info("we are on openshift")
		app.informers.SCC = app.informerFactory.OperatorSCC()
		app.stores.SCCCache = app.informerFactory.OperatorSCC().GetStore()
		app.informers.Route = app.informerFactory.OperatorRoute()
		app.stores.RouteCache = app.informerFactory.OperatorRoute().GetStore()
		app.stores.IsOnOpenshift = true
	} else {
		log.Log.Info("we are on kubernetes")
		app.informers.SCC = app.informerFactory.DummyOperatorSCC()
		app.stores.SCCCache = app.informerFactory.DummyOperatorSCC().GetStore()
		app.informers.Route = app.informerFactory.DummyOperatorRoute()
		app.stores.RouteCache = app.informerFactory.DummyOperatorRoute().GetStore()
	}

	serviceMonitorEnabled, err := util.IsServiceMonitorEnabled(app.clientSet)
	if err != nil {
		golog.Fatalf("Error checking for ServiceMonitor: %v", err)
	}
	if serviceMonitorEnabled {
		log.Log.Info("servicemonitor is defined")
		app.informers.ServiceMonitor = app.informerFactory.OperatorServiceMonitor()
		app.stores.ServiceMonitorCache = app.informerFactory.OperatorServiceMonitor().GetStore()

		app.stores.ServiceMonitorEnabled = true
	} else {
		log.Log.Info("servicemonitor is not defined")
		app.informers.ServiceMonitor = app.informerFactory.DummyOperatorServiceMonitor()
		app.stores.ServiceMonitorCache = app.informerFactory.DummyOperatorServiceMonitor().GetStore()
	}

	prometheusRuleEnabled, err := util.IsPrometheusRuleEnabled(app.clientSet)
	if err != nil {
		golog.Fatalf("Error checking for PrometheusRule: %v", err)
	}
	if prometheusRuleEnabled {
		log.Log.Info("prometheusrule is defined")
		app.informers.PrometheusRule = app.informerFactory.OperatorPrometheusRule()
		app.stores.PrometheusRuleCache = app.informerFactory.OperatorPrometheusRule().GetStore()
		app.stores.PrometheusRulesEnabled = true
	} else {
		log.Log.Info("prometheusrule is not defined")
		app.informers.PrometheusRule = app.informerFactory.DummyOperatorPrometheusRule()
		app.stores.PrometheusRuleCache = app.informerFactory.DummyOperatorPrometheusRule().GetStore()
	}

	app.prepareCertManagers()

	app.kubeVirtRecorder = app.getNewRecorder(k8sv1.NamespaceAll, VirtOperator)
	app.kubeVirtController, err = NewKubeVirtController(app.clientSet, app.aggregatorClient.ApiregistrationV1().APIServices(), app.kubeVirtInformer, app.kubeVirtRecorder, app.stores, app.informers, app.operatorNamespace)
	if err != nil {
		panic(err)
	}

	image := util.GetOperatorImage()
	if image == "" {
		golog.Fatalf("Error getting operator's image: %v", err)
	}
	log.Log.Infof("Operator image: %s", image)

	app.clusterConfig, err = virtconfig.NewClusterConfig(
		app.informerFactory.CRD(),
		app.informerFactory.KubeVirt(),
		app.operatorNamespace,
	)

	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.ctx = ctx

	app.reInitChan = make(chan string, 0)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldChangeLogVerbosity)
	app.clusterConfig.SetConfigModifiedCallback(app.shouldUpdateConfigurationMetrics)

	go app.Run()
	<-app.reInitChan
}

func (app *VirtOperatorApp) Run() {
	promTLSConfig := kvtls.SetupPromTLS(app.operatorCertManager, app.clusterConfig)

	go func() {

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		webService := new(restful.WebService)
		webService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

		componentProfiler := profiler.NewProfileManager(app.clusterConfig)
		webService.Route(webService.GET("/start-profiler").To(componentProfiler.HandleStartProfiler).Doc("start profiler endpoint"))
		webService.Route(webService.GET("/stop-profiler").To(componentProfiler.HandleStopProfiler).Doc("stop profiler endpoint"))
		webService.Route(webService.GET("/dump-profiler").To(componentProfiler.HandleDumpProfiler).Doc("dump profiler results endpoint"))

		restfulContainer := restful.NewContainer()
		restfulContainer.ServeMux = mux
		restfulContainer.Add(webService)

		server := http.Server{
			Addr:      app.ServiceListen.Address(),
			Handler:   mux,
			TLSConfig: promTLSConfig,
			// Disable HTTP/2
			// See CVE-2023-44487
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			golog.Fatal(err)
		}
	}()

	endpointName := VirtOperator

	recorder := app.getNewRecorder(k8sv1.NamespaceAll, endpointName)

	id, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}

	rl, err := resourcelock.New(app.LeaderElection.ResourceLock,
		app.operatorNamespace,
		endpointName,
		app.clientSet.CoreV1(),
		app.clientSet.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		golog.Fatal(err)
	}

	apiAuthConfig := app.informerFactory.ApiAuthConfigMap()

	stop := app.ctx.Done()
	app.informerFactory.Start(stop)

	stopChan := app.ctx.Done()
	cache.WaitForCacheSync(stopChan, app.crdInformer.HasSynced, app.kubeVirtInformer.HasSynced)
	app.clusterConfig.SetConfigModifiedCallback(app.configModificationCallback)

	cache.WaitForCacheSync(stop, apiAuthConfig.HasSynced)

	go app.operatorCertManager.Start()

	caManager := kvtls.NewKubernetesClientCAManager(apiAuthConfig.GetStore())

	tlsConfig := kvtls.SetupTLSWithCertManager(caManager, app.operatorCertManager, tls.VerifyClientCertIfGiven, app.clusterConfig)

	webhookServer := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", app.BindAddress, 8444),
		TLSConfig: tlsConfig,
	}

	var mux http.ServeMux
	mux.HandleFunc("/kubevirt-validate-delete", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validating_webhooks.Serve(w, r, operator_webhooks.NewKubeVirtDeletionAdmitter(app.clientSet))
	}))
	mux.HandleFunc(components.KubeVirtUpdateValidatePath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validating_webhooks.Serve(w, r, operator_webhooks.NewKubeVirtUpdateAdmitter(app.clientSet, app.clusterConfig))
	}))
	mux.HandleFunc(components.KubeVirtCreateValidatePath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validating_webhooks.Serve(w, r, operator_webhooks.NewKubeVirtCreateAdmitter(app.clientSet))
	}))
	webhookServer.Handler = &mux
	go func() {
		err := webhookServer.ListenAndServeTLS("", "")
		if err != nil {
			panic(err)
		}
	}()

	leaderElector, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: app.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: app.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   app.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					leaderGauge.Set(1)
					log.Log.Infof("Started leading")

					// run app
					go app.kubeVirtController.Run(controllerThreads, stop)
				},
				OnStoppedLeading: func() {
					leaderGauge.Set(0)
					log.Log.V(5).Info("stop monitoring the kubevirt-config configMap")
					golog.Fatal("leaderelection lost")
				},
			},
		})
	if err != nil {
		golog.Fatal(err)
	}

	readyGauge.Set(1)
	log.Log.Infof("Attempting to acquire leader status")
	leaderElector.Run(app.ctx)

	panic("unreachable")

}

// Detects if ServiceMonitor or PrometheusRule crd has been applied or deleted that
// re-initializing virt-operator.
func (app *VirtOperatorApp) configModificationCallback() {
	msgf := "Reinitialize virt-operator, %s has been %s"

	smEnabled := app.clusterConfig.HasServiceMonitorAPI()
	if app.stores.ServiceMonitorEnabled != smEnabled {
		if !app.stores.ServiceMonitorEnabled && smEnabled {
			log.Log.Infof(msgf, "ServiceMonitor", "introduced")
		} else {
			log.Log.Infof(msgf, "ServiceMonitor", "removed")
		}
		app.reInitChan <- "reinit"
		return
	}

	prEnabled := app.clusterConfig.HasPrometheusRuleAPI()
	if app.stores.PrometheusRulesEnabled != prEnabled {
		if !app.stores.PrometheusRulesEnabled && prEnabled {
			log.Log.Infof(msgf, "PrometheusRule", "introduced")
		} else {
			log.Log.Infof(msgf, "PrometheusRule", "removed")
		}
		app.reInitChan <- "reinit"
	}
}

func (app *VirtOperatorApp) getNewRecorder(namespace string, componentName string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: app.clientSet.CoreV1().Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: componentName})
}

func (app *VirtOperatorApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()
}

func (app *VirtOperatorApp) prepareCertManagers() {
	app.operatorCertManager = bootstrap.NewFallbackCertificateManager(
		bootstrap.NewSecretCertificateManager(
			components.VirtOperatorCertSecretName,
			app.operatorNamespace,
			app.informers.Secrets.GetStore(),
		),
	)
}

func (app *VirtOperatorApp) shouldChangeLogVerbosity() {
	verbosity := app.clusterConfig.GetVirtOperatorVerbosity(app.host)
	if err := log.Log.SetVerbosityLevel(int(verbosity)); err != nil {
		log.Log.Warningf("failed up update log verbosity to %d: %v", verbosity, err)
	} else {
		log.Log.V(2).Infof("set log verbosity to %d", verbosity)
	}
}

func (app *VirtOperatorApp) shouldUpdateConfigurationMetrics() {
	emulationEnabled := app.clusterConfig.GetDeveloperConfigurationUseEmulation()
	configuration.SetEmulationEnabledMetric(emulationEnabled)
}
