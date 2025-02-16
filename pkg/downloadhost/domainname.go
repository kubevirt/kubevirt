package downloadhost

import (
	"sync"

	configv1 "github.com/openshift/api/config/v1"
)

const (
	CLIDownloadsServiceName = "hyperconverged-cluster-cli-download"
)

type CLIDownloadHost struct {
	DefaultHost configv1.Hostname
	CurrentHost configv1.Hostname
	Cert        string
	Key         string
}

var (
	cliDownloadHost CLIDownloadHost
	lock            = sync.RWMutex{}
)

func Set(info CLIDownloadHost) bool {
	lock.Lock()
	defer lock.Unlock()

	changed := false
	if cliDownloadHost != info {
		changed = true
		cliDownloadHost = info
	}

	return changed
}

func Get() CLIDownloadHost {
	lock.RLock()
	defer lock.RUnlock()

	return cliDownloadHost
}
