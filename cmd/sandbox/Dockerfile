FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/sandbox/go.mod cmd/sandbox/go.sum ./
RUN go mod download
COPY cmd/sandbox/ ./
RUN CGO_ENABLED=0 go build -o /arcana-sandbox .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-sandbox /arcana-sandbox
EXPOSE 8097
ENTRYPOINT ["/arcana-sandbox"]
