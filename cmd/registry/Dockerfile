FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/registry/go.mod cmd/registry/go.sum ./cmd/registry/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/db/go.mod pkg/db/go.sum ./pkg/db/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/registry && go mod download
COPY cmd/registry/ ./cmd/registry/
COPY pkg/common/ ./pkg/common/
COPY pkg/db/ ./pkg/db/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/registry && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-registry .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-registry /arcana-registry
EXPOSE 8104
ENTRYPOINT ["/arcana-registry"]
