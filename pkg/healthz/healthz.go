package healthz

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"k8s.io/client-go/pkg/util/json"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"net/http"
)

func KubeConnectionHealthzFunc(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}
	cli, err := kubecli.Get()
	if err != nil {
		unhealthy(err, response)
		return
	}

	body, err := cli.Core().RESTClient().Get().AbsPath("/version").Do().Raw()
	if err != nil {
		unhealthy(err, response)
		return
	}
	var version interface{}
	err = json.Unmarshal(body, &version)
	if err != nil {
		unhealthy(err, response)
		return
	}
	res["apiserver"] = map[string]interface{}{"connectivity": "ok", "version": version}
	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
	return
}

func unhealthy(err error, response *restful.Response) {
	res := map[string]interface{}{}
	res["apiserver"] = map[string]interface{}{"connectivity": "failed", "error": fmt.Sprintf("%v", err)}
	response.WriteHeaderAndJson(http.StatusInternalServerError, res, restful.MIME_JSON)
}
