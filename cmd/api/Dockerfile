FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/api/go.mod ./
RUN go mod download
COPY cmd/api/ ./
RUN CGO_ENABLED=0 go build -o /arcana-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-api /arcana-api
EXPOSE 8080
ENTRYPOINT ["/arcana-api"]
