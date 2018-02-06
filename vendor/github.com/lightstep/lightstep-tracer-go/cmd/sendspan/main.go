package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/lightstep/lightstep-tracer-go"
)

var (
	flagAccessToken = flag.String("access_token", "", "Access token to use when reporting spans")

	flagHost   = flag.String("collector_host", "", "Hostname of the collector to which reports should be sent")
	flagPort   = flag.Int("collector_port", 0, "Port of the collector to which reports should be sent")
	flagSecure = flag.Bool("secure", true, "Whether or not to use TLS")

	flagUseGRPC = flag.Bool("use_grpc", true, "Whether or not to use gRPC")

	flagOperation = flag.String("operation_name", "test-operation", "The operation to use for the test span")
)

func main() {
	flag.Parse()
	t := lightstep.NewTracer(lightstep.Options{
		AccessToken: *flagAccessToken,
		Collector: lightstep.Endpoint{
			Host:      *flagHost,
			Port:      *flagPort,
			Plaintext: !*flagSecure},
		UseThrift: !*flagUseGRPC})

	fmt.Println("Sending span...")
	span := t.StartSpan(*flagOperation)
	time.Sleep(100 * time.Millisecond)
	span.Finish()

	fmt.Println("Flushing tracer...")
	err := lightstep.FlushLightStepTracer(t)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
