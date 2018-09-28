package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"log"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/pxe"
	"kubevirt.io/kubevirt/pkg/util"
)

func main() {

	var vmiMonitorNamespace string
	var serverNamespace string
	var global bool
	var err error

	flag.StringVar(&serverNamespace, "n", v1.NamespaceAll, "Namespace of the PXE server")
	flag.BoolVar(&global, "a", false, "Answer to requests from outside of the server namespace")

	flag.Parse()

	if serverNamespace == v1.NamespaceAll {
		serverNamespace, err = util.GetNamespace()
		if err != nil {
			panic(err)
		}
	}

	if !global {
		vmiMonitorNamespace = serverNamespace
	} else {
		vmiMonitorNamespace = v1.NamespaceAll
	}

	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	informers := controller.NewKubeInformerFactory(cli.RestClient(), cli)
	p := &pxe.PXE{
		Namespace:   vmiMonitorNamespace,
		PXEInformer: informers.PXE(),
		Cli:         cli,
	}

	stop := make(chan struct{})
	defer close(stop)
	informers.Start(stop)

	cache.WaitForCacheSync(stop, informers.PXE().HasSynced)

	ws := new(restful.WebService)
	ws.Route(ws.GET(filepath.Join("/{namespace}/{name}/", pxe.ConfigDir, "{config}")).To(p.SYSLINUXConfigServer))
	ws.Route(ws.GET("/{namespace}/{name}/images/{filepath:*}").To(p.ImageServer))
	ws.Route(ws.GET("/{namespace}/{name}/{filepath:*}").To(p.SYSLINUXServer))
	ws.Route(ws.GET("/").To(func(request *restful.Request, response *restful.Response) {
		fmt.Println("no match")
		fmt.Println(request.Request.URL.Path)
	}))
	restful.Add(ws)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
