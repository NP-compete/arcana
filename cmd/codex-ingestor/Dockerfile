FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/codex-ingestor/go.mod cmd/codex-ingestor/go.sum ./cmd/codex-ingestor/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/codex-ingestor && go mod download
COPY cmd/codex-ingestor/ ./cmd/codex-ingestor/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/codex-ingestor && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /codex-ingestor .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-ingestor /codex-ingestor
EXPOSE 8092
ENTRYPOINT ["/codex-ingestor"]
