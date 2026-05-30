FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/finops/go.mod cmd/finops/go.sum ./
RUN go mod download
COPY cmd/finops/ ./
RUN CGO_ENABLED=0 go build -o /arcana-finops .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-finops /arcana-finops
EXPOSE 8105
ENTRYPOINT ["/arcana-finops"]
