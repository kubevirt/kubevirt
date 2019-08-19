package operatorstatus

import (
	"fmt"
	"os"
	"reflect"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/version"
)

func MonitorClusterStatus(name string, syncCh chan error, stopCh <-chan struct{}, opClient operatorclient.ClientInterface, configClient configv1client.ConfigV1Interface) {
	var (
		syncs              int
		successfulSyncs    int
		hasClusterOperator bool
	)
	go wait.Until(func() {
		// slow poll until we see a cluster operator API, which could be never
		if !hasClusterOperator {
			opStatusGV := schema.GroupVersion{
				Group:   "config.openshift.io",
				Version: "v1",
			}
			err := discovery.ServerSupportsVersion(opClient.KubernetesInterface().Discovery(), opStatusGV)
			if err != nil {
				log.Infof("ClusterOperator api not present, skipping update (%v)", err)
				time.Sleep(time.Minute)
				return
			}
			hasClusterOperator = true
		}

		// Sample the sync channel and see whether we're successfully retiring syncs as a
		// proxy for "working" (we can't know when we hit level, but we can at least verify
		// we are seeing some syncs succeeding). Once we observe at least one successful
		// sync we can begin reporting available and level.
		select {
		case err, ok := <-syncCh:
			if !ok {
				// syncCh should only close if the Run() loop exits
				time.Sleep(5 * time.Second)
				log.Fatalf("Status sync channel closed but process did not exit in time")
			}
			syncs++
			if err == nil {
				successfulSyncs++
			}
			// grab any other sync events that have accumulated
			for len(syncCh) > 0 {
				if err := <-syncCh; err == nil {
					successfulSyncs++
				}
				syncs++
			}
			// if we haven't yet accumulated enough syncs, wait longer
			// TODO: replace these magic numbers with a better measure of syncs across all queueInformers
			if successfulSyncs < 5 || syncs < 10 {
				log.Printf("Waiting to observe more successful syncs")
				return
			}
		}

		// create the cluster operator in an initial state if it does not exist
		existing, err := configClient.ClusterOperators().Get(name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			log.Info("Existing operator status not found, creating")
			created, createErr := configClient.ClusterOperators().Create(&configv1.ClusterOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Status: configv1.ClusterOperatorStatus{
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:               configv1.OperatorProgressing,
							Status:             configv1.ConditionTrue,
							Message:            fmt.Sprintf("Installing %s", olmversion.OLMVersion),
							LastTransitionTime: metav1.Now(),
						},
						{
							Type:               configv1.OperatorFailing,
							Status:             configv1.ConditionFalse,
							LastTransitionTime: metav1.Now(),
						},
						{
							Type:               configv1.OperatorAvailable,
							Status:             configv1.ConditionFalse,
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			})
			if createErr != nil {
				log.Errorf("Failed to create cluster operator: %v\n", createErr)
				return
			}
			existing = created
			err = nil
		}
		if err != nil {
			log.Errorf("Unable to retrieve cluster operator: %v", err)
			return
		}

		// update the status with the appropriate state
		previousStatus := existing.Status.DeepCopy()
		switch {
		case successfulSyncs > 0:
			setOperatorStatusCondition(&existing.Status.Conditions, configv1.ClusterOperatorStatusCondition{
				Type:   configv1.OperatorFailing,
				Status: configv1.ConditionFalse,
			})
			setOperatorStatusCondition(&existing.Status.Conditions, configv1.ClusterOperatorStatusCondition{
				Type:    configv1.OperatorProgressing,
				Status:  configv1.ConditionFalse,
				Message: fmt.Sprintf("Deployed %s", olmversion.OLMVersion),
			})
			setOperatorStatusCondition(&existing.Status.Conditions, configv1.ClusterOperatorStatusCondition{
				Type:   configv1.OperatorAvailable,
				Status: configv1.ConditionTrue,
			})
			// we set the versions array when all the latest code is deployed and running - in this case,
			// the sync method is responsible for guaranteeing that happens before it returns nil
			if version := os.Getenv("RELEASE_VERSION"); len(version) > 0 {
				existing.Status.Versions = []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: version,
					},
					{
						Name:    "operator-lifecycle-manager",
						Version: olmversion.OLMVersion,
					},
				}
			} else {
				existing.Status.Versions = nil
			}
		default:
			setOperatorStatusCondition(&existing.Status.Conditions, configv1.ClusterOperatorStatusCondition{
				Type:    configv1.OperatorFailing,
				Status:  configv1.ConditionTrue,
				Message: "Waiting for updates to take effect",
			})
			setOperatorStatusCondition(&existing.Status.Conditions, configv1.ClusterOperatorStatusCondition{
				Type:    configv1.OperatorProgressing,
				Status:  configv1.ConditionFalse,
				Message: fmt.Sprintf("Waiting to see update %s succeed", olmversion.OLMVersion),
			})
			// TODO: use % errors within a window to report available
		}

		// update the status
		if !reflect.DeepEqual(previousStatus, &existing.Status) {
			if _, err := configClient.ClusterOperators().UpdateStatus(existing); err != nil {
				log.Errorf("Unable to update cluster operator status: %v", err)
			}
		}

		// if we've reported success, we can sleep longer, otherwise we want to keep watching for
		// successful
		if successfulSyncs > 0 {
			time.Sleep(5 * time.Minute)
		}

	}, 5*time.Second, stopCh)
}

func setOperatorStatusCondition(conditions *[]configv1.ClusterOperatorStatusCondition, newCondition configv1.ClusterOperatorStatusCondition) {
	if conditions == nil {
		conditions = &[]configv1.ClusterOperatorStatusCondition{}
	}
	existingCondition := findOperatorStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = newCondition.LastTransitionTime
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}

func findOperatorStatusCondition(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) *configv1.ClusterOperatorStatusCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}
