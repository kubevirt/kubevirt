package main

import (
	"path/filepath"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/pxe"

	"log"
	"net/http"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// GET http://localhost:8080/hello

func main() {
	ws := new(restful.WebService)
	ws.Route(ws.GET(filepath.Join("/{namespace}/{name}/", pxe.ConfigDir, "{config}")).To(pxe.SYSLINUXConfigServer))
	ws.Route(ws.GET("/{namespace}/{name}/{filepath:*}").To(pxe.SYSLINUXServer))
	restful.Add(ws)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
