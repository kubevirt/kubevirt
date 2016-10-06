# Project structure

```bash
cmd # all the app wiring and app specific data is here, but no logic
cmd/appA # first app
cmd/appA/README.md # info about function, deplying, testing, debugging, ...
cmd/appA/Dockerfile # to build a container
cmd/appA/appA.go # wire the app (all logic in pkg/appA)
cmd/appB # second app
cmd/appB/mydata #  for packaging data with docker, put it the Dockerfile context
cmd/appB/mydata/sampleyaml.yaml
cmd/appB/appB.go # wire the app (all logic in pkg/apB)
cmd/appB/Dockerfile # to build a container
pkg # generally useful packages and app packages with logic are here
pkg/entities # entities for microservice internal use and microservice internal communication
pkg/entities/my_entity.go
pkg/middleware # go-kit middlewares can be stored here
pkg/middleware/my_middleware.go
pkg/appB # logic for appB
pkg/appB/rest # do the rest wiring
pkg/appB/services # app services (some bound to endpoints, some just used internally)
pkg/appB/endpoints # DTOs, encoding, decoding, closing the gap between external communication (e.g. REST and service calls)
pkg/stuff # this should be useful for all services
pkg/other_stuff # this should be useful for all services
```

# Microservice structure

## Communicating with the external world

| abstract layer           | internal location | task                                              | error codes     | direction |
|--------------------------|-------------------|---------------------------------------------------|-----------------|-----------|
| REST LAYER               | mux (gorilla)     | routing                                           | 500, 404, ...   | request   |
| REST LAYER               | TODO              | authentication                                    | 401             | request   |
| REST LAYER               | decode (gokit)    | unmarshal json/xml to RequestDTO                  | 400             | request   |
| REST LAYER               | decode (gokit)    | syntactic validation of json/xml (govalidator)    | 400             | request   |
| TRANSPORT AGNOSTIK LAYER | endpoint (gokit)  | RequestDTO to entity mapping                      | 500             | request   |
| TRANSPORT AGNOSTIK LAYER | endpoint (gokit)  | service parameter validation, precondition checks | 400, 404, 403   | request   |
| SERVICE LAYER            | internal service  | service parameter validation (precond package)    | 500             | request   |
| SERVICE LAYER            | internal service  | execution                                         | 500, ... (TODO) | request   |
| TRANSPORT AGNOSTIK LAYER | endpoint (gokit)  | execution result examination                      | 500, ... (TODO) | response  |
| TRANSPORT AGNOSTIK LAYER | endpoint (gokit)  | result to ResponseDTO mapping                     | 500             | response  |
| REST LAYER               | encode (gokit)    | marshal json/xml to ResponseDTO                   | 500             | response  |

## Microservice internal communication

TODO, but to describe the basic idea: There will be core entities which can be stored in etcd. These core entities belong to virt-controller.
All other componenets can ask virt-controller via it's REST api for data from etcd. virt-controller then provides a DTO representation. For instance a call for fetching all VMs would have the following flow:

 1) REST GET /get/me/my/vms
 2) virt-controller loads the VMs from etcd
 3) virt-cntroller translates the VM into a v1.VM DTO representation
 4) virt-controller returns this v1.VM representation to the caller
