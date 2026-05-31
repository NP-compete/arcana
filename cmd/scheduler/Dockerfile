FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/scheduler/go.mod cmd/scheduler/go.sum ./cmd/scheduler/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/db/go.mod pkg/db/go.sum ./pkg/db/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/scheduler && go mod download
COPY cmd/scheduler/ ./cmd/scheduler/
COPY pkg/common/ ./pkg/common/
COPY pkg/db/ ./pkg/db/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/scheduler && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-scheduler .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-scheduler /arcana-scheduler
EXPOSE 8103
ENTRYPOINT ["/arcana-scheduler"]
