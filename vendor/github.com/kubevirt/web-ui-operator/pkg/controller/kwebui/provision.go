package kwebui

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"strings"

	extenstionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	kubevirtv1alpha1 "github.com/kubevirt/web-ui-operator/pkg/apis/kubevirt/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const InventoryFilePattern = "/tmp/inventory_%s.ini"
const ConfigFilePattern = "/tmp/config_%s"
const PlaybookFile = "/opt/kwebui/kubevirt-web-ui-ansible/playbooks/kubevirt-web-ui/config.yml"
const WebUIContainerName = "console"

const PhaseFreshProvision = "PROVISION_STARTED"
const PhaseProvisioned = "PROVISIONED"
const PhaseProvisionFailed = "PROVISION_FAILED"
const PhaseDeprovision = "DEPROVISION_STARTED"
const PhaseDeprovisioned = "DEPROVISIONED"
const PhaseDeprovisionFailed = "DEPROVISION_FAILED"
const PhaseOtherError = "OTHER_ERROR"
const PhaseNoDeployment = "NOT_DEPLOYED"
const PhaseOwnerReferenceFailed = "OWNER_REFERENCE_FAILED"

func ReconcileExistingDeployment(r *ReconcileKWebUI, request reconcile.Request, instance *kubevirtv1alpha1.KWebUI, deployment *extenstionsv1beta1.Deployment) (reconcile.Result, error) {
	existingVersion := ""
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == WebUIContainerName {
			// quay.io/kubevirt/kubevirt-web-ui:v1.4
			existingVersion = AfterLast(container.Image, ":")
			log.Info(fmt.Sprintf("Existing image tag: %s, from image: %s", existingVersion, container.Image))
			// existingVersion = strings.TrimPrefix(existingVersion, "v")
			if existingVersion == "" {
				log.Info("Failed to read existing image tag")
				return reconcile.Result{}, stderrors.New("failed to read existing image tag")
			}
			break
		}
	}

	// TODO: reconcile based on other parameters, not only on the Version

	if existingVersion == "" {
		log.Info("Can not read deployed container version, giving up.")
		updateStatus(r, request, PhaseOtherError, "Can not read deployed container version.")
		return reconcile.Result{}, nil
	}

	if instance.Spec.Version == existingVersion {
		msg := fmt.Sprintf("Existing version conforms the requested one: %s. Nothing to do.", existingVersion)
		log.Info(msg)
		updateStatus(r, request, PhaseProvisioned, msg)
		return reconcile.Result{}, nil
	}

	if instance.Spec.Version == "" { // deprovision only
		return deprovision(r, request, instance)
	}

	// requested and deployed version are different
	// It should be enough to just re-execute the provision process and restart kubevirt-web-ui pod to read the updated ConfigMap.
	// But deprovision is safer to address potential incompatible changes in the future.
	_, err := deprovision(r, request, instance)
	if err != nil {
		log.Error(err, "Failed to deprovision existing deployment. Can not continue with provision of the requested one.")
		return reconcile.Result{}, err
	}

	return freshProvision(r, request, instance)
}

func runPlaybookWithSetup(namespace string, instance *kubevirtv1alpha1.KWebUI, action string) (reconcile.Result, error) {
	configFile, err := loginClient(namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	defer RemoveFile(configFile)

	inventoryFile, err := generateInventory(instance, namespace, action)
	if err != nil {
		return reconcile.Result{}, err
	}
	defer RemoveFile(inventoryFile)

	err = runPlaybook(inventoryFile, configFile)
	return reconcile.Result{}, err
}

func freshProvision(r *ReconcileKWebUI, request reconcile.Request, instance *kubevirtv1alpha1.KWebUI) (reconcile.Result, error) {
	if instance.Spec.Version == "" {
		log.Info("Removal of kubevirt-web-ui deploymnet is requested but no kubevirt-web-ui deployment found. ")
		updateStatus(r, request, PhaseNoDeployment, "")
		return reconcile.Result{}, nil
	}

	// Kubevirt-web-ui deployment is not present yet
	log.Info("kubevirt-web-ui Deployment is not present. Ansible playbook will be executed to provision it.")
	updateStatus(r, request, PhaseFreshProvision, fmt.Sprintf("Target version: %s", instance.Spec.Version))
	res, err := runPlaybookWithSetup(request.Namespace, instance, "provision")
	if err == nil {
		setOwnerReference(r, request, instance)
		updateStatus(r, request, PhaseProvisioned, "Provision finished.")
	} else {
		updateStatus(r, request, PhaseProvisionFailed, "Failed to provision Kubevirt Web UI. See operator's log for more details.")
	}
	return res, err
}

func deprovision(r *ReconcileKWebUI, request reconcile.Request, instance *kubevirtv1alpha1.KWebUI) (reconcile.Result, error) {
	log.Info("Existing kubevirt-web-ui deployment is about to be deprovisioned.")
	updateStatus(r, request, PhaseDeprovision, "")
	res, err := runPlaybookWithSetup(request.Namespace, instance, "deprovision")
	if err == nil {
		updateStatus(r, request, PhaseDeprovisioned, "Deprovision finished.")
	} else {
		updateStatus(r, request, PhaseDeprovisionFailed, "Failed to deprovision Kubevirt Web UI. See operator's log for more details.")
	}

	return res, err
}

func loginClient(namespace string) (string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get in-cluster config"))
		return "", err
	}

	configFile := fmt.Sprintf(ConfigFilePattern, Unique())
	env := []string{fmt.Sprintf("KUBECONFIG=%s", configFile)}

	cmd, args := "oc", []string{
		"login",
		config.Host,
		fmt.Sprintf("--certificate-authority=%s", config.TLSClientConfig.CAFile),
		fmt.Sprintf("--token=%s", config.BearerToken),
	}

	anonymArgs := append([]string{}, args...)
	err = RunCommand(cmd, args, env, anonymArgs)
	if err != nil {
		return "", err
	}

	cmd, args = "oc", []string{
		"project",
		namespace,
	}
	err = RunCommand(cmd, args, env, args)
	if err != nil {
		return "", err
	}

	return configFile, nil
}

func generateInventory(instance *kubevirtv1alpha1.KWebUI, namespace string, action string) (string, error) {
	log.Info("Writing inventory file")
	inventoryFile := fmt.Sprintf(InventoryFilePattern, Unique())
	f, err := os.Create(inventoryFile)
	if err != nil {
		log.Error(err, "Failed to write inventory file")
		return "", err
	}
	defer f.Close()

	registryUrl := Def(instance.Spec.RegistryUrl, os.Getenv("OPERATOR_REGISTRY"), "quay.io/kubevirt")
	registryNamespace := Def(instance.Spec.RegistryNamespace, "", "")
	version := Def(instance.Spec.Version, os.Getenv("OPERATOR_TAG"),"v1.4")
	branding := Def(instance.Spec.Branding, os.Getenv("BRANDING"), "okdvirt")
	imagePullPolicy := Def(instance.Spec.ImagePullPolicy, os.Getenv("IMAGE_PULL_POLICY"), "IfNotPresent")

	f.WriteString("[OSEv3:children]\nmasters\n\n")
	f.WriteString("[OSEv3:vars]\n")
	f.WriteString("platform=openshift\n")
	f.WriteString(strings.Join([]string{"apb_action=", action, "\n"}, ""))
	f.WriteString(strings.Join([]string{"registry_url=", registryUrl, "\n"}, ""))
	f.WriteString(strings.Join([]string{"registry_namespace=", registryNamespace, "\n"}, ""))
	f.WriteString(strings.Join([]string{"docker_tag=", version, "\n"}, ""))
	f.WriteString(strings.Join([]string{"kubevirt_web_ui_namespace=", Def(namespace, "kubevirt-web-ui", ""), "\n"}, ""))
	f.WriteString(strings.Join([]string{"kubevirt_web_ui_branding=", branding, "\n"}, ""))
	f.WriteString(strings.Join([]string{"image_pull_policy=", imagePullPolicy, "\n"}, ""))
	if action == "deprovision" {
		f.WriteString("preserve_namespace=true\n")
	}
	if instance.Spec.OpenshiftMasterDefaultSubdomain != "" {
		f.WriteString(fmt.Sprintf("openshift_master_default_subdomain=%s\n", instance.Spec.OpenshiftMasterDefaultSubdomain))
	}
	if instance.Spec.PublicMasterHostname != "" {
		f.WriteString(fmt.Sprintf("public_master_hostname=%s\n", instance.Spec.PublicMasterHostname))
	}
	f.WriteString("\n")
	f.WriteString("[masters]\n")
	_, err = f.WriteString("127.0.0.1 ansible_connection=local\n")

	if err != nil {
		log.Error(err, "Failed to write into the inventory file")
		return "", err
	}
	f.Sync()
	log.Info("The inventory file is written.")
	return inventoryFile, nil
}

func setOwnerReference(r *ReconcileKWebUI, request reconcile.Request, instance *kubevirtv1alpha1.KWebUI) error {
	deployment := &extenstionsv1beta1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: "console", Namespace: request.Namespace}, deployment)
	if err != nil {
		msg := "Failed to retrieve the just created kubevirt-web-ui Deployment object to set owner reference."
		log.Error(err, msg)
		updateStatus(r, request, PhaseOwnerReferenceFailed, msg)
		return err
	}

	controllerutil.SetControllerReference(instance, deployment, r.scheme)
	if err != nil {
		msg := "Failed to set Operator CR as the owner of the kubevirt-web-ui Deployment object."
		log.Error(err, msg)
		updateStatus(r, request, PhaseOwnerReferenceFailed, msg)
		return err
	}

	return nil
}

func runPlaybook(inventoryFile, configFile string) error {
	cmd, args := "ansible-playbook", []string{
		"-i",
		inventoryFile,
		PlaybookFile,
		"-vvv",
	}
	env := []string{fmt.Sprintf("KUBECONFIG=%s", configFile)}
	return RunCommand(cmd, args, env, args)
}

func updateStatus(r *ReconcileKWebUI, request reconcile.Request, phase string, msg string) {
	instance := &kubevirtv1alpha1.KWebUI{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get KWebUI object to update status info. Intended to write phase: '%s', message: %s", phase, msg))
		return
	}

	instance.Status.Phase = phase
	instance.Status.Message = msg
	err = r.client.Update(context.TODO(), instance)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to update KWebUI status. Intended to write phase: '%s', message: %s", phase, msg))
	}
}
