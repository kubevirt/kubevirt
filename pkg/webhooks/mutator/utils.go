package mutator

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func getHcoObject(ctx context.Context, cli client.Client, namespace string) (*v1beta1.HyperConverged, error) {
	hco := &v1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoutil.HyperConvergedName,
			Namespace: namespace,
		},
	}

	err := cli.Get(ctx, client.ObjectKeyFromObject(hco), hco)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("HyperConverged CR does not exist")
			return nil, err
		}

		logger.Error(err, "failed getting HyperConverged CR")
		return nil, err
	}

	return hco, nil
}
