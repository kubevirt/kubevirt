package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/components"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/names"
)

const (
	CheckImmediately   = time.Microsecond
	CheckDoNotRepeat   = time.Microsecond
	CheckIgnoreVersion = "IGNORE"
)

func CheckComponentsDeployment(components []Component) {
	for _, component := range components {
		By(fmt.Sprintf("Checking that component %s is deployed", component.ComponentName))
		err := checkForComponent(&component)
		Expect(err).NotTo(HaveOccurred(), "Component has not been fully deployed")
	}
}

func CheckComponentsRemoval(components []Component) {
	for _, component := range components {
		// TODO: Once finalizers are implemented, we should switch to using
		// once-time checks, since after NodeNetworkState removal, no components
		// should be left over.
		By(fmt.Sprintf("Checking that component %s has been removed", component.ComponentName))
		Eventually(func() error {
			return checkForComponentRemoval(&component)
		}, 5*time.Minute, time.Second).ShouldNot(HaveOccurred(), "Component has not been fully removed within the given timeout")
	}
}

func CheckConfigCondition(conditionType ConditionType, conditionStatus ConditionStatus, timeout time.Duration, duration time.Duration) {
	By(fmt.Sprintf("Checking that condition %q status is set to %s", conditionType, conditionStatus))
	config := &opv1alpha1.NetworkAddonsConfig{}

	getAndCheckCondition := func() error {
		err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: names.OPERATOR_CONFIG}, config)
		if err != nil {
			return err
		}
		return checkConfigCondition(config, conditionType, conditionStatus)
	}

	if timeout != CheckImmediately {
		Eventually(getAndCheckCondition, timeout, 10*time.Millisecond).ShouldNot(HaveOccurred(), fmt.Sprintf("Timed out waiting for the condition, current config:\n%v", configToYaml(config)))
	} else {
		Expect(getAndCheckCondition()).NotTo(HaveOccurred(), fmt.Sprintf("Condition is not in the expected state, current config:\n%v", configToYaml(config)))
	}

	if duration != CheckDoNotRepeat {
		Consistently(getAndCheckCondition, duration, 10*time.Millisecond).ShouldNot(HaveOccurred(), fmt.Sprintf("Condition prematurely changed its value, current config:\n%v", configToYaml(config)))
	}
}

func CheckConfigVersions(operatorVersion, observedVersion, targetVersion string, timeout, duration time.Duration) {
	By(fmt.Sprintf("Checking that status contains expected versions Operator: %q, Observed: %q, Target: %q", operatorVersion, observedVersion, targetVersion))
	config := &opv1alpha1.NetworkAddonsConfig{}

	getAndCheckVersions := func() error {
		err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: names.OPERATOR_CONFIG}, config)
		if err != nil {
			return err
		}

		errs := []error{}
		errsAppend := func(err error) {
			if err != nil {
				errs = append(errs, err)
			}
		}

		if operatorVersion != CheckIgnoreVersion && config.Status.OperatorVersion != operatorVersion {
			errsAppend(fmt.Errorf("OperatorVersion %q does not match expected %q", config.Status.OperatorVersion, operatorVersion))
		}

		if observedVersion != CheckIgnoreVersion && config.Status.ObservedVersion != observedVersion {
			errsAppend(fmt.Errorf("ObservedVersion %q does not match expected %q", config.Status.ObservedVersion, observedVersion))
		}

		if targetVersion != CheckIgnoreVersion && config.Status.TargetVersion != targetVersion {
			errsAppend(fmt.Errorf("TargetVersion %q does not match expected %q", config.Status.TargetVersion, targetVersion))
		}

		return errsToErr(errs)
	}

	if timeout != CheckImmediately {
		Eventually(getAndCheckVersions, timeout, time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("Timed out waiting for the expected versions, current config:\n%v", configToYaml(config)))
	} else {
		Expect(getAndCheckVersions()).NotTo(HaveOccurred(), fmt.Sprintf("Versions are not in the expected state, current config:\n%v", configToYaml(config)))
	}

	if duration != CheckDoNotRepeat {
		Consistently(getAndCheckVersions, duration, time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("Versions prematurely changed their values, current config:\n%v", configToYaml(config)))
	}
}

func CheckOperatorIsReady(timeout time.Duration) {
	By("Checking that the operator is up and running")
	if timeout != CheckImmediately {
		Eventually(func() error {
			return checkForDeployment(components.Name)
		}, timeout, time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("Timed out waiting for the operator to become ready"))
	} else {
		Expect(checkForDeployment(components.Name)).ShouldNot(HaveOccurred(), "Operator is not ready")
	}
}

func CheckForLeftoverObjects(currentVersion string) {
	listOptions := client.ListOptions{}
	key := opv1alpha1.SchemeGroupVersion.Group + "/version"
	listOptions.SetLabelSelector(fmt.Sprintf("%s,%s != %s", key, key, currentVersion))

	deployments := appsv1.DeploymentList{}
	err := framework.Global.Client.List(context.Background(), &listOptions, &deployments)
	Expect(err).NotTo(HaveOccurred())
	Expect(deployments.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	daemonSets := appsv1.DaemonSetList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &daemonSets)
	Expect(err).NotTo(HaveOccurred())
	Expect(daemonSets.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	configMaps := corev1.ConfigMapList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &configMaps)
	Expect(err).NotTo(HaveOccurred())
	Expect(configMaps.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	namespaces := corev1.NamespaceList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &namespaces)
	Expect(err).NotTo(HaveOccurred())
	Expect(namespaces.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	secrets := corev1.SecretList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &secrets)
	Expect(err).NotTo(HaveOccurred())
	Expect(secrets.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	clusterRoles := rbacv1.ClusterRoleList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &clusterRoles)
	Expect(err).NotTo(HaveOccurred())
	Expect(clusterRoles.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	clusterRoleBindings := rbacv1.ClusterRoleList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &clusterRoleBindings)
	Expect(err).NotTo(HaveOccurred())
	Expect(clusterRoleBindings.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	roles := rbacv1.RoleList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &roles)
	Expect(err).NotTo(HaveOccurred())
	Expect(roles.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	roleBindings := rbacv1.RoleBindingList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &roleBindings)
	Expect(err).NotTo(HaveOccurred())
	Expect(roleBindings.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")

	serviceAccounts := corev1.ServiceAccountList{}
	err = framework.Global.Client.List(context.Background(), &listOptions, &serviceAccounts)
	Expect(err).NotTo(HaveOccurred())
	Expect(serviceAccounts.Items).To(BeEmpty(), "Found leftover objects from the previous operator version")
}

func KeepCheckingWhile(check func(), while func()) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	done := make(chan bool)

	go func() {
		// Perform some long running operation
		while()

		// Finally close the validator
		close(done)
	}()

	// Keep checking while the goroutine is running
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			check()
		}
	}
}

func checkForComponent(component *Component) error {
	errs := []error{}
	errsAppend := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if component.ClusterRole != "" {
		errsAppend(checkForClusterRole(component.ClusterRole))
	}

	if component.ClusterRoleBinding != "" {
		errsAppend(checkForClusterRoleBinding(component.ClusterRoleBinding))
	}

	if component.SecurityContextConstraints != "" {
		errsAppend(checkForSecurityContextConstraints(component.SecurityContextConstraints))
	}

	for _, daemonSet := range component.DaemonSets {
		errsAppend(checkForDaemonSet(daemonSet))
	}

	for _, deployment := range component.Deployments {
		errsAppend(checkForDeployment(deployment))
	}

	return errsToErr(errs)
}

func checkForComponentRemoval(component *Component) error {
	errs := []error{}
	errsAppend := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if component.ClusterRole != "" {
		errsAppend(checkForClusterRoleRemoval(component.ClusterRole))
	}

	if component.ClusterRoleBinding != "" {
		errsAppend(checkForClusterRoleBindingRemoval(component.ClusterRoleBinding))
	}

	if component.SecurityContextConstraints != "" {
		errsAppend(checkForSecurityContextConstraintsRemoval(component.SecurityContextConstraints))
	}

	return errsToErr(errs)
}

func errsToErr(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	errsStrings := []string{}
	for _, err := range errs {
		errsStrings = append(errsStrings, err.Error())
	}
	return errors.New(strings.Join(errsStrings, "\n"))
}

func checkForClusterRole(name string) error {
	return framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &rbacv1.ClusterRole{})
}

func checkForClusterRoleBinding(name string) error {
	return framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &rbacv1.ClusterRoleBinding{})
}

func checkForSecurityContextConstraints(name string) error {
	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &securityapi.SecurityContextConstraints{})
	if isNotSupportedKind(err) {
		return nil
	}
	return err
}

func checkForDeployment(name string) error {
	deployment := appsv1.Deployment{}

	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: components.Namespace}, &deployment)
	if err != nil {
		return err
	}

	labels := deployment.GetLabels()
	if labels != nil {
		if _, operatorLabelSet := labels[opv1alpha1.SchemeGroupVersion.Group+"/version"]; !operatorLabelSet {
			return fmt.Errorf("Deployment %s/%s is missing operator label", components.Namespace, name)
		}
	}

	if deployment.Status.UnavailableReplicas > 0 || deployment.Status.AvailableReplicas == 0 {
		manifest, err := yaml.Marshal(deployment)
		if err != nil {
			panic(err)
		}
		return fmt.Errorf("Deployment %s/%s is not ready, current state:\n%v", components.Namespace, name, string(manifest))
	}

	return nil
}

func checkForDaemonSet(name string) error {
	daemonSet := appsv1.DaemonSet{}

	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: components.Namespace}, &daemonSet)
	if err != nil {
		return err
	}

	labels := daemonSet.GetLabels()
	if labels != nil {
		if _, operatorLabelSet := labels[opv1alpha1.SchemeGroupVersion.Group+"/version"]; !operatorLabelSet {
			return fmt.Errorf("DaemonSet %s/%s is missing operator label", components.Namespace, name)
		}
	}

	if daemonSet.Status.NumberUnavailable > 0 || daemonSet.Status.NumberAvailable == 0 {
		manifest, err := yaml.Marshal(daemonSet)
		if err != nil {
			panic(err)
		}
		return fmt.Errorf("DaemonSet %s/%s is not ready, current state:\n%v", components.Namespace, name, string(manifest))
	}

	return nil
}

func checkForClusterRoleRemoval(name string) error {
	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &rbacv1.ClusterRole{})
	return isNotFound("ClusterRole", name, err)
}

func checkForClusterRoleBindingRemoval(name string) error {
	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &rbacv1.ClusterRoleBinding{})
	return isNotFound("ClusterRoleBinding", name, err)
}

func checkForSecurityContextConstraintsRemoval(name string) error {
	err := framework.Global.Client.Get(context.Background(), types.NamespacedName{Name: name}, &securityapi.SecurityContextConstraints{})
	if isNotSupportedKind(err) {
		return nil
	}
	return isNotFound("SecurityContextConstraints", name, err)
}

func isNotFound(componentType string, componentName string, clientGetOutput error) error {
	if clientGetOutput != nil {
		if apierrors.IsNotFound(clientGetOutput) {
			return nil
		}
		return clientGetOutput
	}
	return fmt.Errorf("%s %q has been found", componentType, componentName)
}

func checkConfigCondition(conf *opv1alpha1.NetworkAddonsConfig, conditionType ConditionType, conditionStatus ConditionStatus) error {
	for _, condition := range conf.Status.Conditions {
		if condition.Type == conditionsv1.ConditionType(conditionType) {
			if condition.Status == corev1.ConditionStatus(conditionStatus) {
				return nil
			}
			return fmt.Errorf("condition %q is not in expected state %q, obtained state %q, obtained config %v", conditionType, conditionStatus, condition.Status, configToYaml(conf))
		}
	}

	// If a condition is missing, it is considered to be False
	if conditionStatus == ConditionFalse {
		return nil
	}

	return fmt.Errorf("condition %q has not been found in the config", conditionType)
}

func isNotSupportedKind(err error) bool {
	return strings.Contains(err.Error(), "no kind is registered for the type")
}

func configToYaml(config *opv1alpha1.NetworkAddonsConfig) string {
	manifest, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	return string(manifest)
}

// CheckUnicast return an error in case that the given addresses support multicast or is invalid.
func CheckUnicastAndValidity() (string, string) {
	rangeStart, rangeEnd := retrieveRange()
	parsedRangeStart, err := net.ParseMAC(rangeStart)
	Expect(err).ToNot(HaveOccurred())
	checkUnicast(parsedRangeStart)

	parsedRangeEnd, err := net.ParseMAC(rangeEnd)
	Expect(err).ToNot(HaveOccurred())
	checkUnicast(parsedRangeEnd)
	return rangeStart, rangeEnd
}

func checkUnicast(mac net.HardwareAddr) {
	// A bitwise AND between 00000001 and the mac address first octet.
	// In case where the LSB of the first octet (the multicast bit) is on, it will return 1, and 0 otherwise.
	multicastBit := 1 & mac[0]
	Expect(multicastBit).ToNot(BeNumerically("==", 1), "invalid mac address. Multicast addressing is not supported. Unicast addressing must be used. The first octet is %#0X", mac[0])
}

func retrieveRange() (string, string) {
	configMap := &corev1.ConfigMap{}
	Eventually(func() error {

		return framework.Global.Client.Get(context.TODO(),
			types.NamespacedName{Namespace: components.Namespace, Name: names.APPLIED_PREFIX + names.OPERATOR_CONFIG}, configMap)

	}, 50*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

	appliedData, exist := configMap.Data["applied"]
	Expect(exist).To(BeTrue(), "applied data not found in configMap")

	appliedConfig := &opv1alpha1.NetworkAddonsConfigSpec{}
	err := json.Unmarshal([]byte(appliedData), appliedConfig)
	Expect(err).ToNot(HaveOccurred())

	Expect(appliedConfig.KubeMacPool).ToNot(BeNil(), "kubemacpool config doesn't exist")

	rangeStart := appliedConfig.KubeMacPool.RangeStart
	rangeEnd := appliedConfig.KubeMacPool.RangeEnd
	return rangeStart, rangeEnd
}
