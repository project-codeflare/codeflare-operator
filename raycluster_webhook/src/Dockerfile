FROM registry.access.redhat.com/ubi8/go-toolset:1.20.10 as builder

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . /app/

USER root
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o webhook .

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
COPY --from=builder /app/webhook /usr/local/bin/webhook

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/webhook

ENTRYPOINT ["webhook"]
