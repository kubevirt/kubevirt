// webapp: a standalone example Negroni / Gorilla based webapp.
//
// This example demonstrates basic usage of Appdash in a Negroni / Gorilla
// based web application. The entire application is ran locally (i.e. on the
// same server) -- even the Appdash web UI.
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"sourcegraph.com/sourcegraph/appdash"
	appdashtracer "sourcegraph.com/sourcegraph/appdash/opentracing"
	"sourcegraph.com/sourcegraph/appdash/traceapp"

	"github.com/urfave/negroni"
	"github.com/gorilla/mux"
	opentracing "github.com/opentracing/opentracing-go"
)

func main() {
	// Create a recent in-memory store, evicting data after 20s.
	//
	// The store defines where information about traces (i.e. spans and
	// annotations) will be stored during the lifetime of the application. This
	// application uses a MemoryStore store wrapped by a RecentStore with an
	// eviction time of 20s (i.e. all data after 20s is deleted from memory).
	memStore := appdash.NewMemoryStore()
	store := &appdash.RecentStore{
		MinEvictAge: 20 * time.Second,
		DeleteStore: memStore,
	}

	// Start the Appdash web UI on port 8700.
	//
	// This is the actual Appdash web UI -- usable as a Go package itself, We
	// embed it directly into our application such that visiting the web server
	// on HTTP port 8700 will bring us to the web UI, displaying information
	// about this specific web-server (another alternative would be to connect
	// to a centralized Appdash collection server).
	url, err := url.Parse("http://localhost:8700")
	if err != nil {
		log.Fatal(err)
	}
	tapp, err := traceapp.New(nil, url)
	if err != nil {
		log.Fatal(err)
	}
	tapp.Store = store
	tapp.Queryer = memStore
	log.Println("Appdash web UI running on HTTP :8700")
	go func() {
		log.Fatal(http.ListenAndServe(":8700", tapp))
	}()

	// We will use a local collector (as we are running the Appdash web UI
	// embedded within our app).
	//
	// A collector is responsible for collecting the information about traces
	// (i.e. spans and annotations) and placing them into a store. In this app
	// we use a local collector (we could also use a remote collector, sending
	// the information to a remote Appdash collection server).
	collector := appdash.NewLocalCollector(store)

	// Here we use the local collector to create a new opentracing.Tracer
	tracer := appdashtracer.NewTracer(collector)
	opentracing.InitGlobalTracer(tracer)

	// Setup our router (for information, see the gorilla/mux docs):
	router := mux.NewRouter()
	router.HandleFunc("/", Home)
	router.HandleFunc("/endpoint", Endpoint)

	// Setup Negroni for our app (for information, see the negroni docs):
	n := negroni.Classic()
	n.UseHandler(router)
	n.Run(":8699")
}

// Home is the homepage handler for our app.
func Home(w http.ResponseWriter, r *http.Request) {
	// Start a new root Span and therefore a new trace.
	span := opentracing.StartSpan(r.URL.Path)
	defer span.Finish()

	// OpenTracing allows for arbritary tags to be added to a Span.
	span.SetTag("Request.Host", r.Host)
	span.SetTag("Request.Address", r.RemoteAddr)
	addHeaderTags(span, r.Header)

	// Baggage Items are similar to tags, however they are propagated to all
	// children spans, so this will show up in the API calls.
	span.SetBaggageItem("User", os.Getenv("USER"))

	// We're going to make some API request, so we use the default HTTP client
	// to send HTTP requests with trace information placed inside the headers.
	httpClient := http.DefaultClient

	// Make three API requests using our HTTP client.
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("GET", "http://localhost:8699/endpoint", nil)
		if err != nil {
			log.Println("/endpoint:", err)
			continue
		}

		// We inject the span into the request headers before making the request.
		carrier := opentracing.HTTPHeadersCarrier(req.Header)
		span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, carrier)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println("/endpoint:", err)

			// Log the error to the span.
			span.LogEvent(err.Error())
			continue
		}

		span.SetTag("Response.Status", resp.Status)
		resp.Body.Close()
	}

	// Render the page.
	fmt.Fprintf(w, `<p>Three API requests have been made!</p>`)
	fmt.Fprintf(w, `<p><a href="http://localhost:8700/traces" target="_">View the trace</a></p>`)
}

// Endpoint is an example API endpoint. In a real application, the backend of
// your service would be contacting several external and internal API endpoints
// which may be the bottleneck of your application.
//
// For example purposes we just sleep for 200ms before responding to simulate a
// slow API endpoint as the bottleneck of your application.
func Endpoint(w http.ResponseWriter, r *http.Request) {
	// Extract the trace from the headers and join it with a new child span.
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	if err != nil {
		return
	}
	span := opentracing.StartSpan(r.URL.Path, opentracing.ChildOf(spanCtx))
	defer span.Finish()

	span.SetTag("Request.Host", r.Host)
	span.SetTag("Request.Method", r.Method)
	addHeaderTags(span, r.Header)

	time.Sleep(200 * time.Millisecond)
	fmt.Fprintf(w, "Slept for 200ms!")
}

const headerTagPrefix = "Request.Header."

// addHeaderTags adds header key:value pairs to a span as a tag with the prefix
// "Request.Header.*"
func addHeaderTags(span opentracing.Span, h http.Header) {
	for k, v := range h {
		span.SetTag(headerTagPrefix+k, strings.Join(v, ", "))
	}
}
