FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/codex-searcher/go.mod cmd/codex-searcher/go.sum ./cmd/codex-searcher/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/codex-searcher && go mod download
COPY cmd/codex-searcher/ ./cmd/codex-searcher/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/codex-searcher && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /codex-searcher .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-searcher /codex-searcher
EXPOSE 8091
ENTRYPOINT ["/codex-searcher"]
