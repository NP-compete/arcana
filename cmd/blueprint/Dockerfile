FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/blueprint/go.mod cmd/blueprint/go.sum ./cmd/blueprint/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/db/go.mod pkg/db/go.sum ./pkg/db/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/blueprint && go mod download
COPY cmd/blueprint/ ./cmd/blueprint/
COPY pkg/common/ ./pkg/common/
COPY pkg/db/ ./pkg/db/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/blueprint && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-blueprint .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-blueprint /arcana-blueprint
EXPOSE 8088
ENTRYPOINT ["/arcana-blueprint"]
