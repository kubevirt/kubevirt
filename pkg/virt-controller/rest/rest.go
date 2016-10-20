package rest

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Handlers struct {
	RawDomainHandler         http.Handler
	DeleteVMHandler          http.Handler
	PrepareMigrationHandler  http.Handler
	FinalizeMigrationHandler http.Handler
}

func DefineRoutes(handlers *Handlers) http.Handler {
	// TODO, routes and gokit http handlers are strongly tied together, initialize the endpoints and handlers
	// closer to the routing
	router := mux.NewRouter()
	restV1Route := router.PathPrefix("/api/v1/").Subrouter()
	restV1Route.Methods("POST").Path("/domain/raw").Headers("Content-Type", "application/xml").Handler(handlers.RawDomainHandler)
	restV1Route.Methods("DELETE").Path("/vm/{name:[a-z0-9-.]+}").Handler(handlers.DeleteVMHandler)

	jsonPostRouter := restV1Route.Methods("Post").Headers("Content-Type", "application/json").Subrouter()
	jsonPostRouter.Path("/domain/{name:[a-z0-9-.]+}/migration/prepare").Handler(handlers.PrepareMigrationHandler)
	jsonPostRouter.Path("/domain/{name:[a-z0-9-.]+}/migration/finalize").HandlerFunc(func(r http.ResponseWriter, _ *http.Request) {
		// Currently not needed
		r.WriteHeader(http.StatusCreated)
		r.Write(nil)
	})
	return router
}
