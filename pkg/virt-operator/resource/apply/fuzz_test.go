package apply

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	marshalutil "kubevirt.io/kubevirt/tools/util"

	gfh "github.com/AdaLogics/go-fuzz-headers"
)

var (
	objKinds = []string{"ValidatingWebhookConfiguration",
		"MutatingWebhookConfiguration",
		"ValidatingAdmissionPolicyBinding",
		"ValidatingAdmissionPolicy",
		"APIService",
		"Secret",
		"ServiceAccount",
		"ClusterRole",
		"ClusterRoleBinding",
		"Role",
		"RoleBinding",
		"Service",
		"Deployment",
		"DaemonSet",
		"CustomResourceDefinition",
		"SecurityContextConstraints",
		"ServiceMonitor",
		"PrometheusRule",
		"ConfigMap",
		"Route",
		"VirtualMachineClusterInstancetype",
		"VirtualMachineClusterPreference",
	}
)

func loadTargetStrategyForFuzzing(resources []byte, config *util.KubeVirtDeploymentConfig, stores util.Stores) (*install.Strategy, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    config.GetNamespace(),
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
			},
		},
		Data: map[string]string{
			"manifests": string(resources),
		},
	}

	stores.InstallStrategyConfigMapCache.Add(configMap)
	targetStrategy, err := installstrategy.LoadInstallStrategyFromCache(stores, config)
	return targetStrategy, err
}

// FuzzReconciler is a fuzz harness AKA fuzzer for the
// Reconciler. It chooses the Reconcilers method it tests
// in a given iteration with the `callType` parameter.
// Based on its choice, the fuzzer will prepare the
// resources to test the method. Each method requires
// different preparations and the fuzzer will only prepare
// the resources it needs for the particular method.
// The fuzzer prepares the resources early in its iteration
// so that it fails early and before creating the Reconciler
// which requires many caches and clients. 
// At a high level, the fuzzer does three things:
// 1: Creates the resources it needs for the chosen method.
// 2: Creates the Reconciler and the types it needs.
// 3: Invokes the target API. 
func FuzzReconciler(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, callType int) {
		fdp := gfh.NewConsumer(data)
		deployment := &appsv1.Deployment{}
		cachedDeployment := &appsv1.Deployment{}
		daemonSet := &appsv1.DaemonSet{}
		crd1 := &extv1.CustomResourceDefinition{}
		crd2 := &extv1.CustomResourceDefinition{}
		stores := util.Stores{}
		mockPodDisruptionBudgetCacheStore := &MockStore{}
		stores.PodDisruptionBudgetCache = mockPodDisruptionBudgetCacheStore

		mockDSCacheStore := &MockStore{}
		stores.DaemonSetCache = mockDSCacheStore
		stores.OperatorCrdCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		var resourceBytes []byte

		// In this switch statement, the fuzzer prepares the
		// resources needed for calling the target API.
		// The fuzzer does not call its target API in this
		// switch statement. The reason for this is that
		// the fuzzer should generate these random resources
		// before doing too much instantiation of caches and
		// other structures. As such, the fuzzer will know
		// early whether it has random resources that it
		// can use to call the target API later. If it does
		// not, then it should fail fast and try again.
		switch callType % 9 {
		case 0:
			// prepare for testing syncPodDisruptionBudgetForDeployment
			// The preparations are:
			// - Add 2 PodDisruptionBudget resources to the cache
			// - Randomize `deployment` which the fuzzer passes to syncPodDisruptionBudgetForDeployment
			for _ = range 2 {
				requiredPDB := &policyv1.PodDisruptionBudget{}
				err := fdp.GenerateStruct(requiredPDB)
				if err != nil {
					return
				}
				stores.PodDisruptionBudgetCache.Add(requiredPDB)
			}
			err := fdp.GenerateStruct(deployment)
			if err != nil {
				return
			}
		case 1:
			// prepare for testing syncDaemonSet
			// The preparations are:
			// - Add two DaemonSet to the cache.
			// - Randomize `daemonSet` which the fuzzer passes to syncDaemonSet
			for _ = range 2 {
				ds := &appsv1.DaemonSet{}
				err := fdp.GenerateStruct(ds)
				if err != nil {
					return
				}
				stores.DaemonSetCache.Add(ds)
			}
			err := fdp.GenerateStruct(daemonSet)
			if err != nil {
				return
			}
		case 2:
			// prepare for testing syncDeployment
			// The preparations are:
			// - Randomize `deployment`
			// - Randomize `cachedDeployment`
			err := fdp.GenerateStruct(deployment)
			if err != nil {
				return
			}
			err = fdp.GenerateStruct(cachedDeployment)
			if err != nil {
				return
			}
		case 3:
			// prepare for testing createOrUpdateCrds
			// The preparations are:
			// - Randomize 2 CRDs
			// - Add one of the CRDs to the cache

			// Create the two resources first
			// before writing crd1 to bytes;
			// the first priority is to get
			// the crds, so we want to fail
			// fast if we fail to generate them.
			err := fdp.GenerateStruct(crd1)
			if err != nil {
				return
			}
			err = fdp.GenerateStruct(crd2)
			if err != nil {
				return
			}
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			marshalutil.MarshallObject(crd1, writer)
			writer.Flush()
			resourceBytes = b.Bytes()
			stores.OperatorCrdCache.Add(crd2)
		case 4:
			// prepare for testing createOrUpdateRbac
			// The preparations are:
			// - Create resource manifests (resourceBytes)
			// - Add 3 random resources to the cache. This can
			//   be either clusterrole, clusterrolebinding,
			//   role, rolebinding.
			var err error
			resourceBytes, err = createResourcesBytes(fdp)
			if err != nil {
				return
			}
			stores.RoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			stores.RoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			stores.ClusterRoleCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			stores.ClusterRoleBindingCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			// Add random resources to the cache
			for _ = range 3 {
				resourceType, err := fdp.GetInt()
				if err != nil {
					return
				}
				switch resourceType % 4 {
				case 0:
					resource := &rbacv1.ClusterRole{}
					err = fdp.GenerateStruct(resource)
					if err != nil {
						return
					}
					stores.ClusterRoleCache.Add(resource)
				case 1:
					resource := &rbacv1.ClusterRoleBinding{}
					err = fdp.GenerateStruct(resource)
					if err != nil {
						return
					}
					stores.ClusterRoleBindingCache.Add(resource)
				case 2:
					resource := &rbacv1.Role{}
					err = fdp.GenerateStruct(resource)
					if err != nil {
						return
					}
					stores.RoleCache.Add(resource)
				case 3:
					resource := &rbacv1.RoleBinding{}
					err = fdp.GenerateStruct(resource)
					if err != nil {
						return
					}
					stores.RoleBindingCache.Add(resource)
				}
			}
		case 5:
			// prepare for testing createOrUpdateServiceMonitors
			// The preparations are:
			// - Create a list of 2 random ServiceMonitor manifests
			// - Add 2 random ServiceMonitor to the cache.
			stores.ServiceMonitorCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			for _ = range 2 {
				sm1 := &promv1.ServiceMonitor{}
				err := fdp.GenerateStruct(sm1)
				if err != nil {
					return
				}

				marshalutil.MarshallObject(sm1, writer)
				// Split the resources in the manifest
				writer.WriteString("---")

				sm2 := &promv1.ServiceMonitor{}
				err = fdp.GenerateStruct(sm2)
				if err != nil {
					return
				}
				stores.ServiceMonitorCache.Add(sm2)
			}
			writer.Flush()
			resourceBytes = b.Bytes()
		case 6:
			// prepare for testing createOrUpdateInstancetypes
			// The preparations are:
			// Create a manifest of 2 random VirtualMachineClusterInstancetype
			// Add 2 random VirtualMachineClusterInstancetype to the cache
			clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
			clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
			stores.ClusterInstancetype = clusterInstancetypeInformer.GetStore()
			stores.ClusterPreference = clusterPreferenceInformer.GetStore()
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			for _ = range 2 {
				// Create a random instance type for the manifest.
				instancetype1 := &instancetypev1beta1.VirtualMachineClusterInstancetype{}
				err := fdp.GenerateStruct(instancetype1)
				if err != nil {
					return
				}
				marshalutil.MarshallObject(instancetype1, writer)
				// Split the resources in the manifest
				writer.WriteString("---")

				// Create a random instance type and add it to the cache.
				instancetype2 := &instancetypev1beta1.VirtualMachineClusterInstancetype{}
				err = fdp.GenerateStruct(instancetype2)
				if err != nil {
					return
				}
				stores.ClusterInstancetype.Add(instancetype2)
			}
			writer.Flush()
			resourceBytes = b.Bytes()
		case 7:
			// prepare for testing createOrUpdateSCC
			// The preparations are:
			// - Create manifest of 2 random SecurityContextConstraints
			// - Add 2 random SecurityContextConstraints to the cache
			var informers util.Informers
			informers.SCC, _ = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
			stores.SCCCache = informers.SCC.GetStore()

			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			for _ = range 2 {
				// Create a random SecurityContextConstraints
				// for the manifest.
				scc1 := &secv1.SecurityContextConstraints{}
				err := fdp.GenerateStruct(scc1)
				if err != nil {
					return
				}
				marshalutil.MarshallObject(scc1, writer)
				// Split the resources in the manifest
				writer.WriteString("---")

				// Create a random SecurityContextConstraints
				// and add it to the cache.
				scc2 := &secv1.SecurityContextConstraints{}
				err = fdp.GenerateStruct(scc2)
				if err != nil {
					return
				}
				stores.SCCCache.Add(scc2)
			}
			writer.Flush()
			resourceBytes = b.Bytes()
		case 8:
			// prepare for testing createOrUpdateValidatingAdmissionPolicyBindings
			// The preparations are:
			// - Create manifest of 2 random ValidatingAdmissionPolicyBinding
			// - Add 2 random ValidatingAdmissionPolicyBinding to the cache.
			var informers util.Informers
			informers.ValidationWebhook, _ = testutils.NewFakeInformerFor(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
			stores.ValidationWebhookCache = informers.ValidationWebhook.GetStore()
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			for _ = range 2 {
				// Create a random ValidatingAdmissionPolicyBinding
				// for the manifest
				validatingAdmissionPolicyBinding1 := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
				err := fdp.GenerateStruct(validatingAdmissionPolicyBinding1)
				if err != nil {
					return
				}
				marshalutil.MarshallObject(validatingAdmissionPolicyBinding1, writer)
				// Split the resources in the manifest
				writer.WriteString("---")

				// Create a random ValidatingAdmissionPolicyBinding
				// and add it to the cache.
				validatingAdmissionPolicyBinding2 := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
				err = fdp.GenerateStruct(validatingAdmissionPolicyBinding2)
				if err != nil {
					return
				}
				stores.ValidationWebhookCache.Add(validatingAdmissionPolicyBinding2)
			}
			writer.Flush()
			resourceBytes = b.Bytes()
		default:
			return
		}

		// Create the reconciler
		// At this point, the fuzzer has prepared the specific resources
		// for the target API. It can now create the the reconciler
		// and the caches and mocking it needs.
		ctrl := gomock.NewController(t)
		clientset := kubecli.NewMockKubevirtClient(ctrl)
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: Namespace,
			},
		}
		expectations := &util.Expectations{}
		expectations.DaemonSet = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet"))
		expectations.PodDisruptionBudget = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudgets"))

		mockPodCacheStore := &cache.FakeCustomStore{}

		stores.InfrastructurePodCache = mockPodCacheStore
		stores.InstallStrategyConfigMapCache = cache.NewStore(cache.MetaNamespaceKeyFunc)

		pdbClient := fake.NewSimpleClientset()
		dsClient := fake.NewSimpleClientset()

		kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
		clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
		clientset.EXPECT().PolicyV1().Return(pdbClient.PolicyV1()).AnyTimes()
		clientset.EXPECT().AppsV1().Return(dsClient.AppsV1()).AnyTimes()
		secClient := &secv1fake.FakeSecurityV1{
			Fake: &fake.NewSimpleClientset().Fake,
		}
		clientset.EXPECT().SecClient().Return(secClient).AnyTimes()

		r := &Reconciler{
			clientset:    clientset,
			kv:           kv,
			expectations: expectations,
			stores:       stores,
			recorder:     record.NewFakeRecorder(100),
		}

		// At this point the fuzzer has done most of the setup.
		// Some of the cases below do some specific additional
		// setup. At this point, each case corresponds to the
		// cases earlier in the fuzzer. Here, we invoke the
		// target API.
		switch callType % 9 {
		case 0:
			// test syncPodDisruptionBudgetForDeployment
			r.syncPodDisruptionBudgetForDeployment(deployment)
		case 1:
			// test syncDaemonSet
			r.syncDaemonSet(daemonSet)
		case 2:
			// test syncDeployment
			r.stores.DeploymentCache = &MockStore{get: cachedDeployment}
			r.syncDeployment(deployment)
		case 3:
			// test createOrUpdateCrds
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateCrds()
		case 4:
			// test createOrUpdateRbac
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateRbac()
		case 5:
			// test createOrUpdateServiceMonitors
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateServiceMonitors()
		case 6:
			// test createOrUpdateInstancetypes
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateInstancetypes()
		case 7:
			// test createOrUpdateSCC
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateSCC()
		case 8:
			// test createOrUpdateValidatingAdmissionPolicyBindings
			config := getConfig("fake-registry", "v9.9.9")
			targetStrategy, err := loadTargetStrategyForFuzzing(resourceBytes, config, r.stores)
			if err != nil {
				return
			}
			r.targetStrategy = targetStrategy
			r.createOrUpdateValidatingAdmissionPolicyBindings()
		default:
			return
		}
	})
}

// Creates a manifest of 3 random resources.
func createResourcesBytes(fdp *gfh.ConsumeFuzzer) ([]byte, error) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	for _ = range 3 {
		objKindIndex, err := fdp.GetInt()
		if err != nil {
			return b.Bytes(), err
		}

		switch objKinds[objKindIndex%len(objKinds)] {
		case "ValidatingWebhookConfiguration":
			webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
			err := fdp.GenerateStruct(webhook)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(webhook, writer)
			writer.WriteString("---")
		case "MutatingWebhookConfiguration":
			webhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
			err := fdp.GenerateStruct(webhook)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(webhook, writer)
			writer.WriteString("---")
		case "ValidatingAdmissionPolicyBinding":
			validatingAdmissionPolicyBinding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			err := fdp.GenerateStruct(validatingAdmissionPolicyBinding)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(validatingAdmissionPolicyBinding, writer)
			writer.WriteString("---")
		case "ValidatingAdmissionPolicy":
			validatingAdmissionPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			err := fdp.GenerateStruct(validatingAdmissionPolicy)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(validatingAdmissionPolicy, writer)
			writer.WriteString("---")
		case "APIService":
			apiService := &apiregv1.APIService{}
			err := fdp.GenerateStruct(apiService)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(apiService, writer)
			writer.WriteString("---")
		case "Secret":
			secret := &corev1.Secret{}
			err := fdp.GenerateStruct(secret)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(secret, writer)
			writer.WriteString("---")
		case "ServiceAccount":
			sa := &corev1.ServiceAccount{}
			err := fdp.GenerateStruct(sa)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(sa, writer)
			writer.WriteString("---")
		case "ClusterRole":
			cr := &rbacv1.ClusterRole{}
			err := fdp.GenerateStruct(cr)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(cr, writer)
			writer.WriteString("---")
		case "ClusterRoleBinding":
			crb := &rbacv1.ClusterRoleBinding{}
			err := fdp.GenerateStruct(crb)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(crb, writer)
			writer.WriteString("---")
		case "Role":
			r := &rbacv1.Role{}
			err := fdp.GenerateStruct(r)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(r, writer)
			writer.WriteString("---")
		case "RoleBinding":
			rb := &rbacv1.RoleBinding{}
			err := fdp.GenerateStruct(rb)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(rb, writer)
			writer.WriteString("---")
		case "Service":
			s := &corev1.Service{}
			err := fdp.GenerateStruct(s)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(s, writer)
			writer.WriteString("---")
		case "Deployment":
			d := &appsv1.Deployment{}
			err := fdp.GenerateStruct(d)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(d, writer)
			writer.WriteString("---")
		case "DaemonSet":
			d := &appsv1.DaemonSet{}
			err := fdp.GenerateStruct(d)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(d, writer)
			writer.WriteString("---")
		case "CustomResourceDefinition":
			crdv1 := &extv1.CustomResourceDefinition{}
			err := fdp.GenerateStruct(crdv1)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(crdv1, writer)
			writer.WriteString("---")
		case "SecurityContextConstraints":
			s := &secv1.SecurityContextConstraints{}
			err := fdp.GenerateStruct(s)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(s, writer)
			writer.WriteString("---")
		case "ServiceMonitor":
			sm := &promv1.ServiceMonitor{}
			err := fdp.GenerateStruct(sm)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(sm, writer)
			writer.WriteString("---")
		case "PrometheusRule":
			pr := &promv1.PrometheusRule{}
			err := fdp.GenerateStruct(pr)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(pr, writer)
			writer.WriteString("---")
		case "ConfigMap":
			configMap := &corev1.ConfigMap{}
			err := fdp.GenerateStruct(configMap)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(configMap, writer)
			writer.WriteString("---")
		case "Route":
			route := &routev1.Route{}
			err := fdp.GenerateStruct(route)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(route, writer)
			writer.WriteString("---")
		case "VirtualMachineClusterInstancetype":
			instancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{}
			err := fdp.GenerateStruct(instancetype)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(instancetype, writer)
			writer.WriteString("---")
		case "VirtualMachineClusterPreference":
			preference := &instancetypev1beta1.VirtualMachineClusterPreference{}
			err := fdp.GenerateStruct(preference)
			if err != nil {
				return b.Bytes(), err
			}
			marshalutil.MarshallObject(preference, writer)
			writer.WriteString("---")
		default:
			continue

		}
		writer.Flush()

	}
	return b.Bytes(), nil
}
