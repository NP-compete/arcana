FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/sandbox/go.mod cmd/sandbox/go.sum ./cmd/sandbox/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/sandbox && go mod download
COPY cmd/sandbox/ ./cmd/sandbox/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/sandbox && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-sandbox .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-sandbox /arcana-sandbox
EXPOSE 8097
ENTRYPOINT ["/arcana-sandbox"]
