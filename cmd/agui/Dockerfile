FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/agui/go.mod ./
RUN go mod download
COPY cmd/agui/ ./
RUN CGO_ENABLED=0 go build -o /arcana-agui .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-agui /arcana-agui
EXPOSE 8084
ENTRYPOINT ["/arcana-agui"]
