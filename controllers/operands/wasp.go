package operands

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/util/intstr"

	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	// Set dry run mode for e2e testing purposes
	waspDryRunAnnotation = "wasp.hyperconverged.io/dry-run"
)

var volumeMounts = []string{"proc", "opt", "sys", "etc", "run", "tmp"}

func newWaspHandler(Client client.Client, Scheme *runtime.Scheme) Operand {
	return &conditionalHandler{
		operand: &genericOperand{
			Client: Client,
			Scheme: Scheme,
			crType: "DaemonSet",
			hooks:  &waspHooks{},
		},
		getCRWithName: func(hc *hcov1beta1.HyperConverged) client.Object {
			return NewWaspWithNameOnly(hc)
		},
		shouldDeploy: shouldDeployWasp,
	}
}

type waspHooks struct {
	cache *appsv1.DaemonSet
}

func (h *waspHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		h.cache = NewWasp(hc)
	}

	return h.cache, nil
}

func (*waspHooks) getEmptyCr() client.Object { return &appsv1.DaemonSet{} }

func (h *waspHooks) reset() {
	h.cache = nil
}

func (*waspHooks) updateCr(
	req *common.HcoRequest,
	Client client.Client,
	exists runtime.Object,
	required runtime.Object) (bool, bool, error) {
	daemonset, ok1 := required.(*appsv1.DaemonSet)
	found, ok2 := exists.(*appsv1.DaemonSet)

	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to Daemonset")
	}

	if hasCorrectDaemonSetFields(found, daemonset) {
		return false, false, nil
	}

	if req.HCOTriggered {
		req.Logger.Info("Updating existing wasp daemonset to new opinionated values", "name", daemonset.Name)
	} else {
		req.Logger.Info("Reconciling an externally updated wasp daemonset to its opinionated values", "name", daemonset.Name)
	}

	shouldRecreacte := shouldRecreateDaemonset(found, daemonset)
	shouldUpdate := !shouldRecreacte

	var err error
	switch {
	case shouldRecreacte:
		err = recreateDaemonset(found, daemonset, Client, req)
	case shouldUpdate:
		err = updateDaemonset(found, daemonset, Client, req)
	default:
		err = fmt.Errorf("Daemonset can't both recreated and updated at the same time")
	}

	if err != nil {
		return false, false, err
	}

	return true, !req.HCOTriggered, nil

}

func (*waspHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewWasp(hc *hcov1beta1.HyperConverged) *appsv1.DaemonSet {
	waspImage, _ := os.LookupEnv(hcoutil.WaspImageEnvV)
	wasp := NewWaspWithNameOnly(hc)
	wasp.Namespace = hc.Namespace
	wasp.Spec = appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"name": string(hcoutil.AppComponentWasp),
			},
		},
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: ptr.To(intstr.FromString("10%")),
				MaxSurge:       ptr.To(intstr.FromString("0%")),
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"name": string(hcoutil.AppComponentWasp),
				},
			},
			Spec: corev1.PodSpec{
				RestartPolicy:                 corev1.RestartPolicyAlways,
				ServiceAccountName:            string(hcoutil.AppComponentWasp),
				HostPID:                       true,
				HostUsers:                     ptr.To(true),
				TerminationGracePeriodSeconds: ptr.To(int64(30)),
				Containers: []corev1.Container{
					{
						Name:            string(hcoutil.AppComponentWasp),
						Image:           waspImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "FSROOT",
								Value: "/host",
							},
							{
								Name:  "SWAPINNES",
								Value: "5",
							},
							{
								Name:  "IS_OPENSHIFT",
								Value: "true",
							},
							{
								Name: "NODE_NAME",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("50M"),
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: ptr.To(true),
						},
						VolumeMounts: createVolumeMounts(),
					},
				},
				Volumes: createVolumes(),
			},
		},
	}

	injectPlacement(hc.Spec.Workloads.NodePlacement, &wasp.Spec)
	if _, ok := hc.Annotations[waspDryRunAnnotation]; ok {
		wasp.Spec.Template.Spec.Containers[0].Env =
			append(wasp.Spec.Template.Spec.Containers[0].Env,
				corev1.EnvVar{
					Name:  "DRY_RUN",
					Value: "true",
				})
	}
	return wasp
}

func NewWaspWithNameOnly(hc *hcov1beta1.HyperConverged) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(hcoutil.AppComponentWasp),
			Labels:    getLabels(hc, hcoutil.AppComponentWasp),
			Namespace: hc.Namespace,
		},
	}
}

// We need to check only certain fields in the daemonset resource, since some of the fields
// are being set by k8s.
func hasCorrectDaemonSetFields(found *appsv1.DaemonSet, required *appsv1.DaemonSet) bool {
	return hcoutil.CompareLabels(found, required) &&
		reflect.DeepEqual(found.Spec.Selector, required.Spec.Selector) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Containers, required.Spec.Template.Spec.Containers) &&
		reflect.DeepEqual(found.Spec.Template.Spec.ServiceAccountName, required.Spec.Template.Spec.ServiceAccountName) &&
		reflect.DeepEqual(found.Spec.Template.Spec.PriorityClassName, required.Spec.Template.Spec.PriorityClassName) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Affinity, required.Spec.Template.Spec.Affinity) &&
		reflect.DeepEqual(found.Spec.Template.Spec.NodeSelector, required.Spec.Template.Spec.NodeSelector) &&
		reflect.DeepEqual(found.Spec.Template.Spec.Tolerations, required.Spec.Template.Spec.Tolerations)
}

func injectPlacement(nodePlacement *sdkapi.NodePlacement, spec *appsv1.DaemonSetSpec) {
	spec.Template.Spec.NodeSelector = nil
	spec.Template.Spec.Affinity = nil
	spec.Template.Spec.Tolerations = nil

	if nodePlacement != nil {
		if nodePlacement.NodeSelector != nil {
			spec.Template.Spec.NodeSelector = maps.Clone(nodePlacement.NodeSelector)
		}
		if nodePlacement.Affinity != nil {
			spec.Template.Spec.Affinity = nodePlacement.Affinity.DeepCopy()
		}
		if nodePlacement.Tolerations != nil {
			spec.Template.Spec.Tolerations = make([]corev1.Toleration, len(nodePlacement.Tolerations))
			copy(spec.Template.Spec.Tolerations, nodePlacement.Tolerations)
		}
	}
}

func shouldRecreateDaemonset(found, required *appsv1.DaemonSet) bool {
	// updating LabelSelector (it's immutable) would be rejected by API server; create new Deployment instead
	return !reflect.DeepEqual(found.Spec.Selector, required.Spec.Selector)
}

func shouldDeployWasp(hc *hcov1beta1.HyperConverged) bool {
	return hc.Spec.FeatureGates.EnableHigherDensityWithSwap != nil &&
		*hc.Spec.FeatureGates.EnableHigherDensityWithSwap
}

func recreateDaemonset(
	found, daemonset *appsv1.DaemonSet,
	Client client.Client,
	req *common.HcoRequest) error {

	err := Client.Delete(req.Ctx, found, &client.DeleteOptions{})
	if err != nil {
		return err
	}

	err = Client.Create(req.Ctx, daemonset, &client.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func updateDaemonset(
	found, daemonset *appsv1.DaemonSet,
	Client client.Client,
	req *common.HcoRequest) error {

	hcoutil.MergeLabels(&daemonset.ObjectMeta, &found.ObjectMeta)
	daemonset.Spec.DeepCopyInto(&found.Spec)

	err := Client.Update(req.Ctx, found)
	if err != nil {
		return err
	}

	return nil
}

func createVolumes() []corev1.Volume {
	volumes := []corev1.Volume{}
	for _, mnt := range volumeMounts {
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("host%s", mnt),
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: fmt.Sprintf("/%s", mnt),
				},
			},
		})
	}
	return volumes
}

func createVolumeMounts() []corev1.VolumeMount {
	volmnt := []corev1.VolumeMount{}
	for _, mnt := range volumeMounts {
		volmnt = append(volmnt, corev1.VolumeMount{
			Name:      fmt.Sprintf("host%s", mnt),
			MountPath: fmt.Sprintf("/host/%s", mnt),
		})
	}
	return volmnt
}
