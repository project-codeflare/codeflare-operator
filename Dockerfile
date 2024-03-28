# Build the manager binary
#FROM registry.access.redhat.com/ubi8/go-toolset:1.20.10 as builder

# BEGIN -- workaround lack of go-toolset for golang 1.21

# FROM registry-proxy.engineering.redhat.com/rh-osbs/openshift-golang-builder:v1.21 AS golang
FROM golang:1.21 AS golang
FROM registry.access.redhat.com/ubi8/ubi:8.8 AS builder

ARG GOLANG_VERSION=1.21.6

# Install system dependencies
RUN dnf upgrade -y && dnf install -y \
    gcc \
    make \
    openssl-devel \
    && dnf clean all && rm -rf /var/cache/yum

# Install Go
ENV PATH=/usr/local/go/bin:$PATH
COPY --from=golang /usr/local/go /usr/local/go

# END -- workaround lack of go-toolset for golang 1.21

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the Go sources
COPY main.go main.go
COPY pkg/ pkg/

# Build
USER root
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags strictfipsruntime -a -o manager main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
WORKDIR /
COPY --from=builder /workspace/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
