FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.work go.work
COPY cmd/api/ cmd/api/
COPY pkg/ pkg/
RUN cd cmd/api && go build -o /arcana-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-api /arcana-api
EXPOSE 8080
ENTRYPOINT ["/arcana-api"]
