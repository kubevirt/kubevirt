package cache_test

import (
	"io/ioutil"
	"sync"
	"testing"

	"kubevirt.io/kubevirt/pkg/network/cache"

	kfs "kubevirt.io/kubevirt/pkg/os/fs"

	"kubevirt.io/client-go/testutils"
)

func TestCache(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

type tempCacheCreator struct {
	once   sync.Once
	tmpDir string
}

func (c *tempCacheCreator) New(filePath string) *cache.Cache {
	c.once.Do(func() {
		tmpDir, err := ioutil.TempDir("", "temp-cache")
		if err != nil {
			panic("Unable to create temp cache directory")
		}
		c.tmpDir = tmpDir
	})
	return cache.NewCustomCache(filePath, kfs.NewWithRootPath(c.tmpDir))
}
