package cache

import (
	"io/ioutil"

	"github.com/fsnotify/fsnotify"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
)

// NewSocketListWatchFromClient creates a ListWatcher which watches for virt-launcher socket creations, recreations and deletions.
// It is a very special ListWatcher, since it can't be used to stay completely in sync with the file system content.
// Instead of that, it provides at-least-once delivery of events, where the order on an initial sync is not guaranteed.
// While for many tasks this is not good enough, it is a sufficient pattern to use the socket creation as a secondary resource for the VM controller in virt-handler
// TODO: In case Watch is never called, we could leak inotify go-routines, since it is not guaranteed that Stop() would ever be called
// Since the ListWatcher is only created once at start-up that is not an issue right now
func NewSocketListWatchFromClient(socketDir string) cache.ListerWatcher {
	d := &DirectoryListWatcher{socketDir: socketDir}
	return d
}

type DirectoryListWatcher struct {
	socketDir string
	watcher   *fsnotify.Watcher
}

func (d *DirectoryListWatcher) List(options v1.ListOptions) (runtime.Object, error) {
	// Stop the running watcher if necessary
	// This ensures we clean up previous watchers, when we encountered an error or when we resync
	d.Stop()
	var err error
	d.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// This starts the watch already.
	// Starting watching before the actual sync, has the advantage, that we don't mich notifications about file changes.
	// It also means that we can't reliably follow file system changes, because we are informed at least once about changes.
	err = d.watcher.Add(d.socketDir)
	if err != nil {
		return nil, err
	}
	files, err := ioutil.ReadDir(d.socketDir)
	if err != nil {
		d.Stop()
		return nil, err
	}

	domainList := &api.DomainList{
		Items: []api.Domain{},
	}
	for _, file := range files {
		namespace, name, err := isolation.SplitSocketNamespaceNameFunc(file.Name())
		if err != nil {
			logging.DefaultLogger().Error().Reason(err).Msg("Invalid content detected, ignoring and continuing.")
			continue
		}
		domainList.Items = append(domainList.Items, *api.NewMinimalDomainWithNS(namespace, name))

	}
	return domainList, nil
}
func (d *DirectoryListWatcher) Watch(options v1.ListOptions) (watch.Interface, error) {

	return d, nil
}

func (d *DirectoryListWatcher) Stop() {
	if d.watcher != nil {
		d.watcher.Close()
	}
}

func (d *DirectoryListWatcher) ResultChan() <-chan watch.Event {
	c := make(chan watch.Event)
	go func() {
		defer close(c)
		for {
			var e watch.EventType
			var fse fsnotify.Event
			select {
			case event, more := <-d.watcher.Events:
				if !more {
					return
				}
				fse = event
				switch event.Op {
				case fsnotify.Create:
					e = watch.Added
				case fsnotify.Remove:
					e = watch.Deleted
				}

			case err, more := <-d.watcher.Errors:
				if !more {
					return
				}
				c <- watch.Event{Type: watch.Error, Object: &v1.Status{Status: v1.StatusFailure, Message: err.Error()}}
				return
			}
			namespace, name, err := isolation.SplitSocketNamespaceNameFunc(fse.Name)
			if err != nil {
				logging.DefaultLogger().Error().Reason(err).Msg("Invalid content detected, ignoring and continuing.")
				continue
			}
			c <- watch.Event{Type: e, Object: api.NewMinimalDomainWithNS(namespace, name)}
		}
	}()
	return c
}
