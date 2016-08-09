export GO15VENDOREXPERIMENT := 1

all: sync fmt virt-launcher

virt-launcher: *.go
	go build .

clean:
	rm virt-launcher -f

fmt:
	go fmt .

test: virt-launcher
	go test .

run: fmt virt-launcher
	sudo ./virt-launcher --domain-path dom.xml

sync:
	govendor sync

.PHONY: clean test fmt sync
