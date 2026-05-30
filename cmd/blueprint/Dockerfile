FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/blueprint/go.mod cmd/blueprint/go.sum ./
RUN go mod download
COPY cmd/blueprint/ ./
RUN CGO_ENABLED=0 go build -o /arcana-blueprint .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-blueprint /arcana-blueprint
EXPOSE 8088
ENTRYPOINT ["/arcana-blueprint"]
