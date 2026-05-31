FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/codex-scorer/go.mod cmd/codex-scorer/go.sum ./cmd/codex-scorer/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/codex-scorer && go mod download
COPY cmd/codex-scorer/ ./cmd/codex-scorer/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/codex-scorer && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /codex-scorer .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-scorer /codex-scorer
EXPOSE 8093
ENTRYPOINT ["/codex-scorer"]
