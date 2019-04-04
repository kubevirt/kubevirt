# go-appr
This package is a collection of `go-swagger` generated client bindings for `App Registry` which can be used to talk to `quay.io` or [appr](https://github.com/app-registry/appr) to pull down application packages.

Swagger spec file has been obtained from here -  [appr-api-swagger.yaml](https://github.com/app-registry/appr/blob/master/Documentation/server/appr-api-swagger.yaml)

### Example
Below is an example of how you can use the client bindings to connect to an `app registry` server and invoke its APIs. This example assumes that an instance of `appr` is running on your machine (`localhost:5000`). For more information on how to run `appr` server on local environment visit https://github.com/app-registry/appr

```go
package main

import (
	"log"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	apprclient "github.com/operator-framework/go-appr/appregistry"
	"github.com/operator-framework/go-appr/appregistry/package_appr"

	models "github.com/operator-framework/go-appr/models"
)

func main() {
	client := apprclient.New(httptransport.New("localhost:5000", "/cnr", []string{"http"}), strfmt.Default)

	packages, err := listPackages(client)
	if err != nil {
		log.Fatalf("error - %v", err)
	}

	log.Printf("success - found [%d] package(s)\n", len(packages))
}

func listPackages(client *apprclient.Appregistry) (models.Packages, error) {
	params := package_appr.NewListPackagesParams()

	packages, err := client.PackageAppr.ListPackages(params)
	if err != nil {
		return nil, err
	}

	return packages.Payload, nil
}
```

## Generate Client Bindings
```
This section is applicable if you want to regenerate the client bindings.
```

[go-swagger](https://github.com/go-swagger/go-swagger) is a prerequisite before you can generate client bindings. 

First, install `go-swagger`.
```bash
go get -u github.com/go-swagger/go-swagger/cmd/swagger
```

Next, run `go-swagger` to generate client bindings, as shown below.
```bash
# change directory to the root of the go-appr repo.
swagger generate client --spec=./appr.spec.yaml --name=appregistry --api-package=appr --client-package=appregistry
```