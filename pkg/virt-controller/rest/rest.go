package rest

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Handlers struct {
	RawDomainHandler http.Handler
	DeleteVMHandler  http.Handler
}

func DefineRoutes(handlers *Handlers) http.Handler {
	// TODO, routes and gokit http handlers are strongly tied together, initialize the endpoints and handlers
	// closer to the routing
	router := mux.NewRouter()
	restV1Route := router.PathPrefix("/api/v1/").Subrouter()
	restV1Route.Methods("POST").Path("/domain/raw").Headers("Content-Type", "application/xml").Handler(handlers.RawDomainHandler)
	restV1Route.Methods("DELETE").Path("/vm/{id:subdomain:[a-zA-Z0-9]+}").Handler(handlers.DeleteVMHandler)
	return router
}
