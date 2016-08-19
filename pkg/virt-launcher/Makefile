export GO15VENDOREXPERIMENT := 1

all: build
   
build: sync fmt virt-launcher

virt-launcher: *.go
	go build .

clean:
	rm virt-launcher -f

distclean: clean
	find vendor/ -maxdepth 1 -mindepth 1 -not -name vendor.json -exec rm {} -rf \;
	docker rmi kubevirt/virt-launcher || true

fmt:
	go fmt .

test: build
	go test .

run: build
	sudo ./virt-launcher --domain-path dom.xml

sync:
	govendor sync

docker: build
	docker build -t kubevirt/virt-launcher .

.PHONY: clean test fmt sync docker
