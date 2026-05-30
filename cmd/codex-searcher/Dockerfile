FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/codex-searcher/go.mod ./
RUN go mod download
COPY cmd/codex-searcher/ ./
RUN CGO_ENABLED=0 go build -o /codex-searcher .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /codex-searcher /codex-searcher
EXPOSE 8091
ENTRYPOINT ["/codex-searcher"]
