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
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/util"
)

// NamespaceTestAlternative is used to test controller-namespace independently.
var NamespaceTestAlternative = "kubevirt-test-alternative"

// NamespaceTestOperator is used to test if namespaces can still be deleted when kubevirt is uninstalled
var NamespaceTestOperator = "kubevirt-test-operator"

// NamespacePrivileged is used for helper pods that requires to be privileged
var NamespacePrivileged = "kubevirt-test-privileged"

var TestNamespaces = []string{util.NamespaceTestDefault, NamespaceTestAlternative, NamespaceTestOperator, NamespacePrivileged}

type IgnoreDeprecationWarningsLogger struct{}

func (IgnoreDeprecationWarningsLogger) HandleWarningHeader(code int, agent string, message string) {
	if !strings.Contains(message, "VirtualMachineInstancePresets is now deprecated and will be removed in v2") {
		klog.Warning(message)
	}
}

func CleanNamespaces() {
	// Replace the warning handler with a custom one that ignores certain deprecation warnings from KubeVirt
	restConfig, err := kubecli.GetKubevirtClientConfig()
	util.PanicOnError(err)

	restConfig.WarningHandler = IgnoreDeprecationWarningsLogger{}

	virtCli, err := kubecli.GetKubevirtClientFromRESTConfig(restConfig)
	util.PanicOnError(err)

	for _, namespace := range TestNamespaces {
		listOptions := metav1.ListOptions{
			LabelSelector: cleanup.TestLabelForNamespace(namespace),
		}

		_, err := virtCli.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if err != nil {
			continue
		}

		// Clean namespace labels
		err = resetNamespaceLabelsToDefault(virtCli, namespace)
		util.PanicOnError(err)

		clusterinstancetypes, err := virtCli.VirtualMachineClusterInstancetype().List(context.Background(), listOptions)
		util.PanicOnError(err)
		for _, clusterinstancetypes := range clusterinstancetypes.Items {
			util.PanicOnError(virtCli.VirtualMachineClusterInstancetype().Delete(context.Background(), clusterinstancetypes.Name, metav1.DeleteOptions{}))
		}

		instancetype, err := virtCli.VirtualMachineInstancetype(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, instancetype := range instancetype.Items {
			util.PanicOnError(virtCli.VirtualMachineInstancetype(namespace).Delete(context.Background(), instancetype.Name, metav1.DeleteOptions{}))
		}

		clusterPreference, err := virtCli.VirtualMachineClusterPreference().List(context.Background(), listOptions)
		util.PanicOnError(err)
		for _, clusterpreference := range clusterPreference.Items {
			util.PanicOnError(virtCli.VirtualMachineClusterPreference().Delete(context.Background(), clusterpreference.Name, metav1.DeleteOptions{}))
		}

		vmPreference, err := virtCli.VirtualMachinePreference(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, preference := range vmPreference.Items {
			util.PanicOnError(virtCli.VirtualMachinePreference(namespace).Delete(context.Background(), preference.Name, metav1.DeleteOptions{}))
		}

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
		vmis, err := virtCli.VirtualMachineInstance(namespace).List(context.Background(), &metav1.ListOptions{})
		util.PanicOnError(err)
		for _, vmi := range vmis.Items {
			if controller.HasFinalizer(&vmi, v1.VirtualMachineInstanceFinalizer) {
				_, err := virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"), &metav1.PatchOptions{})
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
			err := virtCli.CoreV1().Services(namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				continue
			}
			Expect(err).ToNot(HaveOccurred())
		}

		// Remove all ResourceQuota
		rqList, err := virtCli.CoreV1().ResourceQuotas(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, rq := range rqList.Items {
			err := virtCli.CoreV1().ResourceQuotas(namespace).Delete(context.Background(), rq.Name, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				continue
			}
			Expect(err).ToNot(HaveOccurred())
		}

		// Remove PVCs
		util.PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("persistentvolumeclaims").Do(context.Background()).Error())
		if libstorage.HasCDI() {
			// Remove DataVolumes
			util.PanicOnError(virtCli.CdiClient().CdiV1beta1().RESTClient().Delete().Namespace(namespace).Resource("datavolumes").Do(context.Background()).Error())
		}
		// Remove PVs
		pvs, err := virtCli.CoreV1().PersistentVolumes().List(context.Background(), listOptions)
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
		migrationPolicyList, err := virtCli.MigrationPolicy().List(context.Background(), listOptions)
		util.PanicOnError(err)
		for _, policy := range migrationPolicyList.Items {
			util.PanicOnError(virtCli.MigrationPolicy().Delete(context.Background(), policy.Name, metav1.DeleteOptions{}))
		}

		// Remove clones
		clonesList, err := virtCli.VirtualMachineClone(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, clone := range clonesList.Items {
			util.PanicOnError(virtCli.VirtualMachineClone(namespace).Delete(context.Background(), clone.Name, metav1.DeleteOptions{}))
		}

		// Remove vm snapshots
		util.PanicOnError(virtCli.VirtualMachineSnapshot(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}))
		snapshots, err := virtCli.VirtualMachineSnapshot(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		vmSnapshotFinalizer := "snapshot.kubevirt.io/vmsnapshot-protection"
		for _, snapshot := range snapshots.Items {
			if controller.HasFinalizer(&snapshot, vmSnapshotFinalizer) {
				_, err := virtCli.VirtualMachineSnapshot(snapshot.Namespace).Patch(context.Background(), snapshot.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"), metav1.PatchOptions{})
				if !errors.IsNotFound(err) {
					util.PanicOnError(err)
				}
			}
		}

		util.PanicOnError(virtCli.VirtualMachineSnapshotContent(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}))
		snapshotContentList, err := virtCli.VirtualMachineSnapshotContent(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		vmSnapshotContentFinalizer := "snapshot.kubevirt.io/vmsnapshotcontent-protection"
		for _, snapshotContent := range snapshotContentList.Items {
			if controller.HasFinalizer(&snapshotContent, vmSnapshotContentFinalizer) {
				_, err := virtCli.VirtualMachineSnapshot(snapshotContent.Namespace).Patch(context.Background(), snapshotContent.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"), metav1.PatchOptions{})
				if !errors.IsNotFound(err) {
					util.PanicOnError(err)
				}
			}
		}

		util.PanicOnError(virtCli.VirtualMachineRestore(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}))

		// Remove events
		util.PanicOnError(virtCli.CoreV1().Events(namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}))

		// Remove vmexports
		vmexportList, err := virtCli.VirtualMachineExport(namespace).List(context.Background(), metav1.ListOptions{})
		util.PanicOnError(err)
		for _, export := range vmexportList.Items {
			util.PanicOnError(virtCli.VirtualMachineExport(namespace).Delete(context.Background(), export.Name, metav1.DeleteOptions{}))
		}

	}
}

func removeNamespaces() {
	virtCli := kubevirt.Client()

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
	virtCli := kubevirt.Client()

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
	virtCli := kubevirt.Client()
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

func GetLabelsForNamespace(namespace string) map[string]string {
	labels := map[string]string{
		cleanup.TestLabelForNamespace(namespace): "",
	}
	if namespace == NamespacePrivileged {
		labels["pod-security.kubernetes.io/enforce"] = "privileged"
		labels["pod-security.kubernetes.io/warn"] = "privileged"
		labels["security.openshift.io/scc.podSecurityLabelSync"] = "false"
	}

	return labels
}

func resetNamespaceLabelsToDefault(client kubecli.KubevirtClient, namespace string) error {
	return libnet.PatchNamespace(client, namespace, func(ns *k8sv1.Namespace) {
		if ns.Labels == nil {
			return
		}
		ns.Labels = GetLabelsForNamespace(namespace)
	})
}

func createNamespaces() {
	virtCli := kubevirt.Client()

	// Create a Test Namespaces
	for _, namespace := range TestNamespaces {
		ns := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   namespace,
				Labels: GetLabelsForNamespace(namespace),
			},
		}

		_, err := virtCli.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
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
	NamespacePrivileged = fmt.Sprintf("%s%d", NamespacePrivileged, worker)
	// TODO, that is not needed, just a shortcut to not have to treat this namespace
	// differently when running in parallel
	NamespaceTestOperator = fmt.Sprintf("%s%d", NamespaceTestOperator, worker)
	TestNamespaces = []string{util.NamespaceTestDefault, NamespaceTestAlternative, NamespaceTestOperator, NamespacePrivileged}
}

func GetTestNamespace(object metav1.Object) string {
	if object != nil && object.GetNamespace() != "" {
		return object.GetNamespace()
	}

	if checks.HasFeature(virtconfig.Root) {
		return NamespacePrivileged
	}

	return util.NamespaceTestDefault
}
