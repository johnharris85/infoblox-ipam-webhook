FROM golang:1.13.8 as builder
WORKDIR /workspace

# Run this with docker build --build_arg goproxy=$(go env GOPROXY) to override the goproxy
ARG goproxy=https://proxy.golang.org
# Run this with docker build --build_arg package=./controlplane/kubeadm or --build_arg package=./bootstrap/kubeadm
ENV GOPROXY=$goproxy

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the sources
COPY ./ ./

# Build
ARG package=.

# Do not force rebuild of up-to-date packages (do not use -a)
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags '-extldflags "-static"' \
    -o webhook ${package}

# Production image
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/webhook .
USER nobody
ENTRYPOINT ["/webhook"]

