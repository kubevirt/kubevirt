package rest

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Handlers struct {
	RawDomainHandler http.Handler
}

func DefineRoutes(handlers *Handlers) http.Handler {
	router := mux.NewRouter()
	restV1Route := router.PathPrefix("/api/v1/").Subrouter()
	restV1Route.Methods("POST").Path("/domain/raw").Headers("Content-Type", "application/xml").Handler(handlers.RawDomainHandler)
	return router
}
