FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/audit/go.mod cmd/audit/go.sum ./cmd/audit/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/db/go.mod pkg/db/go.sum ./pkg/db/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/audit && go mod download
COPY cmd/audit/ ./cmd/audit/
COPY pkg/common/ ./pkg/common/
COPY pkg/db/ ./pkg/db/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/audit && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-audit .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-audit /arcana-audit
EXPOSE 8100
ENTRYPOINT ["/arcana-audit"]
