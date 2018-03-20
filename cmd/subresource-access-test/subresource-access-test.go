package main

import (
	"flag"
	"fmt"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

func main() {
	var statusCode int
	flag.Parse()

	// creates the connection
	client, err := kubecli.GetKubevirtSubresourceClient()
	if err != nil {
		panic(err)
	}

	restClient := client.RestClient()

	result := restClient.Get().Resource("virtualmachines").Namespace("default").Name("fake").SubResource("test").Do()

	err = result.Error()
	if err != nil {
		panic(err)
	}

	result.StatusCode(&statusCode)
	if statusCode != 200 {
		panic(fmt.Errorf("http status code is %d", statusCode))
	} else {
		fmt.Println("Subresource Test Endpoint returned 200 OK")
	}
}
