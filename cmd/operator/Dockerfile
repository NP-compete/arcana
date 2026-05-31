FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY cmd/operator/go.mod cmd/operator/go.sum ./cmd/operator/
COPY pkg/common/go.mod ./pkg/common/
COPY pkg/logger/go.mod ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/
COPY pkg/server/go.mod pkg/server/go.sum ./pkg/server/
COPY pkg/tracing/go.mod pkg/tracing/go.sum ./pkg/tracing/
RUN cd cmd/operator && go mod download
COPY cmd/operator/ ./cmd/operator/
COPY pkg/common/ ./pkg/common/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/server/ ./pkg/server/
COPY pkg/tracing/ ./pkg/tracing/
RUN cd cmd/operator && CGO_ENABLED=0 GOFLAGS=-mod=mod go build -o /arcana-operator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-operator /arcana-operator
EXPOSE 8082
ENTRYPOINT ["/arcana-operator"]
