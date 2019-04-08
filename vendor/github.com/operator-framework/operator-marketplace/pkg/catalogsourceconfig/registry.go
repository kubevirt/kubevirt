package catalogsourceconfig

import (
	"context"
	"strconv"
	"strings"
	"time"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	containerName   = "registry-server"
	clusterRoleName = "marketplace-operator-registry-server"
	portNumber      = 50051
	portName        = "grpc"
)

var action = []string{"grpc_health_probe", "-addr=localhost:50051"}

type catalogSourceConfigWrapper struct {
	*marketplace.CatalogSourceConfig
}

func (c *catalogSourceConfigWrapper) key() client.ObjectKey {
	return client.ObjectKey{
		Name:      c.GetName(),
		Namespace: c.GetNamespace(),
	}
}

type registry struct {
	log     *logrus.Entry
	client  client.Client
	reader  datastore.Reader
	csc     catalogSourceConfigWrapper
	image   string
	address string
}

// Registry contains the method that ensures a registry-pod deployment and its
// associated resources are created.
type Registry interface {
	Ensure() error
	GetAddress() string
}

// NewRegistry returns an initialized instance of Registry
func NewRegistry(log *logrus.Entry, client client.Client, reader datastore.Reader, csc *marketplace.CatalogSourceConfig, image string) Registry {
	return &registry{
		log:    log,
		client: client,
		reader: reader,
		csc:    catalogSourceConfigWrapper{csc},
		image:  image,
	}
}

// Ensure ensures a registry-pod deployment and its associated
// resources are created.
func (r *registry) Ensure() error {
	appRegistries, secretIsPresent := r.getAppRegistryCmdLineOptions()

	// We create a ServiceAccount, Role and RoleBindings only if the registry
	// pod needs to access private registry which requires access to a secret
	if secretIsPresent {
		if err := r.ensureServiceAccount(); err != nil {
			return err
		}
		if err := r.ensureRole(); err != nil {
			return err
		}
		if err := r.ensureRoleBinding(); err != nil {
			return err
		}
	}

	if err := r.ensureDeployment(appRegistries, secretIsPresent); err != nil {
		return err
	}
	if err := r.ensureService(); err != nil {
		return err
	}
	return nil
}

func (r *registry) GetAddress() string {
	return r.address
}

// ensureDeployment ensures that registry Deployment is present for serving
// the the grpc interface for the packages from the given app registries.
// needServiceAccount indicates that the deployment is for a private registry
// and the pod requires a Service Account with the Role that allows it to access
// secrets.
func (r *registry) ensureDeployment(appRegistries string, needServiceAccount bool) error {
	registryCommand := getCommand(r.csc.GetPackages(), appRegistries)
	deployment := new(DeploymentBuilder).WithTypeMeta().Deployment()
	if err := r.client.Get(context.TODO(), r.csc.key(), deployment); err != nil {
		deployment = r.newDeployment(registryCommand, needServiceAccount)
		err = r.client.Create(context.TODO(), deployment)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create Deployment %s: %v", deployment.GetName(), err)
			return err
		}
		r.log.Infof("Created Deployment %s with registry command: %s", deployment.GetName(), registryCommand)
	} else {
		// Scale down the deployment. This is required so that we get updates
		// from Quay during the sync cycle when packages have not been added or
		// removed from the spec.
		var replicas int32
		deployment.Spec.Replicas = &replicas
		if err = r.client.Update(context.TODO(), deployment); err != nil {
			r.log.Errorf("Failed to update Deployment %s for scale down: %v", deployment.GetName(), err)
			return err
		}

		// Wait for the deployment to scale down. We need to get the latest version of the object after
		// the update, so we use the object returned here for scaling up.
		if deployment, err = r.waitForDeploymentScaleDown(2*time.Second, 1*time.Minute); err != nil {
			r.log.Errorf("Failed to scale down Deployment %s : %v", deployment.GetName(), err)
			return err
		}

		replicas = 1
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Template = r.newPodTemplateSpec(registryCommand, needServiceAccount)
		if err = r.client.Update(context.TODO(), deployment); err != nil {
			r.log.Errorf("Failed to update Deployment %s : %v", deployment.GetName(), err)
			return err
		}
		r.log.Infof("Updated Deployment %s with registry command: %s", deployment.GetName(), registryCommand)
	}
	return nil
}

// ensureRole ensure that the Role required to access secrets from the registry
// Deployment is present.
func (r *registry) ensureRole() error {
	role := new(RoleBuilder).WithTypeMeta().Role()
	if err := r.client.Get(context.TODO(), r.csc.key(), role); err != nil {
		role = r.newRole()
		err = r.client.Create(context.TODO(), role)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create Role %s: %v", role.GetName(), err)
			return err
		}
		r.log.Infof("Created Role %s", role.GetName())
	} else {
		// Update the Rules to be on the safe side
		role.Rules = getRules()
		err = r.client.Update(context.TODO(), role)
		if err != nil {
			r.log.Errorf("Failed to update Role %s : %v", role.GetName(), err)
			return err
		}
		r.log.Infof("Updated Role %s", role.GetName())
	}
	return nil
}

// ensureRoleBinding ensures that the RoleBinding bound to the Role previously
// created is present.
func (r *registry) ensureRoleBinding() error {
	roleBinding := new(RoleBindingBuilder).WithTypeMeta().RoleBinding()
	if err := r.client.Get(context.TODO(), r.csc.key(), roleBinding); err != nil {
		roleBinding = r.newRoleBinding(r.csc.GetName())
		err = r.client.Create(context.TODO(), roleBinding)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create RoleBinding %s: %v", roleBinding.GetName(), err)
			return err
		}
		r.log.Infof("Created RoleBinding %s", roleBinding.GetName())
	} else {
		// Update the Rules to be on the safe side
		roleBinding.RoleRef = NewRoleRef(r.csc.GetName())
		err = r.client.Update(context.TODO(), roleBinding)
		if err != nil {
			r.log.Errorf("Failed to update RoleBinding %s : %v", roleBinding.GetName(), err)
			return err
		}
		r.log.Infof("Updated RoleBinding %s", roleBinding.GetName())
	}
	return nil
}

// ensureService ensure that the Service for the registry deployment is present.
func (r *registry) ensureService() error {
	service := new(ServiceBuilder).WithTypeMeta().Service()
	// Delete the Service so that we get a new ClusterIP
	if err := r.client.Get(context.TODO(), r.csc.key(), service); err == nil {
		r.log.Infof("Service %s is present", service.GetName())
		err := r.client.Delete(context.TODO(), service)
		if err != nil {
			r.log.Errorf("Failed to delete Service %s", service.GetName())
			// Make a best effort to create the service
		} else {
			r.log.Infof("Deleted Service %s", service.GetName())
		}
	}
	service = r.newService()
	if err := r.client.Create(context.TODO(), service); err != nil && !errors.IsAlreadyExists(err) {
		r.log.Errorf("Failed to create Service %s: %v", service.GetName(), err)
		return err
	}
	r.log.Infof("Created Service %s", service.GetName())

	r.address = service.Spec.ClusterIP + ":" + strconv.Itoa(int(service.Spec.Ports[0].Port))
	return nil
}

// ensureServiceAccount ensure that the ServiceAccount required to be associated
// with the Deployment is present.
func (r *registry) ensureServiceAccount() error {
	serviceAccount := new(ServiceAccountBuilder).WithTypeMeta().ServiceAccount()
	if err := r.client.Get(context.TODO(), r.csc.key(), serviceAccount); err != nil {
		serviceAccount = r.newServiceAccount()
		err = r.client.Create(context.TODO(), serviceAccount)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create ServiceAccount %s: %v", serviceAccount.GetName(), err)
			return err
		}
		r.log.Infof("Created ServiceAccount %s", serviceAccount.GetName())
	} else {
		r.log.Infof("ServiceAccount %s is present", serviceAccount.GetName())
	}
	return nil
}

// getLabels returns the label that must match between the Deployment's
// LabelSelector and the Pod template's label
func (r *registry) getLabel() map[string]string {
	return map[string]string{"marketplace.catalogSourceConfig": r.csc.GetName()}
}

// getAppRegistryCmdLineOptions returns a group of "--registry=" command line
// option(s) required for operator-registry. If one of the packages is from a
// private repository secretIsPresent will be true.
func (r *registry) getAppRegistryCmdLineOptions() (appRegistryOptions string, secretIsPresent bool) {
	for _, packageID := range r.csc.Spec.GetPackageIDs() {
		opsrcMeta, err := r.reader.Read(packageID)
		if err != nil {
			r.log.Errorf("Error %v reading package %s", err, packageID)
			continue
		}
		//--registry="https://quay.io/cnr|community-operators" --regisry="https://quay.io/cnr|custom-operators|mynamespace/mysecret"
		appRegistry := "--registry=" + opsrcMeta.Endpoint + "|" + opsrcMeta.RegistryNamespace
		if opsrcMeta.SecretNamespacedName != "" {
			appRegistry += "|" + opsrcMeta.SecretNamespacedName
			secretIsPresent = true
		}
		if !strings.Contains(appRegistryOptions, appRegistry) {
			appRegistryOptions += appRegistry + " "
		}
	}
	appRegistryOptions = strings.TrimSuffix(appRegistryOptions, " ")
	return
}

// getSubjects returns the Subjects that the RoleBinding should apply to.
func (r *registry) getSubjects() []rbac.Subject {
	return []rbac.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      r.csc.GetName(),
			Namespace: r.csc.GetNamespace(),
		},
	}
}

// newDeployment() returns a Deployment object that can be used to bring up a
// registry deployment
func (r *registry) newDeployment(registryCommand []string, needServiceAccount bool) *apps.Deployment {
	return new(DeploymentBuilder).
		WithMeta(r.csc.GetName(), r.csc.GetNamespace()).
		WithOwnerLabel(r.csc.CatalogSourceConfig).
		WithSpec(1, r.getLabel(), r.newPodTemplateSpec(registryCommand, needServiceAccount)).
		Deployment()
}

// newPodTemplateSpec returns a PodTemplateSpec object that can be used to bring
// up a registry pod
func (r *registry) newPodTemplateSpec(registryCommand []string, needServiceAccount bool) core.PodTemplateSpec {
	podTemplateSpec := core.PodTemplateSpec{
		ObjectMeta: meta.ObjectMeta{
			Name:      r.csc.GetName(),
			Namespace: r.csc.GetNamespace(),
			Labels:    r.getLabel(),
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:    r.csc.GetName(),
					Image:   r.image,
					Command: registryCommand,
					Ports: []core.ContainerPort{
						{
							Name:          portName,
							ContainerPort: portNumber,
						},
					},
					ReadinessProbe: &core.Probe{
						Handler: core.Handler{
							Exec: &core.ExecAction{
								Command: action,
							},
						},
						InitialDelaySeconds: 5,
						FailureThreshold:    30,
					},
					LivenessProbe: &core.Probe{
						Handler: core.Handler{
							Exec: &core.ExecAction{
								Command: action,
							},
						},
						InitialDelaySeconds: 5,
						FailureThreshold:    30,
					},
				},
			},
		},
	}
	if needServiceAccount {
		podTemplateSpec.Spec.ServiceAccountName = r.csc.GetName()
	}
	return podTemplateSpec
}

// newRole returns a Role object with the rules set to access secrets from the
// registry pod
func (r *registry) newRole() *rbac.Role {
	return new(RoleBuilder).
		WithMeta(r.csc.GetName(), r.csc.GetNamespace()).
		WithOwnerLabel(r.csc.CatalogSourceConfig).
		WithRules(getRules()).
		Role()
}

// newRoleBinding returns a RoleBinding object RoleRef set to the given Role.
func (r *registry) newRoleBinding(roleName string) *rbac.RoleBinding {
	return new(RoleBindingBuilder).
		WithMeta(r.csc.GetName(), r.csc.GetNamespace()).
		WithOwnerLabel(r.csc.CatalogSourceConfig).
		WithSubjects(r.getSubjects()).
		WithRoleRef(roleName).
		RoleBinding()
}

// newService returns a new Service object.
func (r *registry) newService() *core.Service {
	return new(ServiceBuilder).
		WithMeta(r.csc.GetName(), r.csc.GetNamespace()).
		WithOwnerLabel(r.csc.CatalogSourceConfig).
		WithSpec(r.newServiceSpec()).
		Service()
}

// newServiceAccount returns a new ServiceAccount object.
func (r *registry) newServiceAccount() *core.ServiceAccount {
	return new(ServiceAccountBuilder).
		WithMeta(r.csc.GetName(), r.csc.GetNamespace()).
		WithOwnerLabel(r.csc.CatalogSourceConfig).
		ServiceAccount()
}

// newServiceSpec returns a ServiceSpec as required to front the registry deployment
func (r *registry) newServiceSpec() core.ServiceSpec {
	return core.ServiceSpec{
		Ports: []core.ServicePort{
			{
				Name:       portName,
				Port:       portNumber,
				TargetPort: intstr.FromInt(portNumber),
			},
		},
		Selector: r.getLabel(),
	}
}

// waitForDeploymentScaleDown waits for the deployment to scale down to zero within the timeout duration.
func (r *registry) waitForDeploymentScaleDown(retryInterval, timeout time.Duration) (*apps.Deployment, error) {
	deployment := apps.Deployment{}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = r.client.Get(context.TODO(), r.csc.key(), &deployment)
		if err != nil {
			r.log.Errorf("Deployment %s not found: %v", deployment.GetName(), err)
			return false, err
		}

		if deployment.Status.AvailableReplicas == 0 {
			return true, nil
		}
		r.log.Infof("Waiting for scale down of Deployment %s (%d/0)\n",
			deployment.GetName(), deployment.Status.AvailableReplicas)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	r.log.Infof("Deployment %s has scaled down (%d/%d)",
		deployment.GetName(), deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
	return &deployment, nil
}

// getCommand returns the command used to launch the registry server
// appregistry-server --registry="<url>|<registry namespace>|<namespaced-secret> -o <packages>"
func getCommand(packages string, registries string) []string {
	return []string{"appregistry-server", registries, "-o", packages}
}

// getRules returns the PolicyRule needed to access secrets from the registry pod
func getRules() []rbac.PolicyRule {
	return []rbac.PolicyRule{
		NewRule([]string{"get"}, []string{""}, []string{"secrets"}, nil),
	}
}
