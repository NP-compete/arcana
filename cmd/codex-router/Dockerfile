FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/codex-router/go.mod cmd/codex-router/go.sum ./cmd/codex-router/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/codex-router && go mod download
COPY cmd/codex-router/ ./cmd/codex-router/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/codex-router && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /codex-router .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-router /codex-router
EXPOSE 8090
ENTRYPOINT ["/codex-router"]
