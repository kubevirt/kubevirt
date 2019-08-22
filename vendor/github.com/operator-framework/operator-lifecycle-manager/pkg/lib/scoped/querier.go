package scoped

import (
	"fmt"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewUserDefinedServiceAccountQuerier returns a new instance of UserDefinedServiceAccountQuerier.
func NewUserDefinedServiceAccountQuerier(logger *logrus.Logger, crclient versioned.Interface) *UserDefinedServiceAccountQuerier {
	return &UserDefinedServiceAccountQuerier{
		logger:   logger,
		crclient: crclient,
	}
}

// UserDefinedServiceAccountQuerier retrieves reference to user defined service account(s).
type UserDefinedServiceAccountQuerier struct {
	crclient versioned.Interface
	logger   *logrus.Logger
}

// NamespaceQuerier returns an instance of ServiceAccountQuerierFunc that can be used by the
// caller to get the reference to the service account associated with the namespace.
func (f *UserDefinedServiceAccountQuerier) NamespaceQuerier(namespace string) ServiceAccountQuerierFunc {
	querierFunc := func() (reference *corev1.ObjectReference, err error) {
		logger := f.logger.WithFields(logrus.Fields{
			"namespace":  namespace,
			logFieldName: logFieldValue,
		})

		return QueryServiceAccountFromNamespace(logger, f.crclient, namespace)
	}

	return querierFunc
}

// QueryServiceAccountFromNamespace will return the reference to a service account
// associated with the operator group for the given namespace.
// - If no operator group is found in the namespace, both reference and err are set to nil.
// - If an operator group found is not managing the namespace then it is ignored.
// - If no operator group is managing this namespace then both reference and err are set to nil.
// - If more than one operator group are managing this namespace then an error is thrown.
func QueryServiceAccountFromNamespace(logger *logrus.Entry, crclient versioned.Interface, namespace string) (reference *corev1.ObjectReference, err error) {
	// TODO: use a lister instead of a noncached client here.
	list, err := crclient.OperatorsV1().OperatorGroups(namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	if len(list.Items) == 0 {
		logger.Warnf("list query returned an empty list")
		return
	}

	groups := make([]*v1.OperatorGroup, 0)
	for _, og := range list.Items {
		if len(og.Status.Namespaces) == 0 {
			logger.Warnf("skipping operator group since it is not managing any namespace og=%s", og.GetName())
			continue
		}

		groups = append(groups, &og)
	}

	if len(groups) == 0 {
		logger.Warn("no operator group found that is managing this namespace")
		return
	}

	if len(groups) > 1 {
		err = fmt.Errorf("more than one operator group(s) are managing this namespace count=%d", len(groups))
		return
	}

	group := groups[0]
	if !group.IsServiceAccountSpecified() {
		// No user defined service account is specified.
		return
	}

	if !group.HasServiceAccountSynced() {
		err = fmt.Errorf("please make sure the service account exists. sa=%s operatorgroup=%s/%s", group.Spec.ServiceAccountName, group.GetNamespace(), group.GetName())
		return
	}

	reference = group.Status.ServiceAccountRef
	return
}
