# Build the manager binary
FROM registry-proxy.engineering.redhat.com/rh-osbs/openshift-golang-builder:v1.23@sha256:ca0c771ecd4f606986253f747e2773fe2960a6b5e8e7a52f6a4797b173ac7f56 AS golang

FROM registry.redhat.io/ubi8/ubi@sha256:fd3bf22d0593e2ed26a1c74ce161c52295711a67de677b5938c87704237e49b0 AS builder
ARG GOLANG_VERSION=1.23.0

# Install system dependencies
RUN dnf upgrade -y && dnf install -y \
    gcc \
    make \
    openssl-devel \
    git \
    && dnf clean all && rm -rf /var/cache/yum

# Install Go
ENV PATH=/usr/local/go/bin:$PATH

COPY --from=golang /usr/lib/golang /usr/local/go
# End of Go versioning workaround

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
