
FROM golang:1.13 AS builder
WORKDIR /gotest
COPY . .
RUN go get
RUN go build -o certtosecret

FROM centos:7
COPY --from=builder /gotest/certtosecret /usr/bin/certtosecret
CMD ["/usr/bin/certtosecret"]
