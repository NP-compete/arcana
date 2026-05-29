FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.work go.work
COPY cmd/engine/ cmd/engine/
COPY pkg/ pkg/
RUN cd cmd/engine && go build -o /arcana-engine .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-engine /arcana-engine
EXPOSE 8081
ENTRYPOINT ["/arcana-engine"]
