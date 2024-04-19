# Build the manager binary

# BEGIN -- workaround lack of go-toolset for golang 1.21

ARG GOLANG_IMAGE=golang:1.21

ARG GOARCH=amd64

FROM ${GOLANG_IMAGE} AS golang
FROM registry.access.redhat.com/ubi8/ubi:8.8 AS builder

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
COPY ./Makefile ./Makefile
RUN go mod download

# Copy the Go sources
COPY main.go main.go
COPY pkg/ pkg/

# Build
USER root
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${GOARCH} make go-build-for-image

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
WORKDIR /
COPY --from=builder /workspace/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
