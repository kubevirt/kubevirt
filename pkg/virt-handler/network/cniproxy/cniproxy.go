package cniproxy

import (
	"fmt"
	"sort"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"

	"strings"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/utils"
)

//TODO: Make it configurable
const (
	CNINetDir     = "/etc/cni/net.d"
	CNIPluginsDir = "/opt/cni/bin"
	LibvirtSocket = "/var/run/libvirt/libvirt-sock"
)

type pidFunc func(string) (int, error)
type confFileFunc func(string, []string) ([]string, error)
type confFromFileFunc func(filename string) (*libcni.NetworkConfig, error)
type getConfFunc func() (*libcni.NetworkConfig, error)

type CNIProxy struct {
	cniConfig   *libcni.CNIConfig
	netConfig   *libcni.NetworkConfig
	runtimeConf *libcni.RuntimeConf
}

func _getCNINetworkConfig(getF confFileFunc, loadF confFromFileFunc) (*libcni.NetworkConfig, error) {
	files, err := getF(CNINetDir, []string{".conf"})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	for _, confFile := range files {
		conf, err := loadF(confFile)
		if err != nil {
			log.Log.Reason(err).Warningf("Error loading CNI config file %s: %v", confFile, err)
			continue
		}
		return conf, nil
	}
	return nil, fmt.Errorf("No valid networks found in %s", CNINetDir)
}

func getCNINetworkConfig() (*libcni.NetworkConfig, error) {
	return _getCNINetworkConfig(libcni.ConfFiles, libcni.ConfFromFile)
}

func _getProxy(getConf getConfFunc, runtime *libcni.RuntimeConf) (*CNIProxy, error) {
	conf, err := getConf()
	if err != nil {
		return nil, err
	}

	cniconf := &libcni.CNIConfig{Path: []string{CNIPluginsDir}}
	cniProxy := &CNIProxy{netConfig: conf, cniConfig: cniconf, runtimeConf: runtime}
	return cniProxy, nil
}

func GetProxy(runtime *libcni.RuntimeConf) (*CNIProxy, error) {
	return _getProxy(getCNINetworkConfig, runtime)
}

func _getLibvirtNS(f pidFunc) (*utils.NSResult, error) {
	pid, err := f(LibvirtSocket)
	if err != nil {
		log.Log.Reason(err).Errorf("Cannot find libvirt socket in %s", LibvirtSocket)
		return nil, err
	}
	log.Log.Debugf("Got libvirt pid %d", pid)
	NS := utils.GetNSFromPid(pid)
	return NS, nil
}

func GetLibvirtNS() (*utils.NSResult, error) {
	return _getLibvirtNS(utils.GetPid)
}

func _buildRuntimeConfig(f pidFunc, ifname string) (*libcni.RuntimeConf, error) {
	libvNS, err := _getLibvirtNS(f)
	if err != nil {
		return nil, err
	}
	log.Log.Reason(err).Errorf("Got namespace path from libvirt pid: %s", libvNS.Net)
	randId := strings.Split(ifname, "-")
	if len(randId) != 2 {
		return nil, fmt.Errorf("invalid interface name: %s", ifname)
	}
	return &libcni.RuntimeConf{
		ContainerID: randId[1],
		NetNS:       libvNS.Net,
		IfName:      ifname,
	}, nil
}

func BuildRuntimeConfig(ifname string) (*libcni.RuntimeConf, error) {
	return _buildRuntimeConfig(utils.GetPid, ifname)
}

func (proxy *CNIProxy) AddToNetwork() (types.Result, error) {
	res, err := proxy.cniConfig.AddNetwork(proxy.netConfig, proxy.runtimeConf)
	if err != nil {
		log.Log.Reason(err).Errorf("Error creating an interface: %s", proxy.runtimeConf.IfName)
		return nil, err
	}

	return res, nil
}

func (proxy *CNIProxy) DeleteFromNetwork() error {
	err := proxy.cniConfig.DelNetwork(proxy.netConfig, proxy.runtimeConf)
	if err != nil {
		log.Log.Reason(err).Errorf("Error deleting an interface: %v", err)
		return err
	}
	return nil
}
