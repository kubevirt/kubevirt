.PHONY: default build test

default: build

build: clean-proto proto

test: build

PROTO_GEN = lightsteppb/lightstep_carrier.pb.go collectorpb/collector.pb.go

.PHONY: proto clean-proto

clean-proto:
	@rm -f $(PROTO_GEN)

proto: $(PROTO_GEN)

collectorpb/collector.pb.go: collector.proto
	docker run --rm -v $(shell pwd):/input:ro -v $(shell pwd)/collectorpb:/output \
	  lightstep/grpc-gateway:latest \
		protoc -I/root/go/src/tmp/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis --go_out=plugins=grpc:/output --proto_path=/input /input/collector.proto

lightsteppb/lightstep_carrier.pb.go: lightstep_carrier.proto
	docker run --rm -v $(shell pwd):/input:ro -v $(shell pwd)/lightsteppb:/output \
	  lightstep/protoc:latest \
	  protoc --go_out=plugins=grpc:/output --proto_path=/input /input/lightstep_carrier.proto
