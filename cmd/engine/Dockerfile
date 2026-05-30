FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/engine/go.mod ./
RUN go mod download
COPY cmd/engine/ ./
RUN CGO_ENABLED=0 go build -o /arcana-engine .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-engine /arcana-engine
EXPOSE 8081
ENTRYPOINT ["/arcana-engine"]
