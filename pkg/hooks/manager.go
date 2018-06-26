package hooks

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/log"
)

type hookClient struct {
	client     interface{}
	version    string
	name       string
	hookPoints []*hooksInfo.HookPoint
}

var manager *Manager
var once sync.Once

type Manager struct {
	collected             bool
	callbacksPerHookPoint map[string][]*hookClient
}

func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{collected: false}
	})
	return manager
}

func (m *Manager) Collect(numberOfRequestedHookSidecars uint) error {
	callbacksPerHookPoint, err := collectSideCarSockets(numberOfRequestedHookSidecars)
	if err != nil {
		return err
	}
	log.Log.Info("Collected all requested hook sidecar sockets")

	sortCallbacksPerHookPoint(callbacksPerHookPoint)
	log.Log.Infof("Sorted all collected sidecar sockets per hook point based on their priority and name: %v", callbacksPerHookPoint)

	m.collected = true
	m.callbacksPerHookPoint = callbacksPerHookPoint

	return nil
}

func collectSideCarSockets(numberOfRequestedHookSidecars uint) (map[string][]*hookClient, error) {
	callbacksPerHookPoint := make(map[string][]*hookClient)
	processedSockets := make(map[string]bool)

	for uint(len(processedSockets)) < numberOfRequestedHookSidecars {
		sockets, err := ioutil.ReadDir(HookSocketsSharedDirectory)
		if err != nil {
			return nil, err
		}

		for _, socket := range sockets {
			if _, processed := processedSockets[socket.Name()]; processed {
				continue
			}

			hookClient, notReady, err := processSideCarSocket(HookSocketsSharedDirectory + "/" + socket.Name())
			if notReady {
				log.Log.Info("Sidecar server might not be ready yet, retrying in the next iteration")
				continue
			} else if err != nil {
				log.Log.Reason(err).Infof("Failed to process sidecar socket: %s", socket.Name())
				return nil, err
			}

			for _, hookPoint := range hookClient.hookPoints {
				callbacksPerHookPoint[hookPoint.GetName()] = append(callbacksPerHookPoint[hookPoint.GetName()], hookClient)
			}

			processedSockets[socket.Name()] = true
		}

		time.Sleep(time.Second)
	}

	return callbacksPerHookPoint, nil
}

func processSideCarSocket(socketPath string) (*hookClient, bool, error) {
	conn, err := grpc.Dial(
		socketPath,
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
	if err != nil {
		log.Log.Reason(err).Infof("Failed to Dial hook socket: %s", socketPath)
		return nil, true, nil
	}
	defer conn.Close()

	infoClient := hooksInfo.NewInfoClient(conn)
	info, err := infoClient.Info(context.Background(), &hooksInfo.InfoParams{})
	if err != nil {
		return nil, false, err
	}

	versionsSet := make(map[string]bool)
	for _, version := range info.GetVersions() {
		versionsSet[version] = true
	}

	if _, found := versionsSet[hooksV1alpha1.Version]; found {
		return &hookClient{
			client:     hooksV1alpha1.NewCallbacksClient(conn),
			name:       info.GetName(),
			version:    hooksV1alpha1.Version,
			hookPoints: info.GetHookPoints(),
		}, false, nil
	} else {
		return nil, false, fmt.Errorf("Hook sidecar does not expose a supported version. Exposed versions: %v, supported versions: %s", versionsSet, hooksV1alpha1.Version)
	}
}

func sortCallbacksPerHookPoint(callbacksPerHookPoint map[string][]*hookClient) {
	for _, callbacks := range callbacksPerHookPoint {
		for _, callback := range callbacks {
			sort.Slice(callbacks, func(i, j int) bool {
				if callback.hookPoints[i].Priority == callback.hookPoints[j].Priority {
					return strings.Compare(callback.hookPoints[i].Name, callback.hookPoints[j].Name) < 0
				} else {
					return callback.hookPoints[i].Priority > callback.hookPoints[j].Priority
				}
			})
		}
	}
}
