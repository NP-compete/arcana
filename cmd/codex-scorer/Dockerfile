FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/codex-scorer/go.mod ./
RUN go mod download
COPY cmd/codex-scorer/ ./
RUN CGO_ENABLED=0 go build -o /codex-scorer .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-scorer /codex-scorer
EXPOSE 8093
ENTRYPOINT ["/codex-scorer"]
