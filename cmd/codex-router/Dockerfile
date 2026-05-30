FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/codex-router/go.mod ./
RUN go mod download
COPY cmd/codex-router/ ./
RUN CGO_ENABLED=0 go build -o /codex-router .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-router /codex-router
EXPOSE 8090
ENTRYPOINT ["/codex-router"]
