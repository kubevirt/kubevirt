package commonTestUtils

import (
	"context"
	"github.com/go-logr/logr"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type EventEmitterMock struct{}

func (EventEmitterMock) Init(_ context.Context, _ manager.Manager, _ hcoutil.ClusterInfo, _ logr.Logger) {
	/* not implemented; mock only */
}

func (EventEmitterMock) EmitEvent(_ runtime.Object, _, _, _ string) {
	/* not implemented; mock only */
}

func (EventEmitterMock) UpdateClient(_ context.Context, _ client.Reader, _ logr.Logger) {
	/* not implemented; mock only */
}
