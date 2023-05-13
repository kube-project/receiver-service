# Build the manager binary
FROM golang:1.20 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY models/ models/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/receiver cmd/root.go

FROM alpine
LABEL Author="Gergely Brautigam"
RUN apk add -u ca-certificates
WORKDIR /app/
COPY --from=builder /workspace/bin/receiver /app/receiver

EXPOSE 8000

ENTRYPOINT [ "/app/receiver" ]
