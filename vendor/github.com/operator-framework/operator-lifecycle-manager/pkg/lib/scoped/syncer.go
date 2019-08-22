package scoped

import (
	"fmt"
	"reflect"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/reference"
)

// NewUserDefinedServiceAccountSyncer returns a new instance of UserDefinedServiceAccountSyncer.
func NewUserDefinedServiceAccountSyncer(logger *logrus.Logger, scheme *runtime.Scheme, client operatorclient.ClientInterface, versioned versioned.Interface) *UserDefinedServiceAccountSyncer {
	return &UserDefinedServiceAccountSyncer{
		logger:    logger,
		versioned: versioned,
		client:    client,
		clock:     &clock.RealClock{},
		scheme:    scheme,
	}
}

// UserDefinedServiceAccountSyncer syncs an operator group appropriately when
// a user defined service account is specified.
type UserDefinedServiceAccountSyncer struct {
	versioned versioned.Interface
	client    operatorclient.ClientInterface
	logger    *logrus.Logger
	clock     clock.Clock
	scheme    *runtime.Scheme
}

const (
	// All logs should in this package should have the following field to make
	// it easy to comb through logs.
	logFieldName  = "mode"
	logFieldValue = "scoped"
)

// SyncOperatorGroup takes appropriate actions when a user specifies a service account.
func (s *UserDefinedServiceAccountSyncer) SyncOperatorGroup(in *v1.OperatorGroup) (out *v1.OperatorGroup, err error) {
	out = in
	namespace := in.GetNamespace()
	serviceAccountName := in.Spec.ServiceAccountName

	logger := s.logger.WithFields(logrus.Fields{
		"operatorGroup": in.GetName(),
		"namespace":     in.GetNamespace(),
		logFieldName:    logFieldValue,
	})

	if serviceAccountName == "" {
		if in.Status.ServiceAccountRef == nil {
			return
		}

		// User must have removed ServiceAccount from the spec. We need to
		// rest Status to a nil reference.
		out, err = s.update(in, nil)
		if err != nil {
			err = fmt.Errorf("failed to reset status.serviceAccount, sa=%s %v", serviceAccountName, err)
		}
		return
	}

	// A service account has been specified, we need to update the status.
	sa, err := s.client.KubernetesInterface().CoreV1().ServiceAccounts(namespace).Get(serviceAccountName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("failed to get service account, sa=%s %v", serviceAccountName, err)
		return
	}

	ref, err := reference.GetReference(s.scheme, sa)
	if err != nil {
		return
	}

	if reflect.DeepEqual(in.Status.ServiceAccountRef, ref) {
		logger.Debugf("status.serviceAccount is in sync with spec sa=%s", serviceAccountName)
		return
	}

	out, err = s.update(in, ref)
	if err != nil {
		err = fmt.Errorf("failed to set status.serviceAccount, sa=%s %v", serviceAccountName, err)
	}

	return
}

func (s *UserDefinedServiceAccountSyncer) update(in *v1.OperatorGroup, ref *corev1.ObjectReference) (out *v1.OperatorGroup, err error) {
	out = in

	status := out.Status.DeepCopy()
	status.ServiceAccountRef = ref
	status.LastUpdated = metav1.NewTime(s.clock.Now())

	out.Status = *status

	out, err = s.versioned.OperatorsV1().OperatorGroups(out.GetNamespace()).UpdateStatus(out)
	return
}
