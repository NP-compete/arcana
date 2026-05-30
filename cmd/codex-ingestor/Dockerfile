FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/codex-ingestor/go.mod ./
RUN go mod download
COPY cmd/codex-ingestor/ ./
RUN CGO_ENABLED=0 go build -o /codex-ingestor .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-ingestor /codex-ingestor
EXPOSE 8092
ENTRYPOINT ["/codex-ingestor"]
