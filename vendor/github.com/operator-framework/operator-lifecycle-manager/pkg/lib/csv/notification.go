package csv

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// WatchNotification is an sink interface that can be used to get notification
// of CSV reconciliation request(s) received by the operator.
type WatchNotification interface {
	// OnAddOrUpdate is invoked when a add or update reconciliation request has
	// been received by the operator.
	OnAddOrUpdate(in *v1alpha1.ClusterServiceVersion)

	// OnDelete is invoked when a delete reconciliation request has
	// been received by the operator.
	OnDelete(in *v1alpha1.ClusterServiceVersion)
}
