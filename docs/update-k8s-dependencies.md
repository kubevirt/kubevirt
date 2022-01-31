# Updating Dependencies

## Updating k8s dependencies

To correctly update k8s dependencies we have to first bump all replace directives of `k8s.io/*` in
* [go.mod](../go.mod)
* [staging/client-go/go.mod](../staging/src/kubevirt.io/client-go/go.mod)
* [staging/src/kubevirt.io/api/go.mod](../staging/src/kubevirt.io/api/go.mod)

Then (if necessary) delete generated code that potentially references deprecated or eliminated stuff (like the mock client) and, on cascade, fix the files[1] that include these (remember which files are affected because at the end you will have to revert the changes you made to restore them).  
Delete generated code is not a problem because it will be regenerated.  
Run `make deps-update` to update dependencies.  
Run `make && make generate` to regenerate the code.  
Don't forget to restore the files edited in [1].  