package pxe

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

const (
	DefaultConfig = `
# Use the simple DefaultConfig
UI DefaultConfig.c32

# Time out and use the default DefaultConfig option. Defined as tenths of a second.
TIMEOUT 600 

# Prompt the user. Set to '1' to automatically choose the default option.
PROMPT 0

# Set the boot DefaultConfig to be 1024x768.
MENU RESOLUTION 1024 768

# These do not need to be set. I set them here to show how you can customize or
# localize your PXE server's dialogue.
MENU TITLE    PXE Boot Server

### Now define the DefaultConfig options
LABEL next
	MENU LABEL Boot local 
	MENU DEFAULT
	localboot

LABEL hdt
MENU LABEL Run Hardware Detection Tool
COM32 /hdt.c32

LABEL reboot
MENU LABEL Reboot
COM32 /reboot.c32

LABEL poweroff
MENU LABEL Power Off
COM32 /poweroff.c32
`
)

const (
	ConfigDir            = "/pxelinux.cfg/"
	DefaultSYSLINUXDir   = "/usr/share/syslinux"
	DefaultImageDir      = "/images"
	ContentTypeTextPlain = "text/plain"
)

type PXE struct {
	PXEInformer cache.SharedIndexInformer
	Namespace   string
	Cli         kubecli.KubevirtClient
}

func (p *PXE) SYSLINUXConfigServer(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	namespace := req.PathParameter("namespace")

	vmi, err := p.Cli.VirtualMachineInstance(namespace).Get(name, &v1.GetOptions{})
	if errors.IsNotFound(err) {
		resp.WriteErrorString(http.StatusNotFound, "no configuration for this VMI present")
		return
	} else if err != nil {
		resp.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}

	pxes, err := p.PXEInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		resp.WriteErrorString(http.StatusInternalServerError, "indexer error")
		return
	}

	config := ""
	configForNamespace := DefaultConfig

	for _, obj := range pxes {
		pxe := obj.(*v12.PXE)
		for _, m := range pxe.Configs {
			selector, err := v1.LabelSelectorAsSelector(m.Selector)
			if err != nil {
				resp.WriteErrorString(http.StatusInternalServerError, "cache error")
				return
			}

			if selector == labels.Everything() {
				configForNamespace = m.Content
				continue
			}

			if selector.Matches(labels.Set(vmi.Labels)) {
				config = m.Content
			}
		}
	}

	if config == "" {
		config = configForNamespace
	}

	resp.Header().Set(restful.HEADER_ContentType, ContentTypeTextPlain)
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(config))
	if err != nil {
		fmt.Println(err)
	}
}

func (*PXE) SYSLINUXServer(req *restful.Request, resp *restful.Response) {
	file := req.PathParameter("filepath")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(DefaultSYSLINUXDir, file))
}

func (*PXE) ImageServer(req *restful.Request, resp *restful.Response) {
	file := req.PathParameter("filepath")
	fmt.Println(file)
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(DefaultImageDir, file))
}
