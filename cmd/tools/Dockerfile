FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/tools/go.mod cmd/tools/go.sum ./
RUN go mod download
COPY cmd/tools/ ./
RUN CGO_ENABLED=0 go build -o /arcana-tools .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-tools /arcana-tools
EXPOSE 8096
ENTRYPOINT ["/arcana-tools"]
