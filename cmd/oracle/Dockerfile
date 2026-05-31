FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/oracle/go.mod cmd/oracle/go.sum ./cmd/oracle/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/oracle && go mod download
COPY cmd/oracle/ ./cmd/oracle/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/oracle && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-oracle .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-oracle /arcana-oracle
EXPOSE 8089
ENTRYPOINT ["/arcana-oracle"]
