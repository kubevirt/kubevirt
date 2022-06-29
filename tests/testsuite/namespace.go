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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package testsuite

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

// NamespaceTestAlternative is used to test controller-namespace independently.
var NamespaceTestAlternative = "kubevirt-test-alternative"

// NamespaceTestOperator is used to test if namespaces can still be deleted when kubevirt is uninstalled
var NamespaceTestOperator = "kubevirt-test-operator"

var TestNamespaces = []string{util.NamespaceTestDefault, NamespaceTestAlternative, NamespaceTestOperator}

func CleanNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	for _, namespace := range TestNamespaces {

		_, err := virtCli.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if err != nil {
			continue
		}

		// Clean namespace labels
		err = libnet.RemoveAllLabelsFromNamespace(virtCli, namespace)
		util.PanicOnError(err)

		//Remove all Jobs
		jobDeleteStrategy := metav1.DeletePropagationOrphan
		jobDeleteOptions := metav1.DeleteOptions{PropagationPolicy: &jobDeleteStrategy}
		util.PanicOnError(virtCli.BatchV1().RESTClient().Delete().Namespace(namespace).Resource("jobs").Body(&jobDeleteOptions).Do(context.Background()).Error())
		//Remove all HPA
		util.PanicOnError(virtCli.AutoscalingV1().RESTClient().Delete().Namespace(namespace).Resource("horizontalpodautoscalers").Do(context.Background()).Error())

		// Remove all VirtualMachinePools
		util.PanicOnError(virtCli.VirtualMachinePool(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}))

		// Remove all VirtualMachines
		util.PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachines").Do(context.Background()).Error())

		// Remove all VirtualMachineReplicaSets
		util.PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancereplicasets").Do(context.Background()).Error())

		// Remove all VMIs
		util.PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstances").Do(context.Background()).Error())
		vmis, err := virtCli.VirtualMachineInstance(namespace).List(&metav1.ListOptions{})
		util.PanicOnError(err)
		for _, vmi := range vmis.Items {
			if controller.HasFinalizer(&vmi, v1.VirtualMachineInstanceFinalizer) {
				_, err := virtCli.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"), &metav1.PatchOptions{})
				if !errors.IsNotFound(err) {
					util.PanicOnError(err)
				}
			}
		}

		// Remove all Pods
		podList, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		var gracePeriod int64 = 0
		for _, pod := range podList.Items {
			err := virtCli.CoreV1().Pods(namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
			if errors.IsNotFound(err) {
				continue
			}
			Expect(err).ToNot(HaveOccurred())
		}

		// Remove all Services
		svcList, err := virtCli.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, svc := range svcList.Items {
			util.PanicOnError(virtCli.CoreV1().Services(namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{}))
		}

		// Remove PVCs
		util.PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("persistentvolumeclaims").Do(context.Background()).Error())
		if libstorage.HasCDI() {
			// Remove DataVolumes
			util.PanicOnError(virtCli.CdiClient().CdiV1beta1().RESTClient().Delete().Namespace(namespace).Resource("datavolumes").Do(context.Background()).Error())
		}
		// Remove PVs
		pvs, err := virtCli.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{
			LabelSelector: cleanup.TestLabelForNamespace(namespace),
		})
		util.PanicOnError(err)
		for _, pv := range pvs.Items {
			err := virtCli.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				util.PanicOnError(err)
			}
		}

		// Remove all VirtualMachineInstance Secrets
		labelSelector := util.SecretLabel
		util.PanicOnError(
			virtCli.CoreV1().Secrets(namespace).DeleteCollection(context.Background(),
				metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector},
			),
		)

		// Remove all VirtualMachineInstance Presets
		util.PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancepresets").Do(context.Background()).Error())
		// Remove all limit ranges
		util.PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("limitranges").Do(context.Background()).Error())

		// Remove all Migration Objects
		util.PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancemigrations").Do(context.Background()).Error())
		migrations, err := virtCli.VirtualMachineInstanceMigration(namespace).List(&metav1.ListOptions{})
		util.PanicOnError(err)
		for _, migration := range migrations.Items {
			if controller.HasFinalizer(&migration, v1.VirtualMachineInstanceMigrationFinalizer) {
				_, err := virtCli.VirtualMachineInstanceMigration(namespace).Patch(migration.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"))
				if !errors.IsNotFound(err) {
					util.PanicOnError(err)
				}
			}
		}
		// Remove all NetworkAttachmentDefinitions
		nets, err := virtCli.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil && !errors.IsNotFound(err) {
			util.PanicOnError(err)
		}
		for _, netDef := range nets.Items {
			util.PanicOnError(virtCli.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Delete(context.Background(), netDef.GetName(), metav1.DeleteOptions{}))
		}

		// Remove all Istio Sidecars, VirtualServices, DestinationRules and Gateways
		for _, res := range []string{"sidecars", "virtualservices", "destinationrules", "gateways"} {
			util.PanicOnError(removeAllGroupVersionResourceFromNamespace(schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1beta1", Resource: res}, namespace))
		}

		// Remove all Istio PeerAuthentications
		util.PanicOnError(removeAllGroupVersionResourceFromNamespace(schema.GroupVersionResource{Group: "security.istio.io", Version: "v1beta1", Resource: "peerauthentications"}, namespace))

		// Remove migration policies
		migrationPolicyList, err := virtCli.MigrationPolicy().List(context.Background(), metav1.ListOptions{
			LabelSelector: cleanup.TestLabelForNamespace(namespace),
		})
		util.PanicOnError(err)
		for _, policy := range migrationPolicyList.Items {
			util.PanicOnError(virtCli.MigrationPolicy().Delete(context.Background(), policy.Name, metav1.DeleteOptions{}))
		}
	}
}

func removeNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	// First send an initial delete to every namespace
	for _, namespace := range TestNamespaces {
		err := virtCli.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			util.PanicOnError(err)
		}
	}

	// Wait until the namespaces are terminated
	fmt.Println("")
	for _, namespace := range TestNamespaces {
		fmt.Printf("Waiting for namespace %s to be removed, this can take a while ...\n", namespace)
		EventuallyWithOffset(1, func() error {
			return virtCli.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
		}, 240*time.Second, 1*time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), fmt.Sprintf("should successfully delete namespace '%s'", namespace))
	}
}

func removeAllGroupVersionResourceFromNamespace(groupVersionResource schema.GroupVersionResource, namespace string) error {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	gvr, err := virtCli.DynamicClient().Resource(groupVersionResource).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, r := range gvr.Items {
		err = virtCli.DynamicClient().Resource(groupVersionResource).Namespace(namespace).Delete(context.Background(), r.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func detectInstallNamespace() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	kvs, err := virtCli.KubeVirt("").List(&metav1.ListOptions{})
	util.PanicOnError(err)
	if len(kvs.Items) == 0 {
		util.PanicOnError(fmt.Errorf("Could not detect a kubevirt installation"))
	}
	if len(kvs.Items) > 1 {
		util.PanicOnError(fmt.Errorf("Invalid kubevirt installation, more than one KubeVirt resource found"))
	}
	flags.KubeVirtInstallNamespace = kvs.Items[0].Namespace
}

func createNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	// Create a Test Namespaces
	for _, namespace := range TestNamespaces {
		ns := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					cleanup.TestLabelForNamespace(namespace): "",
				},
			},
		}
		_, err = virtCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		if err != nil {
			util.PanicOnError(err)
		}
	}
}

// CalculateNamespaces checks on which ginkgo gest node the tests are run and sets the namespaces accordingly
func CalculateNamespaces() {
	worker := GinkgoParallelProcess()
	util.NamespaceTestDefault = fmt.Sprintf("%s%d", util.NamespaceTestDefault, worker)
	NamespaceTestAlternative = fmt.Sprintf("%s%d", NamespaceTestAlternative, worker)
	// TODO, that is not needed, just a shortcut to not have to treat this namespace
	// differently when running in parallel
	NamespaceTestOperator = fmt.Sprintf("%s%d", NamespaceTestOperator, worker)
	TestNamespaces = []string{util.NamespaceTestDefault, NamespaceTestAlternative, NamespaceTestOperator}
}
