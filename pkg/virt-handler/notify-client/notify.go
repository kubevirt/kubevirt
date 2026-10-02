package notifyclient

import (
	"context"

	"google.golang.org/grpc"

	v2 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v2"
	grpcUtil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

type NotifyClient interface {
	DomainEvents(context.Context) (DomainEventsStream, error)
	KubernetesEvents(context.Context) (KubernetesEventsStream, error)
	Close()
}

type DomainEventsStream interface {
	Recv() (*v2.DomainEvent, error)
}

type KubernetesEventsStream interface {
	Recv() (*v2.KubernetesEvent, error)
}

func NewClient(socketPath string) (NotifyClient, error) {
	conn, err := grpcUtil.DialSocket(socketPath)
	if err != nil {
		return nil, err
	}

	return &notifyClient{
		v2.NewNotifyClient(conn),
		conn,
	}, nil
}

type notifyClient struct {
	v2Client v2.NotifyClient
	conn     *grpc.ClientConn
}

var _ NotifyClient = &notifyClient{}

func (c *notifyClient) DomainEvents(ctx context.Context) (DomainEventsStream, error) {
	response, err := c.v2Client.DomainEvents(ctx, &v2.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *notifyClient) KubernetesEvents(ctx context.Context) (KubernetesEventsStream, error) {
	response, err := c.v2Client.KubernetesEvents(ctx, &v2.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *notifyClient) Close() {
	c.conn.Close()
}
