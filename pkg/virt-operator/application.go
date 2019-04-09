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
	"io/ioutil"
	golog "log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/certificates"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/service"
	kvutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/install-strategy"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

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

	kubeVirtController KubeVirtController
	kubeVirtRecorder   record.EventRecorder

	operatorNamespace string

	kubeVirtInformer cache.SharedIndexInformer
	kubeVirtCache    cache.Store

	stores    util.Stores
	informers util.Informers

	LeaderElection leaderelectionconfig.Configuration
}

var _ service.Service = &VirtOperatorApp{}

func Execute() {
	var err error
	app := VirtOperatorApp{}

	dumpInstallStrategy := pflag.Bool("dump-install-strategy", false, "Dump install strategy to configmap and exit")

	service.Setup(&app)

	log.InitializeLogging("virt-operator")

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	app.operatorNamespace, err = kvutil.GetNamespace()
	if err != nil {
		golog.Fatalf("Error searching for namespace: %v", err)
	}

	if *dumpInstallStrategy {
		err = installstrategy.DumpInstallStrategyToConfigMap(app.clientSet)
		if err != nil {
			golog.Fatal(err)
		}
		os.Exit(0)
	}

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet, app.operatorNamespace)

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
		InstallStrategyConfigMap: app.informerFactory.OperatorInstallStrategyConfigMaps(),
		InstallStrategyJob:       app.informerFactory.OperatorInstallStrategyJob(),
		InfrastructurePod:        app.informerFactory.OperatorPod(),
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
		InstallStrategyConfigMapCache: app.informerFactory.OperatorInstallStrategyConfigMaps().GetStore(),
		InstallStrategyJobCache:       app.informerFactory.OperatorInstallStrategyJob().GetStore(),
		InfrastructurePodCache:        app.informerFactory.OperatorPod().GetStore(),
	}

	onOpenShift, err := util.IsOnOpenshift(app.clientSet)
	if err != nil {
		golog.Fatalf("Error determining cluster type: %v", err)
	}
	if onOpenShift {
		log.Log.Info("we are on openshift")
		app.informers.SCC = app.informerFactory.OperatorSCC()
		app.stores.SCCCache = app.informerFactory.OperatorSCC().GetStore()
	} else {
		log.Log.Info("we are on kubernetes")
		app.informers.SCC = app.informerFactory.DummyOperatorSCC()
		app.stores.SCCCache = app.informerFactory.DummyOperatorSCC().GetStore()
	}

	app.kubeVirtRecorder = app.getNewRecorder(k8sv1.NamespaceAll, "virt-operator")
	app.kubeVirtController = *NewKubeVirtController(app.clientSet, app.kubeVirtInformer, app.kubeVirtRecorder, app.stores, app.informers)

	image := os.Getenv(util.OperatorImageEnvName)
	if image == "" {
		golog.Fatalf("Error getting operator's image: %v", err)
	}
	log.Log.Infof("Operator image: %s", image)

	app.Run()
}

func (app *VirtOperatorApp) Run() {

	// prepare certs
	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)

	certStore, err := certificates.GenerateSelfSignedCert(certsDirectory, "virt-operator", app.operatorNamespace)
	if err != nil {
		log.Log.Reason(err).Error("unable to generate certificates")
		panic(err)
	}

	go func() {
		// serve metrics
		http.Handle("/metrics", promhttp.Handler())
		err = http.ListenAndServeTLS(app.ServiceListen.Address(), certStore.CurrentPath(), certStore.CurrentPath(), nil)
		if err != nil {
			log.Log.Reason(err).Error("Serving prometheus failed.")
			panic(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	endpointName := "virt-operator"

	recorder := app.getNewRecorder(k8sv1.NamespaceAll, endpointName)

	id, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}

	rl, err := resourcelock.New(app.LeaderElection.ResourceLock,
		app.operatorNamespace,
		endpointName,
		app.clientSet.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		golog.Fatal(err)
	}

	leaderElector, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: app.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: app.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   app.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					log.Log.Infof("Started leading")
					// run app
					stop := ctx.Done()
					app.informerFactory.Start(stop)
					go app.kubeVirtController.Run(controllerThreads, stop)
				},
				OnStoppedLeading: func() {
					golog.Fatal("leaderelection lost")
				},
			},
		})
	if err != nil {
		golog.Fatal(err)
	}

	log.Log.Infof("Attempting to aquire leader status")
	leaderElector.Run(ctx)
	panic("unreachable")

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
