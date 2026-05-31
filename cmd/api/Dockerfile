FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/api/go.mod cmd/api/go.sum ./cmd/api/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/db/go.mod pkg/db/go.sum ./pkg/db/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/api && go mod download
COPY cmd/api/ ./cmd/api/
COPY pkg/common/ ./pkg/common/
COPY pkg/db/ ./pkg/db/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/api && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-api /arcana-api
EXPOSE 8080
ENTRYPOINT ["/arcana-api"]
