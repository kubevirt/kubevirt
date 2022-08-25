FROM registry.access.redhat.com/ubi8/ubi:8.6-903 as builder
RUN dnf install -y go
COPY . /kubevirt
WORKDIR /kubevirt
RUN go build -o /usr/bin/hook cmd/usb-disk-hook/usb-disk.go

FROM registry.access.redhat.com/ubi8/ubi:8.6-903
COPY --from=builder /usr/bin/hook /usr/bin/hook

