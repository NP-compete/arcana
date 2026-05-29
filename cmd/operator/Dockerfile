FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.work go.work
COPY cmd/operator/ cmd/operator/
COPY pkg/ pkg/
RUN cd cmd/operator && go build -o /arcana-operator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-operator /arcana-operator
EXPOSE 8082
ENTRYPOINT ["/arcana-operator"]
