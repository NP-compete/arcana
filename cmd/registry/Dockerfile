FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/registry/go.mod ./
RUN go mod download
COPY cmd/registry/ ./
RUN CGO_ENABLED=0 go build -o /arcana-registry .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-registry /arcana-registry
EXPOSE 8104
ENTRYPOINT ["/arcana-registry"]
