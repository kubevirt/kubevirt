all: fmt virt-launcher

virt-launcher: *.go
	go build .

clean:
	rm virt-launcher -f

fmt:
	go fmt .

test: virt-launcher
	go test .


.PHONY: clean test fmt
