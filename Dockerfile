# Build the manager binary

FROM registry.access.redhat.com/ubi8/go-toolset:1.22@sha256:e66779cc9bef4b5d3361134fda8f054314eef5dc6b356086bae59e65805a57ad AS builder

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
