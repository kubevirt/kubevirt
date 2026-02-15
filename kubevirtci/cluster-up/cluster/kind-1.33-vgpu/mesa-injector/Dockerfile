# Copyright 2024 The KubeVirt Authors.
# Licensed under the Apache License, Version 2.0.

FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY main.go .

# Initialize module and download dependencies with pinned versions
# Using k8s.io v0.29.x which is compatible with Go 1.22
RUN go mod init mesa-injector && \
    go mod edit -require k8s.io/api@v0.29.0 && \
    go mod edit -require k8s.io/apimachinery@v0.29.0 && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o mesa-injector .

FROM alpine:3.19

COPY --from=builder /app/mesa-injector /mesa-injector

ENTRYPOINT ["/mesa-injector"]
